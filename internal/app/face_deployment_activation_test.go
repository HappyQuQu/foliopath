package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
)

type faceDeploymentCommitStub struct {
	commit face.DeploymentActivationCommit
}

func (r *faceDeploymentCommitStub) CommitFaceDeploymentActivation(_ context.Context, c face.DeploymentActivationCommit) (aimodel.Operation, error) {
	r.commit = c
	return aimodel.Operation{}, nil
}

func TestFaceDeploymentAdapterIdentityAndCommit(t *testing.T) {
	expected := remoteml.ServiceIdentity{Version: 1, Protocol: "foliopath-face-v1", Model: "buffalo_l", Dimension: 512, CacheManifestSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: strings.Repeat("b", 64)}
	var mismatch atomic.Bool
	var hold atomic.Bool
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hold.Load() {
			entered <- struct{}{}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		if r.URL.Path != "/api/v1/face/identity" {
			t.Error(r.URL.Path)
		}
		actual := expected
		if mismatch.Load() {
			actual.Protocol = "foliopath-ml-v1"
		}
		_ = json.NewEncoder(w).Encode(actual)
	}))
	defer server.Close()
	defer unblock()
	client, err := remoteml.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	fc, err := client.BindFace(expected)
	if err != nil {
		t.Fatal(err)
	}
	identity := face.DeploymentIdentity{Version: 1, Protocol: expected.Protocol, Model: expected.Model, Dimension: 512, CacheManifestSHA256: expected.CacheManifestSHA256, RuntimeSHA256: expected.RuntimeContractSHA256,
		PreprocessingSHA256: strings.Repeat("c", 64), DetectorSHA256: strings.Repeat("d", 64), EmbedderSHA256: strings.Repeat("e", 64),
		Threshold: face.ThresholdProfile{SchemaVersion: 1, ProfileID: "test-only", CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2, QualitySummarySHA256: strings.Repeat("f", 64)}}
	repo := &faceDeploymentCommitStub{}
	runtime, err := newFaceDeploymentActivationRuntime(fc, repo, identity)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, _ := identity.Fingerprint()
	now := time.Now().UTC()
	model := aimodel.Model{ID: "aim_face_adapter", Package: aimodel.VerifiedPackage{PackageID: "buffalo_l", Purpose: aimodel.PurposeFaceDetectionEmbedding, Version: "1", Architecture: "arm64", ContentHash: fingerprint, LicenseID: "test-only", PackageSizeByte: 1}, StorageMode: aimodel.StorageService, State: aimodel.StateAvailable, SourceIdentity: fingerprint, AvailabilityRevision: 1, CreatedAt: now, UpdatedAt: now}
	metadata, err := runtime.LoadAndValidateFaceDeployment(context.Background(), model)
	if err != nil {
		t.Fatal(err)
	}
	generation := aimodel.FaceGeneration{ID: "face_adapter_generation", ModelID: model.ID, PackageID: "buffalo_l", DetectorContentHash: metadata.DetectorContentHash, EmbedderContentHash: metadata.EmbedderContentHash, EmbeddingDimension: 512, TransformVersion: aimodel.FaceTransformVersion, ThresholdProfile: metadata.ThresholdProfile, ThresholdProfileHash: metadata.ThresholdProfileHash, State: aimodel.GenerationActive, CreatedAt: now, ActivatedAt: &now, UpdatedAt: now}
	if _, err := runtime.CommitFaceDeploymentActivation(context.Background(), aimodel.FaceActivationCommit{Generation: generation}); err != nil || repo.commit.Identity != identity {
		t.Fatal(err)
	}
	generation.DetectorContentHash = strings.Repeat("0", 64)
	if _, err := runtime.CommitFaceDeploymentActivation(context.Background(), aimodel.FaceActivationCommit{Generation: generation}); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatal(err)
	}
	// Occupy both shared slots. Activation must wait without consuming a
	// failed attempt, and cancellation must still interrupt that wait.
	hold.Store(true)
	held := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { held <- fc.VerifyIdentity(context.Background()) }()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("slots not occupied")
		}
	}
	waitCtx, stop := context.WithTimeout(context.Background(), 30*time.Millisecond)
	_, waitErr := runtime.LoadAndValidateFaceDeployment(waitCtx, model)
	stop()
	if !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Fatal("budget wait must be cancellable", waitErr)
	}
	loaded := make(chan error, 1)
	go func() { _, err := runtime.LoadAndValidateFaceDeployment(context.Background(), model); loaded <- err }()
	select {
	case err := <-loaded:
		t.Fatal("activation returned while busy", err)
	case <-time.After(30 * time.Millisecond):
	}
	hold.Store(false)
	unblock()
	for i := 0; i < 2; i++ {
		if err := <-held; err != nil {
			t.Fatal(err)
		}
	}
	select {
	case err := <-loaded:
		if err != nil {
			t.Fatal("activation did not resume", err)
		}
	case <-time.After(time.Second):
		t.Fatal("activation stuck")
	}
	mismatch.Store(true)
	if _, err := runtime.LoadAndValidateFaceDeployment(context.Background(), model); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runtime.LoadAndValidateFaceDeployment(ctx, model); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	identity.RuntimeSHA256 = strings.Repeat("0", 64)
	if _, err := newFaceDeploymentActivationRuntime(fc, repo, identity); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatal(err)
	}
}
