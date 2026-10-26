package aimodel

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type staticCandidateSource struct {
	items     []RawCandidate
	truncated bool
}

func (source staticCandidateSource) ScanModelPackages(context.Context, int, int, int64) ([]RawCandidate, bool, error) {
	return source.items, source.truncated, nil
}

func TestScannerReplacesCandidatesAndRejectsStaleRevision(t *testing.T) {
	catalog, manifest, facts := catalogFixture(t)
	encoded, err := jsonMarshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{"aic_first", "aic_second"}
	scanner, err := NewScanner(staticCandidateSource{items: []RawCandidate{{
		Manifest: encoded, Files: facts, SourceIdentity: "source:test",
	}}}, catalog, "arm64", func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := scanner.Scan(context.Background())
	if err != nil || first.Revision != 1 || len(first.Candidates) != 1 || first.Candidates[0].Compatibility != "compatible" {
		t.Fatalf("first scan = %#v, %v", first, err)
	}
	resolved, err := scanner.Resolve("aic_first", 1)
	if err != nil || resolved.Package.PackageID != manifest.PackageID {
		t.Fatalf("resolved = %#v, %v", resolved, err)
	}
	if _, err := scanner.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.Resolve("aic_first", 1); !errors.Is(err, ErrCandidateStale) {
		t.Fatalf("stale resolve error = %v", err)
	}
}

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}
