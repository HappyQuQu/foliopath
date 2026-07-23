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
