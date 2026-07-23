package library

import "testing"

func TestNormalizeRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "allowed root", input: "", want: ""},
		{name: "nested", input: "family/2026", want: "family/2026"},
		{name: "literal percent", input: "旅行 100%/literal%20", want: "旅行 100%/literal%20"},
		{name: "absolute", input: "/etc", wantErr: true},
		{name: "traversal", input: "family/../work", wantErr: true},
		{name: "encoded traversal", input: "%252e%252e", wantErr: true},
		{name: "encoded separator", input: "family%252fwork", wantErr: true},
		{name: "empty component", input: "family//work", wantErr: true},
		{name: "backslash", input: `family\work`, wantErr: true},
		{name: "invalid UTF-8", input: string([]byte{'b', 'a', 'd', 0xff}), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeRoot(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("NormalizeRoot(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("NormalizeRoot(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestRootsOverlapUsesComponentBoundaries(t *testing.T) {
	t.Parallel()

	if !RootsOverlap("family", "family/2026") {
		t.Fatal("ancestor and descendant must overlap")
	}
	if RootsOverlap("family", "family-archive") {
		t.Fatal("plain string prefixes must not overlap")
	}
	if !RootsOverlap("", "family") {
		t.Fatal("the allowed root must overlap every descendant")
	}
}
