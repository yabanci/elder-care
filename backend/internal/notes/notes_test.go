package notes_test

import (
	"net/http"
	"testing"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/links"
	"eldercare/backend/internal/notes"
	"eldercare/backend/internal/testutil"
)

const notesSecret = "notes_test_secret"

func setup(t *testing.T) http.Handler {
	t.Helper()
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, notesSecret, 1)
	notesSvc := notes.NewService(pool)
	linksSvc := links.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.POST("/patients/link", linksSvc.Link)
	api.POST("/patients/:patientID/notes", notesSvc.Create)
	api.GET("/patients/:patientID/notes", notesSvc.List)
	return r
}

func register(t *testing.T, r http.Handler, email, role string) (token, id, invite string) {
	t.Helper()
	w := testutil.DoJSON(t, r, "POST", "/api/auth/register", "", map[string]string{
		"email": email, "password": "secret123", "full_name": "Test User", "role": role,
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

func TestNotes_DoctorCreatesAndListsForPatient(t *testing.T) {
	r := setup(t)
	patientTok, patientID, invite := register(t, r, "p-notes@test.kz", "patient")
	doctorTok, doctorID, _ := register(t, r, "d-notes@test.kz", "doctor")

	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", doctorTok, map[string]any{
		"invite_code": invite,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("link: %d %s", w.Code, w.Body.String())
	}

	w = testutil.DoJSON(t, r, "POST", "/api/patients/"+patientID+"/notes", doctorTok, map[string]any{
		"body": "Switched to morning dose",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create note: %d %s", w.Code, w.Body.String())
	}
	var note map[string]any
	testutil.Decode(t, w, &note)
	if note["author_id"] != doctorID {
		t.Errorf("author_id: got %v want %s", note["author_id"], doctorID)
	}
	if note["author_role"] != "doctor" {
		t.Errorf("author_role: got %v want doctor", note["author_role"])
	}

	// Patient sees the note about themselves.
	w = testutil.DoJSON(t, r, "GET", "/api/patients/"+patientID+"/notes", patientTok, nil)
	var list []map[string]any
	testutil.Decode(t, w, &list)
	if len(list) != 1 || list[0]["body"] != "Switched to morning dose" {
		t.Fatalf("expected 1 note visible to patient, got %v", list)
	}
}

func TestNotes_PatientCannotAuthor(t *testing.T) {
	r := setup(t)
	patientTok, patientID, _ := register(t, r, "p-notes2@test.kz", "patient")

	w := testutil.DoJSON(t, r, "POST", "/api/patients/"+patientID+"/notes", patientTok, map[string]any{
		"body": "Self-note attempt",
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("patient self-author should be 403, got %d %s", w.Code, w.Body.String())
	}
}

func TestNotes_UnlinkedDoctorCannotAuthor(t *testing.T) {
	r := setup(t)
	_, patientID, _ := register(t, r, "p-notes3@test.kz", "patient")
	doctorTok, _, _ := register(t, r, "d-notes2@test.kz", "doctor") // not linked

	w := testutil.DoJSON(t, r, "POST", "/api/patients/"+patientID+"/notes", doctorTok, map[string]any{
		"body": "Unauthorized",
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("unlinked doctor should be 403, got %d %s", w.Code, w.Body.String())
	}
}

func TestNotes_RejectsEmptyBody(t *testing.T) {
	r := setup(t)
	patientTok, patientID, invite := register(t, r, "p-notes4@test.kz", "patient")
	doctorTok, _, _ := register(t, r, "d-notes3@test.kz", "doctor")
	_ = patientTok
	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", doctorTok, map[string]any{
		"invite_code": invite,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("link: %d", w.Code)
	}

	w = testutil.DoJSON(t, r, "POST", "/api/patients/"+patientID+"/notes", doctorTok, map[string]any{
		"body": "",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty body should be 400, got %d %s", w.Code, w.Body.String())
	}
}
