package repository

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"keyloop-service-scheduler/internal/scheduling"
)

const (
	seedCustomerJohn     = "11111111-1111-1111-1111-111111111111"
	seedCustomerMaria    = "22222222-2222-2222-2222-222222222222"
	seedVehicleJohn      = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	seedVehicleMaria     = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2"
	seedDealership       = "44444444-4444-4444-4444-444444444444"
	seedServiceOil       = "55555555-5555-5555-5555-555555555551"
	seedServiceBrake     = "55555555-5555-5555-5555-555555555552"
	seedServiceTire      = "55555555-5555-5555-5555-555555555553"
	seedTechAlex         = "66666666-6666-6666-6666-666666666661"
	seedTechTaylor       = "66666666-6666-6666-6666-666666666663"
	seedBayOne           = "88888888-8888-8888-8888-888888888881"
	seedBayTwo           = "88888888-8888-8888-8888-888888888882"
	integrationContainer = "keyloop-appointments-integration-postgres"
)

func TestAppointmentRepositoryIntegration(t *testing.T) {
	if os.Getenv("RUN_DB_INTEGRATION") != "1" {
		t.Skip("set RUN_DB_INTEGRATION=1 to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	databaseURL := startIntegrationPostgres(t, ctx)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	repo := NewPostgresReferenceRepository(pool)

	t.Run("successful appointment creation", func(t *testing.T) {
		appointment, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T14:00:00Z", seedServiceOil, seedCustomerJohn, seedVehicleJohn))
		if err != nil {
			t.Fatal(err)
		}
		if appointment.Status != scheduling.StatusConfirmed {
			t.Fatalf("expected confirmed appointment, got %s", appointment.Status)
		}
	})

	t.Run("reject overlapping technician booking", func(t *testing.T) {
		insertAppointmentForTest(t, pool, "aaaaaaaa-0000-0000-0000-000000000101", seedCustomerJohn, seedVehicleJohn, seedServiceOil, seedTechAlex, seedBayOne, "2026-05-04T15:00:00Z", "2026-05-04T15:30:00Z", scheduling.StatusConfirmed)
		insertAppointmentForTest(t, pool, "aaaaaaaa-0000-0000-0000-000000000102", seedCustomerMaria, seedVehicleMaria, seedServiceOil, seedTechTaylor, seedBayTwo, "2026-05-04T15:00:00Z", "2026-05-04T15:30:00Z", scheduling.StatusConfirmed)

		_, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T15:00:00Z", seedServiceOil, seedCustomerJohn, seedVehicleJohn))
		if err != scheduling.ErrConflict {
			t.Fatalf("expected conflict, got %v", err)
		}
	})

	t.Run("reject overlapping service bay booking", func(t *testing.T) {
		insertAppointmentForTest(t, pool, "aaaaaaaa-0000-0000-0000-000000000201", seedCustomerJohn, seedVehicleJohn, seedServiceOil, seedTechAlex, seedBayOne, "2026-05-04T16:00:00Z", "2026-05-04T16:30:00Z", scheduling.StatusConfirmed)
		insertAppointmentForTest(t, pool, "aaaaaaaa-0000-0000-0000-000000000202", seedCustomerMaria, seedVehicleMaria, seedServiceOil, seedTechTaylor, seedBayTwo, "2026-05-04T16:00:00Z", "2026-05-04T16:30:00Z", scheduling.StatusConfirmed)

		_, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T16:00:00Z", seedServiceBrake, seedCustomerJohn, seedVehicleJohn))
		if err != scheduling.ErrConflict {
			t.Fatalf("expected conflict, got %v", err)
		}
	})

	t.Run("allow adjacent appointment", func(t *testing.T) {
		appointment, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T13:30:00Z", seedServiceOil, seedCustomerMaria, seedVehicleMaria))
		if err != nil {
			t.Fatal(err)
		}
		if appointment.Status != scheduling.StatusConfirmed {
			t.Fatalf("expected confirmed appointment, got %s", appointment.Status)
		}
	})

	t.Run("cancelled appointments do not block", func(t *testing.T) {
		insertAppointmentForTest(t, pool, "aaaaaaaa-0000-0000-0000-000000000301", seedCustomerJohn, seedVehicleJohn, seedServiceOil, seedTechAlex, seedBayOne, "2026-05-04T17:00:00Z", "2026-05-04T17:30:00Z", scheduling.StatusCancelled)
		insertAppointmentForTest(t, pool, "aaaaaaaa-0000-0000-0000-000000000302", seedCustomerMaria, seedVehicleMaria, seedServiceOil, seedTechTaylor, seedBayTwo, "2026-05-04T17:00:00Z", "2026-05-04T17:30:00Z", scheduling.StatusCompleted)

		if _, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T17:00:00Z", seedServiceOil, seedCustomerJohn, seedVehicleJohn)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("technician must have required skill", func(t *testing.T) {
		appointment, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T18:00:00Z", seedServiceTire, seedCustomerMaria, seedVehicleMaria))
		if err != nil {
			t.Fatal(err)
		}
		if appointment.TechnicianID != seedTechTaylor {
			t.Fatalf("expected tire appointment to use Taylor Smith, got %s", appointment.TechnicianID)
		}
	})

	t.Run("vehicle must belong to customer", func(t *testing.T) {
		_, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T19:00:00Z", seedServiceOil, seedCustomerJohn, seedVehicleMaria))
		if err != scheduling.ErrVehicleCustomerMismatch {
			t.Fatalf("expected vehicle/customer mismatch, got %v", err)
		}
	})

	t.Run("unknown resources return not found", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*scheduling.CreateAppointmentRequest)
		}{
			{
				name: "unknown customer",
				mutate: func(request *scheduling.CreateAppointmentRequest) {
					request.CustomerID = "aaaaaaaa-0000-0000-0000-000000009996"
				},
			},
			{
				name: "unknown vehicle",
				mutate: func(request *scheduling.CreateAppointmentRequest) {
					request.VehicleID = "aaaaaaaa-0000-0000-0000-000000009997"
				},
			},
			{
				name: "unknown dealership",
				mutate: func(request *scheduling.CreateAppointmentRequest) {
					request.DealershipID = "aaaaaaaa-0000-0000-0000-000000009998"
				},
			},
			{
				name: "unknown service type",
				mutate: func(request *scheduling.CreateAppointmentRequest) {
					request.ServiceTypeID = "aaaaaaaa-0000-0000-0000-000000009999"
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				request := createRequest("2026-05-04T19:30:00Z", seedServiceOil, seedCustomerJohn, seedVehicleJohn)
				tt.mutate(&request)

				_, err := repo.CreateAppointment(ctx, request)
				if !errors.Is(err, scheduling.ErrNotFound) {
					t.Fatalf("expected not found, got %v", err)
				}
			})
		}
	})

	t.Run("cancel appointment makes slot available", func(t *testing.T) {
		appointment, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T20:00:00Z", seedServiceOil, seedCustomerJohn, seedVehicleJohn))
		if err != nil {
			t.Fatal(err)
		}
		cancelled, err := repo.CancelAppointment(ctx, appointment.ID)
		if err != nil {
			t.Fatal(err)
		}
		if cancelled.Status != scheduling.StatusCancelled {
			t.Fatalf("expected cancelled status, got %s", cancelled.Status)
		}
		if _, err := repo.CreateAppointment(ctx, createRequest("2026-05-04T20:00:00Z", seedServiceOil, seedCustomerMaria, seedVehicleMaria)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("completed appointment cannot be cancelled", func(t *testing.T) {
		insertAppointmentForTest(t, pool, "aaaaaaaa-0000-0000-0000-000000000401", seedCustomerJohn, seedVehicleJohn, seedServiceOil, seedTechAlex, seedBayOne, "2026-05-04T21:00:00Z", "2026-05-04T21:30:00Z", scheduling.StatusCompleted)
		_, err := repo.CancelAppointment(ctx, "aaaaaaaa-0000-0000-0000-000000000401")
		if !errors.Is(err, scheduling.ErrCompletedAppointmentCannotCancel) {
			t.Fatalf("expected completed cancel conflict, got %v", err)
		}
	})

	t.Run("already cancelled appointment returns current appointment", func(t *testing.T) {
		insertAppointmentForTest(t, pool, "aaaaaaaa-0000-0000-0000-000000000402", seedCustomerJohn, seedVehicleJohn, seedServiceOil, seedTechAlex, seedBayOne, "2026-05-04T21:30:00Z", "2026-05-04T22:00:00Z", scheduling.StatusCancelled)

		cancelled, err := repo.CancelAppointment(ctx, "aaaaaaaa-0000-0000-0000-000000000402")
		if err != nil {
			t.Fatal(err)
		}
		if cancelled.Status != scheduling.StatusCancelled {
			t.Fatalf("expected cancelled appointment, got %s", cancelled.Status)
		}
	})

	t.Run("list and detail appointments", func(t *testing.T) {
		list, err := repo.ListAppointments(ctx, scheduling.AppointmentFilters{})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) == 0 {
			t.Fatal("expected appointments")
		}

		detail, err := repo.GetAppointmentDetail(ctx, list[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		if detail.ID != list[0].ID {
			t.Fatalf("expected detail id %s, got %s", list[0].ID, detail.ID)
		}

		_, err = repo.GetAppointmentDetail(ctx, "aaaaaaaa-0000-0000-0000-000000009998")
		if err != scheduling.ErrNotFound {
			t.Fatalf("expected not found, got %v", err)
		}
	})
}

func startIntegrationPostgres(t *testing.T, ctx context.Context) string {
	t.Helper()

	_ = exec.Command("docker", "rm", "-f", integrationContainer).Run()
	cmd := exec.Command(
		"docker",
		"run",
		"-d",
		"--name", integrationContainer,
		"-e", "POSTGRES_USER=postgres",
		"-e", "POSTGRES_PASSWORD=postgres",
		"-e", "POSTGRES_DB=keyloop_scheduler",
		"-p", "5432",
		"postgres:16-alpine",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("start postgres container: %v: %s", err, output)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", integrationContainer).Run()
	})

	for i := 0; i < 30; i++ {
		if err := exec.Command("docker", "exec", integrationContainer, "pg_isready", "-U", "postgres", "-d", "keyloop_scheduler").Run(); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
		if i == 29 {
			t.Fatal("postgres container did not become ready")
		}
	}

	portOutput, err := exec.Command("docker", "port", integrationContainer, "5432/tcp").Output()
	if err != nil {
		t.Fatal(err)
	}
	portLines := strings.Split(strings.TrimSpace(string(portOutput)), "\n")
	_, port, err := net.SplitHostPort(strings.TrimSpace(portLines[0]))
	if err != nil {
		t.Fatal(err)
	}

	runSQLFile(t, "backend/db/migrations/000001_create_scheduler_schema.up.sql")
	runSQLFile(t, "backend/db/seed.sql")

	return fmt.Sprintf("postgres://postgres:postgres@localhost:%s/keyloop_scheduler?sslmode=disable", port)
}

func runSQLFile(t *testing.T, path string) {
	t.Helper()

	root := repositoryRoot(t)
	contents, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("docker", "exec", "-i", integrationContainer, "psql", "-v", "ON_ERROR_STOP=1", "-U", "postgres", "-d", "keyloop_scheduler")
	cmd.Stdin = strings.NewReader(string(contents))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run sql %s: %v: %s", path, err, output)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

func createRequest(startTime, serviceTypeID, customerID, vehicleID string) scheduling.CreateAppointmentRequest {
	parsed, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		panic(err)
	}

	return scheduling.CreateAppointmentRequest{
		CustomerID:    customerID,
		VehicleID:     vehicleID,
		DealershipID:  seedDealership,
		ServiceTypeID: serviceTypeID,
		StartTime:     parsed,
	}
}

func insertAppointmentForTest(t *testing.T, pool *pgxpool.Pool, id, customerID, vehicleID, serviceTypeID, technicianID, serviceBayID, startTime, endTime, status string) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO appointments (
			id,
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, id, customerID, vehicleID, seedDealership, serviceTypeID, technicianID, serviceBayID, startTime, endTime, status)
	if err != nil {
		t.Fatal(err)
	}
}
