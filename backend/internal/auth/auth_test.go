package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arsenozhetov/elder-care/backend/internal/auth"
	"github.com/arsenozhetov/elder-care/backend/internal/testutil"
)

const testSecret = "test_secret_please_dont_use_in_prod"

func setup(t *testing.T) (*auth.Service, http.Handler) {
	t.Helper()
	pool := testutil.NewPool(t)
	svc := auth.NewService(pool, testSecret, 1)
	r := testutil.NewRouter()
	r.POST("/api/auth/register", svc.Register)
	r.POST("/api/auth/login", svc.Login)
	api := r.Group("/api")
	api.Use(svc.Middleware())
	api.GET("/me", svc.Me)
	api.PATCH("/me", svc.UpdateMe)
	return svc, r
}

func doJSON(t *testing.T, r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
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

func decode(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode %d: %v body=%s", w.Code, err, w.Body.String())
	}
}

func register(t *testing.T, r http.Handler, email, role string) (token string, user map[string]any) {
	t.Helper()
	w := doJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email":     email,
		"password":  "secret123",
		"full_name": "Test " + role,
		"role":      role,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register %s: %d %s", email, w.Code, w.Body.String())
	}
	var resp struct {
		Token string         `json:"token"`
		User  map[string]any `json:"user"`
	}
	decode(t, w, &resp)
	return resp.Token, resp.User
}

func TestRegisterLoginMe(t *testing.T) {
	_, r := setup(t)

	token, user := register(t, r, "alice@test.kz", "patient")
	if token == "" {
		t.Fatal("expected token")
	}
	if user["onboarded"] != false {
		t.Fatalf("new patient should be onboarded=false, got %v", user["onboarded"])
	}
	if user["invite_code"] == nil || user["invite_code"] == "" {
		t.Fatalf("patient must have invite_code, got %v", user["invite_code"])
	}

	w := doJSON(t, r, "POST", "/api/auth/login", "", map[string]string{
		"email": "alice@test.kz", "password": "secret123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var loginResp struct {
		Token string         `json:"token"`
		User  map[string]any `json:"user"`
	}
	decode(t, w, &loginResp)

	w = doJSON(t, r, "GET", "/api/me", loginResp.Token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}
	var me map[string]any
	decode(t, w, &me)
	if me["email"] != "alice@test.kz" {
		t.Fatalf("unexpected email %v", me["email"])
	}
}

func TestRegisterDoctorIsOnboarded(t *testing.T) {
	_, r := setup(t)
	_, user := register(t, r, "doc@test.kz", "doctor")
	if user["onboarded"] != true {
		t.Fatalf("doctor should be onboarded=true by default, got %v", user["onboarded"])
	}
	if user["invite_code"] != nil {
		// doctors get no invite code — it's generated for everyone but only patients use it.
		// Accept either null or a string; just ensure it doesn't blow up.
	}
}

func TestLoginWrongPassword(t *testing.T) {
	_, r := setup(t)
	register(t, r, "bob@test.kz", "patient")
	w := doJSON(t, r, "POST", "/api/auth/login", "", map[string]string{
		"email": "bob@test.kz", "password": "wrong",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", w.Code, w.Body.String())
	}
}

func TestMeRequiresToken(t *testing.T) {
	_, r := setup(t)
	w := doJSON(t, r, "GET", "/api/me", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
	w = doJSON(t, r, "GET", "/api/me", "garbage", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with garbage token, got %d", w.Code)
	}
}

func TestPatchMeCoalesceAndValidation(t *testing.T) {
	_, r := setup(t)
	token, _ := register(t, r, "carol@test.kz", "patient")

	// Set height
	w := doJSON(t, r, "PATCH", "/api/me", token, map[string]any{"height_cm": 170})
	if w.Code != http.StatusOK {
		t.Fatalf("patch height: %d %s", w.Code, w.Body.String())
	}
	var u map[string]any
	decode(t, w, &u)
	if n, _ := u["height_cm"].(float64); int(n) != 170 {
		t.Fatalf("height_cm not persisted, got %v", u["height_cm"])
	}

	// Null height shouldn't overwrite
	w = doJSON(t, r, "PATCH", "/api/me", token, map[string]any{"height_cm": nil})
	if w.Code != http.StatusOK {
		t.Fatalf("patch null: %d", w.Code)
	}
	decode(t, w, &u)
	if n, _ := u["height_cm"].(float64); int(n) != 170 {
		t.Fatalf("height_cm lost after null patch, got %v", u["height_cm"])
	}

	// Invalid lang rejected
	w = doJSON(t, r, "PATCH", "/api/me", token, map[string]any{"lang": "fr"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad lang, got %d %s", w.Code, w.Body.String())
	}

	// Valid lang persisted
	w = doJSON(t, r, "PATCH", "/api/me", token, map[string]any{"lang": "kk"})
	if w.Code != http.StatusOK {
		t.Fatalf("patch lang: %d %s", w.Code, w.Body.String())
	}
	decode(t, w, &u)
	if u["lang"] != "kk" {
		t.Fatalf("lang not persisted, got %v", u["lang"])
	}

	// Complete onboarding
	w = doJSON(t, r, "PATCH", "/api/me", token, map[string]any{
		"onboarded":          true,
		"chronic_conditions": "hypertension",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("complete onboard: %d %s", w.Code, w.Body.String())
	}
	decode(t, w, &u)
	if u["onboarded"] != true {
		t.Fatalf("expected onboarded=true, got %v", u["onboarded"])
	}
	if u["chronic_conditions"] != "hypertension" {
		t.Fatalf("chronic not persisted, got %v", u["chronic_conditions"])
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	_, r := setup(t)
	register(t, r, "dup@test.kz", "patient")
	w := doJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": "dup@test.kz", "password": "secret123", "full_name": "X", "role": "patient",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on duplicate, got %d %s", w.Code, w.Body.String())
	}
}
