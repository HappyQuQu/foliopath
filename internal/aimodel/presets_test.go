package aimodel

import (
	"errors"
	"testing"
)

func TestBuiltinPresetsAreInformationalAndIsolated(t *testing.T) {
	presets := BuiltinModelPresets()
	if len(presets) != 3 {
		t.Fatal(presets)
	}
	catalog, err := ParseDownloadCatalog(nil, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, preset := range presets {
		if seen[preset.ID] || preset.Name == "" || preset.SourceLabel == "" || (preset.Readiness != "package_pending" && preset.Readiness != "adapter_pending") {
			t.Fatal(preset)
		}
		seen[preset.ID] = true
		if _, _, err := catalog.candidate(preset.ID); !errors.Is(err, ErrDownloadUnavailable) {
			t.Fatalf("preset authorized an install: %v", err)
		}
	}
	presets[0].Name = "mutated"
	if BuiltinModelPresets()[0].Name == "mutated" {
		t.Fatal("shared mutable presets")
	}
}
