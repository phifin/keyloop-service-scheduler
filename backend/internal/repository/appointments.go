package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"keyloop-service-scheduler/internal/scheduling"
)

func (r *PostgresReferenceRepository) CreateAppointment(ctx context.Context, request scheduling.CreateAppointmentRequest) (appointment scheduling.Appointment, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return scheduling.Appointment{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1::text))`, request.DealershipID); err != nil {
		return scheduling.Appointment{}, err
	}

	if err = verifyCustomerExists(ctx, tx, request.CustomerID); err != nil {
		return scheduling.Appointment{}, err
	}

	if err = verifyVehicleBelongsToCustomer(ctx, tx, request.VehicleID, request.CustomerID); err != nil {
		return scheduling.Appointment{}, err
	}

	if err = verifyDealershipExists(ctx, tx, request.DealershipID); err != nil {
		return scheduling.Appointment{}, err
	}

	serviceType, err := getServiceTypeForBooking(ctx, tx, request.ServiceTypeID)
	if err != nil {
		return scheduling.Appointment{}, err
	}

	endTime := request.StartTime.Add(time.Duration(serviceType.DurationMinutes) * time.Minute)

	technicianID, err := chooseAvailableTechnician(ctx, tx, request.DealershipID, serviceType.RequiredSkillCode, request.StartTime, endTime)
	if err != nil {
		return scheduling.Appointment{}, err
	}

	serviceBayID, err := chooseAvailableServiceBay(ctx, tx, request.DealershipID, request.StartTime, endTime)
	if err != nil {
		return scheduling.Appointment{}, err
	}

	appointment, err = insertAppointment(ctx, tx, request, technicianID, serviceBayID, endTime)
	if err != nil {
		return scheduling.Appointment{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return scheduling.Appointment{}, err
	}

	return appointment, nil
}

func (r *PostgresReferenceRepository) ListAppointments(ctx context.Context, filters scheduling.AppointmentFilters) ([]scheduling.AppointmentListItem, error) {
	args := []any{}
	where := "WHERE TRUE"
	if filters.DealershipID != nil {
		args = append(args, *filters.DealershipID)
		where += fmt.Sprintf(" AND a.dealership_id = $%d", len(args))
	}
	if filters.Status != nil {
		args = append(args, *filters.Status)
		where += fmt.Sprintf(" AND a.status = $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			a.id::text,
			c.name AS customer_name,
			v.vin AS vehicle_vin,
			concat(v.make, ' ', v.model) AS vehicle_label,
			d.name AS dealership_name,
			st.name AS service_type_name,
			t.name AS technician_name,
			sb.name AS service_bay_name,
			a.start_time,
			a.end_time,
			a.status
		FROM appointments a
		INNER JOIN customers c ON c.id = a.customer_id
		INNER JOIN vehicles v ON v.id = a.vehicle_id
		INNER JOIN dealerships d ON d.id = a.dealership_id
		INNER JOIN service_types st ON st.id = a.service_type_id
		INNER JOIN technicians t ON t.id = a.technician_id
		INNER JOIN service_bays sb ON sb.id = a.service_bay_id
		`+where+`
		ORDER BY a.start_time DESC, a.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appointments := []scheduling.AppointmentListItem{}
	for rows.Next() {
		var appointment scheduling.AppointmentListItem
		if err := rows.Scan(
			&appointment.ID,
			&appointment.CustomerName,
			&appointment.VehicleVIN,
			&appointment.VehicleLabel,
			&appointment.DealershipName,
			&appointment.ServiceTypeName,
			&appointment.TechnicianName,
			&appointment.ServiceBayName,
			&appointment.StartTime,
			&appointment.EndTime,
			&appointment.Status,
		); err != nil {
			return nil, err
		}
		appointment.StartTime = appointment.StartTime.UTC()
		appointment.EndTime = appointment.EndTime.UTC()
		appointments = append(appointments, appointment)
	}

	return appointments, rows.Err()
}

func (r *PostgresReferenceRepository) GetAppointmentDetail(ctx context.Context, appointmentID string) (scheduling.AppointmentDetail, error) {
	return getAppointmentDetail(ctx, r.pool, appointmentID)
}

func (r *PostgresReferenceRepository) CancelAppointment(ctx context.Context, appointmentID string) (scheduling.Appointment, error) {
	var status string
	if err := r.pool.QueryRow(ctx, `
		SELECT status
		FROM appointments
		WHERE id = $1
	`, appointmentID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return scheduling.Appointment{}, scheduling.ErrNotFound
		}
		return scheduling.Appointment{}, err
	}

	switch status {
	case scheduling.StatusCancelled:
		return getAppointment(ctx, r.pool, appointmentID)
	case scheduling.StatusCompleted:
		return scheduling.Appointment{}, scheduling.ErrCompletedAppointmentCannotCancel
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE appointments
		SET status = 'CANCELLED'
		WHERE id = $1
		RETURNING
			id::text,
			customer_id::text,
			vehicle_id::text,
			dealership_id::text,
			service_type_id::text,
			technician_id::text,
			service_bay_id::text,
			start_time,
			end_time,
			status
	`, appointmentID)

	return scanAppointment(row)
}

