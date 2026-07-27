package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func openTimedScanStore(t *testing.T, now *time.Time) *Store {
	t.Helper()
	store, err := Open(
		context.Background(),
		filepath.Join(t.TempDir(), "scan-worker.db"),
		Options{
			Now: func() time.Time { return *now },
		},
	)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return store
}

func createWorkerLibrary(
	t *testing.T,
	store *Store,
	name string,
	root string,
) library.Library {
	t.Helper()
	service, err := library.NewService(store)
	if err != nil {
		t.Fatalf("library.NewService() error = %v", err)
	}
	record, err := service.Create(context.Background(), name, root)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", root, err)
	}
	return record
}

func TestListStartupLibraryIDsUsesKeysetAndExcludesActiveRemoval(t *testing.T) {
	now := time.UnixMilli(5_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	first := createWorkerLibrary(t, store, "First", "first")
	second := createWorkerLibrary(t, store, "Second", "second")
	third := createWorkerLibrary(t, store, "Third", "third")
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO library_removals(
			library_id, library_name, status, created_at_ms
		) VALUES (?, ?, 'queued', ?)`,
		second.ID,
		second.Name,
		now.UnixMilli(),
	); err != nil {
		t.Fatal(err)
	}

	page, err := store.ListStartupLibraryIDs(ctx, 0, 1)
	if err != nil {
		t.Fatalf("ListStartupLibraryIDs(first) error = %v", err)
	}
	if len(page) != 1 || page[0] != first.ID {
		t.Fatalf("first startup page = %v", page)
	}
	page, err = store.ListStartupLibraryIDs(ctx, first.ID, 2)
	if err != nil {
		t.Fatalf("ListStartupLibraryIDs(second) error = %v", err)
	}
	if len(page) != 1 || page[0] != third.ID {
		t.Fatalf("second startup page = %v", page)
	}
}

func TestScanQueueClaimUsesDurableOrderAndLease(t *testing.T) {
	now := time.UnixMilli(5_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	firstLibrary := createWorkerLibrary(t, store, "First", "first")
	secondLibrary := createWorkerLibrary(t, store, "Second", "second")
	first, err := store.AdmitFullScan(ctx, firstLibrary.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.AdmitFullScan(ctx, secondLibrary.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx,
		`UPDATE scan_runs SET available_at_ms = 4500 WHERE id = ?`,
		first.Run.ID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx,
		`UPDATE scan_runs SET available_at_ms = 4000 WHERE id = ?`,
		second.Run.ID,
	); err != nil {
		t.Fatal(err)
	}

	claimed, found, err := store.ClaimNextFullScan(ctx, 2*time.Minute)
	if err != nil {
		t.Fatalf("ClaimNextFullScan() error = %v", err)
	}
	if !found || claimed.ID != second.Run.ID ||
		claimed.Status != scanner.RunStatusRunning ||
		claimed.Phase != scanner.PhaseCheckingRoot {
		t.Fatalf("claimed run = %#v, want second queued run", claimed)
	}
	var attemptCount, heartbeatMS, leaseMS, revision int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT attempt_count, heartbeat_at_ms, lease_expires_at_ms, revision
		FROM scan_runs WHERE id = ?`,
		claimed.ID,
	).Scan(&attemptCount, &heartbeatMS, &leaseMS, &revision); err != nil {
		t.Fatal(err)
	}
	if attemptCount != 1 || heartbeatMS != 5_000 || leaseMS != 125_000 {
		t.Fatalf(
			"claim lease = attempt %d, heartbeat %d, expiry %d",
			attemptCount,
			heartbeatMS,
			leaseMS,
		)
	}

	now = time.UnixMilli(10_000)
	refreshed, err := store.RefreshFullScanLease(ctx, claimed.ID, 2*time.Minute)
	if err != nil {
		t.Fatalf("RefreshFullScanLease() error = %v", err)
	}
	if refreshed.Revision != revision {
		t.Fatalf("heartbeat revision = %d, want unchanged %d", refreshed.Revision, revision)
	}
	if err := store.db.QueryRowContext(ctx,
		`SELECT heartbeat_at_ms, lease_expires_at_ms FROM scan_runs WHERE id = ?`,
		claimed.ID,
	).Scan(&heartbeatMS, &leaseMS); err != nil {
		t.Fatal(err)
	}
	if heartbeatMS != 10_000 || leaseMS != 130_000 {
		t.Fatalf("refreshed heartbeat = %d, expiry = %d", heartbeatMS, leaseMS)
	}
}

func TestScanQueueRecoveryRequeuesBeforeThirdExpiry(t *testing.T) {
	now := time.UnixMilli(20_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	firstLibrary := createWorkerLibrary(t, store, "Retry", "retry")
	secondLibrary := createWorkerLibrary(t, store, "Interrupted", "interrupted")
	first, err := store.AdmitFullScan(ctx, firstLibrary.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.AdmitFullScan(ctx, secondLibrary.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	for _, runID := range []int64{first.Run.ID, second.Run.ID} {
		if _, found, err := store.ClaimNextFullScan(ctx, time.Minute); err != nil || !found {
			t.Fatalf("claim run %d: found=%v error=%v", runID, found, err)
		}
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE scan_runs
		SET lease_expires_at_ms = 19000,
		    attempt_count = CASE id WHEN ? THEN 2 ELSE 3 END
		WHERE id IN (?, ?)`,
		first.Run.ID, first.Run.ID, second.Run.ID,
	); err != nil {
		t.Fatal(err)
	}

	summary, err := store.RecoverExpiredFullScans(ctx)
	if err != nil {
		t.Fatalf("RecoverExpiredFullScans() error = %v", err)
	}
	if summary.Requeued != 1 || summary.Interrupted != 1 {
		t.Fatalf("recovery summary = %#v", summary)
	}
	requeued, err := store.GetScanRun(ctx, first.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	interrupted, err := store.GetScanRun(ctx, second.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if requeued.Status != scanner.RunStatusQueued ||
		requeued.StartedAtMS != nil ||
		requeued.ErrorCode != "" {
		t.Fatalf("requeued run = %#v", requeued)
	}
	if interrupted.Status != scanner.RunStatusInterrupted ||
		interrupted.ErrorCode != "scan_interrupted" ||
		interrupted.FinishedAtMS == nil {
		t.Fatalf("interrupted run = %#v", interrupted)
	}
	retryLibrary, err := store.GetLibrary(ctx, firstLibrary.ID)
	if err != nil {
		t.Fatal(err)
	}
	failedLibrary, err := store.GetLibrary(ctx, secondLibrary.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retryLibrary.Status != library.StatusPending {
		t.Fatalf("requeued library status = %q, want pending", retryLibrary.Status)
	}
	if failedLibrary.Status != library.StatusError {
		t.Fatalf("interrupted library status = %q, want error", failedLibrary.Status)
	}
}
