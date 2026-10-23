package face

import (
	"math"
	"testing"
)

func TestSizeWeightedDetectionQuality(t *testing.T) {
	for _, test := range []struct{ score, width, height, want float32 }{{.9, 64, 64, .9}, {.9, 128, 32, .45}, {0, 1, 1, 0}} {
		got, err := SizeWeightedDetectionQuality(test.score, test.width, test.height)
		if err != nil || got != test.want {
			t.Fatal(got, err)
		}
	}
	for _, values := range [][3]float32{{1, 0, 64}, {-1, 64, 64}, {2, 64, 64}, {1, float32(math.Inf(1)), 1}, {float32(math.NaN()), 1, 1}} {
		if _, err := SizeWeightedDetectionQuality(values[0], values[1], values[2]); err == nil {
			t.Fatal("invalid quality accepted")
		}
	}
}
