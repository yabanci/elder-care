package baseline

import (
	"math"
	"testing"
	"time"
)

const float64Tolerance = 1e-6

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < float64Tolerance
}

// readingsAt produces readings spaced one day apart, starting `now-len(values)*day`.
func readingsAt(now time.Time, values ...float64) []Reading {
	out := make([]Reading, len(values))
	for i, v := range values {
		out[i] = Reading{
			Value:      v,
			MeasuredAt: now.Add(-time.Duration(len(values)-i) * 24 * time.Hour),
		}
	}
	return out
}

func TestMeanStd_PopulationVariance(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	// Values 2,4,4,4,5,5,7,9: mean=5, popVar=4 (sd=2), sampleVar=32/7
	in := readingsAt(now, 2, 4, 4, 4, 5, 5, 7, 9)
	mean, std := EstimateMeanStd(in, now)
	if !approxEqual(mean, 5.0) {
		t.Errorf("mean: got %v want 5", mean)
	}
	if !approxEqual(std, 2.0) {
		t.Errorf("std (population): got %v want 2", std)
	}
}

func TestMedianMAD(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	// Values 1,2,3,4,5: median=3, |x-med|=[2,1,0,1,2], MAD=1, sigma=1.4826
	in := readingsAt(now, 1, 2, 3, 4, 5)
	med, sigma := EstimateMedianMAD(in, now)
	if !approxEqual(med, 3.0) {
		t.Errorf("median: got %v want 3", med)
	}
	if !approxEqual(sigma, 1.4826) {
		t.Errorf("MAD-sigma: got %v want 1.4826", sigma)
	}
}

func TestMedianMAD_RobustToOutliers(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	// Same as above plus a 1000 outlier; median is 3, MAD-sigma still ~1.4826.
	in := readingsAt(now, 1, 2, 3, 4, 5, 1000)
	med, sigma := EstimateMedianMAD(in, now)
	if med < 2 || med > 4.5 {
		t.Errorf("median should resist outlier, got %v", med)
	}
	if sigma > 5.0 {
		t.Errorf("MAD-sigma should resist outlier, got %v", sigma)
	}
}

func TestEWMA_SampleVariance(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	// With a very long half-life relative to spacing, EWMA approaches
	// the unweighted sample mean+std. Tolerance accounts for the small
	// residual decay across the 1-day-apart inputs.
	in := readingsAt(now, 2, 4, 4, 4, 5, 5, 7, 9)
	mean, std := EstimateEWMA(in, now, 100000.0)
	if math.Abs(mean-5.0) > 0.01 {
		t.Errorf("mean: got %v want ≈5", mean)
	}
	// Sample sd of [2,4,4,4,5,5,7,9] is sqrt(32/7) ≈ 2.13809.
	if math.Abs(std-2.13809) > 0.01 {
		t.Errorf("sample std: got %v want ≈2.138", std)
	}
}

func TestEWMA_RecentValuesWeightHeavier(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	// Old value 100, recent values cluster around 70.
	// Short half-life → EWMA mean closer to 70 than 100.
	in := []Reading{
		{Value: 100, MeasuredAt: now.Add(-30 * 24 * time.Hour)},
		{Value: 70, MeasuredAt: now.Add(-2 * 24 * time.Hour)},
		{Value: 71, MeasuredAt: now.Add(-1 * 24 * time.Hour)},
		{Value: 70, MeasuredAt: now},
	}
	mean, _ := EstimateEWMA(in, now, 7.0)
	if mean > 80 {
		t.Errorf("expected EWMA closer to recent ~70, got %v", mean)
	}
}

func TestEWMAMAD_RobustResponsive(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	in := readingsAt(now, 70, 71, 72, 70, 71, 72, 70, 71, 72, 71)
	mean, sigma := EstimateEWMAMAD(in, now, 7.0)
	if math.Abs(mean-71) > 1.5 {
		t.Errorf("EWMA-MAD mean off: got %v expected ~71", mean)
	}
	if sigma <= 0 || sigma > 3.0 {
		t.Errorf("EWMA-MAD sigma unreasonable: got %v", sigma)
	}
}

func TestEstimate_DispatchesByKind(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	in := readingsAt(now, 70, 72, 71, 70, 72, 71, 70)

	for _, kind := range []EstimatorKind{EstMeanStd, EstMedianMAD, EstEWMA, EstEWMAMAD} {
		t.Run(string(kind), func(t *testing.T) {
			mean, std := Estimate(kind, in, now)
			if math.IsNaN(mean) || math.IsNaN(std) {
				t.Errorf("NaN result for %s: mean=%v std=%v", kind, mean, std)
			}
			if std < 0 {
				t.Errorf("negative std for %s: %v", kind, std)
			}
		})
	}
}

func TestEstimate_EmptyHistory(t *testing.T) {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	mean, std := Estimate(EstEWMAMAD, nil, now)
	if mean != 0 || std != 0 {
		t.Errorf("empty history should return zeros, got mean=%v std=%v", mean, std)
	}
}
