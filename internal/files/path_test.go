package files

import (
	"errors"
	"testing"
)

func TestNormalizeUsesSharedPolicyAndPreservesSafePath(t *testing.T) {
	t.Parallel()

	const input = "相册/旅行 100%.jpg"
	got, err := Normalize(input)
	if err != nil {
		t.Fatalf("Normalize(%q): %v", input, err)
	}
	if got != input {
		t.Fatalf("Normalize(%q) = %q; raw path was changed", input, got)
	}
}

func TestNormalizeMapsPolicyFailureToFilesError(t *testing.T) {
	t.Parallel()

	_, err := Normalize("safe/%252e%252e/file.jpg")
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Normalize error = %v; want ErrInvalidPath", err)
	}
}
