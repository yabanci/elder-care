package metrics_test

import (
	"net/http"
	"testing"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/metrics"
	"eldercare/backend/internal/testutil"
)

const testSecret = "test_secret_please_dont_use_in_prod"

func setup(t *testing.T) http.Handler {
	t.Helper()
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, testSecret, 1)
	metricsSvc := metrics.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.POST("/metrics", metricsSvc.CreateForSelf)
	api.GET("/metrics", metricsSvc.List)
	api.GET("/metrics/summary", metricsSvc.Summary)
	api.GET("/alerts", metricsSvc.ListAlerts)
	api.POST("/alerts/:id/acknowledge", metricsSvc.AcknowledgeAlert)
	return r
}

func register(t *testing.T, r http.Handler, email string) string {
	t.Helper()
	w := testutil.DoJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": email, "password": "secret123", "full_name": "T", "role": "patient",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register %s: %d %s", email, w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	testutil.Decode(t, w, &resp)
	return resp.Token
}

func TestMetricsCreateAndList(t *testing.T) {
	r := setup(t)
	token := register(t, r, "p-metrics@test.kz")

	for _, v := range []float64{72, 75, 70, 73, 71} {
		w := testutil.DoJSON(t, r, "POST", "/api/metrics", token, map[string]any{
			"kind": "pulse", "value": v,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("create pulse %v: %d %s", v, w.Code, w.Body.String())
		}
	}

	w := testutil.DoJSON(t, r, "GET", "/api/metrics", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d", w.Code)
	}
	var list []map[string]any
	testutil.Decode(t, w, &list)
	if len(list) != 5 {
		t.Fatalf("expected 5, got %d", len(list))
	}

	w = testutil.DoJSON(t, r, "GET", "/api/metrics/summary", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("summary: %d", w.Code)
	}
	var summary []map[string]any
	testutil.Decode(t, w, &summary)
	// Only one kind seeded, so summary should be 1.
	if len(summary) != 1 || summary[0]["kind"] != "pulse" {
		t.Fatalf("unexpected summary: %v", summary)
	}
}

func TestMetricsCreateValidation(t *testing.T) {
	r := setup(t)
	token := register(t, r, "p-mvalid@test.kz")

	// Unknown kind
	w := testutil.DoJSON(t, r, "POST", "/api/metrics", token, map[string]any{
		"kind": "blood_oxygen", "value": 95,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown kind: expected 400, got %d", w.Code)
	}
}

func TestMetricsSafetyTriggersAlert(t *testing.T) {
	r := setup(t)
	token := register(t, r, "p-alert@test.kz")

	// Pulse 180 — well above safety upper bound → critical alert created.
	w := testutil.DoJSON(t, r, "POST", "/api/metrics", token, map[string]any{
		"kind": "pulse", "value": 180,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	w = testutil.DoJSON(t, r, "GET", "/api/alerts", token, nil)
	var alerts []map[string]any
	testutil.Decode(t, w, &alerts)
	if len(alerts) == 0 {
		t.Fatalf("expected at least one alert from out-of-safety pulse, got 0")
	}
	if alerts[0]["severity"] != "critical" {
		t.Fatalf("expected critical, got %v", alerts[0]["severity"])
	}
	if alerts[0]["acknowledged"] != false {
		t.Fatalf("new alert should be unacknowledged")
	}
	id := alerts[0]["id"].(string)

	// Acknowledge.
	w = testutil.DoJSON(t, r, "POST", "/api/alerts/"+id+"/acknowledge", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("ack: %d %s", w.Code, w.Body.String())
	}
	w = testutil.DoJSON(t, r, "GET", "/api/alerts", token, nil)
	testutil.Decode(t, w, &alerts)
	if alerts[0]["acknowledged"] != true {
		t.Fatalf("alert should be acknowledged after POST")
	}
}

func TestMetricsNormalValueProducesNoAlert(t *testing.T) {
	r := setup(t)
	token := register(t, r, "p-noalert@test.kz")

	w := testutil.DoJSON(t, r, "POST", "/api/metrics", token, map[string]any{
		"kind": "pulse", "value": 72,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d", w.Code)
	}
	w = testutil.DoJSON(t, r, "GET", "/api/alerts", token, nil)
	var alerts []map[string]any
	testutil.Decode(t, w, &alerts)
	if len(alerts) != 0 {
		t.Fatalf("normal value should not create alerts, got %d", len(alerts))
	}
}
