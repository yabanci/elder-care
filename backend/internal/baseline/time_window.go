package baseline

import (
	"sort"
	"time"
)

// Default window parameters used by Analyze.
const (
	DefaultWindowDays    = 30
	DefaultMaxSamples    = 60
	DefaultStreakSamples = 10
	DefaultStreakDays    = 14
)

// WindowFilter returns history limited to the last `days` ending at `now`,
// capped at `maxSamples` (newest preferred). Output is chronological
// (oldest first), suitable input for the estimators.
func WindowFilter(history []Reading, now time.Time, days, maxSamples int) []Reading {
	if len(history) == 0 || days <= 0 || maxSamples <= 0 {
		return nil
	}
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	filtered := make([]Reading, 0, len(history))
	for _, r := range history {
		if !r.MeasuredAt.Before(cutoff) {
			filtered = append(filtered, r)
		}
	}
	// Sort newest-first to apply the cap, then flip back to chronological.
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].MeasuredAt.After(filtered[j].MeasuredAt)
	})
	if len(filtered) > maxSamples {
		filtered = filtered[:maxSamples]
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].MeasuredAt.Before(filtered[j].MeasuredAt)
	})
	return filtered
}

// IsStable returns true once the patient has at least minSamples readings
// within the last `days` ending at `now`. Used as the cold-start gate:
// before this is satisfied, Analyze falls back to safety bounds only.
func IsStable(history []Reading, now time.Time, minSamples, days int) bool {
	if len(history) == 0 || minSamples <= 0 || days <= 0 {
		return false
	}
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	count := 0
	for _, r := range history {
		if !r.MeasuredAt.Before(cutoff) {
			count++
			if count >= minSamples {
				return true
			}
		}
	}
	return false
}
