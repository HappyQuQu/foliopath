package app

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

// Composition adapter: one identity-bound HTTP client is shared with encoders. It
// performs no model-file access and leaves activation transactions to SQLite.
type deploymentActivationRuntime struct {
	client      *remoteml.Client
	repository  semantic.DeploymentActivationRepository
	space       semantic.SpaceIdentity
	fingerprint string
}

func newDeploymentActivationRuntime(client *remoteml.Client, repository semantic.DeploymentActivationRepository, space semantic.SpaceIdentity) (*deploymentActivationRuntime, error) {
	if client == nil || repository == nil {
		return nil, aimodel.ErrModelIncompatible
	}
	expected, configured := client.Identity()
	fingerprint, err := space.Fingerprint()
	if err != nil || !configured || space.Protocol != expected.Protocol || space.Model != expected.Model || space.Dimension != expected.Dimension ||
		space.RuntimeSHA256 != expected.RuntimeContractSHA256 || space.CacheManifestSHA256 != expected.CacheManifestSHA256 {
		return nil, aimodel.ErrModelIncompatible
	}
	return &deploymentActivationRuntime{client, repository, space, fingerprint}, nil
}

func (r *deploymentActivationRuntime) LoadAndValidateDeployment(ctx context.Context, model aimodel.Model) (aimodel.RuntimeMetadata, error) {
	if aimodel.ValidateModel(model) != nil || model.StorageMode != aimodel.StorageService || model.Package.ContentHash != r.fingerprint {
		return aimodel.RuntimeMetadata{}, aimodel.ErrModelIncompatible
	}
	if err := r.client.VerifyIdentity(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return aimodel.RuntimeMetadata{}, err
		}
		if errors.Is(err, remoteml.ErrIdentityMismatch) {
			return aimodel.RuntimeMetadata{}, aimodel.ErrModelIncompatible
		}
		return aimodel.RuntimeMetadata{}, aimodel.ErrInferenceRuntimeUnavailable
	}
	return aimodel.RuntimeMetadata{EmbeddingDimension: int64(r.space.Dimension)}, nil
}

func (r *deploymentActivationRuntime) CommitDeploymentActivation(ctx context.Context, commit aimodel.ActivationCommit) (aimodel.Operation, error) {
	return r.repository.CommitDeploymentActivation(ctx, semantic.DeploymentActivationCommit{Activation: commit, Identity: r.space})
}

var _ aimodel.DeploymentActivationRuntime = (*deploymentActivationRuntime)(nil)

func (service *databaseService) CommitDeploymentActivation(ctx context.Context, commit semantic.DeploymentActivationCommit) (aimodel.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Operation{}, errors.New("deployment activation repository is not ready")
	}
	return service.store.CommitDeploymentActivation(ctx, commit)
}
