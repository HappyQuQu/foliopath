package thumbnail

import (
	"context"
	"errors"
	"sync"
	"time"
)

const cacheEvictionBatch = 64

type CacheEntry struct {
	AssetID           int64
	CacheRelativePath string
	ByteSize          int64
}

type PendingCacheDeletion struct {
	ID                int64
	CacheRelativePath string
}

type CacheRepository interface {
	CacheQuota(context.Context) (int64, error)
	ReadyCacheUsage(context.Context) (int64, error)
	ListLRUCacheEntries(context.Context, int) ([]CacheEntry, error)
	DeleteReadyCacheEntry(context.Context, CacheEntry) error
	ListPendingCacheDeletions(context.Context, int) ([]PendingCacheDeletion, error)
	CompleteCacheDeletion(context.Context, PendingCacheDeletion) error
}

type CacheStorage interface {
	AvailableBytes(context.Context) (int64, error)
	Remove(context.Context, string) error
}

type Reservation interface {
	Release()
}

type Capacity interface {
	Reserve(context.Context, int64) (Reservation, error)
}

type CacheManager struct {
	repository CacheRepository
	storage    CacheStorage
	gate       chan struct{}
}

func NewCacheManager(
	repository CacheRepository,
	storage CacheStorage,
) (*CacheManager, error) {
	if repository == nil || storage == nil {
		return nil, errors.New("cache manager dependencies are required")
	}
	manager := &CacheManager{
		repository: repository,
		storage:    storage,
		gate:       make(chan struct{}, 1),
	}
	manager.gate <- struct{}{}
	return manager, nil
}

func (manager *CacheManager) Reserve(
	ctx context.Context,
	incomingBytes int64,
) (Reservation, error) {
	if incomingBytes < 0 {
		return nil, ErrCacheCapacity
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-manager.gate:
	}
	if err := manager.reconcileLocked(ctx, incomingBytes); err != nil {
		manager.gate <- struct{}{}
		return nil, err
	}
	return &cacheReservation{release: func() { manager.gate <- struct{}{} }}, nil
}

func (manager *CacheManager) Reconcile(ctx context.Context) error {
	reservation, err := manager.Reserve(ctx, 0)
	if err != nil {
		return err
	}
	reservation.Release()
	return nil
}

func (manager *CacheManager) Run(
	ctx context.Context,
	notifications <-chan struct{},
) error {
	if notifications == nil {
		return errors.New("cache manager notifications are required")
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := manager.Reconcile(ctx); err != nil &&
			!errors.Is(err, ErrCacheCapacity) &&
			ctx.Err() == nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-notifications:
		case <-ticker.C:
		}
	}
}

func (manager *CacheManager) reconcileLocked(
	ctx context.Context,
	incomingBytes int64,
) error {
	if err := manager.removePending(ctx); err != nil {
		return err
	}
	quota, err := manager.repository.CacheQuota(ctx)
	if err != nil || ValidateCacheQuota(quota) != nil || incomingBytes > quota {
		if err != nil {
			return err
		}
		return ErrCacheCapacity
	}
	usage, err := manager.repository.ReadyCacheUsage(ctx)
	if err != nil {
		return err
	}
	available, err := manager.storage.AvailableBytes(ctx)
	if err != nil {
		return err
	}
	high := quota * CacheHighWaterPercent / 100
	low := quota * CacheLowWaterPercent / 100
	quotaPressure := usage+incomingBytes > high
	targetUsage := high - incomingBytes
	if quotaPressure {
		targetUsage = low - incomingBytes
	}
	if targetUsage < 0 {
		targetUsage = 0
	}

	for usage > targetUsage ||
		available < CacheSafeFreeBytes+incomingBytes {
		candidates, err := manager.repository.ListLRUCacheEntries(
			ctx, cacheEvictionBatch,
		)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return ErrCacheCapacity
		}
		for _, candidate := range candidates {
			if err := manager.storage.Remove(ctx, candidate.CacheRelativePath); err != nil {
				return err
			}
			if err := manager.repository.DeleteReadyCacheEntry(ctx, candidate); err != nil {
				return err
			}
			usage -= candidate.ByteSize
			if usage < 0 {
				usage = 0
			}
			if usage <= targetUsage {
				available, err = manager.storage.AvailableBytes(ctx)
				if err != nil {
					return err
				}
				if available >= CacheSafeFreeBytes+incomingBytes {
					break
				}
			}
		}
		available, err = manager.storage.AvailableBytes(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *CacheManager) removePending(ctx context.Context) error {
	for {
		pending, err := manager.repository.ListPendingCacheDeletions(
			ctx, cacheEvictionBatch,
		)
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			return nil
		}
		for _, candidate := range pending {
			if err := manager.storage.Remove(ctx, candidate.CacheRelativePath); err != nil {
				return err
			}
			if err := manager.repository.CompleteCacheDeletion(ctx, candidate); err != nil {
				return err
			}
		}
	}
}

type cacheReservation struct {
	once    sync.Once
	release func()
}

func (reservation *cacheReservation) Release() {
	reservation.once.Do(func() {
		if reservation.release != nil {
			reservation.release()
		}
	})
}
