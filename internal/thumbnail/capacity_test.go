package thumbnail

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type cacheRepositoryStub struct {
	quota   int64
	usage   int64
	entries []CacheEntry
	pending []PendingCacheDeletion
}

func (stub *cacheRepositoryStub) CacheQuota(context.Context) (int64, error) {
	return stub.quota, nil
}

func (stub *cacheRepositoryStub) ReadyCacheUsage(context.Context) (int64, error) {
	return stub.usage, nil
}

func (stub *cacheRepositoryStub) ListLRUCacheEntries(
	_ context.Context,
	limit int,
) ([]CacheEntry, error) {
	return slices.Clone(stub.entries[:min(limit, len(stub.entries))]), nil
}

func (stub *cacheRepositoryStub) DeleteReadyCacheEntry(
	_ context.Context,
	entry CacheEntry,
) error {
	if len(stub.entries) == 0 || stub.entries[0] != entry {
		return errors.New("cache entries were not evicted in LRU order")
	}
	stub.entries = stub.entries[1:]
	stub.usage -= entry.ByteSize
	return nil
}

func (stub *cacheRepositoryStub) ListPendingCacheDeletions(
	_ context.Context,
	limit int,
) ([]PendingCacheDeletion, error) {
	return slices.Clone(stub.pending[:min(limit, len(stub.pending))]), nil
}

func (stub *cacheRepositoryStub) CompleteCacheDeletion(
	_ context.Context,
	item PendingCacheDeletion,
) error {
	if len(stub.pending) == 0 || stub.pending[0] != item {
		return errors.New("pending cache deletion order changed")
	}
	stub.pending = stub.pending[1:]
	return nil
}

type cacheStorageStub struct {
	available int64
	sizes     map[string]int64
	removed   []string
}

func (stub *cacheStorageStub) AvailableBytes(context.Context) (int64, error) {
	return stub.available, nil
}

func (stub *cacheStorageStub) Remove(_ context.Context, path string) error {
	stub.removed = append(stub.removed, path)
	stub.available += stub.sizes[path]
	return nil
}

func TestCacheManagerEvictsToLowWaterlineInLRUOrder(t *testing.T) {
	repository := &cacheRepositoryStub{
		quota: 1000, usage: 950,
		entries: []CacheEntry{
			{AssetID: 1, CacheRelativePath: "old.webp", ByteSize: 100},
			{AssetID: 2, CacheRelativePath: "new.webp", ByteSize: 100},
		},
	}
	storage := &cacheStorageStub{
		available: CacheSafeFreeBytes + 1000,
		sizes: map[string]int64{
			"old.webp": 100,
			"new.webp": 100,
		},
	}
	manager, err := NewCacheManager(repository, storage)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := manager.Reserve(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	reservation.Release()
	if !slices.Equal(storage.removed, []string{"old.webp", "new.webp"}) ||
		repository.usage != 750 {
		t.Fatalf("removed = %v, usage = %d", storage.removed, repository.usage)
	}
}

func TestCacheManagerProtectsFreeSpaceAndCleansInvalidatedFiles(t *testing.T) {
	repository := &cacheRepositoryStub{
		quota: 1000, usage: 100,
		entries: []CacheEntry{
			{AssetID: 1, CacheRelativePath: "ready.webp", ByteSize: 100},
		},
		pending: []PendingCacheDeletion{
			{ID: 1, CacheRelativePath: "stale.webp"},
		},
	}
	storage := &cacheStorageStub{
		available: CacheSafeFreeBytes - 75,
		sizes: map[string]int64{
			"ready.webp": 100,
			"stale.webp": 25,
		},
	}
	manager, _ := NewCacheManager(repository, storage)
	reservation, err := manager.Reserve(context.Background(), 25)
	if err != nil {
		t.Fatal(err)
	}
	reservation.Release()
	if !slices.Equal(storage.removed, []string{"stale.webp", "ready.webp"}) {
		t.Fatalf("removed = %v", storage.removed)
	}
}

func TestCacheManagerFailsClosedWhenNoReconstructibleSpaceRemains(t *testing.T) {
	repository := &cacheRepositoryStub{quota: 1000}
	storage := &cacheStorageStub{
		available: CacheSafeFreeBytes - 1,
		sizes:     map[string]int64{},
	}
	manager, _ := NewCacheManager(repository, storage)
	if _, err := manager.Reserve(
		context.Background(), 1,
	); !errors.Is(err, ErrCacheCapacity) {
		t.Fatalf("capacity error = %v", err)
	}
}
