package baseline

import "math"

// downOnlyMetrics lists metrics where deviations on the high side are
// clinically irrelevant. SpO2 is the canonical example — there is no
// such thing as "too much oxygen" in this context.
var downOnlyMetrics = map[string]bool{
	"spo2": true,
}

// SafetyCheck applies absolute clinical bounds. Returns (result, fired).
// When fired is true, the result is authoritative and the rest of the
// pipeline is short-circuited. When fired is false, callers continue
// with personalized estimation.
func SafetyCheck(value float64, th Thresholds) (Result, bool) {
	switch {
	case th.CriticalLow != nil && value < *th.CriticalLow:
		return Result{
			Severity:         SeverityCritical,
			ReasonCode:       ReasonSafetyBelowMin,
			AlgorithmVersion: Version,
		}, true
	case th.CriticalHigh != nil && value > *th.CriticalHigh:
		return Result{
			Severity:         SeverityCritical,
			ReasonCode:       ReasonSafetyAboveMax,
			AlgorithmVersion: Version,
		}, true
	case th.WarnLow != nil && value < *th.WarnLow:
		return Result{
			Severity:         SeverityWarning,
			ReasonCode:       ReasonSafetyWarnLow,
			AlgorithmVersion: Version,
		}, true
	case th.WarnHigh != nil && value > *th.WarnHigh:
		return Result{
			Severity:         SeverityWarning,
			ReasonCode:       ReasonSafetyWarnHigh,
			AlgorithmVersion: Version,
		}, true
	}
	return Result{}, false
}

// Decide turns a (value, mean, std, downOnly) tuple into a personal-baseline
// severity decision. Direction-aware: for unidirectional metrics (SpO2)
// only deviations *below* the mean count.
func Decide(value, mean, std float64, downOnly bool) Result {
	res := Result{
		Severity:         SeverityNormal,
		ReasonCode:       ReasonNormal,
		Mean:             mean,
		Std:              std,
		AlgorithmVersion: Version,
	}
	if std <= 0 || math.IsNaN(std) {
		return res
	}
	dev := value - mean
	if downOnly && dev > 0 {
		return res // ignore high-side deviations on down-only metrics
	}
	z := math.Abs(dev) / std
	res.ZScore = z
	switch {
	case z >= 3:
		res.Severity = SeverityCritical
		res.ReasonCode = ReasonBaselineCrit
	case z >= 2:
		res.Severity = SeverityWarning
		res.ReasonCode = ReasonBaselineWarn
	}
	return res
}
