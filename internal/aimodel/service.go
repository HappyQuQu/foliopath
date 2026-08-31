package aimodel

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

type Repository interface {
	ListAIModels(context.Context) (Snapshot, error)
	GetAIModel(context.Context, string) (Model, error)
	RegisterAIModel(context.Context, Model) (Model, bool, error)
	SetAIModelAvailability(context.Context, string, int64, State, time.Time) (Model, error)
}

func (service *Service) Get(ctx context.Context, modelID string) (Model, error) {
	if modelID == "" {
		return Model{}, ErrModelNotFound
	}
	model, err := service.repository.GetAIModel(ctx, modelID)
	if err != nil {
		return Model{}, err
	}
	if err := ValidateModel(model); err != nil {
		return Model{}, err
	}
	return model, nil
}

type IDGenerator func() (string, error)

type Service struct {
	repository Repository
	now        func() time.Time
	newID      IDGenerator
}

func NewService(repository Repository, now func() time.Time, newID IDGenerator) (*Service, error) {
	if repository == nil {
		return nil, errors.New("AI model repository is required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomModelID
	}
	return &Service{repository: repository, now: now, newID: newID}, nil
}

func (service *Service) List(ctx context.Context) (Snapshot, error) {
	snapshot, err := service.repository.ListAIModels(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	if snapshot.Revision < 1 {
		return Snapshot{}, ErrRepositoryState
	}
	activeFound := snapshot.ActiveModelID == ""
	for index := range snapshot.Items {
		if err := ValidateModel(snapshot.Items[index]); err != nil {
			return Snapshot{}, err
		}
		if snapshot.Items[index].Active != (snapshot.Items[index].ID == snapshot.ActiveModelID) {
			return Snapshot{}, ErrRepositoryState
		}
		activeFound = activeFound || snapshot.Items[index].Active
	}
	if !activeFound {
		return Snapshot{}, ErrRepositoryState
	}
	return snapshot, nil
}

func (service *Service) RegisterInstalled(
	ctx context.Context,
	verified VerifiedPackage,
	storageMode StorageMode,
	sourceIdentity string,
) (Model, bool, error) {
	if ValidatePackage(verified) != nil ||
		(storageMode != StorageManaged && storageMode != StorageDirect) ||
		sourceIdentity == "" || len(sourceIdentity) > 256 {
		return Model{}, false, ErrInvalidModel
	}
	id, err := service.newID()
	if err != nil {
		return Model{}, false, fmt.Errorf("generate AI model ID: %w", err)
	}
	now := service.now().UTC()
	model := Model{
		ID:                   id,
		Package:              verified,
		StorageMode:          storageMode,
		State:                StateAvailable,
		SourceIdentity:       sourceIdentity,
		AvailabilityRevision: 1,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	registered, created, err := service.repository.RegisterAIModel(ctx, model)
	if err != nil {
		return Model{}, false, err
	}
	if err := ValidateModel(registered); err != nil {
		return Model{}, false, err
	}
	return registered, created, nil
}

func (service *Service) SetAvailability(
	ctx context.Context,
	modelID string,
	expectedRevision int64,
	available bool,
) (Model, error) {
	if modelID == "" || expectedRevision < 1 {
		return Model{}, ErrInvalidModel
	}
	state := StateUnavailable
	if available {
		state = StateAvailable
	}
	model, err := service.repository.SetAIModelAvailability(
		ctx, modelID, expectedRevision, state, service.now().UTC(),
	)
	if err != nil {
		return Model{}, err
	}
	if err := ValidateModel(model); err != nil {
		return Model{}, err
	}
	return model, nil
}

func randomModelID() (string, error) {
	var value [18]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "aim_" + base64.RawURLEncoding.EncodeToString(value[:]), nil
}
