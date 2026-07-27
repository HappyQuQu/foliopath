package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
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

func TestScanAdmissionEnforcesAndReleasesGlobalCapacity(t *testing.T) {
	now := time.UnixMilli(5_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	active := make([]scanner.AdmissionResult, 0, scanner.MaxActiveFullScans)
	for index := 0; index < scanner.MaxActiveFullScans; index++ {
		record := createWorkerLibrary(
			t,
			store,
			fmt.Sprintf("Library %03d", index),
			fmt.Sprintf("library-%03d", index),
		)
		admitted, err := store.AdmitFullScan(ctx, record.ID, scanner.TriggerManual)
		if err != nil {
			t.Fatalf("admit scan %d: %v", index, err)
		}
		active = append(active, admitted)
	}

	creationAtCapacity := library.CreateCommand{
		Name:             "Creation overflow",
		NameKey:          "creation overflow",
		RootRelativePath: "creation-overflow",
		KeyHash:          [32]byte{0xa1},
		RequestHash:      [32]byte{0xb2},
		RetentionMS:      86_400_000,
	}
	if _, err := store.CreateLibraryWithScan(
		ctx,
		creationAtCapacity,
	); !errors.Is(err, library.ErrScanCapacity) {
		t.Fatalf("creation overflow error = %v, want ErrScanCapacity", err)
	}
	var partialCreation int
	if err := store.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM libraries
		WHERE root_rel_path = ?`,
		creationAtCapacity.RootRelativePath,
	).Scan(&partialCreation); err != nil {
		t.Fatal(err)
	}
	if partialCreation != 0 {
		t.Fatal("capacity-rejected library creation persisted partial state")
	}

	extra := createWorkerLibrary(t, store, "Overflow", "overflow")
	if _, err := store.AdmitFullScan(
		ctx,
		extra.ID,
		scanner.TriggerManual,
	); !errors.Is(err, scanner.ErrAdmissionCapacity) {
		t.Fatalf("overflow admission error = %v, want ErrAdmissionCapacity", err)
	}
	coalesced, err := store.AdmitFullScan(
		ctx,
		active[0].Run.LibraryID,
		scanner.TriggerScheduled,
	)
	if err != nil {
		t.Fatalf("coalesce at capacity: %v", err)
	}
	if !coalesced.Coalesced || coalesced.Run.ID != active[0].Run.ID {
		t.Fatalf("coalesced admission = %#v", coalesced)
	}

	claimed, found, err := store.ClaimNextFullScan(ctx, time.Minute)
	if err != nil || !found || claimed.ID != active[0].Run.ID {
		t.Fatalf("claim capacity release run = %#v, found %t, err %v", claimed, found, err)
	}
	if _, err := store.CancelFullScan(ctx, claimed.ID, scanner.SkipCounts{}); err != nil {
		t.Fatalf("cancel capacity release run: %v", err)
	}
	released, err := store.AdmitFullScan(ctx, extra.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatalf("admit after capacity release: %v", err)
	}
	if released.Coalesced || released.Run.LibraryID != extra.ID {
		t.Fatalf("released-capacity admission = %#v", released)
	}

	var activeCount int
	if err := store.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM scan_runs
		WHERE status IN ('queued', 'running')`,
	).Scan(&activeCount); err != nil {
		t.Fatal(err)
	}
	if activeCount != scanner.MaxActiveFullScans {
		t.Fatalf(
			"active scans after release = %d, want %d",
			activeCount,
			scanner.MaxActiveFullScans,
		)
	}
}

func TestConcurrentAdmissionsCannotExceedGlobalCapacity(t *testing.T) {
	firstStore, filename := openTestStore(t)
	secondStore, err := Open(context.Background(), filename, Options{})
	if err != nil {
		t.Fatalf("Open(second store) error = %v", err)
	}
	t.Cleanup(func() { _ = secondStore.Close() })
	ctx := context.Background()
	for index := 0; index < scanner.MaxActiveFullScans-1; index++ {
		record := createWorkerLibrary(
			t,
			firstStore,
			fmt.Sprintf("Existing %03d", index),
			fmt.Sprintf("existing-%03d", index),
		)
		if _, err := firstStore.AdmitFullScan(
			ctx,
			record.ID,
			scanner.TriggerManual,
		); err != nil {
			t.Fatalf("admit existing scan %d: %v", index, err)
		}
	}
	firstCandidate := createWorkerLibrary(t, firstStore, "Candidate A", "candidate-a")
	secondCandidate := createWorkerLibrary(t, firstStore, "Candidate B", "candidate-b")

	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, candidate := range []struct {
		store     *Store
		libraryID int64
	}{
		{store: firstStore, libraryID: firstCandidate.ID},
		{store: secondStore, libraryID: secondCandidate.ID},
	} {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := candidate.store.AdmitFullScan(
				ctx,
				candidate.libraryID,
				scanner.TriggerManual,
			)
			results <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	var admitted, atCapacity int
	for err := range results {
		switch {
		case err == nil:
			admitted++
		case errors.Is(err, scanner.ErrAdmissionCapacity):
			atCapacity++
		default:
			t.Fatalf("concurrent admission error = %v", err)
		}
	}
	if admitted != 1 || atCapacity != 1 {
		t.Fatalf(
			"concurrent capacity results = admitted %d, at capacity %d",
			admitted,
			atCapacity,
		)
	}
	var activeCount int
	if err := firstStore.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM scan_runs
		WHERE status IN ('queued', 'running')`,
	).Scan(&activeCount); err != nil {
		t.Fatal(err)
	}
	if activeCount != scanner.MaxActiveFullScans {
		t.Fatalf("active scans = %d, want %d", activeCount, scanner.MaxActiveFullScans)
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
