package scanner

import (
	"errors"
	"testing"
)

func TestNormalizeEntryPathUsesSharedEncodedTraversalPolicy(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"%2e%2e",
		"%252e%252e",
		"album%252fsecret",
		"album/%2500.jpg",
		string([]byte{'b', 'a', 'd', 0xff}),
	} {
		if _, err := normalizeEntryPath(value); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("normalizeEntryPath(%q) error = %v, want ErrInvalidEntry", value, err)
		}
	}
	for _, value := range []string{"", "album/photo.jpg", "旅行 100%/literal%20.jpg"} {
		got, err := normalizeEntryPath(value)
		if err != nil || got != value {
			t.Errorf("normalizeEntryPath(%q) = (%q, %v), want unchanged", value, got, err)
		}
	}
}

func TestSkipCountsValidateAndTotal(t *testing.T) {
	counts := SkipCounts{Directories: 2, Files: 3}
	if !counts.Valid() || counts.Total() != 5 {
		t.Fatalf("valid skip counts = %#v, total %d", counts, counts.Total())
	}
	for _, invalid := range []SkipCounts{
		{Directories: -1},
		{Files: -1},
	} {
		if invalid.Valid() {
			t.Fatalf("negative skip counts accepted: %#v", invalid)
		}
	}
}
