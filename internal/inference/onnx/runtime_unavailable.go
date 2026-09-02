//go:build !linux || !cgo || !onnxruntime

package onnx

import (
	"context"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func (*Runtime) LoadAndValidate(
	context.Context,
	aimodel.Model,
	aimodel.Manifest,
	aimodel.RuntimeFileOpener,
) (aimodel.RuntimeMetadata, error) {
	return aimodel.RuntimeMetadata{}, aimodel.ErrInferenceRuntimeUnavailable
}

func (*Runtime) OpenImageSession(
	context.Context,
	aimodel.Manifest,
	aimodel.RuntimeFileOpener,
) (ImageSession, error) {
	return nil, aimodel.ErrInferenceRuntimeUnavailable
}

func (*Runtime) OpenTextSession(
	context.Context,
	aimodel.Manifest,
	aimodel.RuntimeFileOpener,
) (TextSession, error) {
	return nil, aimodel.ErrInferenceRuntimeUnavailable
}

func (*Runtime) OpenFaceEmbeddingSession(
	context.Context,
	aimodel.Manifest,
	aimodel.RuntimeFileOpener,
) (FaceEmbeddingSession, error) {
	return nil, aimodel.ErrInferenceRuntimeUnavailable
}

func (*Runtime) OpenFaceDetectorSession(
	context.Context,
	aimodel.Manifest,
	aimodel.RuntimeFileOpener,
) (FaceDetectorSession, error) {
	return nil, aimodel.ErrInferenceRuntimeUnavailable
}

var _ aimodel.InferenceRuntime = (*Runtime)(nil)
