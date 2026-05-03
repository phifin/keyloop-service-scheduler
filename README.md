# Keyloop Unified Service Scheduler

Technical assessment project for Keyloop, Scenario A: Unified Service Scheduler.

This is a production-oriented MVP with production-minded design choices. It is not claimed to be production ready.

## Project Overview

The application supports a dealership service scheduling workflow:

- Load reference data for dealerships, customers, vehicles, service types, technicians, and service bays.
- Check whether a requested slot has both a qualified technician and an available service bay.
- Create confirmed appointments with a transactional availability re-check.
- List, view, and cancel appointments.
- Demonstrate the workflow through a lightweight frontend.

## Implementation Scope

- Backend primary layer: Go, chi, PostgreSQL, pgx.
- Frontend demo layer: React, Vite, TypeScript, React Router, Tailwind CSS.
- Database: PostgreSQL migrations and stable seed data.
- Documentation: Markdown and Mermaid diagrams.

The backend owns scheduling correctness. The frontend calls the backend APIs and does not duplicate booking business logic.

## Tech Stack

- Backend: Go, chi, pgx pool
- Database: PostgreSQL
- Migrations: golang-migrate
- Frontend: React, Vite, TypeScript, React Router, Tailwind CSS
- Tooling: Docker Compose, Makefile
- Documentation: Markdown, Mermaid

## Repository Structure

```text
.
+-- backend/
|   +-- cmd/api/
|   +-- db/
|   |   +-- migrations/
|   |   +-- seed.sql
|   +-- internal/
|   |   +-- config/
|   |   +-- database/
|   |   +-- http/
|   |   +-- repository/
|   |   +-- scheduling/
|   +-- openapi.yaml
+-- frontend/
|   +-- src/api/
|   +-- src/components/
|   +-- src/pages/
|   +-- src/types/
+-- docs/
|   +-- ai-collaboration.md
|   +-- system-design.md
+-- docker-compose.yml
+-- Makefile
+-- .env.example
+-- README.md
```

## Prerequisites

- Go 1.22 or newer
- Node.js and npm
- Docker and Docker Compose
- PostgreSQL client tools, including `psql`
- golang-migrate CLI

Install golang-migrate if needed:

```sh
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Environment Variables

Copy or reference `.env.example`:

```sh
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/keyloop_scheduler?sslmode=disable
VITE_API_BASE_URL=http://localhost:8080
```

The backend defaults to `PORT=8080`. The frontend uses `VITE_API_BASE_URL` and expects the backend at `http://localhost:8080` by default.

## Run PostgreSQL

```sh
docker compose up -d
```

This starts PostgreSQL with:

- User: `postgres`
- Password: `postgres`
- Database: `keyloop_scheduler`
- Local port: `5432`

If local port `5432` is already in use, change the port mapping in `docker-compose.yml` from `5432:5432` to `5433:5432`, then use:

```sh
DATABASE_URL=postgres://postgres:postgres@localhost:5433/keyloop_scheduler?sslmode=disable
```

## Run Migrations

```sh
make migrate-up
```

Rollback the initial migration:

```sh
make migrate-down
```

## Seed Data

```sh
make seed
```

Seed data uses stable UUID values for repeatable demos and tests. It includes:

- Downtown Keyloop Motors
- Two service bays
- Four service types
- Three technicians with skills
- Three customers and vehicles
- One confirmed appointment for conflict demos

Verify seeded data:

```sh
psql "postgres://postgres:postgres@localhost:5432/keyloop_scheduler?sslmode=disable" \
  -c "select name from dealerships;"
```

## Run Backend

```sh
cd backend
go run ./cmd/api
```

Or from the repository root:

```sh
make run-api
```

Health check:

```sh
curl http://localhost:8080/health
```

Expected response:

```json
{
  "status": "ok"
}
```

## Run Frontend

```sh
cd frontend
npm install
npm run dev
```

The demo routes are:

- `/book`
- `/appointments`
- `/appointments/:appointmentId`

For a custom API URL:

```sh
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

## API Examples

Seeded UUIDs used below:

- Dealership: `44444444-4444-4444-4444-444444444444`
- Customer John Smith: `11111111-1111-1111-1111-111111111111`
- John Smith vehicle: `aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1`
- Oil Change service: `55555555-5555-5555-5555-555555555551`

Reference data:

```sh
curl http://localhost:8080/dealerships
curl http://localhost:8080/customers
curl http://localhost:8080/service-types
curl http://localhost:8080/customers/11111111-1111-1111-1111-111111111111/vehicles
curl "http://localhost:8080/technicians?dealershipId=44444444-4444-4444-4444-444444444444"
curl "http://localhost:8080/service-bays?dealershipId=44444444-4444-4444-4444-444444444444"
```

Check availability:

```sh
curl "http://localhost:8080/availability?dealershipId=44444444-4444-4444-4444-444444444444&serviceTypeId=55555555-5555-5555-5555-555555555551&startTime=2026-05-04T14:00:00Z"
```

Create an appointment:

```sh
curl -X POST http://localhost:8080/appointments \
  -H "Content-Type: application/json" \
  -d '{
    "customerId": "11111111-1111-1111-1111-111111111111",
    "vehicleId": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
    "dealershipId": "44444444-4444-4444-4444-444444444444",
    "serviceTypeId": "55555555-5555-5555-5555-555555555551",
    "startTime": "2026-05-04T14:00:00Z"
  }'
