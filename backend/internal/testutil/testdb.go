// Package testutil provides integration-test helpers that spin up the full
// backend against a real Postgres referenced by the TEST_DATABASE_URL env var.
// Tests skip automatically if the variable is not set.
package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"eldercare/backend/internal/db"
)

// NewPool returns a pgxpool connected to TEST_DATABASE_URL, applies migrations,
// and wipes all domain tables so each test starts from a known state.
func NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}

	// Wipe all domain data but keep schema + migration bookkeeping.
	if _, err := pool.Exec(ctx, `
		TRUNCATE messages, medication_logs, medications, plans, alerts, health_metrics, patient_links, users
		RESTART IDENTITY CASCADE
	`); err != nil {
		pool.Close()
		t.Fatalf("truncate: %v", err)
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

// NewRouter returns a gin.Engine in test mode with default middleware disabled.
func NewRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}
