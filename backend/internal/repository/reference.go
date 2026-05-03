package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"keyloop-service-scheduler/internal/scheduling"
)

var ErrNotFound = errors.New("not found")

type Dealership struct {
	ID       string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Address  string `db:"address" json:"address"`
	Timezone string `db:"timezone" json:"timezone"`
}

type Customer struct {
	ID    string  `db:"id" json:"id"`
	Name  string  `db:"name" json:"name"`
	Email string  `db:"email" json:"email"`
	Phone *string `db:"phone" json:"phone"`
}

type Vehicle struct {
	ID         string `db:"id" json:"id"`
	CustomerID string `db:"customer_id" json:"customerId"`
	VIN        string `db:"vin" json:"vin"`
	Make       string `db:"make" json:"make"`
	Model      string `db:"model" json:"model"`
	Year       int    `db:"year" json:"year"`
}

type ServiceType struct {
	ID                string `db:"id" json:"id"`
	Name              string `db:"name" json:"name"`
	DurationMinutes   int    `db:"duration_minutes" json:"durationMinutes"`
	RequiredSkillCode string `db:"required_skill_code" json:"requiredSkillCode"`
}

type Technician struct {
	ID           string   `db:"id" json:"id"`
	DealershipID string   `db:"dealership_id" json:"dealershipId"`
	Name         string   `db:"name" json:"name"`
	Skills       []string `db:"skills" json:"skills"`
}

type ServiceBay struct {
	ID           string `db:"id" json:"id"`
	DealershipID string `db:"dealership_id" json:"dealershipId"`
	Name         string `db:"name" json:"name"`
}

type ReferenceRepository interface {
	ListDealerships(ctx context.Context) ([]Dealership, error)
	ListCustomers(ctx context.Context) ([]Customer, error)
	ListCustomerVehicles(ctx context.Context, customerID string) ([]Vehicle, error)
	ListServiceTypes(ctx context.Context) ([]ServiceType, error)
	ListTechnicians(ctx context.Context, dealershipID string) ([]Technician, error)
	ListServiceBays(ctx context.Context, dealershipID string) ([]ServiceBay, error)
}

type PostgresReferenceRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresReferenceRepository(pool *pgxpool.Pool) *PostgresReferenceRepository {
	return &PostgresReferenceRepository{pool: pool}
}

func (r *PostgresReferenceRepository) ListDealerships(ctx context.Context) ([]Dealership, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, address, timezone
		FROM dealerships
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[Dealership])
}

func (r *PostgresReferenceRepository) ListCustomers(ctx context.Context) ([]Customer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, email, phone
		FROM customers
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[Customer])
}

func (r *PostgresReferenceRepository) ListCustomerVehicles(ctx context.Context, customerID string) ([]Vehicle, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM customers
			WHERE id = $1
		)
	`, customerID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id::text, customer_id::text AS customer_id, vin, make, model, year
		FROM vehicles
		WHERE customer_id = $1
		ORDER BY make, model, year
	`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[Vehicle])
}

func (r *PostgresReferenceRepository) ListServiceTypes(ctx context.Context) ([]ServiceType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			name,
			duration_minutes,
			required_skill_code
		FROM service_types
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[ServiceType])
}

func (r *PostgresReferenceRepository) ListTechnicians(ctx context.Context, dealershipID string) ([]Technician, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			t.id::text,
			t.dealership_id::text AS dealership_id,
			t.name,
			COALESCE(array_agg(ts.skill_code ORDER BY ts.skill_code) FILTER (WHERE ts.skill_code IS NOT NULL), '{}') AS skills
		FROM technicians t
		LEFT JOIN technician_skills ts ON ts.technician_id = t.id
		WHERE t.dealership_id = $1
		GROUP BY t.id, t.dealership_id, t.name
		ORDER BY t.name
	`, dealershipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[Technician])
}

func (r *PostgresReferenceRepository) ListServiceBays(ctx context.Context, dealershipID string) ([]ServiceBay, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, dealership_id::text AS dealership_id, name
		FROM service_bays
		WHERE dealership_id = $1
		ORDER BY name
	`, dealershipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[ServiceBay])
}

func (r *PostgresReferenceRepository) VerifyDealershipExists(ctx context.Context, dealershipID string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
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

func (r *PostgresReferenceRepository) GetServiceType(ctx context.Context, serviceTypeID string) (scheduling.ServiceType, error) {
	var serviceType scheduling.ServiceType
	if err := r.pool.QueryRow(ctx, `
		SELECT id::text, duration_minutes, required_skill_code
		FROM service_types
		WHERE id = $1
	`, serviceTypeID).Scan(&serviceType.ID, &serviceType.DurationMinutes, &serviceType.RequiredSkillCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return scheduling.ServiceType{}, scheduling.ErrNotFound
		}
		return scheduling.ServiceType{}, err
	}

	return serviceType, nil
}

func (r *PostgresReferenceRepository) ListQualifiedTechnicians(ctx context.Context, dealershipID, requiredSkillCode string) ([]scheduling.Resource, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id::text, t.name
		FROM technicians t
		INNER JOIN technician_skills ts ON ts.technician_id = t.id
		WHERE t.dealership_id = $1
			AND ts.skill_code = $2
		ORDER BY t.name
	`, dealershipID, requiredSkillCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[scheduling.Resource])
}

func (r *PostgresReferenceRepository) ListBusyTechnicianIDs(ctx context.Context, dealershipID string, startTime, endTime time.Time) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT technician_id::text
		FROM appointments
		WHERE dealership_id = $1
			AND status = 'CONFIRMED'
			AND start_time < $3
			AND end_time > $2
	`, dealershipID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (r *PostgresReferenceRepository) ListAvailableServiceBays(ctx context.Context, dealershipID string) ([]scheduling.Resource, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name
		FROM service_bays
		WHERE dealership_id = $1
		ORDER BY name
	`, dealershipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[scheduling.Resource])
}

func (r *PostgresReferenceRepository) ListBusyServiceBayIDs(ctx context.Context, dealershipID string, startTime, endTime time.Time) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT service_bay_id::text
		FROM appointments
		WHERE dealership_id = $1
			AND status = 'CONFIRMED'
			AND start_time < $3
			AND end_time > $2
	`, dealershipID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowTo[string])
}
