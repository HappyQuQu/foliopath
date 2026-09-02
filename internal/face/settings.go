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
	GetFaceLibrarySettings(context.Context, int64) (LibrarySettings, error)
	UpdateFaceLibrarySettings(context.Context, int64, bool, int64, time.Time) (LibrarySettings, error)
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
