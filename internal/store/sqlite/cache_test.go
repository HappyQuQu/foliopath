package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func TestCacheMaintenanceWritesQueueBehindExistingTransaction(t *testing.T) {
	t.Run("ready thumbnail eviction", func(t *testing.T) {
		store, _ := openTestStoreWithOptions(t, Options{
			BusyTimeout: 50 * time.Millisecond,
		})
		assertCacheWriteQueuesBehindGate(t, store, func() error {
			return store.DeleteReadyCacheEntries(
				context.Background(),
				[]thumbnail.CacheEntry{{
					AssetID:           1,
					CacheRelativePath: "libraries/lib_1/aa/evicted.webp",
					ByteSize:          44,
				}},
			)
		})
	})

	t.Run("pending deletion completion", func(t *testing.T) {
		store, _ := openTestStoreWithOptions(t, Options{
			BusyTimeout: 50 * time.Millisecond,
		})
		libraryID := seedBrowseCatalog(t, store)
		result, err := store.db.ExecContext(context.Background(), `
            INSERT INTO cache_deletions(
                library_id, cache_rel_path, byte_size, created_at_ms
            ) VALUES (?, 'libraries/lib_1/aa/pending.webp', 44, 1000)`,
			libraryID,
		)
		if err != nil {
			t.Fatal(err)
		}
		deletionID, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		assertCacheWriteQueuesBehindGate(t, store, func() error {
			return store.CompleteCacheDeletion(
				context.Background(),
				thumbnail.PendingCacheDeletion{
					ID:                deletionID,
					CacheRelativePath: "libraries/lib_1/aa/pending.webp",
				},
			)
		})
	})
}

func TestCacheCleanupStateCoalescesAndResumesDurably(t *testing.T) {
	store, filename := openTestStore(t)
	ctx := context.Background()

	initial, err := store.GetCacheCleanup(ctx)
	if err != nil || initial.Status != thumbnail.CleanupIdle || initial.Revision != 1 {
		t.Fatalf("initial cleanup = %#v, %v", initial, err)
	}
	firstKey := sha256.Sum256([]byte("first-key"))
	secondKey := sha256.Sum256([]byte("second-key"))
	thirdKey := sha256.Sum256([]byte("third-key"))
	first, err := store.RequestCacheCleanup(ctx, firstKey, 1000)
	if err != nil || !first.Created || first.Cleanup.Status != thumbnail.CleanupQueued {
		t.Fatalf("first request = %#v, %v", first, err)
	}
	coalesced, err := store.RequestCacheCleanup(ctx, secondKey, 1001)
	if err != nil || coalesced.Created ||
		coalesced.Cleanup.Revision != first.Cleanup.Revision {
		t.Fatalf("coalesced request = %#v, %v", coalesced, err)
	}
	replayed, err := store.RequestCacheCleanup(ctx, firstKey, 1002)
	if err != nil || !replayed.Replayed {
		t.Fatalf("active replay = %#v, %v", replayed, err)
	}
	running, claimed, err := store.ClaimCacheCleanup(ctx, 500, 1100)
	if err != nil || !claimed || running.Status != thumbnail.CleanupRunning ||
		running.InitialUsageBytes != 500 {
		t.Fatalf("claimed cleanup = %#v, %t, %v", running, claimed, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	resumed, claimed, err := reopened.ClaimCacheCleanup(ctx, 400, 1200)
	if err != nil || !claimed || resumed.Status != thumbnail.CleanupRunning ||
		resumed.StartedAtMS == nil || *resumed.StartedAtMS != 1100 {
		t.Fatalf("resumed cleanup = %#v, %t, %v", resumed, claimed, err)
	}
	finished, err := reopened.FinishCacheCleanup(
		ctx, thumbnail.CleanupSucceeded, nil, 1300,
	)
	if err != nil || finished.Status != thumbnail.CleanupSucceeded {
		t.Fatalf("finished cleanup = %#v, %v", finished, err)
	}
	completedReplay, err := reopened.RequestCacheCleanup(ctx, firstKey, 1400)
	if err != nil || !completedReplay.Replayed || completedReplay.Created {
		t.Fatalf("completed replay = %#v, %v", completedReplay, err)
	}
	next, err := reopened.RequestCacheCleanup(ctx, thirdKey, 1500)
	if err != nil || !next.Created || next.Cleanup.Status != thumbnail.CleanupQueued {
		t.Fatalf("next cleanup = %#v, %v", next, err)
	}
}

func assertCacheWriteQueuesBehindGate(
	t *testing.T,
	store *Store,
	operation func() error,
) {
	t.Helper()

	writeStarted := make(chan struct{})
	releaseWrite := make(chan struct{})
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- store.withWriteTx(context.Background(), func(*sql.Tx) error {
			close(writeStarted)
			<-releaseWrite
			return nil
		})
	}()
	<-writeStarted

	operationDone := make(chan error, 1)
	go func() {
		operationDone <- operation()
	}()

	select {
	case err := <-operationDone:
		close(releaseWrite)
		<-writeDone
		t.Fatalf("cache write returned before write gate released: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseWrite)
	if err := <-writeDone; err != nil {
		t.Fatalf("finish held write: %v", err)
	}
	if err := <-operationDone; err != nil {
		t.Fatalf("cache write after write gate release: %v", err)
	}
}
