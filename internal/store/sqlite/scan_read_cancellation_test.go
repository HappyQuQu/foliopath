package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Mirrors cancelled settings reads while the idle scan queue keeps polling.
// Only temporary synthetic catalog state is used, never a media filesystem.
func TestScanPollingSurvivesCancelledSnapshotReads(t *testing.T) {
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "cancel.db"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	library := createWorkerLibrary(t, store, "Synthetic", "synthetic")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var workers sync.WaitGroup
	failures := make(chan error, 10)
	for i := 0; i < 6; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			iteration := 0
			for ctx.Err() == nil {
				read, abort := context.WithTimeout(ctx, 100*time.Microsecond)
				var err error
				if iteration%2 == 0 {
					_, err = store.GetFaceLibrarySettings(read, library.ID)
				} else {
					var sum int64
					err = store.db.QueryRowContext(read, `WITH RECURSIVE counter(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM counter WHERE n<100000) SELECT sum(n) FROM counter`).Scan(&sum)
				}
				iteration++
				wasCancelled := read.Err() != nil
				abort()
				if err != nil && !wasCancelled && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					failures <- err
					return
				}
			}
		}()
	}
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for ctx.Err() == nil {
				_, _, err := store.ClaimNextFullScan(ctx, time.Minute)
				if err == nil {
					_, err = store.RecoverExpiredFullScans(ctx)
				}
				if err != nil && ctx.Err() == nil {
					failures <- err
					return
				}
			}
		}()
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Errorf("live scan/read context failed: %T %v", err, err)
	}
}
