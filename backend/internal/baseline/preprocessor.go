package baseline

import (
	"math"
	"sort"
)

// Hampel applies a Hampel identifier to history: each point is compared
// against the median of a centered window of size `window`; if the point
// is more than `threshold` × MAD-based-σ-equivalent away, it is replaced
// with the local median. This prevents prior outliers in the baseline
// window from poisoning the current estimator.
//
// Edges that lack a full window are left untouched (we'd rather keep an
// honest possibly-noisy edge than fabricate a median from too-few points).
//
// Returns a new slice; input is not mutated.
func Hampel(history []Reading, window int, threshold float64) []Reading {
	if history == nil {
		return nil
	}
	if len(history) < window {
		// Series too short to apply the window meaningfully; pass through.
		out := make([]Reading, len(history))
		copy(out, history)
		return out
	}
	half := window / 2
	out := make([]Reading, len(history))
	copy(out, history)

	// Reusable scratch slice; Go's sort.Float64s wants a slice.
	scratch := make([]float64, 0, window)

	for i := half; i < len(history)-half; i++ {
		scratch = scratch[:0]
		for j := i - half; j <= i+half; j++ {
			scratch = append(scratch, history[j].Value)
		}
		// Compute median.
		sort.Float64s(scratch)
		med := scratch[len(scratch)/2]
		// Compute MAD on the same window.
		for k := range scratch {
			scratch[k] = math.Abs(scratch[k] - med)
		}
		sort.Float64s(scratch)
		mad := scratch[len(scratch)/2]
		// 1.4826 converts MAD to σ-equivalent under a Gaussian assumption.
		sigma := 1.4826 * mad
		if sigma == 0 {
			continue
		}
		if math.Abs(history[i].Value-med) > threshold*sigma {
			out[i].Value = med
		}
	}
	return out
}
