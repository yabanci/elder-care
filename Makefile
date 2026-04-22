SHELL := /bin/bash
include .env
export

.PHONY: up down migrate seed backend frontend check test lint fmt

up:
	docker compose up -d db
	@echo "Waiting for Postgres..."
	@until docker compose exec -T db pg_isready -U $(POSTGRES_USER) >/dev/null 2>&1; do sleep 1; done
	@echo "Postgres is ready."

down:
	docker compose down

migrate:
	cd backend && go run ./cmd/server --migrate-only

seed:
	cd backend && go run ./cmd/seed

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

install:
	cd backend && go mod download
	cd frontend && npm install

test:
	cd backend && go test ./...

# Integration tests hit a real Postgres via TEST_DATABASE_URL. Uses the dev db from docker compose up.
test-integration:
	cd backend && TEST_DATABASE_URL="postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" go test -p 1 -race -count=1 ./...

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

fmt:
	cd backend && gofmt -w .
	cd frontend && npx prettier --write "app/**/*.{ts,tsx}" "components/**/*.{ts,tsx}" "lib/**/*.{ts,tsx}" 2>/dev/null || true

check: lint test
	cd frontend && npx tsc --noEmit
	@echo "All checks passed."