type serviceTypeForBooking struct {
	DurationMinutes   int
	RequiredSkillCode string
}

type appointmentQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func verifyCustomerExists(ctx context.Context, queryer appointmentQuerier, customerID string) error {
	var exists bool
	if err := queryer.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM customers
			WHERE id = $1
		)
	`, customerID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return scheduling.ErrNotFound
	}

	return nil
}

func verifyVehicleBelongsToCustomer(ctx context.Context, queryer appointmentQuerier, vehicleID, customerID string) error {
	var ownerID string
	if err := queryer.QueryRow(ctx, `
		SELECT customer_id::text
		FROM vehicles
		WHERE id = $1
	`, vehicleID).Scan(&ownerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return scheduling.ErrNotFound
		}
		return err
	}
	if ownerID != customerID {
		return scheduling.ErrVehicleCustomerMismatch
	}

	return nil
}

func verifyDealershipExists(ctx context.Context, queryer appointmentQuerier, dealershipID string) error {
	var exists bool
	if err := queryer.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM dealerships
			WHERE id = $1
		)
	`, dealershipID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return scheduling.ErrNotFound
	}

	return nil
}

func getServiceTypeForBooking(ctx context.Context, queryer appointmentQuerier, serviceTypeID string) (serviceTypeForBooking, error) {
	var serviceType serviceTypeForBooking
	if err := queryer.QueryRow(ctx, `
		SELECT duration_minutes, required_skill_code
		FROM service_types
		WHERE id = $1
	`, serviceTypeID).Scan(&serviceType.DurationMinutes, &serviceType.RequiredSkillCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return serviceTypeForBooking{}, scheduling.ErrNotFound
		}
		return serviceTypeForBooking{}, err
	}

	return serviceType, nil
}

func chooseAvailableTechnician(ctx context.Context, queryer appointmentQuerier, dealershipID, requiredSkillCode string, startTime, endTime time.Time) (string, error) {
	var technicianID string
	if err := queryer.QueryRow(ctx, `
		SELECT t.id::text
		FROM technicians t
		INNER JOIN technician_skills ts ON ts.technician_id = t.id
		WHERE t.dealership_id = $1
			AND ts.skill_code = $2
			AND NOT EXISTS (
				SELECT 1
				FROM appointments a
				WHERE a.technician_id = t.id
					AND a.status = 'CONFIRMED'
					AND a.start_time < $4
					AND a.end_time > $3
			)
		ORDER BY t.name
		LIMIT 1
	`, dealershipID, requiredSkillCode, startTime, endTime).Scan(&technicianID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", scheduling.ErrConflict
		}
		return "", err
	}

	return technicianID, nil
}

func chooseAvailableServiceBay(ctx context.Context, queryer appointmentQuerier, dealershipID string, startTime, endTime time.Time) (string, error) {
	var serviceBayID string
	if err := queryer.QueryRow(ctx, `
		SELECT sb.id::text
		FROM service_bays sb
		WHERE sb.dealership_id = $1
			AND NOT EXISTS (
				SELECT 1
				FROM appointments a
				WHERE a.service_bay_id = sb.id
					AND a.status = 'CONFIRMED'
					AND a.start_time < $3
					AND a.end_time > $2
			)
		ORDER BY sb.name
		LIMIT 1
	`, dealershipID, startTime, endTime).Scan(&serviceBayID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", scheduling.ErrConflict
		}
		return "", err
	}

	return serviceBayID, nil
}

