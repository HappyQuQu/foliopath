package semantic

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSemanticLibraryNotFound  = errors.New("semantic library not found")
	ErrSemanticRevisionConflict = errors.New("semantic settings revision conflict")
	ErrSemanticDisabled         = errors.New("semantic search disabled")
)

type LibraryState string

const (
	LibraryDisabled      LibraryState = "disabled"
	LibraryAwaitingModel LibraryState = "awaiting_model"
	LibraryBuilding      LibraryState = "building"
	LibraryReady         LibraryState = "ready"
	LibraryDegraded      LibraryState = "degraded"
	LibraryClearing      LibraryState = "clearing"
)

type Coverage struct {
	Eligible  int64
	Completed int64
	Failed    int64
	Stale     int64
	Revision  int64
}

func (coverage Coverage) Complete() bool {
	return coverage.Failed == 0 && coverage.Stale == 0 && coverage.Completed == coverage.Eligible
}

type LibrarySettings struct {
	LibraryID          int64
	Enabled            bool
	State              LibraryState
	Revision           int64
	ActiveGenerationID string
	Coverage           Coverage
}

type LibrarySettingsRepository interface {
	GetSemanticLibrarySettings(context.Context, int64) (LibrarySettings, error)
	UpdateSemanticLibrarySettings(context.Context, int64, bool, int64, time.Time) (LibrarySettings, error)
}

type SettingsService struct {
	repository LibrarySettingsRepository
	now        func() time.Time
}

func NewSettingsService(repository LibrarySettingsRepository, now func() time.Time) (*SettingsService, error) {
	if repository == nil {
		return nil, errors.New("semantic settings repository is required")
	}
	if now == nil {
		now = time.Now
	}
	return &SettingsService{repository: repository, now: now}, nil
}

func (service *SettingsService) Get(ctx context.Context, libraryID int64) (LibrarySettings, error) {
	if libraryID < 1 {
		return LibrarySettings{}, ErrSemanticLibraryNotFound
	}
	return service.repository.GetSemanticLibrarySettings(ctx, libraryID)
}

func (service *SettingsService) Update(ctx context.Context, libraryID int64, enabled bool, expectedRevision int64) (LibrarySettings, error) {
	if libraryID < 1 {
		return LibrarySettings{}, ErrSemanticLibraryNotFound
	}
	if expectedRevision < 1 {
		return LibrarySettings{}, ErrSemanticRevisionConflict
	}
	return service.repository.UpdateSemanticLibrarySettings(ctx, libraryID, enabled, expectedRevision, service.now().UTC())
}
