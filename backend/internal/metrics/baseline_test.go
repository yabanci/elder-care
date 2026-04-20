package metrics

import "testing"

func TestAnalyze_NoHistoryUsesSafetyLimits(t *testing.T) {
	r := Analyze("pulse", 150, nil)
	if r.Severity != "critical" {
		t.Fatalf("expected critical, got %s", r.Severity)
	}

	r = Analyze("pulse", 75, nil)
	if r.Severity != "normal" {
		t.Fatalf("expected normal, got %s", r.Severity)
	}
}

func TestAnalyze_PersonalBaselineOutlier(t *testing.T) {
	// patient's stable pulse around 72 ± ~1
	hist := []float64{72, 71, 73, 72, 70, 74, 73, 72, 71, 72}
	r := Analyze("pulse", 88, hist) // within safety but huge z-score
	if !r.UsedHistory {
		t.Fatalf("expected baseline to be used")
	}
	if r.Severity != "critical" {
		t.Fatalf("expected critical personal deviation, got %s (z=%.2f)", r.Severity, r.ZScore)
	}
}

func TestAnalyze_PersonalBaselineWarning(t *testing.T) {
	hist := []float64{120, 125, 115, 128, 118, 123, 116, 127, 119, 122}
	r := Analyze("bp_sys", 131, hist)
	if r.Severity != "warning" {
		t.Fatalf("expected warning, got %s (z=%.2f, mean=%.2f, std=%.2f)",
			r.Severity, r.ZScore, r.Mean, r.Std)
	}
}

func TestAnalyze_SafetyOverridesBaseline(t *testing.T) {
	// even if history is wild, crossing safety limit is critical
	hist := []float64{150, 155, 148, 152, 151, 149, 153}
	r := Analyze("pulse", 145, hist)
	if r.Severity != "critical" {
		t.Fatalf("expected critical from safety limit, got %s", r.Severity)
	}
}

func TestMeanStd(t *testing.T) {
	m, s := meanStd([]float64{2, 4, 4, 4, 5, 5, 7, 9})
	if m != 5 {
		t.Fatalf("mean: want 5, got %v", m)
	}
	if s < 1.9 || s > 2.1 {
		t.Fatalf("std: want ~2, got %v", s)
	}
}
