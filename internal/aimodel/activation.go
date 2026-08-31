package aimodel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

const (
	SemanticTransformVersion    = 1
	SemanticOutputSchemaVersion = 1
	SemanticIndexFormatVersion  = 1
)

type GenerationState string

const (
	GenerationBuilding GenerationState = "building"
	GenerationReady    GenerationState = "ready"
	GenerationActive   GenerationState = "active"
	GenerationRetired  GenerationState = "retired"
	GenerationFailed   GenerationState = "failed"
)

type Generation struct {
	ID                  string
	ModelID             string
	TransformVersion    int64
	OutputSchemaVersion int64
	IndexFormatVersion  int64
	EmbeddingDimension  int64
	State               GenerationState
	CreatedAt           time.Time
	ActivatedAt         *time.Time
	UpdatedAt           time.Time
}

type ActivationWork struct {
	IdempotencyKey               string
	RequestHash                  string
	ModelID                      string
	ExpectedAvailabilityRevision int64
	Operation                    Operation
}

type ActivationCommit struct {
	OperationID                  string
	ExpectedRevision             int64
	ExpectedAvailabilityRevision int64
	Generation                   Generation
	UpdatedAt                    time.Time
}

type ActivationRepository interface {
	FindAIModelActivation(context.Context, string) (ActivationWork, bool, error)
	CreateAIModelActivation(context.Context, ActivationWork) (ActivationWork, bool, error)
	ClaimAIModelActivation(context.Context, time.Time) (ActivationWork, bool, error)
	CommitAIModelActivation(context.Context, ActivationCommit) (Operation, error)
}

type ActivationResult struct {
	Operation Operation
	Created   bool
	Replayed  bool
}

type ActivationAdmission interface {
	ReplayActivation(context.Context, string, int64, string) (ActivationResult, bool, error)
	StartActivation(context.Context, Model, string) (ActivationResult, error)
}

type ActivationAdmissionService struct {
	queue ActivationRepository
	now   func() time.Time
	newID IDGenerator
	wake  interface{ Wake() }
}

func NewActivationAdmissionService(
	queue ActivationRepository,
	wake interface{ Wake() },
	now func() time.Time,
	newID IDGenerator,
) (*ActivationAdmissionService, error) {
	if queue == nil || wake == nil {
		return nil, errors.New("AI model activation admission dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomOperationID
	}
	return &ActivationAdmissionService{queue: queue, wake: wake, now: now, newID: newID}, nil
}

func (service *ActivationAdmissionService) ReplayActivation(
	ctx context.Context,
	modelID string,
	availabilityRevision int64,
	idempotencyKey string,
) (ActivationResult, bool, error) {
	existing, found, err := service.queue.FindAIModelActivation(ctx, idempotencyKey)
	if err != nil || !found {
		return ActivationResult{}, false, err
	}
	if existing.RequestHash != ActivationRequestHash(modelID, availabilityRevision) {
		return ActivationResult{}, false, ErrIdempotencyConflict
	}
	if err := validateActivationWork(existing, modelID, availabilityRevision, idempotencyKey); err != nil {
		return ActivationResult{}, false, err
	}
	return ActivationResult{Operation: existing.Operation, Replayed: true}, true, nil
}

func (service *ActivationAdmissionService) StartActivation(
	ctx context.Context,
	model Model,
	idempotencyKey string,
) (ActivationResult, error) {
	if ValidateModel(model) != nil || len(idempotencyKey) < 8 || len(idempotencyKey) > 128 {
		return ActivationResult{}, ErrInvalidModel
	}
	if model.State != StateAvailable {
		return ActivationResult{}, ErrModelUnavailable
	}
	requestHash := ActivationRequestHash(model.ID, model.AvailabilityRevision)
	if replay, found, err := service.ReplayActivation(ctx, model.ID, model.AvailabilityRevision, idempotencyKey); err != nil || found {
		return replay, err
	}
	id, err := service.newID()
	if err != nil {
		return ActivationResult{}, err
	}
	now := service.now().UTC()
	requested := ActivationWork{
		IdempotencyKey: idempotencyKey, RequestHash: requestHash, ModelID: model.ID,
		ExpectedAvailabilityRevision: model.AvailabilityRevision,
		Operation: Operation{
			ID: id, Kind: OperationModelActivate, State: OperationQueued, Phase: PhaseQueued,
			ModelID: model.ID, Revision: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
	work, created, err := service.queue.CreateAIModelActivation(ctx, requested)
	if err != nil {
		return ActivationResult{}, err
	}
	if work.RequestHash != requestHash {
		return ActivationResult{}, ErrIdempotencyConflict
	}
	if err := validateActivationWork(work, model.ID, model.AvailabilityRevision, idempotencyKey); err != nil {
		return ActivationResult{}, err
	}
	if created && !sameOperationCreation(work.Operation, requested.Operation) {
		return ActivationResult{}, ErrRepositoryState
	}
	if created {
		service.wake.Wake()
	}
	return ActivationResult{Operation: work.Operation, Created: created, Replayed: !created}, nil
}

func validateActivationWork(work ActivationWork, modelID string, availabilityRevision int64, idempotencyKey string) error {
	if work.IdempotencyKey != idempotencyKey || work.ModelID != modelID ||
		work.ExpectedAvailabilityRevision != availabilityRevision || validateOperation(work.Operation) != nil ||
		work.Operation.Kind != OperationModelActivate || work.Operation.ModelID != modelID {
		return ErrRepositoryState
	}
	return nil
}

func ActivationRequestHash(modelID string, availabilityRevision int64) string {
	digest := sha256.Sum256([]byte("foliopath:model-activation:v1\x00" + modelID + "\x00" + decimalInt64(availabilityRevision)))
	return hex.EncodeToString(digest[:])
}

func ValidateGeneration(generation Generation) error {
	if generation.ID == "" || generation.ModelID == "" ||
		generation.TransformVersion != SemanticTransformVersion ||
		generation.OutputSchemaVersion != SemanticOutputSchemaVersion ||
		generation.IndexFormatVersion != SemanticIndexFormatVersion ||
		generation.EmbeddingDimension < 1 || generation.EmbeddingDimension > 65536 ||
		generation.State != GenerationActive || generation.CreatedAt.IsZero() ||
		generation.ActivatedAt == nil || generation.UpdatedAt.Before(generation.CreatedAt) {
		return ErrInvalidModel
	}
	return nil
}

var ErrModelUnavailable = errors.New("AI model unavailable")

func decimalInt64(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}
