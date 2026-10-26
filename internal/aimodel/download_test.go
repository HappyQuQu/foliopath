package aimodel

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func downloadFixture(t *testing.T) DownloadDefinition {
	t.Helper()
	_, manifest, _ := catalogFixture(t)
	return DownloadDefinition{ID: "download_vision", Name: "Vision model", SourceLabel: "Official source",
		Manifest: manifest, ContentHash: strings.Repeat("d", 64), RuntimeArchitectures: []string{"amd64", "arm64"},
		Files: map[string]string{"image_encoder.onnx": "https://models.example/image.onnx", "text_encoder.onnx": "https://models.example/text.onnx", "tokenizer.json": "https://models.example/tokenizer.json"}}
}

func TestDownloadCatalogValidationAndFixedIdentity(t *testing.T) {
	entry := downloadFixture(t)
	data, _ := json.Marshal([]DownloadDefinition{entry})
	catalog, err := ParseDownloadCatalog(data, "arm64")
	if err != nil {
		t.Fatal(err)
	}
	candidate, _, err := catalog.candidate(entry.ID)
	if err != nil || candidate.Package.PackageSizeByte != 36 || candidate.SourceIdentity != "download:"+entry.ContentHash {
		t.Fatalf("candidate: %+v %v", candidate, err)
	}
	if _, _, err := catalog.candidate("https://arbitrary.example"); !errors.Is(err, ErrDownloadUnavailable) {
		t.Fatal(err)
	}
	for _, invalid := range []string{"http://models.example/file", "https://user:password@models.example/file", "https://models.example:8443/file", "file:///tmp/file", "https://models.example/file#fragment"} {
		t.Run(invalid, func(t *testing.T) {
			entry := downloadFixture(t)
			entry.Files["tokenizer.json"] = invalid
			data, _ := json.Marshal([]DownloadDefinition{entry})
			if _, err := ParseDownloadCatalog(data, "arm64"); err == nil {
				t.Fatal("accepted unsafe source")
			}
		})
	}
	for _, invalid := range [][]byte{[]byte(`null`), []byte(`[{"id":"one","id":"two"}]`), []byte(`[] []`), []byte(`[{"unexpected":true}]`), []byte(strings.Repeat(" ", MaxDownloadCatalogBytes+1))} {
		if _, err := ParseDownloadCatalog(invalid, "arm64"); err == nil {
			t.Fatal("accepted invalid config")
		}
	}
	empty, err := ParseDownloadCatalog(nil, "amd64")
	if err != nil || len(empty.entries) != 0 {
		t.Fatalf("unconfigured catalog: %+v %v", empty, err)
	}
	source := &DownloadSource{Catalog: catalog}
	if err := source.ValidateDirectModelSource(context.Background(), candidate.SourceIdentity); !errors.Is(err, ErrModelIncompatible) {
		t.Fatal("remote direct mode allowed")
	}
}
