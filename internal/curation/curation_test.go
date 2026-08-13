package curation

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeTagNameUsesNFCWhitespaceAndCaseFold(t *testing.T) {
	name, key, err := NormalizeTagName("  Cafe\u0301　 TRIP  ")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Café TRIP" || key != "café trip" {
		t.Fatalf("normalized tag = %q / %q", name, key)
	}
}

func TestNormalizeTagNameRejectsControlsEmptyAndOversize(t *testing.T) {
	for _, value := range []string{"", "   ", "trip\nname", strings.Repeat("界", MaxTagRunes+1)} {
		if _, _, err := NormalizeTagName(value); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("NormalizeTagName(%q) error = %v", value, err)
		}
	}
}
