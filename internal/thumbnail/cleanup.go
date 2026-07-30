package thumbnail

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

type CleanupStatus string

const (
	CleanupIdle      CleanupStatus = "idle"
	CleanupQueued    CleanupStatus = "queued"
	CleanupRunning   CleanupStatus = "running"
	CleanupSucceeded CleanupStatus = "succeeded"
	CleanupFailed    CleanupStatus = "failed"
)

type Cleanup struct {
	Revision            int64
	Status              CleanupStatus
	IdempotencyKeyHash  [sha256.Size]byte
	RequestedAtMS       *int64
	StartedAtMS         *int64
	FinishedAtMS        *int64
	InitialUsageBytes   int64
	RemainingUsageBytes int64
	ReclaimedBytes      int64
	DeletedEntries      int64
	ErrorCode           *string
}

type CacheSummary struct {
	UsageBytes         int64
	QuotaBytes         int64
	HighWatermarkBytes int64
	LowWatermarkBytes  int64
	AvailableBytes     int64
	SafeFreeSpaceBytes int64
	Pressure           string
	Cleanup            Cleanup
}

type CleanupRequestResult struct {
	Cleanup  Cleanup
	Created  bool
	Replayed bool
}

type CleanupProgress struct {
	RemainingUsageBytes int64
	ReclaimedBytes      int64
	DeletedEntries      int64
	UpdatedAtMS         int64
}

type CleanupRepository interface {
	GetCacheCleanup(context.Context) (Cleanup, error)
	RequestCacheCleanup(context.Context, [sha256.Size]byte, int64) (CleanupRequestResult, error)
	ClaimCacheCleanup(context.Context, int64, int64) (Cleanup, bool, error)
	UpdateCacheCleanupProgress(context.Context, CleanupProgress) error
	FinishCacheCleanup(context.Context, CleanupStatus, *string, int64) (Cleanup, error)
}

func (manager *CacheManager) Summary(ctx context.Context) (CacheSummary, error) {
	repository, ok := manager.repository.(CleanupRepository)
	if !ok {
		return CacheSummary{}, ErrInvalidState
	}
	quota, err := manager.repository.CacheQuota(ctx)
	if err != nil {
		return CacheSummary{}, err
	}
	usage, err := manager.repository.ReadyCacheUsage(ctx)
	if err != nil {
		return CacheSummary{}, err
	}
	available, err := manager.storage.AvailableBytes(ctx)
	if err != nil {
		return CacheSummary{}, err
	}
	cleanup, err := repository.GetCacheCleanup(ctx)
	if err != nil {
		return CacheSummary{}, err
	}
	high := quota * CacheHighWaterPercent / 100
	low := quota * CacheLowWaterPercent / 100
	quotaPressure := usage > high
	storagePressure := available < CacheSafeFreeBytes
	pressure := "normal"
	switch {
	case quotaPressure && storagePressure:
		pressure = "quota_and_storage"
	case quotaPressure:
		pressure = "quota"
	case storagePressure:
		pressure = "storage"
	}
	return CacheSummary{
		UsageBytes: usage, QuotaBytes: quota,
		HighWatermarkBytes: high, LowWatermarkBytes: low,
		AvailableBytes: available, SafeFreeSpaceBytes: CacheSafeFreeBytes,
		Pressure: pressure, Cleanup: cleanup,
	}, nil
}

func (manager *CacheManager) StartCleanup(
	ctx context.Context,
	idempotencyKey string,
) (CleanupRequestResult, error) {
	repository, ok := manager.repository.(CleanupRepository)
	if !ok || idempotencyKey == "" || len(idempotencyKey) > 128 {
		return CleanupRequestResult{}, ErrInvalidState
	}
	digest := sha256.Sum256([]byte(idempotencyKey))
	result, err := repository.RequestCacheCleanup(
		ctx,
		digest,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		return CleanupRequestResult{}, err
	}
	if result.Created {
		select {
		case manager.cleanupWake <- struct{}{}:
		default:
		}
	}
	return result, nil
}

func (manager *CacheManager) Cleanup(ctx context.Context) (Cleanup, error) {
	repository, ok := manager.repository.(CleanupRepository)
	if !ok {
		return Cleanup{}, ErrInvalidState
	}
	return repository.GetCacheCleanup(ctx)
}

func (manager *CacheManager) processCleanup(ctx context.Context) error {
	repository, ok := manager.repository.(CleanupRepository)
	if !ok {
		return nil
	}
	usage, err := manager.repository.ReadyCacheUsage(ctx)
	if err != nil {
		return err
	}
	cleanup, claimed, err := repository.ClaimCacheCleanup(
		ctx,
		usage,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil || !claimed {
		return err
	}
	reservation, err := manager.reserveGate(ctx)
	if err != nil {
		return err
	}
	defer reservation.Release()

	remaining := usage
	reclaimed := cleanup.ReclaimedBytes
	deleted := cleanup.DeletedEntries
	for {
		entries, listErr := manager.repository.ListLRUCacheEntries(ctx, cacheEvictionBatch)
		if listErr != nil {
			return manager.failCleanup(ctx, repository, listErr)
		}
		if len(entries) == 0 {
			_, finishErr := repository.FinishCacheCleanup(
				ctx, CleanupSucceeded, nil, time.Now().UTC().UnixMilli(),
			)
			return finishErr
		}
		removed := make([]CacheEntry, 0, len(entries))
		for _, entry := range entries {
			if removeErr := manager.storage.Remove(ctx, entry.CacheRelativePath); removeErr != nil {
				if len(removed) > 0 {
					if deleteErr := manager.repository.DeleteReadyCacheEntries(ctx, removed); deleteErr != nil {
						removeErr = errors.Join(removeErr, deleteErr)
					}
				}
				return manager.failCleanup(ctx, repository, removeErr)
			}
			removed = append(removed, entry)
			remaining -= entry.ByteSize
			reclaimed += entry.ByteSize
			deleted++
		}
		if err := manager.repository.DeleteReadyCacheEntries(ctx, removed); err != nil {
			return manager.failCleanup(ctx, repository, err)
		}
		if remaining < 0 {
			remaining = 0
		}
		if err := repository.UpdateCacheCleanupProgress(ctx, CleanupProgress{
			RemainingUsageBytes: remaining,
			ReclaimedBytes:      reclaimed,
			DeletedEntries:      deleted,
			UpdatedAtMS:         time.Now().UTC().UnixMilli(),
		}); err != nil {
			return err
		}
	}
}

func (manager *CacheManager) failCleanup(
	ctx context.Context,
	repository CleanupRepository,
	cause error,
) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	code := "internal_error"
	_, err := repository.FinishCacheCleanup(
		ctx, CleanupFailed, &code, time.Now().UTC().UnixMilli(),
	)
	return errors.Join(fmt.Errorf("cache cleanup failed: %w", cause), err)
}
