package face

import "math"

// SizeWeightedDetectionQuality is the existing size/score heuristic, not a
// calibrated recognition probability. Both local and remote adapters use this
// owner; the generation's threshold profile still controls admission.
func SizeWeightedDetectionQuality(score, width, height float32) (float32, error) {
	for _, value := range []float32{score, width, height} {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return 0, ErrInvalidOutput
		}
	}
	if score < 0 || score > 1 || width <= 0 || height <= 0 {
		return 0, ErrInvalidOutput
	}
	return score * min(float32(1), min(width, height)/64), nil
}
