package face

import (
	"context"
	"errors"
	"time"
)

var (
	ErrFaceLibraryNotFound  = errors.New("face library not found")
	ErrFaceLibraryOffline   = errors.New("face library offline")
	ErrFaceSettingsConflict = errors.New("face settings revision conflict")
	ErrFaceModelUnavailable = errors.New("face model unavailable")
)

type LibrarySettings struct {
	LibraryID          int64
	Enabled            bool
	State              string
	Revision           int64
	ActiveGenerationID string
	Coverage           FaceCoverage
}
type LibrarySettingsRepository interface {
	GetActiveFaceGeneration(context.Context) (string, error)
	GetFaceLibrarySettings(context.Context, int64) (LibrarySettings, error)
	UpdateFaceLibrarySettings(context.Context, int64, bool, int64, time.Time) (LibrarySettings, error)
}

// JobSettings selects the deployed active generation for new work. The
// repository's revision guard protects a concurrent administrator update.
func (s *SettingsService) JobSettings(ctx context.Context, libraryID int64) (LibrarySettings, error) {
	settings, err := s.Get(ctx, libraryID)
	if err != nil || !settings.Enabled {
		return settings, err
	}
	generation, err := s.repository.GetActiveFaceGeneration(ctx)
	if err != nil {
		return LibrarySettings{}, err
	}
	// A failed inference can leave the library awaiting the same generation.
	// Re-enter through the revision-guarded settings owner, which verifies model
	// availability transactionally; never bypass admission's fail-closed guard.
	if generation != settings.ActiveGenerationID || settings.State == "awaiting_model" {
		return s.Update(ctx, libraryID, true, settings.Revision)
	}
	return settings, nil
}

type SettingsService struct {
	repository LibrarySettingsRepository
	now        func() time.Time
}

func NewSettingsService(repository LibrarySettingsRepository, now func() time.Time) (*SettingsService, error) {
	if repository == nil {
		return nil, errors.New("face settings repository is required")
	}
	if now == nil {
		now = time.Now
	}
	return &SettingsService{repository: repository, now: now}, nil
}
func (s *SettingsService) Get(ctx context.Context, libraryID int64) (LibrarySettings, error) {
	if libraryID < 1 {
		return LibrarySettings{}, ErrFaceLibraryNotFound
	}
	return s.repository.GetFaceLibrarySettings(ctx, libraryID)
}
func (s *SettingsService) Update(ctx context.Context, libraryID int64, enabled bool, expectedRevision int64) (LibrarySettings, error) {
	if libraryID < 1 {
		return LibrarySettings{}, ErrFaceLibraryNotFound
	}
	if expectedRevision < 1 {
		return LibrarySettings{}, ErrFaceSettingsConflict
	}
	return s.repository.UpdateFaceLibrarySettings(ctx, libraryID, enabled, expectedRevision, s.now().UTC())
}
