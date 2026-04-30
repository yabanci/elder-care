package baseline

// Analyze is the public top-level entry point. It composes the six layers
// described in the design spec and returns a Result that is ready to be
// persisted to alerts (when Severity != Normal) and to algorithm_runs
// (always).
//
// Pipeline order:
//
//  1. Resolve defaults (Now, Estimator).
//  2. ConditionProfile → effective Thresholds for the metric.
//  3. SafetyOverride — absolute clinical bounds (short-circuit if fired).
//  4. Preprocessor → Hampel filter on history.
//  5. TimeAwareWindow → last 30 days, capped at 60 samples.
//  6. StableStreakGate → require ≥10 readings in last 14 days; otherwise
//     return ColdStart.
//  7. Estimator → mean, std (sample-corrected for EWMA variants).
//  8. DecisionRule → personal baseline z-score severity.
func Analyze(in Input) Result {
	now := resolveNow(in.Now)
	estimator := resolveEstimator(in.Estimator)

	thresholds := ThresholdsFor(in.Kind, in.Profile)

	if r, fired := SafetyCheck(in.Value, thresholds); fired {
		r.Estimator = estimator
		r.HistorySize = len(in.History)
		r.AlgorithmVersion = Version
		return r
	}

	cleaned := Hampel(in.History, 5, 3.0)
	windowed := WindowFilter(cleaned, now, DefaultWindowDays, DefaultMaxSamples)

	if !IsStable(windowed, now, DefaultStreakSamples, DefaultStreakDays) {
		return Result{
			Severity:         SeverityNormal,
			ReasonCode:       ReasonColdStart,
			Estimator:        estimator,
			UsedHistory:      false,
			HistorySize:      len(in.History),
			AlgorithmVersion: Version,
		}
	}

	mean, std := Estimate(estimator, windowed, now)
	res := Decide(in.Value, mean, std, downOnlyMetrics[in.Kind])
	res.Estimator = estimator
	res.UsedHistory = true
	res.HistorySize = len(windowed)
	res.AlgorithmVersion = Version
	return res
}
