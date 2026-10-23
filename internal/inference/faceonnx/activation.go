package faceonnx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
)

type activationSessionFactory interface {
	OpenFaceDetectorSession(context.Context, aimodel.Manifest, aimodel.RuntimeFileOpener) (onnx.FaceDetectorSession, error)
	OpenFaceEmbeddingSession(context.Context, aimodel.Manifest, aimodel.RuntimeFileOpener) (onnx.FaceEmbeddingSession, error)
}

// ActivationRuntime validates the indivisible detector/embedder/profile
// package before the model lifecycle can publish a face generation. It does
// not retain sessions; library workers open generation-bound sessions later.
type ActivationRuntime struct {
	sessions activationSessionFactory
}

func NewActivationRuntime(sessions activationSessionFactory) (*ActivationRuntime, error) {
	if sessions == nil {
		return nil, aimodel.ErrInvalidModel
	}
	return &ActivationRuntime{sessions: sessions}, nil
}

func (runtime *ActivationRuntime) LoadAndValidateFace(
	ctx context.Context,
	model aimodel.Model,
	manifest aimodel.Manifest,
	open aimodel.RuntimeFileOpener,
) (aimodel.FaceRuntimeMetadata, error) {
	if runtime == nil || runtime.sessions == nil || open == nil ||
		model.Package.Purpose != aimodel.PurposeFaceDetectionEmbedding ||
		manifest.FormatVersion != aimodel.FaceFormatVersion ||
		manifest.Purpose != aimodel.PurposeFaceDetectionEmbedding || manifest.Contracts == nil {
		return aimodel.FaceRuntimeMetadata{}, aimodel.ErrModelIncompatible
	}
	if err := ctx.Err(); err != nil {
		return aimodel.FaceRuntimeMetadata{}, err
	}
	detector, err := runtime.sessions.OpenFaceDetectorSession(ctx, manifest, open)
	if err != nil {
		return aimodel.FaceRuntimeMetadata{}, err
	}
	detections, detectorRunErr := detector.DetectFaces(ctx, make([]float32, 3*onnx.FaceDetectorWidth*onnx.FaceDetectorHeight))
	detectorCloseErr := detector.Close()
	if detectorRunErr != nil || detectorCloseErr != nil {
		return aimodel.FaceRuntimeMetadata{}, errors.Join(detectorRunErr, detectorCloseErr)
	}
	for _, detection := range detections {
		values := []float32{detection.X, detection.Y, detection.Width, detection.Height, detection.Score}
		for _, point := range detection.Landmarks {
			values = append(values, point.X, point.Y)
		}
		for _, value := range values {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return aimodel.FaceRuntimeMetadata{}, aimodel.ErrModelIncompatible
			}
		}
	}

	embedder, err := runtime.sessions.OpenFaceEmbeddingSession(ctx, manifest, open)
	if err != nil {
		return aimodel.FaceRuntimeMetadata{}, err
	}
	embedding, embedderRunErr := embedder.EmbedFace(ctx, make([]float32, 3*alignedFaceSize*alignedFaceSize))
	embedderCloseErr := embedder.Close()
	if embedderRunErr != nil || embedderCloseErr != nil {
		return aimodel.FaceRuntimeMetadata{}, errors.Join(embedderRunErr, embedderCloseErr)
	}
	if len(embedding) != int(onnx.FaceEmbeddingDimension) {
		return aimodel.FaceRuntimeMetadata{}, aimodel.ErrModelIncompatible
	}
	var norm float64
	for _, value := range embedding {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return aimodel.FaceRuntimeMetadata{}, aimodel.ErrModelIncompatible
		}
		norm += float64(value) * float64(value)
	}
	if norm == 0 || math.IsNaN(norm) || math.IsInf(norm, 0) {
		return aimodel.FaceRuntimeMetadata{}, aimodel.ErrModelIncompatible
	}

	profile, err := openThresholdProfile(ctx, manifest, open)
	if err != nil {
		return aimodel.FaceRuntimeMetadata{}, err
	}
	return aimodel.FaceRuntimeMetadata{
		EmbeddingDimension: onnx.FaceEmbeddingDimension,
		ThresholdProfile:   profile.ProfileID,
	}, nil
}

func openThresholdProfile(
	ctx context.Context,
	manifest aimodel.Manifest,
	open aimodel.RuntimeFileOpener,
) (face.ThresholdProfile, error) {
	var expected *aimodel.ManifestFile
	for index := range manifest.Files {
		if manifest.Files[index].Role == "face_threshold_profile" {
			if expected != nil {
				return face.ThresholdProfile{}, aimodel.ErrModelIncompatible
			}
			expected = &manifest.Files[index]
		}
	}
	if expected == nil || expected.Size < 1 || expected.Size > face.MaxThresholdProfileBytes {
		return face.ThresholdProfile{}, aimodel.ErrModelIncompatible
	}
	if err := ctx.Err(); err != nil {
		return face.ThresholdProfile{}, err
	}
	file, err := open(ctx, expected.Name)
	if err != nil {
		return face.ThresholdProfile{}, err
	}
	reader, ok := file.(io.Reader)
	if file == nil || !ok || file.Size() != expected.Size {
		if file != nil {
			_ = file.Close()
		}
		return face.ThresholdProfile{}, aimodel.ErrModelIncompatible
	}
	content, readErr := io.ReadAll(io.LimitReader(reader, face.MaxThresholdProfileBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return face.ThresholdProfile{}, errors.Join(readErr, closeErr)
	}
	if int64(len(content)) != expected.Size {
		return face.ThresholdProfile{}, aimodel.ErrModelIncompatible
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != expected.SHA256 {
		return face.ThresholdProfile{}, aimodel.ErrModelIncompatible
	}
	profile, err := face.ParseThresholdProfile(content)
	if err != nil {
		return face.ThresholdProfile{}, aimodel.ErrModelIncompatible
	}
	return profile, nil
}

var _ aimodel.FaceInferenceRuntime = (*ActivationRuntime)(nil)
