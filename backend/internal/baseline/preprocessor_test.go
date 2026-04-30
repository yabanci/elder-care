package baseline

import (
	"math"
	"testing"
	"time"
)

func makeReadings(values ...float64) []Reading {
	out := make([]Reading, len(values))
	base := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	for i, v := range values {
		out[i] = Reading{Value: v, MeasuredAt: base.Add(time.Duration(i) * time.Hour)}
	}
	return out
}

func TestHampel_PassesThroughCleanSeries(t *testing.T) {
	in := makeReadings(70, 71, 72, 70, 71, 72, 70, 71, 72)
	out := Hampel(in, 5, 3.0)
	if len(out) != len(in) {
		t.Fatalf("length mismatch: got %d, want %d", len(out), len(in))
	}
	for i := range in {
		if math.Abs(in[i].Value-out[i].Value) > 1e-9 {
			t.Errorf("idx %d: clean value mutated: %v → %v", i, in[i].Value, out[i].Value)
		}
	}
}

func TestHampel_ReplacesSingleOutlierWithLocalMedian(t *testing.T) {
	// Tight cluster around 70, one wild outlier in the middle.
	in := makeReadings(70, 71, 72, 200, 71, 70, 72)
	out := Hampel(in, 5, 3.0)
	if len(out) != len(in) {
		t.Fatalf("length mismatch")
	}
	if out[3].Value > 100 {
		t.Errorf("outlier 200 was not replaced: got %v", out[3].Value)
	}
	// Local median in window [70,71,72,200,71] is 71. Replacement should equal that.
	if math.Abs(out[3].Value-71) > 1e-9 {
		t.Errorf("expected local median 71, got %v", out[3].Value)
	}
}

func TestHampel_LeavesEdgesIntactWhenWindowDoesNotFit(t *testing.T) {
	in := makeReadings(200, 70, 70, 70, 70)
	out := Hampel(in, 5, 3.0)
	// Edge index 0 has too few neighbors on the left; fallback is to leave it alone.
	if math.Abs(out[0].Value-200) > 1e-9 {
		t.Errorf("expected edge value preserved (no enough window), got %v", out[0].Value)
	}
}

func TestHampel_SmallSeriesIsNoop(t *testing.T) {
	in := makeReadings(70, 71, 72)
	out := Hampel(in, 5, 3.0)
	for i := range in {
		if in[i].Value != out[i].Value {
			t.Errorf("small series mutated at %d", i)
		}
	}
}

func TestHampel_NilInputReturnsNil(t *testing.T) {
	if got := Hampel(nil, 5, 3.0); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestHampel_PreservesTimestamps(t *testing.T) {
	in := makeReadings(70, 71, 72, 200, 71, 70, 72)
	out := Hampel(in, 5, 3.0)
	for i := range in {
		if !out[i].MeasuredAt.Equal(in[i].MeasuredAt) {
			t.Errorf("idx %d: timestamp mutated", i)
		}
	}
}
