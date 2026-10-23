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

func semanticV2CatalogFixture(t *testing.T) (*Catalog, Manifest, []FileFact) {
	t.Helper()
	manifest := Manifest{
		FormatVersion: SemanticFormatVersion,
		PackageID:     "siglip-sentencepiece-v2",
		Purpose:       PurposeSemanticImageText,
		Version:       "1.0.0-candidate",
		Architecture:  "portable-onnx",
		LicenseID:     "Apache-2.0",
		Contracts: &SemanticContracts{
			ImagePreprocess: SemanticImagePreprocessContract, TextCanonical: SemanticTextCanonicalContract,
			Tokenizer: SemanticTokenizerContract, EmbeddingAndStorage: SemanticEmbeddingContract,
		},
		Files: []ManifestFile{
			{Name: "image_encoder.onnx", Size: 11, SHA256: strings.Repeat("a", 64), Role: "image_encoder"},
			{Name: "text_encoder.onnx", Size: 12, SHA256: strings.Repeat("b", 64), Role: "text_encoder"},
			{Name: "spiece.model", Size: 13, SHA256: strings.Repeat("c", 64), Role: "sentencepiece_model"},
		},
	}
	catalog, err := NewCatalog([]CatalogEntry{{
		Manifest: manifest, ContentHash: strings.Repeat("e", 64), RuntimeArchitectures: []string{"amd64", "arm64"},
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

func TestCatalogAcceptsExactSemanticV2WithoutReinterpretingV1(t *testing.T) {
	catalog, manifest, facts := semanticV2CatalogFixture(t)
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := catalog.Verify(encoded, facts, "amd64")
	if err != nil || verified.Purpose != PurposeSemanticImageText || verified.PackageSizeByte != 36 {
		t.Fatalf("verified=%#v error=%v", verified, err)
	}
	returned, found := catalog.Manifest(manifest.PackageID)
	if !found || returned.Contracts == nil || returned.Contracts.Tokenizer != SemanticTokenizerContract {
		t.Fatalf("manifest=%#v found=%v", returned, found)
	}
	returned.Contracts.Tokenizer = "mutated"
	again, _ := catalog.Manifest(manifest.PackageID)
	if again.Contracts.Tokenizer != SemanticTokenizerContract {
		t.Fatal("catalog leaked mutable semantic contracts")
	}

	confused := manifest
	confused.FormatVersion = 1
	encoded, _ = json.Marshal(confused)
	if _, err := catalog.Verify(encoded, facts, "amd64"); !errors.Is(err, ErrModelIncompatible) {
		t.Fatalf("v1 confusion error=%v", err)
	}
}

func TestCatalogRejectsHostileOrIncompleteSemanticV2(t *testing.T) {
	catalog, valid, facts := semanticV2CatalogFixture(t)
	for name, mutate := range map[string]func(*Manifest){
		"missing tokenizer":   func(value *Manifest) { value.Files = value.Files[:2] },
		"legacy role":         func(value *Manifest) { value.Files[2].Role = "tokenizer" },
		"duplicate role":      func(value *Manifest) { value.Files[2].Role = "text_encoder" },
		"unknown contract":    func(value *Manifest) { value.Contracts.Tokenizer = "latest" },
		"face contract mixed": func(value *Manifest) { value.Contracts.Decode = FaceDecodeContract },
		"per-file license":    func(value *Manifest) { value.Files[0].LicenseID = "MIT" },
		"nested path":         func(value *Manifest) { value.Files[2].Name = "tokenizer/spiece.model" },
	} {
		t.Run(name, func(t *testing.T) {
			value := valid
			contracts := *valid.Contracts
			value.Contracts = &contracts
			value.Files = append([]ManifestFile(nil), valid.Files...)
			mutate(&value)
			encoded, _ := json.Marshal(value)
			if _, err := catalog.Verify(encoded, facts, "arm64"); !errors.Is(err, ErrModelIncompatible) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func faceCatalogFixture(t *testing.T) (*Catalog, Manifest, []FileFact) {
	t.Helper()
	manifest := Manifest{
		FormatVersion: FaceFormatVersion,
		PackageID:     "face-yunet-auraface-v3",
		Purpose:       PurposeFaceDetectionEmbedding,
		Version:       "1.0.0-candidate",
		Architecture:  "portable-onnx",
		Contracts: &FaceContracts{
			Decode: FaceDecodeContract, Detector: FaceDetectorContract, Postprocess: FacePostprocessContract,
			Alignment: FaceAlignmentContract, Embedding: FaceEmbeddingContract, Storage: FaceStorageContract,
			ThresholdProfile: FaceThresholdContract,
		},
		Files: []ManifestFile{
			{Name: "face_detector.onnx", Size: 11, SHA256: strings.Repeat("a", 64), Role: "face_detector", LicenseID: "MIT"},
			{Name: "face_embedder.onnx", Size: 12, SHA256: strings.Repeat("b", 64), Role: "face_embedder", LicenseID: "Apache-2.0"},
			{Name: "threshold_profile.json", Size: 13, SHA256: strings.Repeat("c", 64), Role: "face_threshold_profile", LicenseID: "Apache-2.0"},
		},
	}
	catalog, err := NewCatalog([]CatalogEntry{{
		Manifest: manifest, ContentHash: strings.Repeat("d", 64), RuntimeArchitectures: []string{"amd64", "arm64"},
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

func TestCatalogAcceptsExactCombinedFacePackageWithoutReinterpretingSemanticV1(t *testing.T) {
	catalog, manifest, facts := faceCatalogFixture(t)
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := catalog.Verify(encoded, facts, "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if verified.Purpose != PurposeFaceDetectionEmbedding || verified.PackageSizeByte != 36 ||
		verified.LicenseID != "Apache-2.0 AND MIT" {
		t.Fatalf("verified=%#v", verified)
	}
	returned, found := catalog.Manifest(manifest.PackageID)
	if !found || returned.Contracts == nil || returned.Contracts.Alignment != FaceAlignmentContract {
		t.Fatalf("manifest=%#v found=%v", returned, found)
	}
	returned.Contracts.Alignment = "mutated"
	again, _ := catalog.Manifest(manifest.PackageID)
	if again.Contracts.Alignment != FaceAlignmentContract {
		t.Fatal("catalog leaked mutable face contracts")
	}

	confused := manifest
	confused.FormatVersion = 1
	confused.Purpose = PurposeSemanticImageText
	encoded, _ = json.Marshal(confused)
	if _, err := catalog.Verify(encoded, facts, "arm64"); !errors.Is(err, ErrModelIncompatible) {
		t.Fatalf("semantic confusion error=%v", err)
	}
}

func TestCatalogRejectsHostileOrIncompleteFacePackage(t *testing.T) {
	catalog, valid, facts := faceCatalogFixture(t)
	for name, mutate := range map[string]func(*Manifest){
		"missing threshold": func(value *Manifest) { value.Files = value.Files[:2] },
		"duplicate role":    func(value *Manifest) { value.Files[2].Role = "face_embedder" },
		"unknown contract":  func(value *Manifest) { value.Contracts.Alignment = "latest" },
		"missing license":   func(value *Manifest) { value.Files[1].LicenseID = "" },
		"nested path":       func(value *Manifest) { value.Files[0].Name = "models/detector.onnx" },
		"path package":      func(value *Manifest) { value.PackageID = "../face" },
		"unicode version":   func(value *Manifest) { value.Version = "候选" },
		"top license":       func(value *Manifest) { value.LicenseID = "Apache-2.0" },
	} {
		t.Run(name, func(t *testing.T) {
			value := valid
			contracts := *valid.Contracts
			value.Contracts = &contracts
			value.Files = append([]ManifestFile(nil), valid.Files...)
			mutate(&value)
			encoded, _ := json.Marshal(value)
			if _, err := catalog.Verify(encoded, facts, "arm64"); !errors.Is(err, ErrModelIncompatible) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	encoded, _ := json.Marshal(valid)
	unknown := append(encoded[:len(encoded)-1], []byte(`,"redistributionApproved":true}`)...)
	duplicate := []byte(`{"formatVersion":3,"formatVersion":3}`)
	trailing := append(encoded, []byte(` {}`)...)
	for _, value := range [][]byte{unknown, duplicate, trailing} {
		if _, err := catalog.Verify(value, facts, "arm64"); !errors.Is(err, ErrModelIncompatible) {
			t.Fatalf("hostile JSON error=%v", err)
		}
	}
}
