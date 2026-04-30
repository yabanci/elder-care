package baseline

import "testing"

func TestParseProfile_Russian(t *testing.T) {
	cases := []struct {
		text string
		want Profile
	}{
		{"гипертония", Profile{Hypertension: true}},
		{"артериальная гипертензия", Profile{Hypertension: true}},
		{"сахарный диабет 2 типа", Profile{T2D: true}},
		{"СД2", Profile{T2D: true}},
		{"ХОБЛ", Profile{COPD: true}},
		{"гипертония и сахарный диабет", Profile{Hypertension: true, T2D: true}},
		{"", Profile{}},
		{"нет", Profile{}},
	}
	for _, c := range cases {
		got := ParseProfile(c.text)
		if got != c.want {
			t.Errorf("ParseProfile(%q) = %+v, want %+v", c.text, got, c.want)
		}
	}
}

func TestParseProfile_Kazakh(t *testing.T) {
	cases := []struct {
		text string
		want Profile
	}{
		{"қант диабеті", Profile{T2D: true}},
		{"қан қысымы", Profile{Hypertension: true}},
	}
	for _, c := range cases {
		got := ParseProfile(c.text)
		if got != c.want {
			t.Errorf("ParseProfile(%q) = %+v, want %+v", c.text, got, c.want)
		}
	}
}

func TestParseProfile_English(t *testing.T) {
	cases := []struct {
		text string
		want Profile
	}{
		{"hypertension", Profile{Hypertension: true}},
		{"Type 2 diabetes", Profile{T2D: true}},
		{"COPD", Profile{COPD: true}},
		{"hypertension, copd", Profile{Hypertension: true, COPD: true}},
	}
	for _, c := range cases {
		got := ParseProfile(c.text)
		if got != c.want {
			t.Errorf("ParseProfile(%q) = %+v, want %+v", c.text, got, c.want)
		}
	}
}

func TestThresholdsFor_DefaultsWhenNoProfile(t *testing.T) {
	prof := Profile{}
	th := ThresholdsFor("bp_sys", prof)
	if th.WarnHigh == nil || *th.WarnHigh != 150 {
		t.Errorf("default bp_sys WarnHigh: got %v want 150", th.WarnHigh)
	}
}

func TestThresholdsFor_HypertensionWidensBPSysWarn(t *testing.T) {
	prof := Profile{Hypertension: true}
	th := ThresholdsFor("bp_sys", prof)
	if th.WarnHigh == nil || *th.WarnHigh != 170 {
		t.Errorf("hypertension bp_sys WarnHigh: got %v want 170", th.WarnHigh)
	}
}

func TestThresholdsFor_T2DLoosensGlucoseBand(t *testing.T) {
	prof := Profile{T2D: true}
	th := ThresholdsFor("glucose", prof)
	if th.WarnHigh == nil || *th.WarnHigh != 12.0 {
		t.Errorf("t2d glucose WarnHigh: got %v want 12.0", th.WarnHigh)
	}
	if th.WarnLow == nil || *th.WarnLow != 3.5 {
		t.Errorf("t2d glucose WarnLow: got %v want 3.5", th.WarnLow)
	}
}

func TestThresholdsFor_COPDLoosensSpO2(t *testing.T) {
	prof := Profile{COPD: true}
	th := ThresholdsFor("spo2", prof)
	if th.WarnLow == nil || *th.WarnLow != 90 {
		t.Errorf("copd spo2 WarnLow: got %v want 90", th.WarnLow)
	}
}

func TestThresholdsFor_MultiProfileTakesWiderBound(t *testing.T) {
	// Both hypertension (widens BP) and T2D (widens glucose). bp_sys
	// reflects hypertension's 170; glucose stays at default since T2D
	// only overrides glucose.
	prof := Profile{Hypertension: true, T2D: true}
	th := ThresholdsFor("bp_sys", prof)
	if th.WarnHigh == nil || *th.WarnHigh != 170 {
		t.Errorf("expected hypertension widen on bp_sys, got %v", th.WarnHigh)
	}
}

func TestThresholdsFor_UnknownMetricReturnsEmpty(t *testing.T) {
	th := ThresholdsFor("imaginary", Profile{})
	if th.WarnHigh != nil || th.WarnLow != nil || th.CriticalHigh != nil || th.CriticalLow != nil {
		t.Errorf("unknown metric should return empty thresholds, got %+v", th)
	}
}
