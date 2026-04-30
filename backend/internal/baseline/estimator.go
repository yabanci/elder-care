package baseline

import (
	"math"
	"sort"
	"time"
)

// DefaultEWMAHalfLifeDays controls how fast EWMA forgets old readings.
// Half-life 7 days means a reading 7 days old contributes half as much as
// today's. Tunable in eval ablations; production uses this constant.
const DefaultEWMAHalfLifeDays = 7.0

// Estimate dispatches to the requested estimator. Returns (mean, sigma).
// For EWMA variants, sigma is sample-corrected (Kish effective N).
// For empty history returns (0, 0); callers should guard with len(history)>0.
func Estimate(kind EstimatorKind, history []Reading, now time.Time) (float64, float64) {
	if len(history) == 0 {
		return 0, 0
	}
	switch kind {
	case EstMeanStd:
		return EstimateMeanStd(history, now)
	case EstMedianMAD:
		return EstimateMedianMAD(history, now)
	case EstEWMA:
		return EstimateEWMA(history, now, DefaultEWMAHalfLifeDays)
	case EstEWMAMAD:
		return EstimateEWMAMAD(history, now, DefaultEWMAHalfLifeDays)
	default:
		return EstimateEWMAMAD(history, now, DefaultEWMAHalfLifeDays)
	}
}

// EstimateMeanStd uses arithmetic mean + population standard deviation
// (1/N variance). Faithfully reproduces v1-as-shipped — kept for
// ablation comparisons. New estimators use sample variance.
func EstimateMeanStd(history []Reading, _ time.Time) (float64, float64) {
	n := len(history)
	if n == 0 {
		return 0, 0
	}
	var sum float64
	for _, r := range history {
		sum += r.Value
	}
	mean := sum / float64(n)
	var sq float64
	for _, r := range history {
		d := r.Value - mean
		sq += d * d
	}
	std := math.Sqrt(sq / float64(n))
	return mean, std
}

// EstimateMedianMAD uses median + 1.4826·MAD as σ-equivalent.
// Robust against historical outliers — a single 200-bpm typo in 30
// readings does not blow up the scale estimator the way mean+SD does.
func EstimateMedianMAD(history []Reading, _ time.Time) (float64, float64) {
	n := len(history)
	if n == 0 {
		return 0, 0
	}
	values := make([]float64, n)
	for i, r := range history {
		values[i] = r.Value
	}
	sort.Float64s(values)
	med := values[n/2]
	for i := range values {
		values[i] = math.Abs(values[i] - med)
	}
	sort.Float64s(values)
	mad := values[n/2]
	return med, 1.4826 * mad
}

// EstimateEWMA computes time-decayed weighted mean and sample-corrected
// standard deviation. Weight w_i = exp(-ln(2) · Δt_days / halfLifeDays).
//
// Sample correction uses Kish's effective sample size: N_eff = (Σw)² / Σw².
// When N_eff > 1, variance is multiplied by N_eff / (N_eff - 1) to convert
// from population to sample variance. When N_eff ≤ 1 the population value
// is returned (sample correction is undefined).
func EstimateEWMA(history []Reading, now time.Time, halfLifeDays float64) (float64, float64) {
	if len(history) == 0 || halfLifeDays <= 0 {
		return 0, 0
	}
	lambda := math.Ln2 / halfLifeDays
	var sumW, sumWX, sumW2 float64
	for _, r := range history {
		dtDays := now.Sub(r.MeasuredAt).Hours() / 24.0
		if dtDays < 0 {
			dtDays = 0
		}
		w := math.Exp(-lambda * dtDays)
		sumW += w
		sumWX += w * r.Value
		sumW2 += w * w
	}
	if sumW == 0 {
		return 0, 0
	}
	mean := sumWX / sumW
	var sumWD2 float64
	for _, r := range history {
		dtDays := now.Sub(r.MeasuredAt).Hours() / 24.0
		if dtDays < 0 {
			dtDays = 0
		}
		w := math.Exp(-lambda * dtDays)
		d := r.Value - mean
		sumWD2 += w * d * d
	}
	popVar := sumWD2 / sumW
	nEff := (sumW * sumW) / sumW2
	if nEff > 1 {
		popVar *= nEff / (nEff - 1)
	}
	return mean, math.Sqrt(popVar)
}

// EstimateEWMAMAD pairs EWMA mean with EWMA-of-absolute-deviations as
// the scale estimator. Time-decayed and more robust to occasional spikes
// than EWMA's squared-deviation variance, while still responsive to
// gradual drift. This is the production default estimator.
func EstimateEWMAMAD(history []Reading, now time.Time, halfLifeDays float64) (float64, float64) {
	if len(history) == 0 || halfLifeDays <= 0 {
		return 0, 0
	}
	lambda := math.Ln2 / halfLifeDays
	var sumW, sumWX float64
	for _, r := range history {
		dtDays := now.Sub(r.MeasuredAt).Hours() / 24.0
		if dtDays < 0 {
			dtDays = 0
		}
		w := math.Exp(-lambda * dtDays)
		sumW += w
		sumWX += w * r.Value
	}
	if sumW == 0 {
		return 0, 0
	}
	mean := sumWX / sumW
	var sumWAbs float64
	for _, r := range history {
		dtDays := now.Sub(r.MeasuredAt).Hours() / 24.0
		if dtDays < 0 {
			dtDays = 0
		}
		w := math.Exp(-lambda * dtDays)
		sumWAbs += w * math.Abs(r.Value-mean)
	}
	mad := sumWAbs / sumW
	return mean, 1.4826 * mad
}
