package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/testutil"
)

func setupWithLogout(t *testing.T) http.Handler {
	t.Helper()
	pool := testutil.NewPool(t)
	svc := auth.NewService(pool, testSecret, 1)
	r := testutil.NewRouter()
	r.POST("/api/auth/register", svc.Register)
	r.POST("/api/auth/login", svc.Login)
	r.POST("/api/auth/logout", svc.Logout)
	api := r.Group("/api")
	api.Use(svc.Middleware())
	api.GET("/me", svc.Me)
	return r
}

func registerAndExtract(t *testing.T, r http.Handler, email string) (token string, cookie *http.Cookie) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"email": email, "password": "secret123", "full_name": "T", "role": "patient",
	})
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			cookie = c
		}
	}
	return resp.Token, cookie
}

func TestRegisterSetsAuthCookie(t *testing.T) {
	r := setupWithLogout(t)
	_, cookie := registerAndExtract(t, r, "p-cookie@test.kz")
	if cookie == nil {
		t.Fatal("expected eldercare_token cookie on register response")
	}
	if !cookie.HttpOnly {
		t.Error("cookie must be HttpOnly")
	}
	if cookie.Path != "/" {
		t.Errorf("cookie path: got %q want /", cookie.Path)
	}
	if cookie.MaxAge <= 0 {
		t.Errorf("cookie MaxAge should be positive (TTL), got %d", cookie.MaxAge)
	}
	if cookie.Value == "" {
		t.Error("cookie value is empty")
	}
}

func TestMiddlewareAcceptsCookie(t *testing.T) {
	r := setupWithLogout(t)
	_, cookie := registerAndExtract(t, r, "p-cookie-mw@test.kz")
	if cookie == nil {
		t.Fatal("no cookie returned")
	}

	// Hit /api/me using ONLY the cookie (no Authorization header).
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("/api/me with cookie should succeed: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "p-cookie-mw@test.kz") {
		t.Errorf("response should contain user email: %s", w.Body.String())
	}
}

func TestMiddlewareAcceptsBearer(t *testing.T) {
	r := setupWithLogout(t)
	token, _ := registerAndExtract(t, r, "p-bearer@test.kz")

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("/api/me with bearer should succeed: %d %s", w.Code, w.Body.String())
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	r := setupWithLogout(t)
	_, cookie := registerAndExtract(t, r, "p-logout@test.kz")

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", w.Code, w.Body.String())
	}
	var cleared *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			cleared = c
		}
	}
	if cleared == nil {
		t.Fatal("logout response should set a cookie clearing token")
	}
	if cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Errorf("cookie should be cleared (empty value, MaxAge<=0), got value=%q MaxAge=%d", cleared.Value, cleared.MaxAge)
	}
}

func TestMiddlewareRejectsNoCredential(t *testing.T) {
	r := setupWithLogout(t)
	req := httptest.NewRequest("GET", "/api/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no token should be 401, got %d", w.Code)
	}
}
