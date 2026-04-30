package baseline

import (
	"testing"
	"time"
)

func TestWindowFilter_DropsOldReadings(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	in := []Reading{
		{Value: 60, MeasuredAt: now.Add(-60 * 24 * time.Hour)}, // older than 30d
		{Value: 70, MeasuredAt: now.Add(-20 * 24 * time.Hour)},
		{Value: 75, MeasuredAt: now.Add(-1 * 24 * time.Hour)},
	}
	out := WindowFilter(in, now, 30, 100)
	if len(out) != 2 {
		t.Fatalf("expected 2 readings within 30d, got %d", len(out))
	}
	for _, r := range out {
		if r.Value == 60 {
			t.Errorf("60 should have been filtered (older than 30d)")
		}
	}
}

func TestWindowFilter_CapsAtMaxSamples(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	var in []Reading
	for i := 0; i < 100; i++ {
		in = append(in, Reading{
			Value:      float64(i),
			MeasuredAt: now.Add(-time.Duration(i) * time.Hour),
		})
	}
	out := WindowFilter(in, now, 30, 60)
	if len(out) != 60 {
		t.Fatalf("expected 60 readings (cap), got %d", len(out))
	}
	// The most recent should be present (newest values are 0..59 hours old).
	hasNewest := false
	for _, r := range out {
		if r.Value == 0 {
			hasNewest = true
		}
	}
	if !hasNewest {
		t.Errorf("most-recent reading was dropped")
	}
}

func TestWindowFilter_ReturnsChronological(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	in := []Reading{
		{Value: 70, MeasuredAt: now.Add(-3 * 24 * time.Hour)},
		{Value: 71, MeasuredAt: now.Add(-1 * 24 * time.Hour)},
		{Value: 72, MeasuredAt: now.Add(-2 * 24 * time.Hour)},
	}
	out := WindowFilter(in, now, 30, 100)
	for i := 1; i < len(out); i++ {
		if out[i].MeasuredAt.Before(out[i-1].MeasuredAt) {
			t.Errorf("out of order at %d: %v before %v", i, out[i].MeasuredAt, out[i-1].MeasuredAt)
		}
	}
}

func TestWindowFilter_NilInput(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	if got := WindowFilter(nil, now, 30, 60); len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestIsStable_BelowThreshold(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	in := readingsAt(now, 70, 71, 72, 70, 71, 72, 70, 71) // 8 readings
	if IsStable(in, now, 10, 14) {
		t.Errorf("8 readings should not satisfy ≥10")
	}
}

func TestIsStable_AtThreshold(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	in := readingsAt(now, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11) // 11 readings, 1d apart
	if !IsStable(in, now, 10, 14) {
		t.Errorf("11 readings within 14d should be stable")
	}
}

func TestIsStable_OutsideWindow(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	// 11 readings but ≥30d ago.
	in := []Reading{}
	for i := 0; i < 11; i++ {
		in = append(in, Reading{
			Value:      70,
			MeasuredAt: now.Add(-time.Duration(40+i) * 24 * time.Hour),
		})
	}
	if IsStable(in, now, 10, 14) {
		t.Errorf("readings outside 14d window should not count")
	}
}
