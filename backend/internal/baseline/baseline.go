// Package baseline implements the v2 personal-baseline alert algorithm
// for the ElderCare master's-thesis MVP. It replaces the v1 algorithm in
// internal/metrics/baseline.go and is structured as a layered pipeline so
// each contribution (preprocessing, robust estimator, time-aware window,
// stable-streak gate, condition profile, decision rule, safety override)
// can be evaluated independently in offline ablation studies.
//
// The package is pure: no DB, no I/O. Persistence and HTTP wiring live in
// internal/metrics. Long-form rationale: docs/superpowers/specs/2026-05-01-thesis-baseline-v2-design.md
package baseline

import "time"

// Version is recorded on every alert and algorithm_runs row so that an
// offline replay can tell which algorithm produced a given result.
const Version = "v2"

// Reading is a single historical sample fed into the algorithm.
type Reading struct {
	Value      float64   `json:"value"`
	MeasuredAt time.Time `json:"measured_at"`
}

// Profile is the set of chronic-condition profiles a patient matches. A
// patient may have several; thresholds compose by taking the narrower
// bound per metric (more sensitive — see condition_profile.go).
type Profile struct {
	Hypertension bool `json:"hypertension"`
	T2D          bool `json:"t2d"`
	COPD         bool `json:"copd"`
}

// EstimatorKind selects which baseline estimator runs.
type EstimatorKind string

const (
	EstMeanStd   EstimatorKind = "mean_std"
	EstMedianMAD EstimatorKind = "median_mad"
	EstEWMA      EstimatorKind = "ewma"
	EstEWMAMAD   EstimatorKind = "ewma_mad"
)

// DefaultEstimator is the production default. Eval ablations override it.
const DefaultEstimator = EstEWMAMAD

// Severity values stored on the alert row. Mirror health_metrics CHECK
// constraint values plus "normal" (which means: no alert row written).
const (
	SeverityNormal   = "normal"
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Stable i18n keys persisted in alerts.reason_code. Frontend dictionary
// maps each to a localized string (lib/i18n.ts).
const (
	ReasonNormal         = "normal"
	ReasonSafetyBelowMin = "safety_below_min"
	ReasonSafetyAboveMax = "safety_above_max"
	ReasonSafetyWarnLow  = "safety_warn_low"
	ReasonSafetyWarnHigh = "safety_warn_high"
	ReasonBaselineWarn   = "baseline_warn_z2"
	ReasonBaselineCrit   = "baseline_crit_z3"
	ReasonConditionWarn  = "condition_warn"
	ReasonConditionCrit  = "condition_crit"
	ReasonColdStart      = "cold_start"
	ReasonLegacy         = "legacy"
)

// Input is a single Analyze invocation.
type Input struct {
	Kind      string        `json:"kind"`
	Value     float64       `json:"value"`
	History   []Reading     `json:"history"`
	Profile   Profile       `json:"profile"`
	Estimator EstimatorKind `json:"estimator,omitempty"` // zero ⇒ DefaultEstimator
	Now       time.Time     `json:"now,omitempty"`       // zero ⇒ time.Now()
}

// Result is what Analyze returns. Callers persist these fields onto
// alerts (when severity != normal) and onto algorithm_runs (always).
type Result struct {
	Severity         string        `json:"severity"`
	ReasonCode       string        `json:"reason_code"`
	Mean             float64       `json:"mean"`
	Std              float64       `json:"std"`
	ZScore           float64       `json:"z_score"`
	Estimator        EstimatorKind `json:"estimator"`
	UsedHistory      bool          `json:"used_history"`
	HistorySize      int           `json:"history_size"`
	AlgorithmVersion string        `json:"algorithm_version"`
}

// resolveNow returns the input's Now field or the current wall clock if zero.
func resolveNow(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

// resolveEstimator returns the input's estimator or the production default.
func resolveEstimator(e EstimatorKind) EstimatorKind {
	if e == "" {
		return DefaultEstimator
	}
	return e
}
