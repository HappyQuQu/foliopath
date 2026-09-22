package app

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
)

type faceDeploymentActivationRuntime struct {
	client      *remoteml.FaceClient
	repository  face.DeploymentActivationRepository
	identity    face.DeploymentIdentity
	fingerprint string
}

func newFaceDeploymentActivationRuntime(client *remoteml.FaceClient, repository face.DeploymentActivationRepository, identity face.DeploymentIdentity) (*faceDeploymentActivationRuntime, error) {
	if client == nil || repository == nil {
		return nil, aimodel.ErrModelIncompatible
	}
	fingerprint, err := identity.Fingerprint()
	expected := client.Identity()
	if err != nil || expected.Protocol != identity.Protocol || expected.Model != identity.Model || expected.Dimension != identity.Dimension ||
		expected.RuntimeContractSHA256 != identity.RuntimeSHA256 || expected.CacheManifestSHA256 != identity.CacheManifestSHA256 {
		return nil, aimodel.ErrModelIncompatible
	}
	return &faceDeploymentActivationRuntime{client, repository, identity, fingerprint}, nil
}

func (r *faceDeploymentActivationRuntime) LoadAndValidateFaceDeployment(ctx context.Context, model aimodel.Model) (aimodel.DeploymentFaceMetadata, error) {
	if aimodel.ValidateModel(model) != nil || model.StorageMode != aimodel.StorageService || model.Package.Purpose != aimodel.PurposeFaceDetectionEmbedding ||
		model.Package.PackageID != r.identity.Model || model.Package.ContentHash != r.fingerprint {
		return aimodel.DeploymentFaceMetadata{}, aimodel.ErrModelIncompatible
	}
	if err := r.client.WaitVerifyIdentity(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return aimodel.DeploymentFaceMetadata{}, err
		}
		if errors.Is(err, remoteml.ErrIdentityMismatch) {
			return aimodel.DeploymentFaceMetadata{}, aimodel.ErrModelIncompatible
		}
		return aimodel.DeploymentFaceMetadata{}, aimodel.ErrInferenceRuntimeUnavailable
	}
	return r.identity.Metadata()
}

func (r *faceDeploymentActivationRuntime) CommitFaceDeploymentActivation(ctx context.Context, commit aimodel.FaceActivationCommit) (aimodel.Operation, error) {
	if !r.identity.MatchesGeneration(commit.Generation) {
		return aimodel.Operation{}, aimodel.ErrModelIncompatible
	}
	return r.repository.CommitFaceDeploymentActivation(ctx, face.DeploymentActivationCommit{Activation: commit, Identity: r.identity})
}

var _ aimodel.DeploymentFaceActivationRuntime = (*faceDeploymentActivationRuntime)(nil)

func (service *databaseService) CommitFaceDeploymentActivation(ctx context.Context, commit face.DeploymentActivationCommit) (aimodel.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Operation{}, errors.New("face deployment repository is not ready")
	}
	return service.store.CommitFaceDeploymentActivation(ctx, commit)
}
