package thumbnail

import (
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"strconv"
	"testing"
)

type cacheRepositoryStub struct {
	quota              int64
	usage              int64
	entries            []CacheEntry
	pending            []PendingCacheDeletion
	batches            [][]CacheEntry
	cleanup            Cleanup
	lastCleanupKeyHash [sha256.Size]byte
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

func (stub *cacheRepositoryStub) GetCacheCleanup(context.Context) (Cleanup, error) {
	return stub.cleanup, nil
}

func (stub *cacheRepositoryStub) RequestCacheCleanup(
	_ context.Context,
	keyHash [sha256.Size]byte,
	requestedAtMS int64,
) (CleanupRequestResult, error) {
	if stub.cleanup.Status == CleanupQueued || stub.cleanup.Status == CleanupRunning {
		return CleanupRequestResult{Cleanup: stub.cleanup}, nil
	}
	stub.cleanup = Cleanup{
		Revision: 2, Status: CleanupQueued, IdempotencyKeyHash: keyHash,
		RequestedAtMS: &requestedAtMS,
	}
	stub.lastCleanupKeyHash = keyHash
	return CleanupRequestResult{Cleanup: stub.cleanup, Created: true}, nil
}

func (stub *cacheRepositoryStub) ClaimCacheCleanup(
	_ context.Context,
	usageBytes, startedAtMS int64,
) (Cleanup, bool, error) {
	if stub.cleanup.Status != CleanupQueued && stub.cleanup.Status != CleanupRunning {
		return stub.cleanup, false, nil
	}
	stub.cleanup.Status = CleanupRunning
	stub.cleanup.StartedAtMS = &startedAtMS
	stub.cleanup.InitialUsageBytes = usageBytes
	stub.cleanup.RemainingUsageBytes = usageBytes
	return stub.cleanup, true, nil
}

func (stub *cacheRepositoryStub) UpdateCacheCleanupProgress(
	_ context.Context,
	progress CleanupProgress,
) error {
	stub.cleanup.RemainingUsageBytes = progress.RemainingUsageBytes
	stub.cleanup.ReclaimedBytes = progress.ReclaimedBytes
	stub.cleanup.DeletedEntries = progress.DeletedEntries
	return nil
}

func (stub *cacheRepositoryStub) FinishCacheCleanup(
	_ context.Context,
	status CleanupStatus,
	errorCode *string,
	finishedAtMS int64,
) (Cleanup, error) {
	stub.cleanup.Status = status
	stub.cleanup.ErrorCode = errorCode
	stub.cleanup.FinishedAtMS = &finishedAtMS
	stub.cleanup.Revision++
	return stub.cleanup, nil
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

func TestCacheManagerCleanupDeletesOnlyReadyDerivedEntries(t *testing.T) {
	repository := &cacheRepositoryStub{
		quota: 1000,
		usage: 300,
		entries: []CacheEntry{
			{AssetID: 1, CacheRelativePath: "one.webp", ByteSize: 100},
			{AssetID: 2, CacheRelativePath: "two.webp", ByteSize: 200},
		},
		cleanup: Cleanup{Revision: 1, Status: CleanupIdle},
	}
	storage := &cacheStorageStub{
		available: CacheSafeFreeBytes,
		sizes:     map[string]int64{"one.webp": 100, "two.webp": 200},
	}
	manager, _ := NewCacheManager(repository, storage)
	result, err := manager.StartCleanup(context.Background(), "cleanup-key-1")
	if err != nil || !result.Created {
		t.Fatalf("StartCleanup() = %#v, %v", result, err)
	}
	if repository.lastCleanupKeyHash != sha256.Sum256([]byte("cleanup-key-1")) {
		t.Fatal("idempotency key digest was not retained")
	}
	if err := manager.processCleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.cleanup.Status != CleanupSucceeded ||
		repository.cleanup.RemainingUsageBytes != 0 ||
		repository.cleanup.ReclaimedBytes != 300 ||
		repository.cleanup.DeletedEntries != 2 ||
		repository.usage != 0 ||
		!slices.Equal(storage.removed, []string{"one.webp", "two.webp"}) {
		t.Fatalf("cleanup = %#v, usage = %d, removed = %v",
			repository.cleanup, repository.usage, storage.removed)
	}
}

func TestCacheManagerCleanupPersistsSanitizedFailureAndRemainsRetryable(t *testing.T) {
	repository := &cacheRepositoryStub{
		quota: 1000, usage: 100,
		entries: []CacheEntry{
			{AssetID: 1, CacheRelativePath: "denied.webp", ByteSize: 100},
		},
		cleanup: Cleanup{Revision: 1, Status: CleanupQueued},
	}
	storage := &cacheStorageStub{
		available: CacheSafeFreeBytes,
		sizes:     map[string]int64{"denied.webp": 100},
		removeErr: errors.New("permission denied at /secret/cache/path"),
	}
	manager, _ := NewCacheManager(repository, storage)
	if err := manager.processCleanup(context.Background()); err == nil {
		t.Fatal("cleanup filesystem failure unexpectedly succeeded")
	}
	if repository.cleanup.Status != CleanupFailed ||
		repository.cleanup.ErrorCode == nil ||
		*repository.cleanup.ErrorCode != "internal_error" ||
		repository.usage != 100 ||
		len(repository.entries) != 1 {
		t.Fatalf("failed cleanup = %#v, usage = %d, entries = %v",
			repository.cleanup, repository.usage, repository.entries)
	}
}

func TestCacheManagerCleanupUsesBoundedBatchesAtHundredThousandEntries(t *testing.T) {
	const itemCount = 100_000
	entries := make([]CacheEntry, itemCount)
	sizes := make(map[string]int64, itemCount)
	for index := range entries {
		name := strconv.Itoa(index) + ".webp"
		entries[index] = CacheEntry{
			AssetID: int64(index + 1), CacheRelativePath: name, ByteSize: 1,
		}
		sizes[name] = 1
	}
	repository := &cacheRepositoryStub{
		quota: itemCount, usage: itemCount, entries: entries,
		cleanup: Cleanup{Revision: 1, Status: CleanupQueued},
	}
	manager, _ := NewCacheManager(repository, &cacheStorageStub{
		available: CacheSafeFreeBytes,
		sizes:     sizes,
	})
	if err := manager.processCleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.cleanup.DeletedEntries != itemCount ||
		repository.cleanup.Status != CleanupSucceeded {
		t.Fatalf("100k cleanup = %#v", repository.cleanup)
	}
	for index, batch := range repository.batches {
		if len(batch) < 1 || len(batch) > cacheEvictionBatch {
			t.Fatalf("batch %d size = %d", index, len(batch))
		}
	}
}
