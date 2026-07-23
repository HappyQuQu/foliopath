package pathpolicy

import (
	"errors"
	"testing"
)

func TestNormalizePreservesSafeRawPath(t *testing.T) {
	t.Parallel()

	paths := []string{
		"",
		"album/photo.jpg",
		"相册/旅行 100%.jpg",
		"literal%GG.jpg",
		"literal%20name.jpg",
		"100%25-real.jpg",
		"photo%2Ejpg",
		"caf%C3%A9.jpg",
	}
	for _, input := range paths {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			got, err := Normalize(input)
			if err != nil {
				t.Fatalf("Normalize(%q): %v", input, err)
			}
			if got != input {
				t.Fatalf("Normalize(%q) = %q; raw path was changed", input, got)
			}
		})
	}
}

func TestNormalizeRejectsUnsafePathAndEncodedViews(t *testing.T) {
	t.Parallel()

	paths := []string{
		"/absolute",
		"//server/share",
		`album\photo.jpg`,
		".",
		"..",
		"album/./photo.jpg",
		"album/../photo.jpg",
		"album//photo.jpg",
		"album/",
		"nul\x00byte",
		string([]byte{'b', 'a', 'd', 0xff}),
		"%2e",
		"%2E%2e",
		"%252e%252e",
		"%25252E%25252e",
		"%2fetc",
		"%2Fetc",
		"%252fetc",
		"%5cwindows",
		"%255Cwindows",
		"%00",
		"%2500",
		"safe/%2e%2e/file.jpg",
	}
	for _, input := range paths {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			_, err := Normalize(input)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("Normalize(%q) error = %v; want ErrInvalid", input, err)
			}
		})
	}
}
