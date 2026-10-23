package aimodel

import (
	"context"
	"errors"
	"io"
	"time"
)

type RuntimeMetadata struct {
	EmbeddingDimension int64
}

type FaceRuntimeMetadata struct {
	EmbeddingDimension int64
	ThresholdProfile   string
}

type RuntimeModelFile interface {
	io.Closer
	RuntimePath() string
	Size() int64
}

type RuntimeFileOpener func(context.Context, string) (RuntimeModelFile, error)

type ActivationPackageSource interface {
	ValidateActivationSource(context.Context, Model, Manifest) error
	OpenActivationModelFile(context.Context, Model, string) (RuntimeModelFile, error)
}

type InferenceRuntime interface {
	LoadAndValidate(context.Context, Model, Manifest, RuntimeFileOpener) (RuntimeMetadata, error)
}

type FaceInferenceRuntime interface {
	LoadAndValidateFace(context.Context, Model, Manifest, RuntimeFileOpener) (FaceRuntimeMetadata, error)
}

type AvailabilityMarker interface {
	MarkUnavailable(context.Context, string, int64) error
}

// DeploymentActivationRuntime owns service identity verification and routes the
// commit through the space-bound repository. It must never open local model files.
type DeploymentActivationRuntime interface {
	LoadAndValidateDeployment(context.Context, Model) (RuntimeMetadata, error)
	CommitDeploymentActivation(context.Context, ActivationCommit) (Operation, error)
}

type DeploymentFaceMetadata struct {
	FaceRuntimeMetadata
	DetectorContentHash, EmbedderContentHash, ThresholdProfileHash string
}

// DeploymentFaceActivationRuntime verifies the service's complete face identity
// and routes commit through a matching space-bound repository, never local files.
type DeploymentFaceActivationRuntime interface {
	LoadAndValidateFaceDeployment(context.Context, Model) (DeploymentFaceMetadata, error)
	CommitFaceDeploymentActivation(context.Context, FaceActivationCommit) (Operation, error)
}

type ActivationWorker struct {
	queue          ActivationRepository
	models         *Service
	operations     *OperationService
	catalog        *Catalog
	source         ActivationPackageSource
	runtime        InferenceRuntime
	faceRuntime    FaceInferenceRuntime
	deployment     DeploymentActivationRuntime
	faceDeployment DeploymentFaceActivationRuntime
	availability   AvailabilityMarker
	notifications  <-chan struct{}
	pollInterval   time.Duration
	now            func() time.Time
	newID          IDGenerator
}

func NewActivationWorker(
	queue ActivationRepository,
	models *Service,
	operations *OperationService,
	catalog *Catalog,
	source ActivationPackageSource,
	runtime InferenceRuntime,
	availability AvailabilityMarker,
	notifications <-chan struct{},
	pollInterval time.Duration,
	now func() time.Time,
	newID IDGenerator,
) (*ActivationWorker, error) {
	return NewActivationWorkerWithFaceRuntime(queue, models, operations, catalog, source, runtime, nil,
		availability, notifications, pollInterval, now, newID)
}

func NewActivationWorkerWithFaceRuntime(
	queue ActivationRepository,
	models *Service,
	operations *OperationService,
	catalog *Catalog,
	source ActivationPackageSource,
	runtime InferenceRuntime,
	faceRuntime FaceInferenceRuntime,
	availability AvailabilityMarker,
	notifications <-chan struct{},
	pollInterval time.Duration,
	now func() time.Time,
	newID IDGenerator,
) (*ActivationWorker, error) {
	if queue == nil || models == nil || operations == nil || catalog == nil || source == nil || runtime == nil || availability == nil || notifications == nil {
		return nil, errors.New("AI activation worker dependencies are required")
	}
	if pollInterval == 0 {
		pollInterval = time.Second
	}
	if pollInterval < 10*time.Millisecond || pollInterval > time.Minute {
		return nil, errors.New("AI activation worker poll interval is invalid")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomGenerationID
	}
	return &ActivationWorker{queue: queue, models: models, operations: operations, catalog: catalog,
		source: source, runtime: runtime, faceRuntime: faceRuntime, availability: availability,
		notifications: notifications, pollInterval: pollInterval, now: now, newID: newID}, nil
}

