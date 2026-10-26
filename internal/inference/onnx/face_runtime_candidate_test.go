//go:build linux && cgo && onnxruntime

package onnx

import (
	"context"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

const faceCandidateModelEnv = "FOLIOPATH_ORT_FACE_MODEL"
const faceDetectorCandidateModelEnv = "FOLIOPATH_ORT_FACE_DETECTOR"

func TestNativeFaceEmbeddingCandidate(t *testing.T) {
	modelPath := os.Getenv(faceCandidateModelEnv)
	if modelPath == "" {
		t.Skip("set FOLIOPATH_ORT_FACE_MODEL to the reviewed face embedder fixture")
	}
	file, err := os.Open(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	manifest := aimodel.Manifest{Files: []aimodel.ManifestFile{{Name: "face_embedder.onnx", Role: "face_embedder", Size: info.Size()}}}
	session, err := New().OpenFaceEmbeddingSession(context.Background(), manifest, func(context.Context, string) (aimodel.RuntimeModelFile, error) {
		return &nativeFaceRuntimeFile{File: file, size: info.Size()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	output, err := session.EmbedFace(context.Background(), make([]float32, faceEmbeddingTensorElements))
	if err != nil || len(output) != int(FaceEmbeddingDimension) {
		t.Fatalf("face embedding dimension=%d err=%v", len(output), err)
	}
	var norm float64
	for _, value := range output {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			t.Fatal("face embedder returned non-finite output")
		}
		norm += float64(value) * float64(value)
	}
	if norm == 0 {
		t.Fatal("face embedder returned a zero vector")
	}
}

func TestNativeFaceDetectorCandidate(t *testing.T) {
	modelPath := os.Getenv(faceDetectorCandidateModelEnv)
	if modelPath == "" {
		t.Skip("set FOLIOPATH_ORT_FACE_DETECTOR to the reviewed face detector fixture")
	}
	file, err := os.Open(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	manifest := aimodel.Manifest{Files: []aimodel.ManifestFile{{Name: "face_detector.onnx", Role: "face_detector", Size: info.Size()}}}
	session, err := New().OpenFaceDetectorSession(context.Background(), manifest, func(context.Context, string) (aimodel.RuntimeModelFile, error) {
		return &nativeFaceRuntimeFile{File: file, size: info.Size()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	detections, err := session.DetectFaces(context.Background(), make([]float32, faceDetectorTensorElements))
	if err != nil {
		t.Fatal(err)
	}
	for _, detection := range detections {
		values := []float32{detection.X, detection.Y, detection.Width, detection.Height, detection.Score}
		for _, point := range detection.Landmarks {
			values = append(values, point.X, point.Y)
		}
		for _, value := range values {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				t.Fatal("face detector returned non-finite output")
			}
		}
	}
}

type nativeFaceRuntimeFile struct {
	*os.File
	size int64
}

func (file *nativeFaceRuntimeFile) RuntimePath() string {
	return "/proc/self/fd/" + strconv.FormatUint(uint64(file.Fd()), 10)
}
func (file *nativeFaceRuntimeFile) Size() int64 { return file.size }
