package aimodel

import (
	"context"
	"errors"
	"time"
)

var (
	ErrOperationNotFound        = errors.New("AI operation not found")
	ErrOperationAlreadyFinished = errors.New("AI operation already finished")
	ErrInvalidTransition        = errors.New("invalid AI operation transition")
)

type OperationKind string

const (
	OperationModelInstall         OperationKind = "model_install"
	OperationModelActivate        OperationKind = "model_activate"
	OperationSemanticMissing      OperationKind = "semantic_missing"
	OperationSemanticRebuild      OperationKind = "semantic_rebuild"
	OperationSemanticClear        OperationKind = "semantic_clear"
	OperationTagSuggestionMissing OperationKind = "tag_suggestion_missing"
	OperationTagSuggestionRebuild OperationKind = "tag_suggestion_rebuild"
	OperationTagReviewClear       OperationKind = "tag_review_clear"
	OperationVideoSemanticMissing OperationKind = "video_semantic_missing"
	OperationVideoSemanticRebuild OperationKind = "video_semantic_rebuild"
)

type OperationState string

const (
	OperationQueued     OperationState = "queued"
	OperationRunning    OperationState = "running"
	OperationCancelling OperationState = "cancelling"
	OperationSucceeded  OperationState = "succeeded"
	OperationFailed     OperationState = "failed"
	OperationCancelled  OperationState = "cancelled"
)

type OperationPhase string

const (
	PhaseQueued     OperationPhase = "queued"
	PhaseScanning   OperationPhase = "scanning"
	PhaseVerifying  OperationPhase = "verifying"
	PhaseCopying    OperationPhase = "copying"
	PhaseLoading    OperationPhase = "loading"
	PhaseBuilding   OperationPhase = "building"
	PhaseValidating OperationPhase = "validating"
	PhaseClearing   OperationPhase = "clearing"
	PhaseFinalizing OperationPhase = "finalizing"
	PhaseCompleted  OperationPhase = "completed"
)

type Operation struct {
	ID             string
	Kind           OperationKind
	State          OperationState
	Phase          OperationPhase
	ModelID        string
	LibraryID      int64
	CompletedItems int64
	TotalItems     *int64
	ErrorCode      string
	Revision       int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	FinishedAt     *time.Time
}

type OperationTransition struct {
	ExpectedRevision int64
	State            OperationState
	Phase            OperationPhase
	CompletedItems   int64
	TotalItems       *int64
	ErrorCode        string
	FinishedAt       *time.Time
	UpdatedAt        time.Time
}

type OperationRepository interface {
	CreateAIOperation(context.Context, Operation) (Operation, error)
	GetAIOperation(context.Context, string) (Operation, error)
	TransitionAIOperation(context.Context, string, OperationTransition) (Operation, error)
	RecoverInterruptedAIOperations(context.Context, time.Time) (int64, error)
}

type OperationService struct {
	repository OperationRepository
	now        func() time.Time
	newID      IDGenerator
}

