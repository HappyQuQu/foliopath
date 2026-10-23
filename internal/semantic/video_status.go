package semantic

import (
	"context"
	"errors"
)

// VideoStatus preserves recorded coverage even while a library or model is offline.
// Nil Coverage means no progress has been recorded for the active generation.
type VideoStatus struct {
	LibraryID    int64
	Enabled      bool
	Offline      bool
	GenerationID string
	Coverage     *VideoCoverage
}

type VideoStatusRepository interface {
	GetVideoStatus(context.Context, int64) (VideoStatus, error)
}

type VideoStatusService struct{ repository VideoStatusRepository }

func NewVideoStatusService(repository VideoStatusRepository) (*VideoStatusService, error) {
	if repository == nil {
		return nil, errors.New("video status repository is required")
	}
	return &VideoStatusService{repository: repository}, nil
}

func (service *VideoStatusService) Get(ctx context.Context, libraryID int64) (VideoStatus, error) {
	if libraryID < 1 {
		return VideoStatus{}, ErrSemanticLibraryNotFound
	}
	return service.repository.GetVideoStatus(ctx, libraryID)
}
