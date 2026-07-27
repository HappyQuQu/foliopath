package library

import (
	"context"
	"errors"
	"time"
)

type RemovalRepository interface {
	ClaimNextLibraryRemoval(context.Context) (Removal, bool, error)
	LibraryRemovalReady(context.Context, int64) (bool, error)
	CleanupLibraryRemovalBatch(context.Context, int64, int) (bool, error)
	FailLibraryRemoval(context.Context, int64, string) error
}

type DerivedCacheCleaner interface {
	RemoveLibraryCache(context.Context, int64) error
}

type RemovalWorker struct {
	repository RemovalRepository
	cache      DerivedCacheCleaner
	wake       chan struct{}
	batchSize  int
}

func NewRemovalWorker(
	repository RemovalRepository,
	cache DerivedCacheCleaner,
	batchSize int,
) (*RemovalWorker, error) {
	if repository == nil || cache == nil {
		return nil, errors.New("library removal worker dependencies are required")
	}
	if batchSize == 0 {
		batchSize = 500
	}
	if batchSize < 1 || batchSize > 1000 {
		return nil, errors.New("library removal batch size must be between 1 and 1000")
	}
	return &RemovalWorker{
		repository: repository,
		cache:      cache,
		wake:       make(chan struct{}, 1),
		batchSize:  batchSize,
	}, nil
}

func (worker *RemovalWorker) Wake() {
	select {
	case worker.wake <- struct{}{}:
	default:
	}
}

func (worker *RemovalWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	worker.Wake()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-worker.wake:
		case <-ticker.C:
		}
		for {
			progressed, err := worker.runOne(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				break
			}
			if !progressed {
				break
			}
		}
	}
}

func (worker *RemovalWorker) runOne(ctx context.Context) (bool, error) {
	removal, found, err := worker.repository.ClaimNextLibraryRemoval(ctx)
	if err != nil || !found {
		return found, err
	}
	ready, err := worker.repository.LibraryRemovalReady(ctx, removal.ID)
	if err != nil {
		_ = worker.repository.FailLibraryRemoval(ctx, removal.ID, "application_data_unavailable")
		return false, err
	}
	if !ready {
		return false, nil
	}
	if err := worker.cache.RemoveLibraryCache(ctx, removal.LibraryID); err != nil {
		_ = worker.repository.FailLibraryRemoval(ctx, removal.ID, "application_data_unavailable")
		return false, err
	}
	done, err := worker.repository.CleanupLibraryRemovalBatch(
		ctx,
		removal.ID,
		worker.batchSize,
	)
	if err != nil {
		_ = worker.repository.FailLibraryRemoval(ctx, removal.ID, "cleanup_interrupted")
		return false, err
	}
	return !done, nil
}
