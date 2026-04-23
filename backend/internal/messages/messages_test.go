package messages_test

import (
	"net/http"
	"testing"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/links"
	"eldercare/backend/internal/messages"
	"eldercare/backend/internal/testutil"
)

const testSecret = "test_secret_please_dont_use_in_prod"

func setup(t *testing.T) http.Handler {
	t.Helper()
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, testSecret, 1)
	msgSvc := messages.NewService(pool)
	linksSvc := links.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.POST("/messages", msgSvc.Send)
	api.GET("/messages/:otherID", msgSvc.Thread)
	api.POST("/patients/link", linksSvc.Link)
	return r
}

func register(t *testing.T, r http.Handler, email, role string) (token, id, invite string) {
	t.Helper()
	w := testutil.DoJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": email, "password": "secret123", "full_name": "T", "role": role,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register %s: %d %s", email, w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
		User  struct {
			ID         string  `json:"id"`
			InviteCode *string `json:"invite_code"`
		} `json:"user"`
	}
	testutil.Decode(t, w, &resp)
	if resp.User.InviteCode != nil {
		invite = *resp.User.InviteCode
	}
	return resp.Token, resp.User.ID, invite
}

func TestMessagesRequireLink(t *testing.T) {
	r := setup(t)
	pToken, pID, _ := register(t, r, "p-msg-unlinked@test.kz", "patient")
	dToken, dID, _ := register(t, r, "d-msg-unlinked@test.kz", "doctor")

	// Unlinked pair cannot message.
	w := testutil.DoJSON(t, r, "POST", "/api/messages", pToken, map[string]any{
		"recipient_id": dID,
		"body":         "hi",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("unlinked send: expected 403, got %d %s", w.Code, w.Body.String())
	}

	w = testutil.DoJSON(t, r, "GET", "/api/messages/"+pID, dToken, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("unlinked thread: expected 403, got %d", w.Code)
	}
}

func TestMessagesLinkedFlow(t *testing.T) {
	r := setup(t)
	pToken, pID, pInvite := register(t, r, "p-msg@test.kz", "patient")
	dToken, dID, _ := register(t, r, "d-msg@test.kz", "doctor")

	// Link doctor to patient.
	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", dToken, map[string]any{"invite_code": pInvite})
	if w.Code != http.StatusOK {
		t.Fatalf("link: %d %s", w.Code, w.Body.String())
	}

	// Doctor sends to patient.
	w = testutil.DoJSON(t, r, "POST", "/api/messages", dToken, map[string]any{
		"recipient_id": pID, "body": "How are you feeling?",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("doctor send: %d %s", w.Code, w.Body.String())
	}
	// Patient replies.
	w = testutil.DoJSON(t, r, "POST", "/api/messages", pToken, map[string]any{
		"recipient_id": dID, "body": "Better, thanks",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("patient reply: %d %s", w.Code, w.Body.String())
	}

	// Patient reads the thread — both messages present in chronological order.
	w = testutil.DoJSON(t, r, "GET", "/api/messages/"+dID, pToken, nil)
	var thread []map[string]any
	testutil.Decode(t, w, &thread)
	if len(thread) != 2 {
		t.Fatalf("expected 2 messages, got %d: %s", len(thread), w.Body.String())
	}
	if thread[0]["body"] != "How are you feeling?" || thread[1]["body"] != "Better, thanks" {
		t.Fatalf("order/bodies wrong: %v", thread)
	}
	// First message came from doctor, so it gets marked read after patient loads the thread.
	// The doctor's load sees his own outgoing message with no read_at surprise, and the patient's reply
	// to him will mark-as-read on his fetch:
	w = testutil.DoJSON(t, r, "GET", "/api/messages/"+pID, dToken, nil)
	testutil.Decode(t, w, &thread)
	if len(thread) != 2 {
		t.Fatalf("doctor thread length: expected 2, got %d", len(thread))
	}
}

func TestMessagesValidation(t *testing.T) {
	r := setup(t)
	token, _, _ := register(t, r, "p-val-msg@test.kz", "patient")

	// Non-UUID recipient → 400
	w := testutil.DoJSON(t, r, "POST", "/api/messages", token, map[string]any{
		"recipient_id": "not-a-uuid", "body": "x",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad uuid: expected 400, got %d", w.Code)
	}

	// Empty body → 400
	_, otherID, _ := register(t, r, "p-other-msg@test.kz", "patient")
	w = testutil.DoJSON(t, r, "POST", "/api/messages", token, map[string]any{
		"recipient_id": otherID, "body": "",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty body: expected 400, got %d", w.Code)
	}
}
