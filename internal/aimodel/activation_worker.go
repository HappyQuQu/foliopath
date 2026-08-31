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

type AvailabilityMarker interface {
	MarkUnavailable(context.Context, string, int64) error
}

type ActivationWorker struct {
	queue         ActivationRepository
	models        *Service
	operations    *OperationService
	catalog       *Catalog
	source        ActivationPackageSource
	runtime       InferenceRuntime
	availability  AvailabilityMarker
	notifications <-chan struct{}
	pollInterval  time.Duration
	now           func() time.Time
	newID         IDGenerator
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
		source: source, runtime: runtime, availability: availability, notifications: notifications, pollInterval: pollInterval, now: now, newID: newID}, nil
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
	metadata, loadErr := worker.load(ctx, work)
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
	generation := Generation{ID: id, ModelID: work.ModelID, TransformVersion: SemanticTransformVersion,
		OutputSchemaVersion: SemanticOutputSchemaVersion, IndexFormatVersion: SemanticIndexFormatVersion,
		EmbeddingDimension: metadata.EmbeddingDimension, State: GenerationActive,
		CreatedAt: now, ActivatedAt: &now, UpdatedAt: now}
	if ValidateGeneration(generation) != nil {
		settleWorkerFailure(parent, worker.operations, current.ID, "model_incompatible")
		return
	}
	_, commitErr := worker.queue.CommitAIModelActivation(parent, ActivationCommit{
		OperationID: current.ID, ExpectedRevision: current.Revision,
		ExpectedAvailabilityRevision: work.ExpectedAvailabilityRevision,
		Generation:                   generation, UpdatedAt: now,
	})
	if commitErr == nil {
		return
	}
	settleWorkerFailure(context.WithoutCancel(parent), worker.operations, current.ID, activationErrorCode(commitErr))
}

func (worker *ActivationWorker) load(ctx context.Context, work ActivationWork) (RuntimeMetadata, error) {
	model, err := worker.models.Get(ctx, work.ModelID)
	if err != nil {
		return RuntimeMetadata{}, err
	}
	if model.State != StateAvailable || model.AvailabilityRevision != work.ExpectedAvailabilityRevision {
		return RuntimeMetadata{}, ErrModelUnavailable
	}
	manifest, exists := worker.catalog.Manifest(model.Package.PackageID)
	if !exists {
		return RuntimeMetadata{}, ErrModelIncompatible
	}
	if err := worker.source.ValidateActivationSource(ctx, model, manifest); err != nil {
		return RuntimeMetadata{}, errors.Join(err, worker.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	metadata, err := worker.runtime.LoadAndValidate(ctx, model, manifest, func(openCtx context.Context, name string) (RuntimeModelFile, error) {
		return worker.source.OpenActivationModelFile(openCtx, model, name)
	})
	if err != nil && (errors.Is(err, ErrModelSourceUnavailable) || errors.Is(err, ErrModelIncompatible)) {
		err = errors.Join(err, worker.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	return metadata, err
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
	case errors.Is(err, ErrModelUnavailable), errors.Is(err, ErrModelNotFound), errors.Is(err, ErrPreconditionFailed):
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