func insertAppointment(ctx context.Context, queryer appointmentQuerier, request scheduling.CreateAppointmentRequest, technicianID, serviceBayID string, endTime time.Time) (scheduling.Appointment, error) {
	row := queryer.QueryRow(ctx, `
		INSERT INTO appointments (
			customer_id,
			vehicle_id,
			dealership_id,
			service_type_id,
			technician_id,
			service_bay_id,
			start_time,
			end_time,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'CONFIRMED')
		RETURNING
			id::text,
			customer_id::text,
			vehicle_id::text,
			dealership_id::text,
			service_type_id::text,
			technician_id::text,
			service_bay_id::text,
			start_time,
			end_time,
			status
	`, request.CustomerID, request.VehicleID, request.DealershipID, request.ServiceTypeID, technicianID, serviceBayID, request.StartTime, endTime)

	return scanAppointment(row)
}

func getAppointment(ctx context.Context, queryer appointmentQuerier, appointmentID string) (scheduling.Appointment, error) {
	row := queryer.QueryRow(ctx, `
		SELECT
			id::text,
			customer_id::text,
			vehicle_id::text,
			dealership_id::text,
			service_type_id::text,
			technician_id::text,
			service_bay_id::text,
			start_time,
			end_time,
			status
		FROM appointments
		WHERE id = $1
	`, appointmentID)

	appointment, err := scanAppointment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return scheduling.Appointment{}, scheduling.ErrNotFound
	}
	return appointment, err
}

func getAppointmentDetail(ctx context.Context, queryer appointmentQuerier, appointmentID string) (scheduling.AppointmentDetail, error) {
	var detail scheduling.AppointmentDetail
	if err := queryer.QueryRow(ctx, `
		SELECT
			a.id::text,
			a.status,
			a.start_time,
			a.end_time,
			c.id::text,
			c.name,
			v.id::text,
			v.vin,
			v.make,
			v.model,
			v.year,
			d.id::text,
			d.name,
			st.id::text,
			st.name,
			t.id::text,
			t.name,
			sb.id::text,
			sb.name
		FROM appointments a
		INNER JOIN customers c ON c.id = a.customer_id
		INNER JOIN vehicles v ON v.id = a.vehicle_id
		INNER JOIN dealerships d ON d.id = a.dealership_id
		INNER JOIN service_types st ON st.id = a.service_type_id
		INNER JOIN technicians t ON t.id = a.technician_id
		INNER JOIN service_bays sb ON sb.id = a.service_bay_id
		WHERE a.id = $1
	`, appointmentID).Scan(
		&detail.ID,
		&detail.Status,
		&detail.StartTime,
		&detail.EndTime,
		&detail.Customer.ID,
		&detail.Customer.Name,
		&detail.Vehicle.ID,
		&detail.Vehicle.VIN,
		&detail.Vehicle.Make,
		&detail.Vehicle.Model,
		&detail.Vehicle.Year,
		&detail.Dealership.ID,
		&detail.Dealership.Name,
		&detail.ServiceType.ID,
		&detail.ServiceType.Name,
		&detail.Technician.ID,
		&detail.Technician.Name,
		&detail.ServiceBay.ID,
		&detail.ServiceBay.Name,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return scheduling.AppointmentDetail{}, scheduling.ErrNotFound
		}
		return scheduling.AppointmentDetail{}, err
	}

	detail.StartTime = detail.StartTime.UTC()
	detail.EndTime = detail.EndTime.UTC()

	return detail, nil
}

func scanAppointment(row pgx.Row) (scheduling.Appointment, error) {
	var appointment scheduling.Appointment
	if err := row.Scan(
		&appointment.ID,
		&appointment.CustomerID,
		&appointment.VehicleID,
		&appointment.DealershipID,
		&appointment.ServiceTypeID,
		&appointment.TechnicianID,
		&appointment.ServiceBayID,
		&appointment.StartTime,
		&appointment.EndTime,
		&appointment.Status,
	); err != nil {
		return scheduling.Appointment{}, err
	}

	appointment.StartTime = appointment.StartTime.UTC()
	appointment.EndTime = appointment.EndTime.UTC()

	return appointment, nil
}
