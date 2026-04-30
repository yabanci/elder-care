package baseline

import "testing"

func TestSafetyCheck_BelowMin(t *testing.T) {
	th := defaultThresholds["pulse"]
	res, ok := SafetyCheck(30.0, th)
	if !ok {
		t.Fatalf("expected safety to fire for pulse=30")
	}
	if res.Severity != SeverityCritical {
		t.Errorf("severity: got %v want critical", res.Severity)
	}
	if res.ReasonCode != ReasonSafetyBelowMin {
		t.Errorf("reason: got %v want %v", res.ReasonCode, ReasonSafetyBelowMin)
	}
}

func TestSafetyCheck_AboveMax(t *testing.T) {
	th := defaultThresholds["pulse"]
	res, ok := SafetyCheck(160.0, th)
	if !ok || res.Severity != SeverityCritical || res.ReasonCode != ReasonSafetyAboveMax {
		t.Errorf("got %+v ok=%v", res, ok)
	}
}

func TestSafetyCheck_WarnHigh(t *testing.T) {
	th := defaultThresholds["pulse"]
	res, ok := SafetyCheck(120.0, th)
	if !ok || res.Severity != SeverityWarning || res.ReasonCode != ReasonSafetyWarnHigh {
		t.Errorf("got %+v ok=%v", res, ok)
	}
}

func TestSafetyCheck_NoFireInsideBounds(t *testing.T) {
	th := defaultThresholds["pulse"]
	if _, ok := SafetyCheck(72.0, th); ok {
		t.Errorf("expected safety not to fire for pulse=72")
	}
}

func TestDecide_NormalWithinZ(t *testing.T) {
	res := Decide(72, 70, 5, false)
	if res.Severity != SeverityNormal {
		t.Errorf("got %v want normal", res.Severity)
	}
	if res.ReasonCode != ReasonNormal {
		t.Errorf("reason: got %v want %v", res.ReasonCode, ReasonNormal)
	}
}

func TestDecide_WarnAtZ2(t *testing.T) {
	// value=80, mean=70, std=5 → z=2 exactly.
	res := Decide(80, 70, 5, false)
	if res.Severity != SeverityWarning {
		t.Errorf("z=2 expected warning, got %v", res.Severity)
	}
	if res.ReasonCode != ReasonBaselineWarn {
		t.Errorf("reason: got %v want %v", res.ReasonCode, ReasonBaselineWarn)
	}
	if res.ZScore < 1.99 || res.ZScore > 2.01 {
		t.Errorf("zscore: got %v want ≈2", res.ZScore)
	}
}

func TestDecide_CritAtZ3(t *testing.T) {
	// value=85, mean=70, std=5 → z=3.
	res := Decide(85, 70, 5, false)
	if res.Severity != SeverityCritical {
		t.Errorf("z=3 expected critical, got %v", res.Severity)
	}
	if res.ReasonCode != ReasonBaselineCrit {
		t.Errorf("reason: got %v want %v", res.ReasonCode, ReasonBaselineCrit)
	}
}

func TestDecide_DownOnlyMetric_OnlyFiresOnDip(t *testing.T) {
	// SpO2: only the low side counts. value=99, mean=96, std=1 → z=3 high
	// but should NOT fire because direction is down-only.
	res := Decide(99, 96, 1, true)
	if res.Severity != SeverityNormal {
		t.Errorf("down-only high deviation should be normal, got %v", res.Severity)
	}
	// value=92, mean=96, std=1 → z=4 low → critical.
	res = Decide(92, 96, 1, true)
	if res.Severity != SeverityCritical {
		t.Errorf("down-only low deviation z=4 should be critical, got %v", res.Severity)
	}
}

func TestDecide_StdZeroIsNormal(t *testing.T) {
	res := Decide(80, 80, 0, false)
	if res.Severity != SeverityNormal {
		t.Errorf("std=0 should be normal, got %v", res.Severity)
	}
}
