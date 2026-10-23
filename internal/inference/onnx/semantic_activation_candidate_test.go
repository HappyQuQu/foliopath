//go:build linux && cgo && onnxruntime && sentencepiece && inferencecandidate

package onnx

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestNativeSemanticCandidateRuntimeValidation(t *testing.T) {
	directory := os.Getenv("FOLIOPATH_SIGLIP_CANDIDATE_DIR")
	if directory == "" {
		t.Skip("set FOLIOPATH_SIGLIP_CANDIDATE_DIR to the isolated converted fixture")
	}
	manifest := aimodel.Manifest{FormatVersion: aimodel.SemanticFormatVersion, Files: []aimodel.ManifestFile{
		{Name: "image_encoder.onnx", Role: "image_encoder", Size: 371674312, SHA256: "941bc2649234f3dbbc4ae3c77059bb9969f2aa47ca57d6b7a4e081bb05157a2e"},
		{Name: "text_encoder.onnx", Role: "text_encoder", Size: 441295454, SHA256: "0b9cdf435028e23ddee07c0943c881ab3b017203a409c9918742f63a866ed91f"},
		{Name: "spiece.model", Role: "sentencepiece_model", Size: 798330, SHA256: "1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec"},
	}}
	opener := func(_ context.Context, name string) (aimodel.RuntimeModelFile, error) {
		for _, expected := range manifest.Files {
			if name != expected.Name {
				continue
			}
			file, err := os.Open(filepath.Join(directory, name))
			if err != nil {
				return nil, err
			}
			hash := sha256.New()
			size, err := io.Copy(hash, file)
			if err != nil || size != expected.Size || fmt.Sprintf("%x", hash.Sum(nil)) != expected.SHA256 {
				file.Close()
				return nil, fmt.Errorf("candidate hash/size mismatch")
			}
			return &semanticCandidateFile{File: file, size: size}, nil
		}
		return nil, fmt.Errorf("unexpected fixture file")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	metadata, err := New().LoadAndValidate(ctx, aimodel.Model{}, manifest, opener)
	if err != nil || metadata.EmbeddingDimension != 768 {
		t.Fatalf("runtime validation: %+v %v", metadata, err)
	}
	// No activation transaction, registry admission, or generation commit occurs here.
}
