package medications_test

import (
	"net/http"
	"testing"
	"time"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/links"
	"eldercare/backend/internal/medications"
	"eldercare/backend/internal/testutil"
)

const testSecret = "test_secret_please_dont_use_in_prod"

func setup(t *testing.T) http.Handler {
	t.Helper()
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, testSecret, 1)
	medSvc := medications.NewService(pool)
	linksSvc := links.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.POST("/medications", medSvc.Create)
	api.GET("/medications", medSvc.List)
	api.GET("/medications/today", medSvc.Today)
	api.DELETE("/medications/:id", medSvc.Deactivate)
	api.POST("/medications/:id/log", medSvc.LogDose)
	api.POST("/patients/link", linksSvc.Link)
	api.POST("/patients/:patientID/medications", medSvc.Create)
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

func TestMedicationsCRUDForPatient(t *testing.T) {
	r := setup(t)
	token, _, _ := register(t, r, "p-med@test.kz", "patient")

	w := testutil.DoJSON(t, r, "POST", "/api/medications", token, map[string]any{
		"name":         "Metformin",
		"dosage":       "500 mg",
		"times_of_day": []string{"08:00", "20:00"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	testutil.Decode(t, w, &created)
	id := created["id"].(string)

	w = testutil.DoJSON(t, r, "GET", "/api/medications", token, nil)
	var list []map[string]any
	testutil.Decode(t, w, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 med, got %d", len(list))
	}

	// Today's schedule should expand into 2 pending items.
	w = testutil.DoJSON(t, r, "GET", "/api/medications/today", token, nil)
	var schedule []map[string]any
	testutil.Decode(t, w, &schedule)
	if len(schedule) != 2 {
		t.Fatalf("expected 2 scheduled items, got %d: %s", len(schedule), w.Body.String())
	}

	// Log the first dose as taken.
	w = testutil.DoJSON(t, r, "POST", "/api/medications/"+id+"/log", token, map[string]any{
		"scheduled_at": schedule[0]["scheduled_at"],
		"status":       "taken",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("log: %d %s", w.Code, w.Body.String())
	}

	// Deactivate.
	w = testutil.DoJSON(t, r, "DELETE", "/api/medications/"+id, token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("deactivate: %d %s", w.Code, w.Body.String())
	}
	// After deactivation List filters active=true → empty.
	w = testutil.DoJSON(t, r, "GET", "/api/medications", token, nil)
	testutil.Decode(t, w, &list)
	if len(list) != 0 {
		t.Fatalf("list after deactivate: expected 0, got %d", len(list))
	}
}

func TestMedicationsCreateValidation(t *testing.T) {
	r := setup(t)
	token, _, _ := register(t, r, "p-val@test.kz", "patient")

	// Missing required name
	w := testutil.DoJSON(t, r, "POST", "/api/medications", token, map[string]any{
		"times_of_day": []string{"08:00"},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing name: expected 400, got %d", w.Code)
	}

	// Invalid start_date
	w = testutil.DoJSON(t, r, "POST", "/api/medications", token, map[string]any{
		"name":         "X",
		"times_of_day": []string{"08:00"},
		"start_date":   "not-a-date",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad date: expected 400, got %d", w.Code)
	}

	// Invalid log status
	w = testutil.DoJSON(t, r, "POST", "/api/medications", token, map[string]any{
		"name":         "Y",
		"times_of_day": []string{"08:00"},
	})
	var med map[string]any
	testutil.Decode(t, w, &med)
	id := med["id"].(string)

	w = testutil.DoJSON(t, r, "POST", "/api/medications/"+id+"/log", token, map[string]any{
		"scheduled_at": time.Now().Format(time.RFC3339),
		"status":       "bogus",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad status: expected 400, got %d", w.Code)
	}
}

func TestMedicationsDoctorAccessRequiresLink(t *testing.T) {
	r := setup(t)
	pToken, pID, pInvite := register(t, r, "p-acc@test.kz", "patient")
	dToken, _, _ := register(t, r, "d-acc@test.kz", "doctor")

	// Unlinked doctor → 403 on patient-scoped create.
	w := testutil.DoJSON(t, r, "POST", "/api/patients/"+pID+"/medications", dToken, map[string]any{
		"name":         "X",
		"times_of_day": []string{"09:00"},
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("unlinked doctor: expected 403, got %d", w.Code)
	}

	// Link doctor to patient.
	w = testutil.DoJSON(t, r, "POST", "/api/patients/link", dToken, map[string]any{
		"invite_code": pInvite,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("link: %d %s", w.Code, w.Body.String())
	}

	// Linked doctor → 200.
	w = testutil.DoJSON(t, r, "POST", "/api/patients/"+pID+"/medications", dToken, map[string]any{
		"name":         "X",
		"times_of_day": []string{"09:00"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("linked doctor create: %d %s", w.Code, w.Body.String())
	}

	// Patient still sees the med in their own list.
	w = testutil.DoJSON(t, r, "GET", "/api/medications", pToken, nil)
	var list []map[string]any
	testutil.Decode(t, w, &list)
	if len(list) != 1 {
		t.Fatalf("patient list: expected 1, got %d", len(list))
	}
}
