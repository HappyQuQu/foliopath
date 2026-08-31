package aimodel

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func catalogFixture(t *testing.T) (*Catalog, Manifest, []FileFact) {
	t.Helper()
	manifest := Manifest{
		FormatVersion: 1,
		PackageID:     "semantic-test-v1",
		Purpose:       PurposeSemanticImageText,
		Version:       "1.0.0",
		Architecture:  "portable-onnx",
		LicenseID:     "Apache-2.0",
		Files: []ManifestFile{
			{Name: "image_encoder.onnx", Size: 11, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Role: "image_encoder"},
			{Name: "text_encoder.onnx", Size: 12, SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Role: "text_encoder"},
			{Name: "tokenizer.json", Size: 13, SHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Role: "tokenizer"},
		},
	}
	catalog, err := NewCatalog([]CatalogEntry{{
		Manifest:             manifest,
		ContentHash:          "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		RuntimeArchitectures: []string{"amd64", "arm64"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	facts := make([]FileFact, len(manifest.Files))
	for index, file := range manifest.Files {
		facts[index] = FileFact{Name: file.Name, Size: file.Size, SHA256: file.SHA256, Regular: true}
	}
	return catalog, manifest, facts
}

func TestCatalogAcceptsOnlyExactBuiltInPackage(t *testing.T) {
	catalog, manifest, facts := catalogFixture(t)
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := catalog.Verify(encoded, facts, "arm64")
	if err != nil || verified.PackageID != manifest.PackageID || verified.PackageSizeByte != 36 {
		t.Fatalf("verified = %#v, %v", verified, err)
	}
	facts[0].SHA256 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	if _, err := catalog.Verify(encoded, facts, "arm64"); !errors.Is(err, ErrModelIncompatible) {
		t.Fatalf("wrong hash error = %v", err)
	}
	byHash, byHashManifest, found := catalog.PackageByContentHash(strings.Repeat("d", 64), "arm64")
	if !found || byHash.PackageID != manifest.PackageID || byHash.PackageSizeByte != 36 || byHashManifest.PackageID != manifest.PackageID {
		t.Fatalf("content hash lookup = %#v %#v found=%v", byHash, byHashManifest, found)
	}
}

func TestCatalogRejectsDuplicateUnknownAndWrongArchitecture(t *testing.T) {
	catalog, manifest, facts := catalogFixture(t)
	duplicate := []byte(`{"formatVersion":1,"formatVersion":1,"packageId":"semantic-test-v1","purpose":"semantic_image_text","version":"1.0.0","architecture":"portable-onnx","licenseId":"Apache-2.0","files":[]}`)
	if _, err := catalog.Verify(duplicate, facts, "arm64"); !errors.Is(err, ErrModelIncompatible) {
		t.Fatalf("duplicate key error = %v", err)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Verify(encoded, facts, "riscv64"); !errors.Is(err, ErrModelIncompatible) {
		t.Fatalf("wrong architecture error = %v", err)
	}
	unknown := append(encoded[:len(encoded)-1], []byte(`,"extra":true}`)...)
	if _, err := catalog.Verify(unknown, facts, "arm64"); !errors.Is(err, ErrModelIncompatible) {
		t.Fatalf("unknown field error = %v", err)
	}
	duplicateHashManifest := manifest
	duplicateHashManifest.PackageID = "semantic-test-v2"
	if _, err := NewCatalog([]CatalogEntry{
		{Manifest: manifest, ContentHash: strings.Repeat("d", 64), RuntimeArchitectures: []string{"arm64"}},
		{Manifest: duplicateHashManifest, ContentHash: strings.Repeat("d", 64), RuntimeArchitectures: []string{"arm64"}},
	}); !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("duplicate content hash error = %v", err)
	}
	if _, _, found := catalog.PackageByContentHash(strings.Repeat("d", 64), "riscv64"); found {
		t.Fatal("content hash lookup accepted unsupported architecture")
	}
}
