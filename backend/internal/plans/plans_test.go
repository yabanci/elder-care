package plans_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/plans"
	"eldercare/backend/internal/testutil"
)

const testSecret = "test_secret_please_dont_use_in_prod"

func setup(t *testing.T) (*auth.Service, http.Handler) {
	t.Helper()
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, testSecret, 1)
	plansSvc := plans.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	r.POST("/api/auth/login", authSvc.Login)

	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.GET("/me", authSvc.Me)

	// Same layout as cmd/server/main.go — guarded group.
	plansGroup := api.Group("/plans", auth.RequireRole("patient"))
	plansGroup.GET("", plansSvc.List)
	plansGroup.POST("", plansSvc.Create)
	plansGroup.PATCH("/:id", plansSvc.Update)
	plansGroup.DELETE("/:id", plansSvc.Delete)

	return authSvc, r
}

func doJSON(t *testing.T, r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func registerAndToken(t *testing.T, r http.Handler, email, role string) string {
	t.Helper()
	w := doJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": email, "password": "secret123", "full_name": "T " + role, "role": role,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register %s: %d %s", email, w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Token
}

func TestPlansCRUD(t *testing.T) {
	_, r := setup(t)
	token := registerAndToken(t, r, "patient-crud@test.kz", "patient")

	// Initially empty.
	w := doJSON(t, r, "GET", "/api/plans", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list empty: %d %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "[]" {
		t.Fatalf("expected empty list, got %s", w.Body.String())
	}

	// Create.
	w = doJSON(t, r, "POST", "/api/plans", token, map[string]any{
		"day_of_week": 2,
		"title":       "Cardiologist visit",
		"plan_type":   "doctor_visit",
		"time_of_day": "10:00",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing id in create response")
	}

	// List now returns one.
	w = doJSON(t, r, "GET", "/api/plans", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d", w.Code)
	}
	var items []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 || items[0]["title"] != "Cardiologist visit" {
		t.Fatalf("unexpected list: %v", items)
	}

	// Update title.
	w = doJSON(t, r, "PATCH", "/api/plans/"+id, token, map[string]any{"title": "Updated"})
	if w.Code != http.StatusNoContent {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, "GET", "/api/plans", token, nil)
	_ = json.Unmarshal(w.Body.Bytes(), &items)
	if items[0]["title"] != "Updated" {
		t.Fatalf("update not persisted: %v", items[0])
	}

	// Delete.
	w = doJSON(t, r, "DELETE", "/api/plans/"+id, token, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", w.Code)
	}

	// 404 on repeated delete.
	w = doJSON(t, r, "DELETE", "/api/plans/"+id, token, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing, got %d", w.Code)
	}
}

func TestPlansValidation(t *testing.T) {
	_, r := setup(t)
	token := registerAndToken(t, r, "patient-val@test.kz", "patient")

	cases := []struct {
		name string
		body map[string]any
	}{
		{"day too high", map[string]any{"day_of_week": 7, "title": "x", "plan_type": "rest"}},
		{"day negative", map[string]any{"day_of_week": -1, "title": "x", "plan_type": "rest"}},
		{"unknown type", map[string]any{"day_of_week": 1, "title": "x", "plan_type": "bogus"}},
		{"missing title", map[string]any{"day_of_week": 1, "plan_type": "rest"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := doJSON(t, r, "POST", "/api/plans", token, c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestPlansRoleGuard(t *testing.T) {
	_, r := setup(t)
	docToken := registerAndToken(t, r, "doc-plans@test.kz", "doctor")
	famToken := registerAndToken(t, r, "fam-plans@test.kz", "family")

	// Doctor and family must be blocked with 403 — the group requires role=patient.
	for _, tc := range []struct {
		role  string
		token string
	}{{"doctor", docToken}, {"family", famToken}} {
		t.Run(tc.role, func(t *testing.T) {
			w := doJSON(t, r, "GET", "/api/plans", tc.token, nil)
			if w.Code != http.StatusForbidden {
				t.Fatalf("%s GET: expected 403, got %d %s", tc.role, w.Code, w.Body.String())
			}
			w = doJSON(t, r, "POST", "/api/plans", tc.token, map[string]any{
				"day_of_week": 1, "title": "x", "plan_type": "rest",
			})
			if w.Code != http.StatusForbidden {
				t.Fatalf("%s POST: expected 403, got %d", tc.role, w.Code)
			}
		})
	}
}

func TestPlansCrossPatientIsolation(t *testing.T) {
	_, r := setup(t)
	tokenA := registerAndToken(t, r, "pa@test.kz", "patient")
	tokenB := registerAndToken(t, r, "pb@test.kz", "patient")

	// A creates plan
	w := doJSON(t, r, "POST", "/api/plans", tokenA, map[string]any{
		"day_of_week": 1, "title": "A's plan", "plan_type": "other",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("A create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// B should not see A's plan
	w = doJSON(t, r, "GET", "/api/plans", tokenB, nil)
	if w.Body.String() != "[]" {
		t.Fatalf("B should see empty list, got %s", w.Body.String())
	}

	// B cannot update A's plan
	w = doJSON(t, r, "PATCH", "/api/plans/"+id, tokenB, map[string]any{"title": "hijack"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("B update should 404, got %d %s", w.Code, w.Body.String())
	}

	// B cannot delete A's plan
	w = doJSON(t, r, "DELETE", "/api/plans/"+id, tokenB, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("B delete should 404, got %d", w.Code)
	}

	// A's plan still intact
	w = doJSON(t, r, "GET", "/api/plans", tokenA, nil)
	var items []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) != 1 || items[0]["title"] != "A's plan" {
		t.Fatalf("A's plan tampered: %v", items)
	}
}
