package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCandidateComponentBOMPinsArtifactsAndDispositions(t *testing.T) {
	raw, err := os.ReadFile("model-candidates-component.cdx.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Format      string `json:"bomFormat"`
		SpecVersion string `json:"specVersion"`
		Components  []struct {
			Ref    string `json:"bom-ref"`
			Hashes []struct {
				Algorithm string `json:"alg"`
				Content   string `json:"content"`
			} `json:"hashes"`
			Properties []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"properties"`
		} `json:"components"`
		Compositions []struct {
			Aggregate  string   `json:"aggregate"`
			Assemblies []string `json:"assemblies"`
		} `json:"compositions"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if document.Format != "CycloneDX" || document.SpecVersion != "1.6" {
		t.Fatalf("unexpected BOM identity: %q %q", document.Format, document.SpecVersion)
	}
	wantHashes := map[string]string{
		"pkg:generic/google-siglip-base-patch16-224@7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed":                    "2c63cb7d1f2e95ba501893cbb8faeb4ea9a3af295498d35097126228659c2af8",
		"pkg:generic/foliopath-siglip1-image-encoder@7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed?precision=float16": "4d4260477eaf57ce263d8f65656b030384fd6854513896254b9cae9566432b3d",
		"pkg:generic/foliopath-siglip1-text-encoder@7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed?precision=float16":  "19f28e2bdb2e1bfb51ec1733b371899784ae0eb4af14d22980e7a6b89a15ef4f",
		"pkg:generic/opencv-zoo-yunet@2023mar?revision=47534e27c9851bb1128ccc0102f1145e27f23f98":                 "8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4",
		"pkg:generic/opencv-zoo-sface@2021dec?revision=47534e27c9851bb1128ccc0102f1145e27f23f98":                 "0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79",
	}
	if len(document.Components) != len(wantHashes) {
		t.Fatalf("components = %d, want %d", len(document.Components), len(wantHashes))
	}
	seen := make(map[string]bool, len(document.Components))
	for _, component := range document.Components {
		if seen[component.Ref] {
			t.Fatalf("duplicate component ref %q", component.Ref)
		}
		seen[component.Ref] = true
		wantHash, exists := wantHashes[component.Ref]
		if !exists {
			t.Fatalf("unexpected component ref %q", component.Ref)
		}
		if len(component.Hashes) == 0 || component.Hashes[0].Algorithm != "SHA-256" || component.Hashes[0].Content != wantHash {
			t.Fatalf("component %q does not pin expected SHA-256", component.Ref)
		}
		if strings.Contains(component.Ref, "opencv-zoo-sface") {
			disposition := ""
			for _, property := range component.Properties {
				if property.Name == "foliopath:int001:disposition" {
					disposition = property.Value
				}
			}
			if !strings.Contains(disposition, "production-hold") {
				t.Fatalf("SFace disposition = %q, want production-hold", disposition)
			}
		}
	}
	if len(document.Compositions) != 1 || document.Compositions[0].Aggregate != "incomplete" {
		t.Fatalf("candidate BOM must retain one incomplete composition")
	}
	if len(document.Compositions[0].Assemblies) != len(wantHashes) {
		t.Fatalf("composition assemblies = %d, want %d", len(document.Compositions[0].Assemblies), len(wantHashes))
	}
}
