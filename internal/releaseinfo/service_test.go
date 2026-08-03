package releaseinfo

import "testing"

func TestNewerUsesStableSemanticVersions(t *testing.T) {
	for _, test := range []struct {
		candidate string
		current   string
		want      bool
	}{
		{"v0.2.0", "v0.1.9", true},
		{"v1.0.0", "0.9.9", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.1.0-beta.1", "v0.0.9", false},
		{"v0.1.0", "dev", false},
	} {
		if got := newer(test.candidate, test.current); got != test.want {
			t.Errorf("newer(%q, %q) = %t, want %t", test.candidate, test.current, got, test.want)
		}
	}
}
