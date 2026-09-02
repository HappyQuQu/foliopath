//go:build !linux || !cgo || !onnxruntime

package onnx

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestUnavailableRuntimeFailsClosed(t *testing.T) {
	t.Parallel()
	_, err := New().LoadAndValidate(context.Background(), aimodel.Model{}, testManifest(), nil)
	if !errors.Is(err, aimodel.ErrInferenceRuntimeUnavailable) {
		t.Fatalf("LoadAndValidate error = %v", err)
	}
	if session, err := New().OpenImageSession(context.Background(), testManifest(), nil); session != nil || !errors.Is(err, aimodel.ErrInferenceRuntimeUnavailable) {
		t.Fatalf("OpenImageSession = %#v, %v", session, err)
	}
	if session, err := New().OpenFaceEmbeddingSession(context.Background(), testManifest(), nil); session != nil || !errors.Is(err, aimodel.ErrInferenceRuntimeUnavailable) {
		t.Fatalf("OpenFaceEmbeddingSession = %#v, %v", session, err)
	}
	if session, err := New().OpenFaceDetectorSession(context.Background(), testManifest(), nil); session != nil || !errors.Is(err, aimodel.ErrInferenceRuntimeUnavailable) {
		t.Fatalf("OpenFaceDetectorSession = %#v, %v", session, err)
	}
}
