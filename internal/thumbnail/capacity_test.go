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
	batches [][]CacheEntry
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

func (stub *cacheRepositoryStub) DeleteReadyCacheEntries(
	_ context.Context,
	entries []CacheEntry,
) error {
	if len(entries) == 0 || len(entries) > len(stub.entries) ||
		!slices.Equal(stub.entries[:len(entries)], entries) {
		return errors.New("cache entries were not evicted in LRU order")
	}
	for _, entry := range entries {
		stub.usage -= entry.ByteSize
	}
	stub.batches = append(stub.batches, slices.Clone(entries))
	stub.entries = stub.entries[len(entries):]
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
	removeErr error
	failPath  string
}

func (stub *cacheStorageStub) AvailableBytes(context.Context) (int64, error) {
	return stub.available, nil
}

func (stub *cacheStorageStub) Remove(_ context.Context, path string) error {
	if stub.removeErr != nil && (stub.failPath == "" || stub.failPath == path) {
		return stub.removeErr
	}
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
	if len(repository.batches) != 1 || len(repository.batches[0]) != 2 {
		t.Fatalf("eviction batches = %v", repository.batches)
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

func TestCacheManagerSerializesReservationsAndHonorsCancellation(t *testing.T) {
	manager, _ := NewCacheManager(
		&cacheRepositoryStub{quota: DefaultCacheQuotaBytes},
		&cacheStorageStub{
			available: CacheSafeFreeBytes + DefaultCacheQuotaBytes,
			sizes:     map[string]int64{},
		},
	)
	first, err := manager.Reserve(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := manager.Reserve(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("blocked reservation error = %v", err)
	}
	first.Release()
}

func TestCacheManagerPreservesDatabaseStateWhenEvictionFails(t *testing.T) {
	repository := &cacheRepositoryStub{
		quota: 1000, usage: 100,
		entries: []CacheEntry{
			{AssetID: 1, CacheRelativePath: "ready.webp", ByteSize: 100},
		},
	}
	storage := &cacheStorageStub{
		available: CacheSafeFreeBytes - 1,
		sizes:     map[string]int64{"ready.webp": 100},
		removeErr: errors.New("injected cache filesystem failure"),
	}
	manager, _ := NewCacheManager(repository, storage)
	if _, err := manager.Reserve(context.Background(), 1); err == nil {
		t.Fatal("cache filesystem failure unexpectedly succeeded")
	}
	if repository.usage != 100 || len(repository.entries) != 1 {
		t.Fatalf(
			"database state changed after failed deletion: usage %d entries %v",
			repository.usage, repository.entries,
		)
	}
}

func TestCacheManagerReconcilesRemovedPrefixWhenEvictionFails(t *testing.T) {
	repository := &cacheRepositoryStub{
		quota: 1000, usage: 950,
		entries: []CacheEntry{
			{AssetID: 1, CacheRelativePath: "removed.webp", ByteSize: 100},
			{AssetID: 2, CacheRelativePath: "failed.webp", ByteSize: 100},
		},
	}
	storage := &cacheStorageStub{
		available: CacheSafeFreeBytes - 1,
		sizes: map[string]int64{
			"removed.webp": 100,
			"failed.webp":  100,
		},
		removeErr: errors.New("injected cache filesystem failure"),
		failPath:  "failed.webp",
	}
	manager, _ := NewCacheManager(repository, storage)
	if _, err := manager.Reserve(context.Background(), 1); err == nil {
		t.Fatal("cache filesystem failure unexpectedly succeeded")
	}
	if repository.usage != 850 ||
		len(repository.entries) != 1 ||
		repository.entries[0].CacheRelativePath != "failed.webp" {
		t.Fatalf(
			"removed prefix did not reconcile: usage %d entries %v",
			repository.usage, repository.entries,
		)
	}
}
