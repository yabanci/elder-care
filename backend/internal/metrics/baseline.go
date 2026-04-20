package metrics

import "math"

// BaselineResult reports whether a new value deviates from a patient's personal baseline.
// Severity rules (z = |value - mean| / std):
//
//	z <  2   → normal
//	2 ≤ z< 3 → warning
//	z ≥ 3    → critical
//
// Absolute safety thresholds always override baseline (e.g. pulse > 140 is critical regardless).
type BaselineResult struct {
	Severity    string  // "normal" | "warning" | "critical"
	Reason      string
	Mean        float64
	Std         float64
	ZScore      float64
	UsedHistory bool
}

type Thresholds struct {
	CriticalLow  *float64
	CriticalHigh *float64
	WarningLow   *float64
	WarningHigh  *float64
}

func f(v float64) *float64 { return &v }

// SafetyLimits defines absolute medical bounds. These fire even with no history.
var SafetyLimits = map[string]Thresholds{
	"pulse":       {CriticalLow: f(40), CriticalHigh: f(140), WarningLow: f(50), WarningHigh: f(110)},
	"bp_sys":      {CriticalLow: f(80), CriticalHigh: f(180), WarningLow: f(100), WarningHigh: f(150)},
	"bp_dia":      {CriticalLow: f(50), CriticalHigh: f(110), WarningLow: f(60), WarningHigh: f(95)},
	"glucose":     {CriticalLow: f(3.0), CriticalHigh: f(15.0), WarningLow: f(4.0), WarningHigh: f(10.0)},
	"temperature": {CriticalLow: f(35.0), CriticalHigh: f(39.0), WarningLow: f(35.5), WarningHigh: f(37.8)},
	"spo2":        {CriticalLow: f(88), WarningLow: f(93)},
	"weight":      {},
}

// Analyze returns the severity of a reading given the patient's recent history.
// history should be chronological, newest last; only values of the same kind.
func Analyze(kind string, value float64, history []float64) BaselineResult {
	res := BaselineResult{Severity: "normal"}

	if t, ok := SafetyLimits[kind]; ok {
		if t.CriticalLow != nil && value < *t.CriticalLow {
			return BaselineResult{Severity: "critical", Reason: "value below safe minimum"}
		}
		if t.CriticalHigh != nil && value > *t.CriticalHigh {
			return BaselineResult{Severity: "critical", Reason: "value above safe maximum"}
		}
		if t.WarningLow != nil && value < *t.WarningLow {
			res.Severity = "warning"
			res.Reason = "value below normal range"
		}
		if t.WarningHigh != nil && value > *t.WarningHigh {
			res.Severity = "warning"
			res.Reason = "value above normal range"
		}
	}

	if len(history) < 5 {
		return res
	}

	mean, std := meanStd(history)
	res.Mean = mean
	res.Std = std
	res.UsedHistory = true

	if std < 1e-9 {
		return res
	}

	z := math.Abs(value-mean) / std
	res.ZScore = z

	switch {
	case z >= 3:
		res.Severity = "critical"
		res.Reason = "значительное отклонение от личной нормы (z≥3)"
	case z >= 2 && res.Severity != "critical":
		if res.Severity != "warning" {
			res.Severity = "warning"
		}
		res.Reason = "отклонение от личной нормы (z≥2)"
	}
	return res
}

func meanStd(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	mean := sum / float64(len(xs))

	var sq float64
	for _, v := range xs {
		d := v - mean
		sq += d * d
	}
	std := math.Sqrt(sq / float64(len(xs)))
	return mean, std
}
