package media

import (
	"errors"
	"testing"
)

func TestSourceFingerprintUsesSizeAndNanosecondMTime(t *testing.T) {
	t.Parallel()

	first, err := NewSourceFingerprint(42, 1_700_000_000_000_000_001)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := first.String(), "v1:42:1700000000000000001"; got != want {
		t.Fatalf("fingerprint = %q, want %q", got, want)
	}
	if !first.Matches(42, 1_700_000_000_000_000_001) {
		t.Fatal("fingerprint does not match its source metadata")
	}
	if !first.Valid() {
		t.Fatal("canonical fingerprint is not valid")
	}
	if first.Matches(43, 1_700_000_000_000_000_001) ||
		first.Matches(42, 1_700_000_000_000_000_002) {
		t.Fatal("fingerprint did not change with size or nanosecond mtime")
	}
}

func TestSourceFingerprintValidationRejectsNonCanonicalValues(t *testing.T) {
	t.Parallel()
	for _, value := range []SourceFingerprint{
		"", "v2:1:2", "v1:-1:2", "v1:1", "v1:one:2", "v1:1:two",
	} {
		if value.Valid() {
			t.Fatalf("%q unexpectedly valid", value)
		}
	}
}

func TestSourceFingerprintRejectsNegativeSize(t *testing.T) {
	t.Parallel()

	if _, err := NewSourceFingerprint(-1, 0); !errors.Is(err, ErrInvalidSourceMetadata) {
		t.Fatalf("negative size error = %v, want ErrInvalidSourceMetadata", err)
	}
}
