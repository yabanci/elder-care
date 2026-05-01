package medications_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/medications"
	"eldercare/backend/internal/testutil"
)

const tzTestSecret = "tz_test_secret"

// TestPerUserTZ_SchedulesUseLocalDate verifies that two patients on
// opposite sides of the UTC date boundary each see their medication's
// "today" anchored to their own timezone — not the server's local TZ
// nor UTC.
func TestPerUserTZ_SchedulesUseLocalDate(t *testing.T) {
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, tzTestSecret, 1)
	medSvc := medications.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.PATCH("/me", authSvc.UpdateMe)
	api.POST("/medications", medSvc.Create)
	api.GET("/medications/today", medSvc.Today)

	w := testutil.DoJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": "p-tz-almaty@test.kz", "password": "secret123", "full_name": "T", "role": "patient",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	testutil.Decode(t, w, &resp)

	w = testutil.DoJSON(t, r, "PATCH", "/api/me", resp.Token, map[string]any{
		"tz": "Asia/Almaty",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("set tz: %d %s", w.Code, w.Body.String())
	}

	w = testutil.DoJSON(t, r, "POST", "/api/medications", resp.Token, map[string]any{
		"name": "Test", "times_of_day": []string{"08:00", "20:00"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create med: %d %s", w.Code, w.Body.String())
	}

	w = testutil.DoJSON(t, r, "GET", "/api/medications/today", resp.Token, nil)
	var schedule []map[string]any
	testutil.Decode(t, w, &schedule)
	if len(schedule) != 2 {
		t.Fatalf("expected 2 scheduled items in Almaty TZ, got %d: %s", len(schedule), w.Body.String())
	}

	// scheduled_at is serialized in UTC. Parse and re-project to Almaty;
	// the wall-clock hour should be 08 or 20 in that zone, regardless of
	// what UTC clock-time it converts to.
	loc, _ := time.LoadLocation("Asia/Almaty")
	for _, item := range schedule {
		ts, ok := item["scheduled_at"].(string)
		if !ok {
			t.Fatalf("scheduled_at missing/wrong type: %v", item)
		}
		parsed, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			t.Fatalf("parse %s: %v", ts, err)
		}
		hourLocal := parsed.In(loc).Hour()
		if hourLocal != 8 && hourLocal != 20 {
			t.Errorf("scheduled_at=%s reprojects to Almaty hour %d, expected 8 or 20 (raw: %s)", ts, hourLocal, ts)
		}
	}
}

// TestPerUserTZ_RejectsBadTimezone verifies the validator catches garbage.
func TestPerUserTZ_RejectsBadTimezone(t *testing.T) {
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, tzTestSecret, 1)
	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.PATCH("/me", authSvc.UpdateMe)

	w := testutil.DoJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": "p-tz-bad@test.kz", "password": "secret123", "full_name": "T", "role": "patient",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register: %d", w.Code)
	}
	var resp struct {
		Token string `json:"token"`
	}
	testutil.Decode(t, w, &resp)

	w = testutil.DoJSON(t, r, "PATCH", "/api/me", resp.Token, map[string]any{
		"tz": "Mars/Olympus_Mons",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad tz, got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid tz") {
		t.Errorf("expected error mentioning 'invalid tz', got %s", w.Body.String())
	}
}
