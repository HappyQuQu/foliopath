//go:build linux && cgo && onnxruntime && inferencecandidate

package onnx

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

// Opt-in candidate evidence only; neither this test nor its fixture authorizes a catalog entry.
func TestNativeSemanticConvertedCandidate(t *testing.T) {
	directory := os.Getenv("FOLIOPATH_SIGLIP_CANDIDATE_DIR")
	if directory == "" {
		t.Skip("set FOLIOPATH_SIGLIP_CANDIDATE_DIR to the isolated converted fixture")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, fixture := range []struct{ name, role, hash string }{
		{"image_encoder.onnx", "image_encoder", "941bc2649234f3dbbc4ae3c77059bb9969f2aa47ca57d6b7a4e081bb05157a2e"},
		{"text_encoder.onnx", "text_encoder", "0b9cdf435028e23ddee07c0943c881ab3b017203a409c9918742f63a866ed91f"},
	} {
		t.Run(fixture.role, func(t *testing.T) {
			file, err := os.Open(filepath.Join(directory, fixture.name))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			hash := sha256.New()
			size, err := io.Copy(hash, file)
			if err != nil || fmt.Sprintf("%x", hash.Sum(nil)) != fixture.hash {
				t.Fatal("candidate hash mismatch", err)
			}
			manifest := aimodel.Manifest{Files: []aimodel.ManifestFile{{Name: fixture.name, Role: fixture.role, Size: size}}}
			opener := func(context.Context, string) (aimodel.RuntimeModelFile, error) {
				return &semanticCandidateFile{File: file, size: size}, nil
			}
			var values []float32
			if fixture.role == "image_encoder" {
				session, err := New().OpenImageSession(ctx, manifest, opener)
				if err != nil {
					t.Fatal(err)
				}
				defer session.Close()
				values, err = session.Encode(ctx, make([]float32, 3*224*224))
				if err != nil {
					t.Fatal(err)
				}
			} else {
				session, err := New().OpenTextSession(ctx, manifest, opener)
				if err != nil {
					t.Fatal(err)
				}
				defer session.Close()
				ids := make([]int64, 64)
				for i := range ids {
					ids[i] = 1
				}
				ids[0], ids[1], ids[2] = 1226, 14431, 9391
				values, err = session.EncodeText(ctx, ids)
				if err != nil {
					t.Fatal(err)
				}
			}
			if len(values) != int(EmbeddingDimension) {
				t.Fatalf("dimension: %d", len(values))
			}
			var norm float64
			for _, value := range values {
				if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
					t.Fatal("non-finite embedding")
				}
				norm += float64(value) * float64(value)
			}
			if norm <= 0 {
				t.Fatal("zero embedding")
			}
		})
	}
}

type semanticCandidateFile struct {
	*os.File
	size int64
}

func (file *semanticCandidateFile) RuntimePath() string {
	return fmt.Sprintf("/proc/self/fd/%d", file.Fd())
}
func (file *semanticCandidateFile) Size() int64 { return file.size }
