package medications_test

import (
	"net/http"
	"testing"

	"eldercare/backend/internal/testutil"
)

// TestDoctorPrescribesForPatient verifies the doctor → patient prescribe
// flow: link, then POST /api/patients/:id/medications, then patient sees
// the medication labelled with the doctor's user_id in `prescribed_by`.
func TestDoctorPrescribesForPatient(t *testing.T) {
	r := setup(t)

	patientTok, patientID, invite := register(t, r, "p-rx@test.kz", "patient")
	doctorTok, doctorID, _ := register(t, r, "d-rx@test.kz", "doctor")

	// Doctor links to patient.
	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", doctorTok, map[string]any{
		"invite_code": invite,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("link: %d %s", w.Code, w.Body.String())
	}

	// Doctor prescribes.
	w = testutil.DoJSON(t, r, "POST", "/api/patients/"+patientID+"/medications", doctorTok, map[string]any{
		"name": "Лизиноприл", "dosage": "10 мг", "times_of_day": []string{"08:00"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("prescribe: %d %s", w.Code, w.Body.String())
	}
	var med map[string]any
	testutil.Decode(t, w, &med)
	if med["prescribed_by"] != doctorID {
		t.Errorf("prescribed_by: got %v want %s", med["prescribed_by"], doctorID)
	}
	if med["prescribed_at"] == nil {
		t.Error("prescribed_at should be set")
	}

	// Patient sees the medication on their own list.
	w = testutil.DoJSON(t, r, "GET", "/api/medications", patientTok, nil)
	var list []map[string]any
	testutil.Decode(t, w, &list)
	if len(list) != 1 || list[0]["name"] != "Лизиноприл" {
		t.Fatalf("expected 1 prescribed med visible to patient, got %v", list)
	}
	if list[0]["prescribed_by"] != doctorID {
		t.Errorf("patient view: prescribed_by mismatch: %v", list[0]["prescribed_by"])
	}
}

// TestFamilyCannotPrescribe — family member is linked but role-checked
// out of medication writes.
func TestFamilyCannotPrescribe(t *testing.T) {
	r := setup(t)

	_, patientID, invite := register(t, r, "p-rx2@test.kz", "patient")
	familyTok, _, _ := register(t, r, "f-rx@test.kz", "family")

	w := testutil.DoJSON(t, r, "POST", "/api/patients/link", familyTok, map[string]any{
		"invite_code": invite,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("link: %d %s", w.Code, w.Body.String())
	}

	w = testutil.DoJSON(t, r, "POST", "/api/patients/"+patientID+"/medications", familyTok, map[string]any{
		"name": "Аспирин",
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("family prescription should be 403, got %d %s", w.Code, w.Body.String())
	}
}

// TestSelfPrescribeStaysUnattributed — the patient adding their own
// medication should NOT have prescribed_by set (the patient is implicit).
func TestSelfPrescribeStaysUnattributed(t *testing.T) {
	r := setup(t)

	patientTok, _, _ := register(t, r, "p-self-rx@test.kz", "patient")
	w := testutil.DoJSON(t, r, "POST", "/api/medications", patientTok, map[string]any{
		"name": "Витамин D",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("self-prescribe: %d %s", w.Code, w.Body.String())
	}
	var med map[string]any
	testutil.Decode(t, w, &med)
	if pb, ok := med["prescribed_by"]; ok && pb != nil {
		t.Errorf("self-prescribed med should not have prescribed_by, got %v", pb)
	}
}
