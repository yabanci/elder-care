package baseline

import "strings"

// Thresholds describes the safety / warning bounds for a single metric.
// Any field may be nil ⇒ "no bound on this side".
type Thresholds struct {
	CriticalLow  *float64
	CriticalHigh *float64
	WarnLow      *float64
	WarnHigh     *float64
}

func f(v float64) *float64 { return &v }

// minPtr returns the smaller of a or b (the more sensitive lower bound).
// nil means "no bound" — pick the non-nil; if both nil, return nil.
func minPtr(a, b *float64) *float64 {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	case *a < *b:
		return a
	default:
		return b
	}
}

// maxPtr returns the larger of a or b — used for *low* bounds when
// composing profiles, since a higher floor on SpO2 is more sensitive
// (alerts earlier on dips).
func maxPtr(a, b *float64) *float64 {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	case *a > *b:
		return a
	default:
		return b
	}
}

// defaultThresholds is the safety / warning table for the general
// population, mirroring the v1 SafetyLimits map.
var defaultThresholds = map[string]Thresholds{
	"pulse":       {CriticalLow: f(40), CriticalHigh: f(140), WarnLow: f(50), WarnHigh: f(110)},
	"bp_sys":      {CriticalLow: f(80), CriticalHigh: f(180), WarnLow: f(100), WarnHigh: f(150)},
	"bp_dia":      {CriticalLow: f(50), CriticalHigh: f(110), WarnLow: f(60), WarnHigh: f(95)},
	"glucose":     {CriticalLow: f(3.0), CriticalHigh: f(15.0), WarnLow: f(4.0), WarnHigh: f(10.0)},
	"temperature": {CriticalLow: f(35.0), CriticalHigh: f(39.0), WarnLow: f(35.5), WarnHigh: f(37.8)},
	"spo2":        {CriticalLow: f(88), WarnLow: f(93)},
	"weight":      {},
}

// profileOverrides describes the narrower bounds applied when a specific
// chronic-condition profile is active. Bounds compose by taking the more
// sensitive direction (min for high-side warnings, max for low-side).
var profileOverrides = map[string]map[string]Thresholds{
	"hypertension": {
		"bp_sys": {WarnHigh: f(140)},
		"bp_dia": {WarnHigh: f(90)},
		"pulse":  {WarnHigh: f(105)},
	},
	"t2d": {
		"glucose": {WarnHigh: f(9.0), WarnLow: f(4.5)},
		"bp_sys":  {WarnHigh: f(145)},
	},
	"copd": {
		"spo2":  {WarnLow: f(95)},
		"pulse": {WarnHigh: f(115)},
	},
}

// ThresholdsFor returns the effective thresholds for a metric given a
// patient's condition profile. Multiple matching profiles compose by
// taking the narrower bound (more sensitive).
func ThresholdsFor(kind string, prof Profile) Thresholds {
	base, ok := defaultThresholds[kind]
	if !ok {
		return Thresholds{}
	}
	out := base // value copy

	apply := func(name string) {
		ov, ok := profileOverrides[name][kind]
		if !ok {
			return
		}
		out.CriticalHigh = minPtr(out.CriticalHigh, ov.CriticalHigh)
		out.CriticalLow = maxPtr(out.CriticalLow, ov.CriticalLow)
		out.WarnHigh = minPtr(out.WarnHigh, ov.WarnHigh)
		out.WarnLow = maxPtr(out.WarnLow, ov.WarnLow)
	}
	if prof.Hypertension {
		apply("hypertension")
	}
	if prof.T2D {
		apply("t2d")
	}
	if prof.COPD {
		apply("copd")
	}
	return out
}

// keywordMatches returns true if the haystack (lowercased) contains the
// needle (lowercased). Case-insensitive, substring-style; handles cyrillic
// and latin uniformly via strings.ToLower (Unicode-aware).
func keywordMatches(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// hypertensionKeywords / t2dKeywords / copdKeywords are matched
// case-insensitively. Add synonyms here when patient cohort surfaces them.
var (
	hypertensionKeywords = []string{
		"гипертония",
		"гипертензия",
		"артериальн",
		"hypertension",
		"қан қысымы",
	}
	t2dKeywords = []string{
		"диабет",
		"сд2",
		"diabetes",
		"t2d",
		"қант диабет",
	}
	copdKeywords = []string{
		"хобл",
		"copd",
		"өкпенің созылмалы",
	}
)

// ParseProfile inspects the free-text chronic-conditions field and
// returns the set of profiles that match. Defaults to none — safe
// fallback: patient is treated as the general population.
func ParseProfile(chronicConditions string) Profile {
	var p Profile
	for _, kw := range hypertensionKeywords {
		if keywordMatches(chronicConditions, kw) {
			p.Hypertension = true
			break
		}
	}
	for _, kw := range t2dKeywords {
		if keywordMatches(chronicConditions, kw) {
			p.T2D = true
			break
		}
	}
	for _, kw := range copdKeywords {
		if keywordMatches(chronicConditions, kw) {
			p.COPD = true
			break
		}
	}
	return p
}
