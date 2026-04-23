package links_test

import (
	"net/http"
	"testing"

	"eldercare/backend/internal/auth"
	"eldercare/backend/internal/links"
	"eldercare/backend/internal/testutil"
)

const testSecret = "test_secret_please_dont_use_in_prod"

func setup(t *testing.T) http.Handler {
	t.Helper()
	pool := testutil.NewPool(t)
	authSvc := auth.NewService(pool, testSecret, 1)
	linksSvc := links.NewService(pool)

	r := testutil.NewRouter()
	r.POST("/api/auth/register", authSvc.Register)
	api := r.Group("/api")
	api.Use(authSvc.Middleware())
	api.GET("/patients", linksSvc.MyPatients)
	api.GET("/caregivers", linksSvc.MyCaregivers)
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

func TestLinkHappyPath(t *testing.T) {
	r := setup(t)
	pToken, pID, pInvite := register(t, r, "p-link@test.kz", "patient")
	dToken, dID, _ := register(t, r, "d-link@test.kz", "doctor")

	// Doctor links via invite code.
	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", dToken, map[string]any{
		"invite_code": pInvite,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("link: %d %s", w.Code, w.Body.String())
	}

	// Doctor's /patients now lists this patient.
	w = testutil.DoJSON(t, r, "GET", "/api/patients", dToken, nil)
	var patients []map[string]any
	testutil.Decode(t, w, &patients)
	if len(patients) != 1 || patients[0]["patient_id"] != pID || patients[0]["relation"] != "doctor" {
		t.Fatalf("unexpected patients: %v", patients)
	}

	// Patient's /caregivers lists the doctor.
	w = testutil.DoJSON(t, r, "GET", "/api/caregivers", pToken, nil)
	var caregivers []map[string]any
	testutil.Decode(t, w, &caregivers)
	if len(caregivers) != 1 || caregivers[0]["id"] != dID {
		t.Fatalf("unexpected caregivers: %v", caregivers)
	}

	// Idempotent: linking again must not create a duplicate (ON CONFLICT DO NOTHING).
	w = testutil.DoJSON(t, r, "POST", "/api/patients/link", dToken, map[string]any{
		"invite_code": pInvite,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("re-link: %d", w.Code)
	}
	w = testutil.DoJSON(t, r, "GET", "/api/patients", dToken, nil)
	testutil.Decode(t, w, &patients)
	if len(patients) != 1 {
		t.Fatalf("duplicate link: expected 1 row, got %d", len(patients))
	}
}

func TestLinkInviteCodeCaseInsensitive(t *testing.T) {
	r := setup(t)
	_, _, invite := register(t, r, "p-case@test.kz", "patient")
	fToken, _, _ := register(t, r, "f-case@test.kz", "family")

	// Lower-cased invite should still resolve (handler uppercases it).
	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", fToken, map[string]any{
		"invite_code": stringToLower(invite),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("lower-case invite: %d %s", w.Code, w.Body.String())
	}
}

func TestLinkUnknownInvite(t *testing.T) {
	r := setup(t)
	dToken, _, _ := register(t, r, "d-404@test.kz", "doctor")
	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", dToken, map[string]any{
		"invite_code": "DEADBEEF",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", w.Code, w.Body.String())
	}
}

func TestLinkRejectsPatientCaller(t *testing.T) {
	r := setup(t)
	_, _, invite := register(t, r, "p-self@test.kz", "patient")
	pToken, _, _ := register(t, r, "p-caller@test.kz", "patient")

	// Patients can't use the link endpoint.
	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", pToken, map[string]any{
		"invite_code": invite,
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", w.Code, w.Body.String())
	}
}

func TestLinkRejectsDoctorInvite(t *testing.T) {
	r := setup(t)
	// Register a second doctor solely to produce a doctor invite_code, then try to link to it.
	_, _, docInvite := register(t, r, "d-target@test.kz", "doctor")
	dToken, _, _ := register(t, r, "d-caller@test.kz", "doctor")
	if docInvite == "" {
		t.Skip("doctor has no invite code in this build")
	}
	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", dToken, map[string]any{
		"invite_code": docInvite,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (invite not a patient), got %d %s", w.Code, w.Body.String())
	}
}

func stringToLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 32
		}
		out[i] = c
	}
	return string(out)
}
