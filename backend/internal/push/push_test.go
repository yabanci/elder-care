package push_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"eldercare/backend/internal/push"
	"eldercare/backend/internal/testutil"
)

func TestPublicKey_DisabledReportsEmpty(t *testing.T) {
	pool := testutil.NewPool(t)
	svc := push.NewService(pool, "", "", "")

	r := testutil.NewRouter()
	r.GET("/api/push/public-key", svc.PublicKey)

	req := httptest.NewRequest("GET", "/api/push/public-key", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["enabled"] != false {
		t.Errorf("enabled should be false when no keys, got %v", resp["enabled"])
	}
	if resp["public_key"] != "" {
		t.Errorf("public_key should be empty, got %v", resp["public_key"])
	}
}

func TestSendToUser_DisabledIsNoop(t *testing.T) {
	pool := testutil.NewPool(t)
	svc := push.NewService(pool, "", "", "")

	// Should not panic, should not query the DB.
	svc.SendToUser(context.Background(), "00000000-0000-0000-0000-000000000000", push.AlertPayload{
		Title: "x", Body: "y", Severity: "critical",
	})
}

func TestEnabled_RequiresBothKeys(t *testing.T) {
	pool := testutil.NewPool(t)
	if push.NewService(pool, "", "", "").Enabled() {
		t.Error("empty keys → not enabled")
	}
	if push.NewService(pool, "pub", "", "").Enabled() {
		t.Error("only public → not enabled")
	}
	if !push.NewService(pool, "pub", "priv", "").Enabled() {
		t.Error("both keys → enabled")
	}
}
