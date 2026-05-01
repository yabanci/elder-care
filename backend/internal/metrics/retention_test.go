package metrics_test

import (
	"context"
	"testing"
	"time"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/metrics"
	"eldercare/backend/internal/testutil"
)

// TestRetentionSweep_DropsOnlyOldRows seeds two algorithm_runs rows with
// different ages and verifies the sweep removes only the one past the
// retention window. Hits a real Postgres so we catch SQL syntax drift.
func TestRetentionSweep_DropsOnlyOldRows(t *testing.T) {
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, "ret_test_secret", 1)

	// Need a real user + metric so the FK constraints on algorithm_runs hold.
	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	metricsSvc := metrics.NewService(pool)
	api.POST("/metrics", metricsSvc.CreateForSelf)

	w := testutil.DoJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": "p-ret@test.kz", "password": "secret123", "full_name": "T", "role": "patient",
	})
	if w.Code != 200 {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	testutil.Decode(t, w, &resp)

	w = testutil.DoJSON(t, r, "POST", "/api/metrics", resp.Token, map[string]any{
		"kind": "pulse", "value": 200, // safety_above_max → algorithm_runs row written
	})
	if w.Code != 200 {
		t.Fatalf("create metric: %d %s", w.Code, w.Body.String())
	}

	// Backdate that row past the retention window.
	_, err := pool.Exec(context.Background(),
		"UPDATE algorithm_runs SET created_at = now() - interval '120 days'")
	if err != nil {
		t.Fatalf("backdate: %v", err)
	}

	// Insert a fresh second row by submitting another metric.
	w = testutil.DoJSON(t, r, "POST", "/api/metrics", resp.Token, map[string]any{
		"kind": "pulse", "value": 75, // normal — still creates an algorithm_runs row
	})
	if w.Code != 200 {
		t.Fatalf("second metric: %d %s", w.Code, w.Body.String())
	}

	var beforeCount int
	if err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM algorithm_runs").Scan(&beforeCount); err != nil {
		t.Fatal(err)
	}
	if beforeCount != 2 {
		t.Fatalf("expected 2 rows pre-sweep, got %d", beforeCount)
	}

	// Run the sweep synchronously by invoking StartRetentionSweep then
	// cancelling immediately (initial sweep runs before the ticker loop
	// even starts; the goroutine then exits on the cancelled context).
	ctx, cancel := context.WithCancel(context.Background())
	metrics.StartRetentionSweep(ctx, pool)
	// Give the goroutine a moment to perform the initial sweep.
	time.Sleep(200 * time.Millisecond)
	cancel()

	var afterCount int
	if err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM algorithm_runs").Scan(&afterCount); err != nil {
		t.Fatal(err)
	}
	if afterCount != 1 {
		t.Errorf("expected 1 row after sweep (the fresh one), got %d", afterCount)
	}
}
