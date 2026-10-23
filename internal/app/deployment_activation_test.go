package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type deploymentCommitStub struct {
	commit semantic.DeploymentActivationCommit
}

func (r *deploymentCommitStub) CommitDeploymentActivation(_ context.Context, c semantic.DeploymentActivationCommit) (aimodel.Operation, error) {
	r.commit = c
	return aimodel.Operation{}, nil
}

func TestDeploymentActivationAdapterRechecksIdentity(t *testing.T) {
	ctx := context.Background()
	identity := remoteml.ServiceIdentity{Version: 1, Protocol: "foliopath-ml-v1", Model: remoteml.ModelName, Dimension: 512, CacheManifestSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: strings.Repeat("b", 64)}
	var mismatch atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actual := identity
		if mismatch.Load() {
			actual.Dimension = 768
		}
		_ = json.NewEncoder(w).Encode(actual)
	}))
	defer server.Close()
	client, err := remoteml.NewVerified(ctx, server.URL, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	space := semantic.SpaceIdentity{Version: 1, Protocol: identity.Protocol, Model: identity.Model, Dimension: 512, RuntimeSHA256: identity.RuntimeContractSHA256, CacheManifestSHA256: identity.CacheManifestSHA256, PreprocessingSHA256: strings.Repeat("c", 64)}
	repository := &deploymentCommitStub{}
	runtime, err := newDeploymentActivationRuntime(client, repository, space)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := space.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	model := aimodel.Model{ID: "aim_deployment_adapter", Package: aimodel.VerifiedPackage{PackageID: "service", Purpose: aimodel.PurposeSemanticImageText, Version: "1", Architecture: "arm64", ContentHash: fingerprint, LicenseID: "MIT", PackageSizeByte: 1}, StorageMode: aimodel.StorageService, State: aimodel.StateAvailable, SourceIdentity: fingerprint, AvailabilityRevision: 1, CreatedAt: now, UpdatedAt: now}
	if metadata, err := runtime.LoadAndValidateDeployment(ctx, model); err != nil || metadata.EmbeddingDimension != 512 {
		t.Fatalf("metadata=%+v %v", metadata, err)
	}
	commit := aimodel.ActivationCommit{OperationID: "aio_deployment_adapter"}
	if _, err := runtime.CommitDeploymentActivation(ctx, commit); err != nil || repository.commit.Identity != space || repository.commit.Activation.OperationID != commit.OperationID {
		t.Fatal("lost bound commit identity")
	}
	mismatch.Store(true)
	if _, err := runtime.LoadAndValidateDeployment(ctx, model); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatalf("mismatch accepted: %v", err)
	}
	wrong := space
	wrong.Model = "other"
	if _, err := newDeploymentActivationRuntime(client, repository, wrong); err == nil {
		t.Fatal("constructed incompatible adapter")
	}
	raw, err := remoteml.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := newDeploymentActivationRuntime(raw, repository, space); err == nil {
		t.Fatal("accepted unverified client")
	}
}
