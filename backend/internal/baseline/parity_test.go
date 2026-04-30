package baseline

import (
	"bufio"
	"encoding/json"
	"flag"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// updateParity controls whether TestParityFixture regenerates the golden
// JSONL file. Run with `go test -tags=updateparity` or set the flag
// `-update` to refresh after intentional algorithm changes.
var updateParity = flag.Bool("update", false, "regenerate testdata/parity_v2.jsonl")

// parityCase is one (input, expected output) row in the parity fixture.
// The fixture is consumed by both Go (regression guard) and the Python
// evaluation harness — Python decodes each line as Input, runs the
// algo-runner, and compares Result against the saved expected output.
// Any drift in the algorithm output fails both Go's `TestParityFixture`
// and Python's `runner.parity` check.
type parityCase struct {
	Name           string `json:"name"`
	Input          Input  `json:"input"`
	ExpectedResult Result `json:"expected"`
}

// TestParityFixture verifies that the committed parity_v2.jsonl matches
// what the current algorithm produces. The fixture itself is produced by
// running the test with -update (or by deleting the file and re-running);
// any drift after that is a regression and must be reviewed.
func TestParityFixture(t *testing.T) {
	cases := buildParityCases()

	path := filepath.Join("testdata", "parity_v2.jsonl")

	if *updateParity {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		w := bufio.NewWriter(f)
		enc := json.NewEncoder(w)
		for _, c := range cases {
			c.ExpectedResult = Analyze(c.Input)
			if err := enc.Encode(c); err != nil {
				t.Fatal(err)
			}
		}
		_ = w.Flush()
		t.Logf("regenerated %s with %d cases", path, len(cases))
		return
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v (run with -update to generate)", path, err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	idx := 0
	for {
		var saved parityCase
		if err := dec.Decode(&saved); err != nil {
			break
		}
		if idx >= len(cases) {
			t.Fatalf("fixture has more cases than buildParityCases produces; regenerate with -update")
		}
		if saved.Name != cases[idx].Name {
			t.Errorf("case %d: name mismatch: fixture=%s, want=%s (regenerate with -update?)",
				idx, saved.Name, cases[idx].Name)
		}
		got := Analyze(saved.Input)
		if !resultsEqual(got, saved.ExpectedResult) {
			t.Errorf("case %q drifted:\n  expected: %+v\n  got:      %+v", saved.Name, saved.ExpectedResult, got)
		}
		idx++
	}
	if idx < len(cases) {
		t.Errorf("fixture has %d cases, buildParityCases produces %d (regenerate with -update)", idx, len(cases))
	}
}

func resultsEqual(a, b Result) bool {
	if a.Severity != b.Severity ||
		a.ReasonCode != b.ReasonCode ||
		a.Estimator != b.Estimator ||
		a.UsedHistory != b.UsedHistory ||
		a.HistorySize != b.HistorySize ||
		a.AlgorithmVersion != b.AlgorithmVersion {
		return false
	}
	const tol = 1e-9
	return math.Abs(a.Mean-b.Mean) < tol &&
		math.Abs(a.Std-b.Std) < tol &&
		math.Abs(a.ZScore-b.ZScore) < tol
}

// buildParityCases returns a deterministic set of fixture inputs spanning
// every code path in Analyze. Add new cases at the end (so existing
// indices stay stable) and rerun with -update.
func buildParityCases() []parityCase {
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)

	makeHistory := func(n int, mean, jitter float64, spacing time.Duration) []Reading {
		rs := make([]Reading, n)
		for i := 0; i < n; i++ {
			rs[i] = Reading{
				Value:      mean + jitter*float64((i%5)-2)/2.0,
				MeasuredAt: now.Add(-time.Duration(n-i) * spacing),
			}
		}
		return rs
	}

	stable := makeHistory(15, 70, 1.5, 24*time.Hour)
	wideBP := makeHistory(15, 130, 8, 24*time.Hour)
	stableGlucose := makeHistory(15, 5.5, 0.3, 24*time.Hour)
	stableSpO2 := makeHistory(15, 97, 1, 12*time.Hour)
	tinyHistory := makeHistory(3, 70, 1, 24*time.Hour)

	hyper := Profile{Hypertension: true}
	t2d := Profile{T2D: true}
	copd := Profile{COPD: true}

	cases := []parityCase{
		{Name: "safety/pulse_critical_high", Input: Input{Kind: "pulse", Value: 200, History: stable, Now: now}},
		{Name: "safety/pulse_critical_low", Input: Input{Kind: "pulse", Value: 30, History: stable, Now: now}},
		{Name: "safety/bp_sys_critical_high", Input: Input{Kind: "bp_sys", Value: 200, History: wideBP, Now: now}},
		{Name: "safety/bp_sys_warn_high", Input: Input{Kind: "bp_sys", Value: 155, History: wideBP, Now: now}},
		{Name: "safety/glucose_critical_low", Input: Input{Kind: "glucose", Value: 2.5, History: stableGlucose, Now: now}},
		{Name: "safety/spo2_critical_low", Input: Input{Kind: "spo2", Value: 85, History: stableSpO2, Now: now}},
		{Name: "safety/temperature_warn_high", Input: Input{Kind: "temperature", Value: 38.0, History: makeHistory(15, 36.6, 0.2, 12*time.Hour), Now: now}},

		{Name: "cold_start/three_readings_normal_value", Input: Input{Kind: "pulse", Value: 72, History: tinyHistory, Now: now}},
		{Name: "cold_start/no_history_normal_value", Input: Input{Kind: "pulse", Value: 72, History: nil, Now: now}},
		{Name: "cold_start/insufficient_streak", Input: Input{Kind: "pulse", Value: 72, History: makeHistory(8, 70, 1, 24*time.Hour), Now: now}},

		{Name: "baseline/personal_outlier_critical", Input: Input{Kind: "pulse", Value: 90, History: stable, Now: now}},
		{Name: "baseline/personal_outlier_warning", Input: Input{Kind: "pulse", Value: 75, History: stable, Now: now}},
		{Name: "baseline/in_range_quiet", Input: Input{Kind: "pulse", Value: 71, History: stable, Now: now}},

		{Name: "spo2/down_only_high_quiet", Input: Input{Kind: "spo2", Value: 99, History: stableSpO2, Now: now}},
		{Name: "spo2/down_only_dip_critical", Input: Input{Kind: "spo2", Value: 92, History: stableSpO2, Now: now}},

		{Name: "condition/hypertension_widened_quiet_at_155", Input: Input{Kind: "bp_sys", Value: 155, History: wideBP, Profile: hyper, Now: now}},
		{Name: "condition/default_warns_at_155", Input: Input{Kind: "bp_sys", Value: 155, History: wideBP, Now: now}},
		{Name: "condition/t2d_widened_quiet_at_11", Input: Input{Kind: "glucose", Value: 11.0, History: stableGlucose, Profile: t2d, Now: now}},
		{Name: "condition/default_warns_at_11_glucose", Input: Input{Kind: "glucose", Value: 11.0, History: stableGlucose, Now: now}},
		{Name: "condition/copd_widened_quiet_at_91_spo2", Input: Input{Kind: "spo2", Value: 91, History: stableSpO2, Profile: copd, Now: now}},
		{Name: "condition/default_warns_at_91_spo2", Input: Input{Kind: "spo2", Value: 91, History: stableSpO2, Now: now}},

		{Name: "estimator/mean_std_explicit", Input: Input{Kind: "pulse", Value: 90, History: stable, Estimator: EstMeanStd, Now: now}},
		{Name: "estimator/median_mad_explicit", Input: Input{Kind: "pulse", Value: 90, History: stable, Estimator: EstMedianMAD, Now: now}},
		{Name: "estimator/ewma_explicit", Input: Input{Kind: "pulse", Value: 90, History: stable, Estimator: EstEWMA, Now: now}},
	}
	return cases
}