func (worker *ActivationWorker) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-worker.notifications:
		case <-timer.C:
		}
		for {
			work, found, err := worker.queue.ClaimAIModelActivation(ctx, worker.now().UTC())
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			if !found {
				timer.Reset(worker.pollInterval)
				break
			}
			if err := validateClaimedActivationWork(work); err != nil {
				return err
			}
			worker.process(ctx, work)
		}
	}
}

// EnableDeploymentRuntime is composition-time only, before Run starts. Runtime
// replacement is deliberately forbidden; reconfiguration requires a new worker.
func (worker *ActivationWorker) EnableDeploymentRuntime(runtime DeploymentActivationRuntime) error {
	if runtime == nil || worker.deployment != nil {
		return errors.New("invalid deployment activation runtime")
	}
	worker.deployment = runtime
	return nil
}

func (worker *ActivationWorker) EnableFaceDeploymentRuntime(runtime DeploymentFaceActivationRuntime) error {
	if runtime == nil || worker.faceDeployment != nil {
		return errors.New("invalid face deployment activation runtime")
	}
	worker.faceDeployment = runtime
	return nil
}

func validateClaimedActivationWork(work ActivationWork) error {
	if work.ModelID == "" || work.ExpectedAvailabilityRevision < 1 ||
		len(work.IdempotencyKey) < 8 || len(work.IdempotencyKey) > 128 ||
		work.RequestHash != ActivationRequestHash(work.ModelID, work.ExpectedAvailabilityRevision) ||
		validateActivationWork(work, work.ModelID, work.ExpectedAvailabilityRevision, work.IdempotencyKey) != nil ||
		work.Operation.State != OperationRunning || work.Operation.Phase != PhaseLoading ||
		work.Operation.Revision != 2 || work.Operation.CompletedItems != 0 ||
		work.Operation.TotalItems != nil || work.Operation.LibraryID != 0 {
		return ErrRepositoryState
	}
	return nil
}

func (worker *ActivationWorker) process(parent context.Context, work ActivationWork) {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go watchOperationCancellation(ctx, cancel, worker.operations, work.Operation.ID, worker.pollInterval, done)
	loaded, loadErr := worker.load(ctx, work)
	cancel()
	<-done
	current, err := worker.operations.Get(context.WithoutCancel(parent), work.Operation.ID)
	if err != nil {
		return
	}
	if current.State == OperationCancelling {
		settleWorkerFailure(context.WithoutCancel(parent), worker.operations, current.ID, "internal_error")
		return
	}
	if parent.Err() != nil {
		return
	}
	if loadErr != nil {
		settleWorkerFailure(parent, worker.operations, current.ID, activationErrorCode(loadErr))
		return
	}
	id, err := worker.newID()
	if err != nil {
		settleWorkerFailure(parent, worker.operations, current.ID, "internal_error")
		return
	}
	now := worker.now().UTC()
	commitErr := worker.commit(parent, work, current, id, now, loaded)
	if commitErr == nil {
		return
	}
	settleWorkerFailure(context.WithoutCancel(parent), worker.operations, current.ID, activationErrorCode(commitErr))
}

type activationLoadResult struct {
	model       Model
	manifest    Manifest
	semantic    *RuntimeMetadata
	face        *FaceRuntimeMetadata
	faceService *DeploymentFaceMetadata
}

