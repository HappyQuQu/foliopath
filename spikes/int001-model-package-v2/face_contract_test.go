package modelpackagev2

import (
	"encoding/json"
	"strings"
	"testing"
)

func validFaceManifestFixture() FaceManifest {
	return FaceManifest{
		FormatVersion: FaceFormatVersion,
		PackageID:     "face-yunet-auraface-candidate-v3",
		Purpose:       "face_detection_embedding",
		Version:       "0.0.0-candidate",
		Architecture:  "portable-onnx",
		Contracts: FaceContracts{
			Decode: FaceDecodeContract, Detector: FaceDetectorContract, Postprocess: FacePostprocessContract,
			Alignment: FaceAlignmentContract, Embedding: FaceEmbeddingContract, Storage: FaceStorageContract,
			ThresholdProfile: FaceThresholdContract,
		},
		Files: []FaceFile{
			{Name: "face_detector.onnx", Size: 232589, SHA256: strings.Repeat("a", 64), Role: "face_detector", LicenseID: "MIT"},
			{Name: "face_embedder.onnx", Size: 260694151, SHA256: strings.Repeat("b", 64), Role: "face_embedder", LicenseID: "Apache-2.0"},
			{Name: "threshold_profile.json", Size: 512, SHA256: strings.Repeat("c", 64), Role: faceThresholdProfileRole, LicenseID: "Apache-2.0"},
		},
	}
}

func TestProposedFaceV3AcceptsOnlyCompleteCombinedPackage(t *testing.T) {
	encoded, err := json.Marshal(validFaceManifestFixture())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFaceV3(encoded); err != nil {
		t.Fatal(err)
	}
}

func TestProposedFaceV3RejectsConfusedOrUnboundComponents(t *testing.T) {
	for name, mutate := range map[string]func(*FaceManifest){
		"semantic format":    func(value *FaceManifest) { value.FormatVersion = 2 },
		"semantic purpose":   func(value *FaceManifest) { value.Purpose = "semantic_image_text" },
		"missing thresholds": func(value *FaceManifest) { value.Files = value.Files[:2] },
		"duplicate role":     func(value *FaceManifest) { value.Files[2].Role = "face_embedder" },
		"unknown alignment":  func(value *FaceManifest) { value.Contracts.Alignment = "latest" },
		"missing license":    func(value *FaceManifest) { value.Files[1].LicenseID = "" },
		"nested model":       func(value *FaceManifest) { value.Files[0].Name = "models/detector.onnx" },
		"path package id":    func(value *FaceManifest) { value.PackageID = "../face" },
		"control package id": func(value *FaceManifest) { value.PackageID = "face\nmodel" },
		"unicode version":    func(value *FaceManifest) { value.Version = "候选" },
	} {
		t.Run(name, func(t *testing.T) {
			value := validFaceManifestFixture()
			mutate(&value)
			encoded, _ := json.Marshal(value)
			if _, err := ParseFaceV3(encoded); err == nil {
				t.Fatal("invalid face package was accepted")
			}
		})
	}
}

func TestProposedFaceV3RejectsUnknownDuplicateAndTrailingJSON(t *testing.T) {
	valid, _ := json.Marshal(validFaceManifestFixture())
	unknown := append(valid[:len(valid)-1], []byte(`,"redistributionApproved":true}`)...)
	duplicate := []byte(`{"formatVersion":3,"formatVersion":3}`)
	trailing := append(valid, []byte(` {}`)...)
	for _, value := range [][]byte{unknown, duplicate, trailing} {
		if _, err := ParseFaceV3(value); err == nil {
			t.Fatal("hostile face manifest shape was accepted")
		}
	}
}
