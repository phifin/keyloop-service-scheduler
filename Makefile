DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/keyloop_scheduler?sslmode=disable
MIGRATIONS_PATH := backend/db/migrations

.PHONY: migrate-up migrate-down seed test run-api

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

seed:
	psql "$(DATABASE_URL)" -f backend/db/seed.sql

test:
	cd backend && go test ./...

run-api:
	cd backend && go run ./cmd/api
