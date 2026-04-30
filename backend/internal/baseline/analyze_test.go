package baseline

import (
	"testing"
	"time"
)

func TestAnalyze_SafetyOverrideShortCircuits(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	in := Input{
		Kind:    "pulse",
		Value:   200,
		History: readingsAt(now, 70, 71, 72, 70, 71, 72, 70, 71, 72, 71, 70),
		Now:     now,
	}
	r := Analyze(in)
	if r.Severity != SeverityCritical || r.ReasonCode != ReasonSafetyAboveMax {
		t.Errorf("expected safety_above_max critical, got %+v", r)
	}
}

func TestAnalyze_ColdStartFallback(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	in := Input{
		Kind:    "pulse",
		Value:   72, // within safety + warn
		History: readingsAt(now, 70, 71, 72), // only 3 readings → cold start
		Now:     now,
	}
	r := Analyze(in)
	if r.Severity != SeverityNormal {
		t.Errorf("cold start with normal value should be Normal, got %v", r.Severity)
	}
	if r.ReasonCode != ReasonColdStart {
		t.Errorf("reason: got %v want %v", r.ReasonCode, ReasonColdStart)
	}
	if r.UsedHistory {
		t.Errorf("cold start should not use history")
	}
}

func TestAnalyze_PersonalBaselineFiresOnZ3(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	// 11 stable pulse readings around 70±1; current = 80 → z high vs personal baseline.
	hist := readingsAt(now, 70, 71, 70, 69, 71, 70, 71, 70, 70, 71, 70)
	in := Input{
		Kind:    "pulse",
		Value:   80,
		History: hist,
		Now:     now,
	}
	r := Analyze(in)
	if r.Severity != SeverityCritical {
		t.Errorf("expected critical for 80 vs 70±1, got %v (z=%v)", r.Severity, r.ZScore)
	}
	if r.ReasonCode != ReasonBaselineCrit {
		t.Errorf("reason: got %v want %v", r.ReasonCode, ReasonBaselineCrit)
	}
	if !r.UsedHistory {
		t.Errorf("should report used_history true")
	}
	if r.HistorySize != len(hist) {
		t.Errorf("history_size: got %v want %v", r.HistorySize, len(hist))
	}
	if r.AlgorithmVersion != Version {
		t.Errorf("algorithm_version: got %v want %v", r.AlgorithmVersion, Version)
	}
}

func TestAnalyze_PersonalBaselineQuietWhenStable(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	hist := readingsAt(now, 70, 71, 70, 69, 71, 70, 71, 70, 70, 71, 70)
	in := Input{
		Kind:    "pulse",
		Value:   71, // within personal baseline
		History: hist,
		Now:     now,
	}
	r := Analyze(in)
	if r.Severity != SeverityNormal {
		t.Errorf("expected normal, got %v", r.Severity)
	}
}

func TestAnalyze_ConditionProfileWidensSafetyForChronic(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	// BP-sys = 155: above default warn (150), below hypertensive warn (170).
	// History spans 120-150 so personal-baseline z-score does NOT fire on 155.
	hist := readingsAt(now, 120, 135, 145, 130, 140, 125, 150, 130, 135, 140, 145, 130)
	defaultIn := Input{
		Kind:    "bp_sys",
		Value:   155,
		History: hist,
		Profile: Profile{},
		Now:     now,
	}
	hyperIn := defaultIn
	hyperIn.Profile = Profile{Hypertension: true}

	defaultR := Analyze(defaultIn)
	hyperR := Analyze(hyperIn)

	// Default profile patient gets the nuisance alert at 155 (above 150 warn).
	if defaultR.Severity != SeverityWarning {
		t.Errorf("default profile bp_sys=155 SHOULD fire warn (above 150): got %v reason=%v z=%v",
			defaultR.Severity, defaultR.ReasonCode, defaultR.ZScore)
	}
	// Hypertensive patient: 155 is normal-for-them, should NOT fire.
	if hyperR.Severity != SeverityNormal {
		t.Errorf("hypertensive profile bp_sys=155 should NOT fire (under widened 170): got %v reason=%v",
			hyperR.Severity, hyperR.ReasonCode)
	}
}

func TestAnalyze_SpO2DownOnly(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	hist := readingsAt(now, 97, 98, 97, 98, 97, 98, 97, 97, 98, 97, 97)
	// 99 is high but spo2 is down-only and within safety
	r := Analyze(Input{Kind: "spo2", Value: 99, History: hist, Now: now})
	if r.Severity != SeverityNormal {
		t.Errorf("high spo2 should be normal (down-only), got %v", r.Severity)
	}
	// Personal-baseline dip without crossing safety: 92 vs 97±1 → z=5 → critical
	r = Analyze(Input{Kind: "spo2", Value: 92, History: hist, Now: now})
	if r.Severity == SeverityNormal {
		t.Errorf("low spo2 dip should fire, got %v", r.Severity)
	}
}

func TestAnalyze_DefaultsFillIn(t *testing.T) {
	r := Analyze(Input{Kind: "pulse", Value: 72})
	if r.AlgorithmVersion != Version {
		t.Errorf("version not set: %v", r.AlgorithmVersion)
	}
	if r.Estimator == "" {
		t.Errorf("estimator not defaulted: %v", r.Estimator)
	}
}
