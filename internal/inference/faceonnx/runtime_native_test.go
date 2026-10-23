//go:build linux && cgo && libvips && onnxruntime

package faceonnx

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
)

func TestNativeFacePipelineCandidate(t *testing.T) {
	detectorPath := os.Getenv("FOLIOPATH_ORT_FACE_DETECTOR")
	embedderPath := os.Getenv("FOLIOPATH_ORT_FACE_MODEL")
	imagePath := os.Getenv("FOLIOPATH_FACE_IMAGE")
	if detectorPath == "" || embedderPath == "" || imagePath == "" {
		t.Skip("set the reviewed detector, embedder, and face image fixtures")
	}
	runtime := imagevips.NewRuntime()
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runtime.Shutdown)
	detector := openNativeDetector(t, detectorPath)
	defer detector.Close()
	embedder := openNativeEmbedder(t, embedderPath)
	defer embedder.Close()
	pipeline, err := New(imagevips.New(), detector, embedder)
	if err != nil {
		t.Fatal(err)
	}
	image, err := os.Open(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	candidates, err := pipeline.AnalyzeFaces(context.Background(), image, media.FormatJPEG, 250*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) == 0 || len(candidates) > 64 {
		t.Fatalf("candidate count=%d", len(candidates))
	}
	for _, candidate := range candidates {
		if len(candidate.Embedding) != int(onnx.FaceEmbeddingDimension) {
			t.Fatalf("embedding dimension=%d", len(candidate.Embedding))
		}
		var norm float64
		for _, value := range candidate.Embedding {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				t.Fatal("non-finite face embedding")
			}
			norm += float64(value) * float64(value)
		}
		if norm == 0 {
			t.Fatal("zero face embedding")
		}
	}
	t.Logf(
		"candidate_count=%d quantized_1e3_sha256=%x",
		len(candidates),
		quantizedCandidateFingerprint(candidates),
	)
}

func quantizedCandidateFingerprint(candidates []face.Candidate) [sha256.Size]byte {
	hash := sha256.New()
	var encoded [8]byte
	writeInt64 := func(value int64) {
		binary.LittleEndian.PutUint64(encoded[:], uint64(value))
		_, _ = hash.Write(encoded[:])
	}
	writeFloat := func(value float32) {
		writeInt64(int64(math.Round(float64(value) * 1000)))
	}

	writeInt64(int64(len(candidates)))
	for _, candidate := range candidates {
		writeInt64(int64(len(candidate.Embedding)))
		writeFloat(candidate.Box.X)
		writeFloat(candidate.Box.Y)
		writeFloat(candidate.Box.Width)
		writeFloat(candidate.Box.Height)
		writeFloat(candidate.Detection)
		writeFloat(candidate.Quality)
		for _, value := range candidate.Embedding {
			writeFloat(value)
		}
	}

	var fingerprint [sha256.Size]byte
	copy(fingerprint[:], hash.Sum(nil))
	return fingerprint
}

func openNativeDetector(t *testing.T, path string) onnx.FaceDetectorSession {
	t.Helper()
	file, info := openNativeModel(t, path)
	manifest := aimodel.Manifest{Files: []aimodel.ManifestFile{{Name: "face_detector.onnx", Role: "face_detector", Size: info.Size()}}}
	session, err := onnx.New().OpenFaceDetectorSession(context.Background(), manifest, func(context.Context, string) (aimodel.RuntimeModelFile, error) {
		return &nativeRuntimeFile{File: file, size: info.Size()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func openNativeEmbedder(t *testing.T, path string) onnx.FaceEmbeddingSession {
	t.Helper()
	file, info := openNativeModel(t, path)
	manifest := aimodel.Manifest{Files: []aimodel.ManifestFile{{Name: "face_embedder.onnx", Role: "face_embedder", Size: info.Size()}}}
	session, err := onnx.New().OpenFaceEmbeddingSession(context.Background(), manifest, func(context.Context, string) (aimodel.RuntimeModelFile, error) {
		return &nativeRuntimeFile{File: file, size: info.Size()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func openNativeModel(t *testing.T, path string) (*os.File, os.FileInfo) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	return file, info
}

type nativeRuntimeFile struct {
	*os.File
	size int64
}

func (file *nativeRuntimeFile) RuntimePath() string {
	return "/proc/self/fd/" + strconv.FormatUint(uint64(file.Fd()), 10)
}
func (file *nativeRuntimeFile) Size() int64 { return file.size }
