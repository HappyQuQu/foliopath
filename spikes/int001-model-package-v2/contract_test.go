package modelpackagev2

import (
	"encoding/json"
	"strings"
	"testing"
)

func validManifest() Manifest {
	return Manifest{
		FormatVersion: 2, PackageID: "semantic-test-v2", Purpose: "semantic_image_text", Version: "1.0.0",
		Architecture: "portable-onnx", LicenseID: "Apache-2.0",
		Contracts: Contracts{ImagePreprocess: ImagePreprocessContract, TextCanonical: TextCanonicalContract,
			Tokenizer: TokenizerContract, EmbeddingAndStorage: EmbeddingContract},
		Files: []File{
			{Name: "image_encoder.onnx", Size: 11, SHA256: strings.Repeat("a", 64), Role: "image_encoder"},
			{Name: "text_encoder.onnx", Size: 12, SHA256: strings.Repeat("b", 64), Role: "text_encoder"},
			{Name: "spiece.model", Size: 13, SHA256: strings.Repeat("c", 64), Role: "sentencepiece_model"},
		},
	}
}

func TestProposedV2AcceptsOnlyExactContract(t *testing.T) {
	encoded, err := json.Marshal(validManifest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(encoded); err != nil {
		t.Fatal(err)
	}
}

func TestProposedV2DoesNotReinterpretV1(t *testing.T) {
	for name, mutate := range map[string]func(*Manifest){
		"v1 number":         func(value *Manifest) { value.FormatVersion = 1 },
		"v1 tokenizer role": func(value *Manifest) { value.Files[2].Role = "tokenizer" },
		"unknown contract":  func(value *Manifest) { value.Contracts.Tokenizer = "sentencepiece-latest" },
		"nested path":       func(value *Manifest) { value.Files[0].Name = "models/image.onnx" },
		"duplicate role":    func(value *Manifest) { value.Files[2].Role = "text_encoder" },
	} {
		t.Run(name, func(t *testing.T) {
			value := validManifest()
			mutate(&value)
			encoded, _ := json.Marshal(value)
			if _, err := Parse(encoded); err == nil {
				t.Fatal("invalid manifest was accepted")
			}
		})
	}
}

func TestProposedV2RejectsUnknownDuplicateAndTrailingJSON(t *testing.T) {
	valid, _ := json.Marshal(validManifest())
	unknown := append(valid[:len(valid)-1], []byte(`,"extra":true}`)...)
	duplicate := []byte(`{"formatVersion":2,"formatVersion":2}`)
	trailing := append(valid, []byte(` {}`)...)
	for _, value := range [][]byte{unknown, duplicate, trailing} {
		if _, err := Parse(value); err == nil {
			t.Fatal("hostile JSON shape was accepted")
		}
	}
}
