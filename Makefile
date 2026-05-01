SHELL := /bin/bash
include .env
export

.PHONY: up down migrate seed backend mobile mobile-test mobile-build install check test lint fmt test-integration

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

# Run the Flutter app on the default device (chrome / connected handset).
# For an Android emulator: flutter emulators --launch <id> first.
mobile:
	cd mobile && flutter run

mobile-test:
	cd mobile && flutter test

mobile-build:
	cd mobile && flutter build apk --release

install:
	cd backend && go mod download
	cd mobile && flutter pub get

test:
	cd backend && go test ./...

# Integration tests hit a real Postgres via TEST_DATABASE_URL. Uses the dev db from docker compose up.
test-integration:
	cd backend && TEST_DATABASE_URL="postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" go test -p 1 -race -count=1 ./...

lint:
	cd backend && go vet ./...
	cd mobile && flutter analyze --no-fatal-infos

fmt:
	cd backend && gofmt -w .
	cd mobile && dart format .

check: lint test mobile-test
	@echo "All checks passed."
