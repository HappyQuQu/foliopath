package faceonnx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
)

type activationSessionFactoryStub struct {
	detector onnx.FaceDetectorSession
	embedder onnx.FaceEmbeddingSession
}

func (stub activationSessionFactoryStub) OpenFaceDetectorSession(
	ctx context.Context, manifest aimodel.Manifest, open aimodel.RuntimeFileOpener,
) (onnx.FaceDetectorSession, error) {
	file, err := openActivationRole(ctx, manifest, "face_detector", open)
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	return stub.detector, nil
}

func (stub activationSessionFactoryStub) OpenFaceEmbeddingSession(
	ctx context.Context, manifest aimodel.Manifest, open aimodel.RuntimeFileOpener,
) (onnx.FaceEmbeddingSession, error) {
	file, err := openActivationRole(ctx, manifest, "face_embedder", open)
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	return stub.embedder, nil
}

func openActivationRole(ctx context.Context, manifest aimodel.Manifest, role string, open aimodel.RuntimeFileOpener) (aimodel.RuntimeModelFile, error) {
	for _, file := range manifest.Files {
		if file.Role == role {
			return open(ctx, file.Name)
		}
	}
	return nil, aimodel.ErrModelIncompatible
}

type activationRuntimeFile struct {
	*bytes.Reader
	size   int64
	closed bool
}

func (file *activationRuntimeFile) Close() error        { file.closed = true; return nil }
func (file *activationRuntimeFile) RuntimePath() string { return "/proc/self/fd/9" }
func (file *activationRuntimeFile) Size() int64         { return file.size }

func TestActivationRuntimeValidatesBothGraphsAndGovernedThresholdProfile(t *testing.T) {
	profileBytes, manifest := activationFaceManifest(t)
	detector := &detectorStub{}
	embedding := make([]float32, onnx.FaceEmbeddingDimension)
	embedding[0] = 1
	embedder := &embedderStub{output: embedding}
	runtime, err := NewActivationRuntime(activationSessionFactoryStub{detector: detector, embedder: embedder})
	if err != nil {
		t.Fatal(err)
	}
	opened := []string{}
	opener := func(_ context.Context, name string) (aimodel.RuntimeModelFile, error) {
		opened = append(opened, name)
		content := []byte("model")
		if name == "threshold_profile.json" {
			content = profileBytes
		}
		return &activationRuntimeFile{Reader: bytes.NewReader(content), size: int64(len(content))}, nil
	}
	now := time.Date(2026, 9, 1, 22, 0, 0, 0, time.UTC)
	metadata, err := runtime.LoadAndValidateFace(context.Background(), aimodel.Model{
		ID: "aim_face_activation", Package: aimodel.VerifiedPackage{Purpose: aimodel.PurposeFaceDetectionEmbedding},
		CreatedAt: now, UpdatedAt: now,
	}, manifest, opener)
	if err != nil || metadata.EmbeddingDimension != 512 || metadata.ThresholdProfile != "face-reviewed-v1" ||
		len(opened) != 3 || !detector.closed || !embedder.closed {
		t.Fatalf("metadata=%+v err=%v opened=%v detector=%v embedder=%v", metadata, err, opened, detector.closed, embedder.closed)
	}
}

func TestActivationRuntimeRejectsThresholdDigestAndEmbeddingFailures(t *testing.T) {
	profileBytes, manifest := activationFaceManifest(t)
	validEmbedding := make([]float32, onnx.FaceEmbeddingDimension)
	validEmbedding[0] = 1
	for name, testCase := range map[string]struct {
		embedder      *embedderStub
		mutateProfile func([]byte) []byte
	}{
		"zero embedding":  {embedder: &embedderStub{output: make([]float32, onnx.FaceEmbeddingDimension)}, mutateProfile: func(value []byte) []byte { return value }},
		"changed profile": {embedder: &embedderStub{output: validEmbedding}, mutateProfile: func(value []byte) []byte { return append(append([]byte(nil), value...), ' ') }},
	} {
		t.Run(name, func(t *testing.T) {
			runtime, err := NewActivationRuntime(activationSessionFactoryStub{detector: &detectorStub{}, embedder: testCase.embedder})
			if err != nil {
				t.Fatal(err)
			}
			opener := func(_ context.Context, filename string) (aimodel.RuntimeModelFile, error) {
				content := []byte("model")
				if filename == "threshold_profile.json" {
					content = testCase.mutateProfile(profileBytes)
				}
				return &activationRuntimeFile{Reader: bytes.NewReader(content), size: int64(len(content))}, nil
			}
			now := time.Now().UTC()
			_, err = runtime.LoadAndValidateFace(context.Background(), aimodel.Model{
				ID: "aim_face_activation", Package: aimodel.VerifiedPackage{Purpose: aimodel.PurposeFaceDetectionEmbedding},
				CreatedAt: now, UpdatedAt: now,
			}, manifest, opener)
			if !errors.Is(err, aimodel.ErrModelIncompatible) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func activationFaceManifest(t *testing.T) ([]byte, aimodel.Manifest) {
	t.Helper()
	profileBytes, err := json.Marshal(face.ThresholdProfile{
		SchemaVersion: 1, ProfileID: "face-reviewed-v1", CoreSimilarity: .75, EdgeSimilarity: .65,
		MinCoreSize: 2, QualitySummarySHA256: strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(profileBytes)
	contracts := &aimodel.FaceContracts{
		Decode: aimodel.FaceDecodeContract, Detector: aimodel.FaceDetectorContract,
		Postprocess: aimodel.FacePostprocessContract, Alignment: aimodel.FaceAlignmentContract,
		Embedding: aimodel.FaceEmbeddingContract, Storage: aimodel.FaceStorageContract,
		ThresholdProfile: aimodel.FaceThresholdContract,
	}
	manifest := aimodel.Manifest{FormatVersion: aimodel.FaceFormatVersion, PackageID: "face-package-v3",
		Purpose: aimodel.PurposeFaceDetectionEmbedding, Version: "1.0.0", Architecture: "portable-onnx",
		Contracts: contracts, Files: []aimodel.ManifestFile{
			{Name: "face_detector.onnx", Size: 5, SHA256: strings.Repeat("b", 64), Role: "face_detector", LicenseID: "MIT"},
			{Name: "face_embedder.onnx", Size: 5, SHA256: strings.Repeat("c", 64), Role: "face_embedder", LicenseID: "Apache-2.0"},
			{Name: "threshold_profile.json", Size: int64(len(profileBytes)), SHA256: hex.EncodeToString(digest[:]), Role: "face_threshold_profile", LicenseID: "Apache-2.0"},
		}}
	return profileBytes, manifest
}

var _ io.ReadCloser = (*activationRuntimeFile)(nil)
