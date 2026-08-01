package thumbnail

import (
	"context"
	"errors"
)

var ErrProgressLibraryNotFound = errors.New("media processing library not found")

type JobProgress struct {
	Queued    int64
	Running   int64
	Succeeded int64
	Failed    int64
}

func (progress JobProgress) Total() int64 {
	return progress.Queued + progress.Running + progress.Succeeded + progress.Failed
}

func (progress JobProgress) Processed() int64 {
	return progress.Succeeded + progress.Failed
}

func (progress JobProgress) Active() bool {
	return progress.Queued > 0 || progress.Running > 0
}

type ProcessingProgress struct {
	Grid                         JobProgress
	Storyboard                   JobProgress
	StoryboardPendingEligibility int64
}

func (progress ProcessingProgress) Active() bool {
	return progress.Grid.Active() || progress.Storyboard.Active() ||
		progress.StoryboardPendingEligibility > 0
}

type ProgressRepository interface {
	GetMediaProcessingProgress(context.Context, int64) (ProcessingProgress, bool, error)
}

type ProgressService struct {
	repository ProgressRepository
}

func NewProgressService(repository ProgressRepository) (*ProgressService, error) {
	if repository == nil {
		return nil, errors.New("media processing progress repository is required")
	}
	return &ProgressService{repository: repository}, nil
}

func (service *ProgressService) Get(
	ctx context.Context,
	libraryID int64,
) (ProcessingProgress, error) {
	if libraryID <= 0 {
		return ProcessingProgress{}, ErrProgressLibraryNotFound
	}
	progress, found, err := service.repository.GetMediaProcessingProgress(ctx, libraryID)
	if err != nil {
		return ProcessingProgress{}, err
	}
	if !found {
		return ProcessingProgress{}, ErrProgressLibraryNotFound
	}
	return progress, nil
}
