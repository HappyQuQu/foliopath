package sqlite

import (
	"context"
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
