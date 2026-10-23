package imagevips

import "testing"

func TestFaceDecodeReclaimPolicy(t *testing.T) {
	for _, tc := range []struct {
		name          string
		size          int64
		width, height int
		want          bool
	}{
		{"small", 1 << 20, 1179, 2556, false},
		{"bytes below", largeImageReclaimBytes - 1, 0, 0, false},
		{"bytes boundary", largeImageReclaimBytes, 0, 0, true},
		{"pixels below", 1 << 20, 4000, 3999, false},
		{"pixels boundary", 1 << 20, 4000, 4000, true},
		{"wide multiplication", 1, 100000, 100000, true},
		{"no dimensions", 1, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := largeImageDecodeNeedsReclaim(tc.size, tc.width, tc.height); got != tc.want {
				t.Fatalf("reclaim=%v, want %v", got, tc.want)
			}
		})
	}
}