```

List appointments:

```sh
curl http://localhost:8080/appointments
curl "http://localhost:8080/appointments?dealershipId=44444444-4444-4444-4444-444444444444&status=CONFIRMED"
```

Get appointment detail:

```sh
curl http://localhost:8080/appointments/{appointmentId}
```

Cancel an appointment:

```sh
curl -X PATCH http://localhost:8080/appointments/{appointmentId}/cancel
```

## Scheduling Rules

Availability uses this overlap rule:

```sql
existing.start_time < new_end
AND existing.end_time > new_start
AND existing.status = 'CONFIRMED'
```

Adjacent appointments are allowed. An appointment ending at 10:00 does not conflict with another appointment starting at 10:00.

Only `CONFIRMED` appointments block resources. `CANCELLED` and `COMPLETED` appointments do not block availability or future booking.

`GET /availability` is a snapshot for user experience. `POST /appointments` is the source of truth and re-checks availability inside a PostgreSQL transaction before inserting a confirmed appointment.

Appointment creation uses a transaction-level advisory lock scoped to `dealershipId`:

```sql
SELECT pg_advisory_xact_lock(hashtext($1::text));
```

A production-grade improvement would add PostgreSQL exclusion constraints using `tstzrange(start_time, end_time)`.

## Run Tests

Backend unit and handler tests:

```sh
cd backend
go test ./...
```

Frontend production build:

```sh
cd frontend
npm run build
```

Optional repository integration tests use a temporary PostgreSQL container:

```sh
cd backend
RUN_DB_INTEGRATION=1 go test ./internal/repository -run TestAppointmentRepositoryIntegration -count=1
```

## Design Documentation

- [System Design](docs/system-design.md)
- [AI Collaboration Narrative](docs/ai-collaboration.md)
- [OpenAPI Specification](backend/openapi.yaml)

## Final Deliverable Checklist

- System Design Document: [docs/system-design.md](docs/system-design.md)
- AI Collaboration Narrative: [docs/ai-collaboration.md](docs/ai-collaboration.md)
- Backend implementation: Go API with reference data, availability, appointment creation, listing, detail, and cancellation.
- Frontend demo: React/Vite workflow for booking, appointment list, appointment detail, and cancellation.
- Tests: backend tests with `go test ./...`; frontend build with `npm run build`; optional PostgreSQL repository integration tests.
- Video submission: to be recorded separately.

## Assumptions

- Service type duration is fixed.
- Service type requires one skill code.
- Technician skill matching is based on skill code equality.
- Any bay at the selected dealership can support any seeded service type.
- The MVP does not model operating hours, technician shifts, holidays, or time-off.
- Authentication and authorization are out of scope for this assessment build.

## Future Improvements

- Add PostgreSQL exclusion constraints for database-enforced conflict prevention.
- Add dealership business hours, technician shifts, holidays, and time-off.
- Add authentication, authorization, and audit logging.
- Add appointment rescheduling and completion workflows.
- Add pagination for appointment listing.
- Add CI for backend tests, optional integration tests, frontend builds, and OpenAPI validation.
- Add OpenTelemetry tracing and application metrics.
- Add browser end-to-end tests for the demo booking workflow.

## Troubleshooting

### Port 5432 Already in Use

Change `docker-compose.yml` from:

```yaml
ports:
  - "5432:5432"
```

to:

```yaml
ports:
  - "5433:5432"
```

Then run commands with:

```sh
DATABASE_URL=postgres://postgres:postgres@localhost:5433/keyloop_scheduler?sslmode=disable
```

### Backend Cannot Connect to Database

Check that PostgreSQL is running:

```sh
docker compose ps
```

Confirm the database URL:

```sh
echo "$DATABASE_URL"
```

Apply migrations and seed data if the schema or records are missing:

```sh
make migrate-up
make seed
```

### Frontend Cannot Connect to Backend

Confirm the backend is running:

```sh
curl http://localhost:8080/health
```

Confirm the frontend API base URL:

```sh
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

If the backend uses a different port, update `VITE_API_BASE_URL` to match it.
