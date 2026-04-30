package metrics_test

import (
	"net/http"
	"testing"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/baseline"
	"eldercare/backend/internal/metrics"
	"eldercare/backend/internal/testutil"
)

// setupWithProfile is like setup but exposes /api/me PATCH so tests can set
// chronic_conditions to drive the condition-profile branch.
func setupWithProfile(t *testing.T) http.Handler {
	t.Helper()
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, testSecret, 1)
	metricsSvc := metrics.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.PATCH("/me", authSvc.UpdateMe)
	api.POST("/metrics", metricsSvc.CreateForSelf)
	api.GET("/alerts", metricsSvc.ListAlerts)
	return r
}

// TestBaselineV2_AlertsCarryReasonCodeAndVersion verifies alerts now include
// reason_code and algorithm_version in the API payload.
func TestBaselineV2_AlertsCarryReasonCodeAndVersion(t *testing.T) {
	r := setup(t)
	token := register(t, r, "p-rcode@test.kz")

	// Pulse 200 → safety_above_max critical.
	w := testutil.DoJSON(t, r, "POST", "/api/metrics", token, map[string]any{
		"kind": "pulse", "value": 200,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	w = testutil.DoJSON(t, r, "GET", "/api/alerts", token, nil)
	var alerts []map[string]any
	testutil.Decode(t, w, &alerts)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0]["reason_code"] != string(baseline.ReasonSafetyAboveMax) {
		t.Errorf("reason_code: got %v want %v", alerts[0]["reason_code"], baseline.ReasonSafetyAboveMax)
	}
	if alerts[0]["algorithm_version"] != baseline.Version {
		t.Errorf("algorithm_version: got %v want %v", alerts[0]["algorithm_version"], baseline.Version)
	}
}

// TestBaselineV2_ColdStartProducesNoAlerts checks that a brand-new patient
// (no history) does not get false alarms from the personal baseline before
// the streak gate is satisfied.
func TestBaselineV2_ColdStartProducesNoAlerts(t *testing.T) {
	r := setup(t)
	token := register(t, r, "p-cold@test.kz")

	// 5 normal pulse readings — below the 10-reading streak gate.
	for _, v := range []float64{72, 74, 71, 73, 72} {
		w := testutil.DoJSON(t, r, "POST", "/api/metrics", token, map[string]any{
			"kind": "pulse", "value": v,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("create: %d %s", w.Code, w.Body.String())
		}
	}
	// 6th reading: 80 — within safety, but with v1 would have triggered z-score
	// because std collapses to ~1; the streak gate prevents this.
	w := testutil.DoJSON(t, r, "POST", "/api/metrics", token, map[string]any{
		"kind": "pulse", "value": 80,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create 80: %d %s", w.Code, w.Body.String())
	}

	w = testutil.DoJSON(t, r, "GET", "/api/alerts", token, nil)
	var alerts []map[string]any
	testutil.Decode(t, w, &alerts)
	if len(alerts) != 0 {
		t.Fatalf("cold-start should suppress baseline alerts, got %d: %s",
			len(alerts), w.Body.String())
	}
}

// TestBaselineV2_ConditionProfileNarrowsBPWarn proves Claim C: a hypertensive
// patient is alerted at bp_sys=142 while a default-profile patient is not.
func TestBaselineV2_ConditionProfileNarrowsBPWarn(t *testing.T) {
	r := setupWithProfile(t)

	// Patient A: default profile.
	defaultTok := register(t, r, "p-default@test.kz")
	// Patient B: hypertension.
	hyperTok := register(t, r, "p-hyper@test.kz")
	w := testutil.DoJSON(t, r, "PATCH", "/api/me", hyperTok, map[string]any{
		"chronic_conditions": "артериальная гипертензия",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("set chronic: %d %s", w.Code, w.Body.String())
	}

	// Both: 142 mmHg systolic, no history → cold-start path; only the
	// safety/warn-band layer sees it. Default warn high = 150, hypertension
	// narrows to 140 → only patient B should fire warning.
	for _, tok := range []string{defaultTok, hyperTok} {
		w := testutil.DoJSON(t, r, "POST", "/api/metrics", tok, map[string]any{
			"kind": "bp_sys", "value": 142,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("create: %d %s", w.Code, w.Body.String())
		}
	}

	w = testutil.DoJSON(t, r, "GET", "/api/alerts", defaultTok, nil)
	var defaultAlerts []map[string]any
	testutil.Decode(t, w, &defaultAlerts)
	if len(defaultAlerts) != 0 {
		t.Errorf("default profile: bp_sys=142 should not alert, got %d", len(defaultAlerts))
	}

	w = testutil.DoJSON(t, r, "GET", "/api/alerts", hyperTok, nil)
	var hyperAlerts []map[string]any
	testutil.Decode(t, w, &hyperAlerts)
	if len(hyperAlerts) != 1 {
		t.Fatalf("hypertensive profile: bp_sys=142 should alert, got %d: %s",
			len(hyperAlerts), w.Body.String())
	}
	if hyperAlerts[0]["severity"] != baseline.SeverityWarning {
		t.Errorf("severity: got %v want warning", hyperAlerts[0]["severity"])
	}
	if hyperAlerts[0]["reason_code"] != baseline.ReasonSafetyWarnHigh {
		t.Errorf("reason_code: got %v want %v",
			hyperAlerts[0]["reason_code"], baseline.ReasonSafetyWarnHigh)
	}
}