func NewOperationService(repository OperationRepository, now func() time.Time, newID IDGenerator) (*OperationService, error) {
	if repository == nil {
		return nil, errors.New("AI operation repository is required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomOperationID
	}
	return &OperationService{repository: repository, now: now, newID: newID}, nil
}

func (service *OperationService) Create(
	ctx context.Context,
	kind OperationKind,
	modelID string,
	libraryID int64,
	totalItems *int64,
) (Operation, error) {
	if !validOperationKind(kind) || libraryID < 0 || (totalItems != nil && *totalItems < 0) {
		return Operation{}, ErrInvalidTransition
	}
	id, err := service.newID()
	if err != nil {
		return Operation{}, err
	}
	now := service.now().UTC()
	requested := Operation{
		ID: id, Kind: kind, State: OperationQueued, Phase: PhaseQueued,
		ModelID: modelID, LibraryID: libraryID, TotalItems: cloneInt64(totalItems),
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	operation, err := service.repository.CreateAIOperation(ctx, requested)
	if err != nil {
		return Operation{}, err
	}
	if err := validateOperation(operation); err != nil || !sameOperationCreation(operation, requested) {
		if err != nil {
			return Operation{}, err
		}
		return Operation{}, ErrRepositoryState
	}
	return operation, nil
}

func sameOperationCreation(value, requested Operation) bool {
	return value.ID == requested.ID && value.Kind == requested.Kind && value.ModelID == requested.ModelID &&
		value.LibraryID == requested.LibraryID && value.State == requested.State && value.Phase == requested.Phase &&
		value.CompletedItems == requested.CompletedItems && sameOptionalInt64(value.TotalItems, requested.TotalItems) &&
		value.ErrorCode == requested.ErrorCode && value.Revision == requested.Revision
}

func sameOptionalInt64(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func validateRequestedOperation(value Operation, operationID string) error {
	if err := validateOperation(value); err != nil {
		return err
	}
	if value.ID != operationID {
		return ErrRepositoryState
	}
	return nil
}

func (service *OperationService) Get(ctx context.Context, operationID string) (Operation, error) {
	if operationID == "" {
		return Operation{}, ErrOperationNotFound
	}
	operation, err := service.repository.GetAIOperation(ctx, operationID)
	if err != nil {
		return Operation{}, err
	}
	if err := validateRequestedOperation(operation, operationID); err != nil {
		return Operation{}, err
	}
	return operation, nil
}

func (service *OperationService) Start(ctx context.Context, id string, revision int64, phase OperationPhase) (Operation, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if current.Revision != revision {
		return Operation{}, ErrPreconditionFailed
	}
	if current.State != OperationQueued || phase == PhaseQueued || phase == PhaseCompleted || !validOperationPhase(phase) {
		return Operation{}, ErrInvalidTransition
	}
	return service.transition(ctx, current, OperationRunning, phase, current.CompletedItems, current.TotalItems, "", nil)
}

func (service *OperationService) Progress(
	ctx context.Context,
	id string,
	revision int64,
	phase OperationPhase,
	completed int64,
	total *int64,
) (Operation, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if current.Revision != revision {
		return Operation{}, ErrPreconditionFailed
	}
	if current.State != OperationRunning || !validActivePhase(phase) || completed < current.CompletedItems ||
		(total != nil && (completed > *total || *total < 0)) ||
		(current.TotalItems != nil && (total == nil || *total != *current.TotalItems)) {
		return Operation{}, ErrInvalidTransition
	}
	return service.transition(ctx, current, OperationRunning, phase, completed, total, "", nil)
}

func (service *OperationService) RequestCancel(ctx context.Context, id string, revision int64) (Operation, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if current.Revision != revision {
		return Operation{}, ErrPreconditionFailed
	}
	switch current.State {
	case OperationQueued:
		finished := service.now().UTC()
		return service.transition(ctx, current, OperationCancelled, PhaseCompleted, current.CompletedItems, current.TotalItems, "cancelled", &finished)
	case OperationRunning:
		return service.transition(ctx, current, OperationCancelling, current.Phase, current.CompletedItems, current.TotalItems, "", nil)
	case OperationCancelling:
		return current, nil
	default:
		return Operation{}, ErrOperationAlreadyFinished
	}
}

func (service *OperationService) Succeed(ctx context.Context, id string, revision int64) (Operation, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if current.Revision != revision {
		return Operation{}, ErrPreconditionFailed
	}
	if current.State != OperationRunning {
		return Operation{}, ErrInvalidTransition
	}
	finished := service.now().UTC()
	return service.transition(ctx, current, OperationSucceeded, PhaseCompleted, current.CompletedItems, current.TotalItems, "", &finished)
}

func (service *OperationService) FinishCancelled(ctx context.Context, id string, revision int64) (Operation, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if current.Revision != revision {
		return Operation{}, ErrPreconditionFailed
	}
	if current.State != OperationCancelling && current.State != OperationRunning {
		return Operation{}, ErrInvalidTransition
	}
	finished := service.now().UTC()
	return service.transition(ctx, current, OperationCancelled, PhaseCompleted, current.CompletedItems, current.TotalItems, "cancelled", &finished)
}

func (service *OperationService) Fail(ctx context.Context, id string, revision int64, errorCode string) (Operation, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if current.Revision != revision {
		return Operation{}, ErrPreconditionFailed
	}
	if isTerminalOperationState(current.State) || errorCode == "" || len(errorCode) > 128 {
		return Operation{}, ErrInvalidTransition
	}
	finished := service.now().UTC()
	return service.transition(ctx, current, OperationFailed, PhaseCompleted, current.CompletedItems, current.TotalItems, errorCode, &finished)
}

func (service *OperationService) RecoverInterrupted(ctx context.Context) (int64, error) {
	return service.repository.RecoverInterruptedAIOperations(ctx, service.now().UTC())
}

// settleWorkerFailure gives a concurrent cancellation priority over a worker
// failure. A cancellation is the only expected competing transition after a
// worker has claimed non-restart-safe work, so one retry closes that CAS race.
func settleWorkerFailure(ctx context.Context, service *OperationService, operationID, errorCode string) {
	for attempt := 0; attempt < 2; attempt++ {
		current, err := service.Get(ctx, operationID)
		if err != nil {
			return
		}
		switch current.State {
		case OperationCancelling:
			_, err = service.FinishCancelled(ctx, current.ID, current.Revision)
		case OperationRunning:
			_, err = service.Fail(ctx, current.ID, current.Revision, errorCode)
		default:
			return
		}
		if err == nil || !errors.Is(err, ErrPreconditionFailed) {
			return
		}
	}
}

func (service *OperationService) transition(
	ctx context.Context,
	current Operation,
	state OperationState,
	phase OperationPhase,
	completed int64,
	total *int64,
	errorCode string,
	finished *time.Time,
) (Operation, error) {
	operation, err := service.repository.TransitionAIOperation(ctx, current.ID, OperationTransition{
		ExpectedRevision: current.Revision,
		State:            state, Phase: phase, CompletedItems: completed, TotalItems: cloneInt64(total),
		ErrorCode: errorCode, FinishedAt: finished, UpdatedAt: service.now().UTC(),
	})
	if err != nil {
		return Operation{}, err
	}
	if err := validateOperation(operation); err != nil {
		return Operation{}, err
	}
	if operation.ID != current.ID || operation.Kind != current.Kind || operation.ModelID != current.ModelID ||
		operation.LibraryID != current.LibraryID || operation.State != state || operation.Phase != phase ||
		operation.CompletedItems != completed || !sameOptionalInt64(operation.TotalItems, total) ||
		operation.ErrorCode != errorCode || operation.Revision != current.Revision+1 ||
		!operation.CreatedAt.Equal(current.CreatedAt) || operation.UpdatedAt.Before(current.UpdatedAt) {
		return Operation{}, ErrRepositoryState
	}
	return operation, nil
}

func validateOperation(value Operation) error {
	if value.ID == "" || !validOperationKind(value.Kind) || !validOperationState(value.State) ||
		!validOperationPhase(value.Phase) || value.CompletedItems < 0 || value.Revision < 1 ||
		value.CreatedAt.IsZero() || value.UpdatedAt.Before(value.CreatedAt) || len(value.ErrorCode) > 128 {
		return ErrRepositoryState
	}
	if value.TotalItems != nil && (*value.TotalItems < 0 || value.CompletedItems > *value.TotalItems) {
		return ErrRepositoryState
	}
	terminal := isTerminalOperationState(value.State)
	if terminal != (value.FinishedAt != nil) || (terminal && value.Phase != PhaseCompleted) {
		return ErrRepositoryState
	}
	switch value.State {
	case OperationQueued:
		if value.Phase != PhaseQueued || value.ErrorCode != "" {
			return ErrRepositoryState
		}
	case OperationRunning, OperationCancelling:
		if !validActivePhase(value.Phase) || value.ErrorCode != "" {
			return ErrRepositoryState
		}
	case OperationSucceeded:
		if value.ErrorCode != "" {
			return ErrRepositoryState
		}
	case OperationFailed, OperationCancelled:
		if value.ErrorCode == "" {
			return ErrRepositoryState
		}
	}
	return nil
}

func validOperationKind(value OperationKind) bool {
	switch value {
	case OperationModelInstall, OperationModelActivate, OperationSemanticMissing, OperationSemanticRebuild, OperationSemanticClear,
		OperationTagSuggestionMissing, OperationTagSuggestionRebuild, OperationTagReviewClear,
		OperationVideoSemanticMissing, OperationVideoSemanticRebuild:
		return true
	default:
		return false
	}
}

func validOperationState(value OperationState) bool {
	switch value {
	case OperationQueued, OperationRunning, OperationCancelling, OperationSucceeded, OperationFailed, OperationCancelled:
		return true
	default:
		return false
	}
}

func validOperationPhase(value OperationPhase) bool {
	switch value {
	case PhaseQueued, PhaseScanning, PhaseVerifying, PhaseCopying, PhaseLoading, PhaseBuilding, PhaseValidating, PhaseClearing, PhaseFinalizing, PhaseCompleted:
		return true
	default:
		return false
	}
}

func validActivePhase(value OperationPhase) bool {
	return validOperationPhase(value) && value != PhaseQueued && value != PhaseCompleted
}

func isTerminalOperationState(value OperationState) bool {
	return value == OperationSucceeded || value == OperationFailed || value == OperationCancelled
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func randomOperationID() (string, error) {
	id, err := randomModelID()
	if err != nil {
		return "", err
	}
	return "aio_" + id[4:], nil
}
