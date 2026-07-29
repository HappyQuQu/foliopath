package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func TestEnqueueReconcileCoalescesAndPreservesRunningWatermark(t *testing.T) {
	now := time.UnixMilli(1_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	item := createWorkerLibrary(t, store, "Archive", "archive")

	first, err := store.EnqueueReconcile(
		ctx,
		item.ID,
		"albums",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != scanner.ReconcileQueued ||
		first.RequestedRevision != 1 ||
		first.AvailableAtMS != 1_750 {
		t.Fatalf("first reconciliation job = %#v", first)
	}

	now = now.Add(500 * time.Millisecond)
	second, err := store.EnqueueReconcile(
		ctx,
		item.ID,
		"albums",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID ||
		second.RequestedRevision != 2 ||
		second.AvailableAtMS != 2_250 {
		t.Fatalf("coalesced reconciliation job = %#v", second)
	}

	if _, err := store.db.ExecContext(ctx, `
		UPDATE catalog_reconcile_jobs
		SET status = 'running',
		    claimed_revision = requested_revision,
		    lease_expires_at_ms = 10_000,
		    attempt_count = 1
		WHERE id = ?
	`, first.ID); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	running, err := store.EnqueueReconcile(
		ctx,
		item.ID,
		"albums",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	)
	if err != nil {
		t.Fatal(err)
	}
	if running.Status != scanner.ReconcileRunning ||
		running.RequestedRevision != 3 ||
		running.ClaimedRevision == nil ||
		*running.ClaimedRevision != 2 {
		t.Fatalf("running reconciliation watermark = %#v", running)
	}
}

func TestEnqueueReconcileRejectsUnknownLibraryAndInvalidTiming(t *testing.T) {
	store, _ := openTestStore(t)
	for _, target := range []string{"../escape", "a/../escape", `a\b`} {
		if _, err := store.EnqueueReconcile(
			context.Background(),
			1,
			target,
			scanner.ReconcileDebounce,
			scanner.ReconcileMaximumDebounce,
		); err == nil {
			t.Errorf("unsafe target %q unexpectedly accepted", target)
		}
	}
	if _, err := store.EnqueueReconcile(
		context.Background(),
		99,
		"",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	); err == nil {
		t.Fatal("unknown library unexpectedly accepted")
	}
	if _, err := store.EnqueueReconcile(
		context.Background(),
		1,
		"",
		time.Second,
		time.Millisecond,
	); err == nil {
		t.Fatal("invalid debounce timing unexpectedly accepted")
	}
}

func TestEnqueueReconcileEnforcesGlobalDirtyDirectoryCapacity(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	item := createWorkerLibrary(t, store, "Archive", "archive")
	if _, err := store.db.ExecContext(ctx, `
		WITH RECURSIVE sequence(value) AS (
			SELECT 1
			UNION ALL
			SELECT value + 1
			FROM sequence
			WHERE value < ?
		)
		INSERT INTO catalog_reconcile_jobs(
			library_id, relative_dir_path, status, requested_revision,
			available_at_ms, attempt_count, created_at_ms, updated_at_ms
		)
		SELECT ?, printf('directory-%d', value), 'queued', 1,
		       1, 0, 1, 1
		FROM sequence
	`, scanner.MaxDirtyDirectories, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.EnqueueReconcile(
		ctx,
		item.ID,
		"over-capacity",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	); !errors.Is(err, scanner.ErrReconcileCapacity) {
		t.Fatalf("new reconciliation over capacity error = %v", err)
	}
	coalesced, err := store.EnqueueReconcile(
		ctx,
		item.ID,
		"directory-1",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	)
	if err != nil || coalesced.RequestedRevision != 2 {
		t.Fatalf("existing reconciliation at capacity = %#v err=%v", coalesced, err)
	}
}

func TestReconcileClaimLeaseAndExpiredRecovery(t *testing.T) {
	now := time.UnixMilli(5_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	item := createWorkerLibrary(t, store, "Archive", "archive")
	if _, err := store.EnqueueReconcile(
		ctx,
		item.ID,
		"",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.ClaimNextReconcile(ctx, time.Minute); err != nil || found {
		t.Fatalf("early claim found=%t err=%v", found, err)
	}
	now = now.Add(time.Second)
	claimed, found, err := store.ClaimNextReconcile(ctx, time.Minute)
	if err != nil || !found || claimed.Status != scanner.ReconcileRunning ||
		claimed.ClaimedRevision == nil || claimed.AttemptCount != 1 {
		t.Fatalf("claimed reconciliation = %#v found=%t err=%v", claimed, found, err)
	}
	refreshed, err := store.RefreshReconcileLease(ctx, claimed, 2*time.Minute)
	if err != nil || refreshed.LeaseExpiresAtMS == nil ||
		*refreshed.LeaseExpiresAtMS != now.Add(2*time.Minute).UnixMilli() {
		t.Fatalf("refreshed reconciliation = %#v err=%v", refreshed, err)
	}
	now = now.Add(3 * time.Minute)
	summary, err := store.RecoverExpiredReconciles(ctx)
	if err != nil || summary.Requeued != 1 || summary.Interrupted != 0 {
		t.Fatalf("recovery summary = %#v err=%v", summary, err)
	}
}

func TestReconcileClaimsAtMostOneJobPerLibraryAndUsesOtherLibraries(
	t *testing.T,
) {
	now := time.UnixMilli(5_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	firstLibrary := createWorkerLibrary(t, store, "Archive", "archive")
	secondLibrary := createWorkerLibrary(t, store, "Family", "family")
	for _, target := range []struct {
		libraryID int64
		path      string
	}{
		{firstLibrary.ID, "a"},
		{firstLibrary.ID, "b"},
		{secondLibrary.ID, "c"},
	} {
		if _, err := store.EnqueueReconcile(
			ctx,
			target.libraryID,
			target.path,
			scanner.ReconcileDebounce,
			scanner.ReconcileMaximumDebounce,
		); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(time.Second)
	first, found, err := store.ClaimNextReconcile(ctx, time.Minute)
	if err != nil || !found || first.LibraryID != firstLibrary.ID {
		t.Fatalf("first claim = %#v found=%t err=%v", first, found, err)
	}
	second, found, err := store.ClaimNextReconcile(ctx, time.Minute)
	if err != nil || !found || second.LibraryID != secondLibrary.ID {
		t.Fatalf("second claim = %#v found=%t err=%v", second, found, err)
	}
	if third, found, err := store.ClaimNextReconcile(
		ctx,
		time.Minute,
	); err != nil || found {
		t.Fatalf("third concurrent claim = %#v found=%t err=%v", third, found, err)
	}
}

func TestCommitDirectoryReconcileReplacesOnlyConfirmedDirectChildren(t *testing.T) {
	now := time.UnixMilli(10_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	libraryID := initializeReconcileCatalog(t, store)

	var globalBefore, libraryBefore int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT
		    (SELECT content_revision FROM catalog_search_state WHERE singleton_key = 1),
		    (SELECT content_revision FROM libraries WHERE id = ?)
	`, libraryID).Scan(&globalBefore, &libraryBefore); err != nil {
		t.Fatal(err)
	}

	if _, err := store.EnqueueReconcile(
		ctx,
		libraryID,
		"album",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	job, found, err := store.ClaimNextReconcile(ctx, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim reconciliation found=%t err=%v", found, err)
	}
	result, err := store.CommitDirectoryReconcile(ctx, job, []scanner.CatalogEntry{
		{
			Kind:         scanner.CatalogEntryDirectory,
			RelativePath: "album/new-folder",
			ParentPath:   "album",
			Name:         "new-folder",
			MTimeNS:      11,
		},
		{
			Kind:         scanner.CatalogEntryAsset,
			RelativePath: "album/new.jpg",
			ParentPath:   "album",
			Name:         "new.jpg",
			MTimeNS:      12,
			SizeBytes:    42,
			AssetKind:    scanner.AssetKindImage,
			MediaFormat:  scanner.MediaFormatJPEG,
			MIMEType:     "image/jpeg",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Requeued ||
		len(result.NewDirectories) != 1 ||
		result.NewDirectories[0] != "album/new-folder" {
		t.Fatalf("commit result = %#v", result)
	}

	var oldAssets, newAssets, childDirectories int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT
		    (SELECT count(*) FROM assets
		     WHERE library_id = ? AND relative_path = 'album/old.jpg'),
		    (SELECT count(*) FROM assets
		     WHERE library_id = ? AND relative_path = 'album/new.jpg'),
		    (SELECT count(*) FROM directories
		     WHERE library_id = ? AND relative_path = 'album/new-folder')
	`, libraryID, libraryID, libraryID).Scan(
		&oldAssets,
		&newAssets,
		&childDirectories,
	); err != nil {
		t.Fatal(err)
	}
	if oldAssets != 0 || newAssets != 1 || childDirectories != 1 {
		t.Fatalf(
			"reconciled rows = old %d new %d child %d",
			oldAssets,
			newAssets,
			childDirectories,
		)
	}
	var (
		rootDirect, rootRecursive   int64
		albumDirect, albumRecursive int64
		globalAfter, libraryAfter   int64
		status                      string
	)
	if err := store.db.QueryRowContext(ctx, `
		SELECT
		    root.direct_asset_count, root.recursive_asset_count,
		    album.direct_asset_count, album.recursive_asset_count,
		    state.content_revision, library.content_revision,
		    library.automatic_discovery_status
		FROM directories AS root
		JOIN directories AS album
		  ON album.library_id = root.library_id
		 AND album.relative_path = 'album'
		JOIN libraries AS library ON library.id = root.library_id
		CROSS JOIN catalog_search_state AS state
		WHERE root.library_id = ? AND root.relative_path = ''
		  AND state.singleton_key = 1
	`, libraryID).Scan(
		&rootDirect,
		&rootRecursive,
		&albumDirect,
		&albumRecursive,
		&globalAfter,
		&libraryAfter,
		&status,
	); err != nil {
		t.Fatal(err)
	}
	if rootDirect != 0 || rootRecursive != 1 ||
		albumDirect != 1 || albumRecursive != 1 ||
		globalAfter != globalBefore+1 ||
		libraryAfter != libraryBefore+1 ||
		status != "active" {
		t.Fatalf(
			"counts/revisions = root %d/%d album %d/%d global %d library %d status %q",
			rootDirect,
			rootRecursive,
			albumDirect,
			albumRecursive,
			globalAfter,
			libraryAfter,
			status,
		)
	}
}

func TestCommitDirectoryReconcileKeepsNewerRunningEventQueued(t *testing.T) {
	now := time.UnixMilli(20_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	libraryID := initializeReconcileCatalog(t, store)
	if _, err := store.EnqueueReconcile(
		ctx,
		libraryID,
		"album",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	job, found, err := store.ClaimNextReconcile(ctx, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim reconciliation found=%t err=%v", found, err)
	}
	if _, err := store.EnqueueReconcile(
		ctx,
		libraryID,
		"album",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	); err != nil {
		t.Fatal(err)
	}
	result, err := store.CommitDirectoryReconcile(ctx, job, []scanner.CatalogEntry{
		{
			Kind:         scanner.CatalogEntryAsset,
			RelativePath: "album/old.jpg",
			ParentPath:   "album",
			Name:         "old.jpg",
			MTimeNS:      2,
			SizeBytes:    1,
			AssetKind:    scanner.AssetKindImage,
			MediaFormat:  scanner.MediaFormatJPEG,
			MIMEType:     "image/jpeg",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Requeued {
		t.Fatal("new event during running reconciliation was absorbed")
	}
	var status string
	var requested int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT status, requested_revision
		FROM catalog_reconcile_jobs
		WHERE id = ?
	`, job.ID).Scan(&status, &requested); err != nil {
		t.Fatal(err)
	}
	if status != "queued" || requested != 2 {
		t.Fatalf("requeued watermark = %q/%d, want queued/2", status, requested)
	}
}

func initializeReconcileCatalog(t *testing.T, store *Store) int64 {
	t.Helper()
	ctx := context.Background()
	item := createWorkerLibrary(t, store, "Archive", "archive")
	admitted, err := store.AdmitFullScan(ctx, item.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	run, found, err := store.ClaimNextFullScan(ctx, time.Minute)
	if err != nil || !found || run.ID != admitted.Run.ID {
		t.Fatalf("claim initial scan = %#v found=%t err=%v", run, found, err)
	}
	if err := store.UpsertCatalogBatch(ctx, run.ID, []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind:         scanner.CatalogEntryDirectory,
			RelativePath: "album",
			ParentPath:   "",
			Name:         "album",
			MTimeNS:      1,
		},
		{
			Kind:         scanner.CatalogEntryAsset,
			RelativePath: "album/old.jpg",
			ParentPath:   "album",
			Name:         "old.jpg",
			MTimeNS:      2,
			SizeBytes:    1,
			AssetKind:    scanner.AssetKindImage,
			MediaFormat:  scanner.MediaFormatJPEG,
			MIMEType:     "image/jpeg",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}
	return item.ID
}