func (worker *ActivationWorker) load(ctx context.Context, work ActivationWork) (activationLoadResult, error) {
	model, err := worker.models.Get(ctx, work.ModelID)
	if err != nil {
		return activationLoadResult{}, err
	}
	if model.State != StateAvailable || model.AvailabilityRevision != work.ExpectedAvailabilityRevision {
		return activationLoadResult{}, ErrModelUnavailable
	}
	if model.StorageMode == StorageService {
		if model.Package.Purpose == PurposeFaceDetectionEmbedding {
			if worker.faceDeployment == nil {
				return activationLoadResult{}, ErrInferenceRuntimeUnavailable
			}
			metadata, err := worker.faceDeployment.LoadAndValidateFaceDeployment(ctx, model)
			if err != nil {
				return activationLoadResult{}, err
			}
			return activationLoadResult{model: model, face: &metadata.FaceRuntimeMetadata, faceService: &metadata}, nil
		}
		if worker.deployment == nil {
			return activationLoadResult{}, ErrInferenceRuntimeUnavailable
		}
		metadata, err := worker.deployment.LoadAndValidateDeployment(ctx, model)
		if err != nil {
			return activationLoadResult{}, err
		}
		return activationLoadResult{model: model, semantic: &metadata}, nil
	}
	manifest, exists := worker.catalog.Manifest(model.Package.PackageID)
	if !exists {
		return activationLoadResult{}, ErrModelIncompatible
	}
	if err := worker.source.ValidateActivationSource(ctx, model, manifest); err != nil {
		return activationLoadResult{}, errors.Join(err, worker.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	opener := func(openCtx context.Context, name string) (RuntimeModelFile, error) {
		return worker.source.OpenActivationModelFile(openCtx, model, name)
	}
	result := activationLoadResult{model: model, manifest: manifest}
	switch model.Package.Purpose {
	case PurposeSemanticImageText:
		if manifest.FormatVersion != SemanticFormatVersion {
			return activationLoadResult{}, ErrModelIncompatible
		}
		metadata, runtimeErr := worker.runtime.LoadAndValidate(ctx, model, manifest, opener)
		result.semantic, err = &metadata, runtimeErr
	case PurposeFaceDetectionEmbedding:
		if worker.faceRuntime == nil {
			return activationLoadResult{}, ErrInferenceRuntimeUnavailable
		}
		metadata, runtimeErr := worker.faceRuntime.LoadAndValidateFace(ctx, model, manifest, opener)
		result.face, err = &metadata, runtimeErr
	default:
		return activationLoadResult{}, ErrModelIncompatible
	}
	if err != nil && (errors.Is(err, ErrModelSourceUnavailable) || errors.Is(err, ErrModelIncompatible)) {
		err = errors.Join(err, worker.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	return result, err
}

func (worker *ActivationWorker) commit(
	ctx context.Context,
	work ActivationWork,
	operation Operation,
	id string,
	now time.Time,
	loaded activationLoadResult,
) error {
	switch loaded.model.Package.Purpose {
	case PurposeSemanticImageText:
		if loaded.semantic == nil {
			return ErrModelIncompatible
		}
		generation := Generation{ID: id, ModelID: work.ModelID, TransformVersion: SemanticTransformVersion,
			OutputSchemaVersion: SemanticOutputSchemaVersion, IndexFormatVersion: SemanticIndexFormatVersion,
			EmbeddingDimension: loaded.semantic.EmbeddingDimension, State: GenerationActive,
			CreatedAt: now, ActivatedAt: &now, UpdatedAt: now}
		if ValidateGeneration(generation) != nil {
			return ErrModelIncompatible
		}
		commit := ActivationCommit{
			OperationID: operation.ID, ExpectedRevision: operation.Revision,
			ExpectedAvailabilityRevision: work.ExpectedAvailabilityRevision,
			Generation:                   generation, UpdatedAt: now,
		}
		if loaded.model.StorageMode == StorageService {
			if worker.deployment == nil {
				return ErrInferenceRuntimeUnavailable
			}
			_, err := worker.deployment.CommitDeploymentActivation(ctx, commit)
			return err
		}
		_, err := worker.queue.CommitAIModelActivation(ctx, commit)
		return err
	case PurposeFaceDetectionEmbedding:
		if loaded.face == nil {
			return ErrModelIncompatible
		}
		if loaded.model.StorageMode == StorageService {
			if worker.faceDeployment == nil || loaded.faceService == nil {
				return ErrInferenceRuntimeUnavailable
			}
			metadata := loaded.faceService
			generation := FaceGeneration{ID: id, ModelID: work.ModelID, PackageID: loaded.model.Package.PackageID,
				DetectorContentHash: metadata.DetectorContentHash, EmbedderContentHash: metadata.EmbedderContentHash,
				EmbeddingDimension: metadata.EmbeddingDimension, TransformVersion: FaceTransformVersion,
				ThresholdProfile: metadata.ThresholdProfile, ThresholdProfileHash: metadata.ThresholdProfileHash,
				State: GenerationActive, CreatedAt: now, ActivatedAt: &now, UpdatedAt: now}
			if ValidateFaceGeneration(generation) != nil {
				return ErrModelIncompatible
			}
			_, err := worker.faceDeployment.CommitFaceDeploymentActivation(ctx, FaceActivationCommit{
				OperationID: operation.ID, ExpectedRevision: operation.Revision, ExpectedAvailabilityRevision: work.ExpectedAvailabilityRevision,
				Generation: generation, UpdatedAt: now})
			return err
		}
		repository, ok := worker.queue.(FaceActivationRepository)
		if !ok {
			return ErrInferenceRuntimeUnavailable
		}
		detector, embedder, threshold, ok := faceManifestComponents(loaded.manifest)
		if !ok {
			return ErrModelIncompatible
		}
		generation := FaceGeneration{
			ID: id, ModelID: work.ModelID, PackageID: loaded.manifest.PackageID,
			DetectorContentHash: detector.SHA256, EmbedderContentHash: embedder.SHA256,
			EmbeddingDimension: loaded.face.EmbeddingDimension, TransformVersion: FaceTransformVersion,
			ThresholdProfile: loaded.face.ThresholdProfile, ThresholdProfileHash: threshold.SHA256,
			State: GenerationActive, CreatedAt: now, ActivatedAt: &now, UpdatedAt: now,
		}
		if ValidateFaceGeneration(generation) != nil {
			return ErrModelIncompatible
		}
		_, err := repository.CommitFaceModelActivation(ctx, FaceActivationCommit{
			OperationID: operation.ID, ExpectedRevision: operation.Revision,
			ExpectedAvailabilityRevision: work.ExpectedAvailabilityRevision,
			Generation:                   generation, UpdatedAt: now,
		})
		return err
	default:
		return ErrModelIncompatible
	}
}

func faceManifestComponents(manifest Manifest) (ManifestFile, ManifestFile, ManifestFile, bool) {
	var detector, embedder, threshold ManifestFile
	for _, file := range manifest.Files {
		switch file.Role {
		case "face_detector":
			detector = file
		case "face_embedder":
			embedder = file
		case "face_threshold_profile":
			threshold = file
		}
	}
	return detector, embedder, threshold,
		detector.Name != "" && embedder.Name != "" && threshold.Name != ""
}

func (worker *ActivationWorker) markUnavailableAfterValidationFailure(ctx context.Context, model Model, validationErr error) error {
	if errors.Is(validationErr, context.Canceled) || errors.Is(validationErr, context.DeadlineExceeded) || ctx.Err() != nil {
		return nil
	}
	return worker.availability.MarkUnavailable(ctx, model.ID, model.AvailabilityRevision)
}

func watchOperationCancellation(ctx context.Context, cancel context.CancelFunc, operations *OperationService, id string, interval time.Duration, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			op, err := operations.Get(ctx, id)
			if err == nil && op.State == OperationCancelling {
				cancel()
				return
			}
		}
	}
}

func activationErrorCode(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, ErrModelUnavailable), errors.Is(err, ErrModelNotFound), errors.Is(err, ErrPreconditionFailed),
		errors.Is(err, ErrInferenceRuntimeUnavailable):
		return "model_unavailable"
	case errors.Is(err, ErrModelIncompatible), errors.Is(err, ErrInvalidModel):
		return "model_incompatible"
	case errors.Is(err, ErrModelSourceUnavailable):
		return "model_source_unavailable"
	default:
		return "internal_error"
	}
}

func randomGenerationID() (string, error) {
	id, err := randomModelID()
	if err != nil {
		return "", err
	}
	return "aig_" + id[4:], nil
}
