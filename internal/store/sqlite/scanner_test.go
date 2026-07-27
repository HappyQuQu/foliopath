package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func createTestLibrary(t *testing.T, store *Store) library.Library {
	t.Helper()
	service, err := library.NewService(store)
	if err != nil {
		t.Fatalf("library.NewService() error = %v", err)
	}
	created, err := service.Create(context.Background(), "Library", "photos")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return created
}

func catalogFixture(assetPath string) []scanner.CatalogEntry {
	entries := []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "album",
			ParentPath: "", Name: "album", MTimeNS: 1,
		},
	}
	if assetPath != "" {
		entries = append(entries, scanner.CatalogEntry{
			Kind: scanner.CatalogEntryAsset, RelativePath: assetPath,
			ParentPath: "album", Name: "photo.jpg", MTimeNS: 2,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 12,
		})
	}
	return entries
}

func countCatalog(t *testing.T, store *Store, libraryID int64) (directories, assets int64) {
	t.Helper()
	ctx := context.Background()
	if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM directories WHERE library_id = ?`, libraryID).Scan(&directories); err != nil {
		t.Fatalf("count directories: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM assets WHERE library_id = ?`, libraryID).Scan(&assets); err != nil {
		t.Fatalf("count assets: %v", err)
	}
	return directories, assets
}

func TestScanAdmissionIsDurableCoalescedAndAllowsOfflineRetry(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	record := createTestLibrary(t, store)

	first, err := store.AdmitFullScan(ctx, record.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatalf("AdmitFullScan() error = %v", err)
	}
	if first.Coalesced || first.Run.Status != scanner.RunStatusQueued ||
		first.Run.Phase != "queued" || first.Run.Generation != 1 {
		t.Fatalf("first admission = %#v", first)
	}
	replayed, err := store.AdmitFullScan(ctx, record.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatalf("coalesced AdmitFullScan() error = %v", err)
	}
	if !replayed.Coalesced || replayed.Run.ID != first.Run.ID {
		t.Fatalf("coalesced admission = %#v, first = %#v", replayed, first)
	}

	if _, err := store.db.ExecContext(ctx, `
        UPDATE scan_runs
        SET status = 'offline', phase = 'completed', finished_at_ms = created_at_ms,
            error_code = 'library_root_unavailable', revision = revision + 1
        WHERE id = ?`,
		first.Run.ID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx,
		`UPDATE libraries SET status = 'offline' WHERE id = ?`,
		record.ID,
	); err != nil {
		t.Fatal(err)
	}
	retry, err := store.AdmitFullScan(ctx, record.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatalf("offline retry admission error = %v", err)
	}
	if retry.Coalesced || retry.Run.Generation != 2 ||
		retry.Run.Status != scanner.RunStatusQueued {
		t.Fatalf("offline retry = %#v", retry)
	}

	if _, err := store.db.ExecContext(ctx, `
        UPDATE scan_runs
        SET status = 'cancelled', phase = 'completed', finished_at_ms = created_at_ms
        WHERE id = ?`,
		retry.Run.ID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO library_removals(
            library_id, library_name, status, created_at_ms
        ) VALUES (?, ?, 'queued', 1000)`,
		record.ID, record.Name,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdmitFullScan(ctx, record.ID, scanner.TriggerManual); !errors.Is(
		err,
		scanner.ErrAdmissionConflict,
	) {
		t.Fatalf("admission during removal error = %v", err)
	}
}

func TestQueuedScanReservesLibraryAndHasNoStartedTime(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	result, err := store.db.ExecContext(ctx, `
		INSERT INTO scan_runs(
			library_id, generation, trigger_kind, status, created_at_ms, started_at_ms
		)
		VALUES (?, 1, 'manual', 'queued', 1234, NULL)`, libraryRecord.ID)
	if err != nil {
		t.Fatalf("insert queued scan: %v", err)
	}
	runID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read queued scan id: %v", err)
	}
	run, err := store.GetScanRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetScanRun(queued) error = %v", err)
	}
	if run.Status != scanner.RunStatusQueued || run.CreatedAtMS != 1234 || run.StartedAtMS != nil {
		t.Fatalf("queued run = %#v, want queued with createdAt and no startedAt", run)
	}
	if _, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual); !errors.Is(err, scanner.ErrScanActive) {
		t.Fatalf("BeginFullScan with queued admission error = %v, want ErrScanActive", err)
	}
}

func TestAnimatedAssetKindPersists(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	run, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatalf("BeginFullScan() error = %v", err)
	}
	entries := catalogFixture("")
	entries = append(entries, scanner.CatalogEntry{
		Kind: scanner.CatalogEntryAsset, RelativePath: "album/animation.gif",
		ParentPath: "album", Name: "animation.gif", MTimeNS: 2,
		AssetKind: scanner.AssetKindAnimated, MediaFormat: scanner.MediaFormatGIF,
		MIMEType: "image/gif", SizeBytes: 12,
	})
	if err := store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
		t.Fatalf("UpsertCatalogBatch() error = %v", err)
	}
	if _, err := store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
		t.Fatalf("CompleteFullScan() error = %v", err)
	}
	var kind string
	if err := store.db.QueryRowContext(ctx, `
		SELECT kind
		FROM assets
		WHERE library_id = ? AND relative_path = 'album/animation.gif'`,
		libraryRecord.ID,
	).Scan(&kind); err != nil {
		t.Fatalf("read animated asset kind: %v", err)
	}
	if kind != string(scanner.AssetKindAnimated) {
		t.Fatalf("animated asset kind = %q, want %q", kind, scanner.AssetKindAnimated)
	}
}

func TestCatalogForeignKeysRejectCrossLibraryRelationships(t *testing.T) {
	store, _ := openTestStore(t)
	libraries, err := library.NewService(store)
	if err != nil {
		t.Fatalf("library.NewService() error = %v", err)
	}
	ctx := context.Background()
	firstLibrary, err := libraries.Create(ctx, "First", "first")
	if err != nil {
		t.Fatalf("create first library: %v", err)
	}
	secondLibrary, err := libraries.Create(ctx, "Second", "second")
	if err != nil {
		t.Fatalf("create second library: %v", err)
	}
	firstRun, err := store.BeginFullScan(ctx, firstLibrary.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatalf("begin first scan: %v", err)
	}
	secondRun, err := store.BeginFullScan(ctx, secondLibrary.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatalf("begin second scan: %v", err)
	}
	if err := store.UpsertCatalogBatch(ctx, firstRun.ID, []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}}); err != nil {
		t.Fatalf("insert first root: %v", err)
	}
	if err := store.UpsertCatalogBatch(ctx, secondRun.ID, []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}}); err != nil {
		t.Fatalf("insert second root: %v", err)
	}
	var firstRootID, secondRootID int64
	if err := store.db.QueryRowContext(ctx,
		`SELECT id FROM directories WHERE library_id = ? AND relative_path = ''`,
		firstLibrary.ID,
	).Scan(&firstRootID); err != nil {
		t.Fatalf("read first root: %v", err)
	}
	if err := store.db.QueryRowContext(ctx,
		`SELECT id FROM directories WHERE library_id = ? AND relative_path = ''`,
		secondLibrary.ID,
	).Scan(&secondRootID); err != nil {
		t.Fatalf("read second root: %v", err)
	}
	if _, err := store.db.ExecContext(ctx,
		`UPDATE directories SET parent_id = ? WHERE id = ?`,
		secondRootID, firstRootID,
	); err == nil {
		t.Fatal("cross-library directory parent was accepted")
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO assets(
			library_id, directory_id, relative_path, name, kind, media_format,
			mime_type, size_bytes, mtime_ns, last_seen_generation
		)
		VALUES (?, ?, 'cross.jpg', 'cross.jpg', 'image', 'jpeg',
			'image/jpeg', 1, 1, 1)`,
		firstLibrary.ID, secondRootID,
	); err == nil {
		t.Fatal("cross-library asset directory was accepted")
	}
}

func TestFailedScanPreservesOldIndexAndSuccessfulScanCleansStale(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	first, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatalf("BeginFullScan(first) error = %v", err)
	}
	if first.Generation != 1 {
		t.Fatalf("first generation = %d, want 1", first.Generation)
	}
	if err := store.UpsertCatalogBatch(ctx, first.ID, catalogFixture("album/photo.jpg")); err != nil {
		t.Fatalf("UpsertCatalogBatch(first) error = %v", err)
	}
	if _, err := store.CompleteFullScan(ctx, first.ID, scanner.SkipCounts{}); err != nil {
		t.Fatalf("CompleteFullScan(first) error = %v", err)
	}
	var rootRecursive, albumDirect, albumRecursive int64
	if err := store.db.QueryRowContext(ctx, `
        SELECT recursive_asset_count FROM directories
        WHERE library_id = ? AND relative_path = ''`, libraryRecord.ID).Scan(&rootRecursive); err != nil {
		t.Fatalf("read root recursive count: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `
        SELECT direct_asset_count, recursive_asset_count FROM directories
        WHERE library_id = ? AND relative_path = 'album'`, libraryRecord.ID).Scan(&albumDirect, &albumRecursive); err != nil {
		t.Fatalf("read album counts: %v", err)
	}
	if rootRecursive != 1 || albumDirect != 1 || albumRecursive != 1 {
		t.Fatalf("directory counts root=%d album=(%d,%d), want 1 and (1,1)", rootRecursive, albumDirect, albumRecursive)
	}

	second, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatalf("BeginFullScan(second) error = %v", err)
	}
	if second.Generation != 2 {
		t.Fatalf("second generation = %d, want 2", second.Generation)
	}
	if err := store.UpsertCatalogBatch(ctx, second.ID, catalogFixture("")); err != nil {
		t.Fatalf("UpsertCatalogBatch(second) error = %v", err)
	}
	failed, err := store.FailFullScan(ctx, second.ID, scanner.SkipCounts{Files: 3}, "walk_failed")
	if err != nil {
		t.Fatalf("FailFullScan() error = %v", err)
	}
	if failed.Status != scanner.RunStatusFailed ||
		failed.SkippedCount != 3 ||
		failed.SkippedDirectories != 0 ||
		failed.SkippedFiles != 3 {
		t.Fatalf("failed run = %#v", failed)
	}
	_, assets := countCatalog(t, store, libraryRecord.ID)
	if assets != 1 {
		t.Fatalf("assets after failed scan = %d, want 1", assets)
	}
	current, err := store.GetLibrary(ctx, libraryRecord.ID)
	if err != nil {
		t.Fatalf("GetLibrary(after failure) error = %v", err)
	}
	if current.CurrentGeneration != 1 {
		t.Fatalf("current generation after failure = %d, want 1", current.CurrentGeneration)
	}

	third, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatalf("BeginFullScan(third) error = %v", err)
	}
	if third.Generation != 3 {
		t.Fatalf("third generation = %d, want 3", third.Generation)
	}
	if err := store.UpsertCatalogBatch(ctx, third.ID, catalogFixture("")); err != nil {
		t.Fatalf("UpsertCatalogBatch(third) error = %v", err)
	}
	completed, err := store.CompleteFullScan(ctx, third.ID, scanner.SkipCounts{})
	if err != nil {
		t.Fatalf("CompleteFullScan(third) error = %v", err)
	}
	if completed.Status != scanner.RunStatusSucceeded {
		t.Fatalf("completed status = %q", completed.Status)
	}
	_, assets = countCatalog(t, store, libraryRecord.ID)
	if assets != 0 {
		t.Fatalf("assets after successful stale cleanup = %d, want 0", assets)
	}
	current, err = store.GetLibrary(ctx, libraryRecord.ID)
	if err != nil {
		t.Fatalf("GetLibrary(after completion) error = %v", err)
	}
	if current.CurrentGeneration != 3 {
		t.Fatalf("current generation after completion = %d, want 3", current.CurrentGeneration)
	}
}

func TestCompleteFullScanRollsBackCleanupWhenFinalCommitFails(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	first, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, first.ID, catalogFixture("album/old.jpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(ctx, first.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}

	second, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, second.ID, catalogFixture("")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		CREATE TRIGGER reject_generation_two
		BEFORE UPDATE OF current_generation ON libraries
		WHEN NEW.current_generation = 2
		BEGIN
			SELECT RAISE(ABORT, 'injected finalization failure');
		END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	if _, err := store.CompleteFullScan(ctx, second.ID, scanner.SkipCounts{}); err == nil {
		t.Fatal("CompleteFullScan() succeeded despite injected finalization failure")
	}
	_, assets := countCatalog(t, store, libraryRecord.ID)
	if assets != 1 {
		t.Fatalf("assets after rolled-back finalization = %d, want old row preserved", assets)
	}
	current, err := store.GetLibrary(ctx, libraryRecord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentGeneration != 1 {
		t.Fatalf("current generation after rolled-back finalization = %d, want 1", current.CurrentGeneration)
	}
	run, err := store.GetScanRun(ctx, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != scanner.RunStatusRunning {
		t.Fatalf("run status after rolled-back finalization = %q, want running for recovery", run.Status)
	}
}

func TestCompleteFullScanRequiresRootMarkerBeforeCleanup(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	first, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, first.ID, catalogFixture("album/old.jpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(ctx, first.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}

	second, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(ctx, second.ID, scanner.SkipCounts{}); !errors.Is(err, scanner.ErrInvalidEntry) {
		t.Fatalf("CompleteFullScan without root marker error = %v, want ErrInvalidEntry", err)
	}
	_, assets := countCatalog(t, store, libraryRecord.ID)
	if assets != 1 {
		t.Fatalf("assets after rejected rootless completion = %d, want old row preserved", assets)
	}
	current, err := store.GetLibrary(ctx, libraryRecord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentGeneration != 1 {
		t.Fatalf("current generation after rejected rootless completion = %d, want 1", current.CurrentGeneration)
	}
}

func TestCompleteFullScanRollsUpDeepChain(t *testing.T) {
	const depth = 128

	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	run, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	directories := make([]scanner.CatalogEntry, 0, depth+1)
	directories = append(directories, scanner.CatalogEntry{Kind: scanner.CatalogEntryDirectory})
	paths := make([]string, 0, depth)
	parent := ""
	for index := 0; index < depth; index++ {
		name := fmt.Sprintf("d%03d", index)
		relativePath := name
		if parent != "" {
			relativePath = parent + "/" + name
		}
		directories = append(directories, scanner.CatalogEntry{
			Kind:         scanner.CatalogEntryDirectory,
			RelativePath: relativePath,
			ParentPath:   parent,
			Name:         name,
		})
		paths = append(paths, relativePath)
		parent = relativePath
	}
	upsertTestEntries(t, store, run.ID, directories)

	assets := make([]scanner.CatalogEntry, 0, depth+1)
	assets = append(assets, scanner.CatalogEntry{
		Kind:         scanner.CatalogEntryAsset,
		RelativePath: "root.jpg",
		Name:         "root.jpg",
		AssetKind:    scanner.AssetKindImage,
		MediaFormat:  scanner.MediaFormatJPEG,
		MIMEType:     "image/jpeg",
		SizeBytes:    4,
	})
	for index, directoryPath := range paths {
		assets = append(assets, scanner.CatalogEntry{
			Kind:         scanner.CatalogEntryAsset,
			RelativePath: directoryPath + "/asset.jpg",
			ParentPath:   directoryPath,
			Name:         fmt.Sprintf("asset-%03d.jpg", index),
			AssetKind:    scanner.AssetKindImage,
			MediaFormat:  scanner.MediaFormatJPEG,
			MIMEType:     "image/jpeg",
			SizeBytes:    4,
		})
	}
	upsertTestEntries(t, store, run.ID, assets)

	if _, err := store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
		t.Fatalf("CompleteFullScan() error = %v", err)
	}
	assertDirectoryCounts(t, store, libraryRecord.ID, "", 1, depth+1)
	for index, directoryPath := range paths {
		assertDirectoryCounts(t, store, libraryRecord.ID, directoryPath, 1, depth-index)
	}
}

func TestCompleteFullScanRejectsDirectoryCycleAndRollsBack(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	run, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	entries := []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "a",
			Name: "a",
		},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "a/b",
			ParentPath: "a", Name: "b",
		},
	}
	if err := store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
		t.Fatal(err)
	}
	var aID, bID int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT id FROM directories WHERE library_id = ? AND relative_path = 'a'`,
		libraryRecord.ID).Scan(&aID); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `
		SELECT id FROM directories WHERE library_id = ? AND relative_path = 'a/b'`,
		libraryRecord.ID).Scan(&bID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE directories SET parent_id = ? WHERE id = ?`, bID, aID); err != nil {
		t.Fatalf("inject same-library cycle: %v", err)
	}

	if _, err := store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); !errors.Is(err, scanner.ErrInvalidEntry) {
		t.Fatalf("CompleteFullScan() error = %v, want ErrInvalidEntry", err)
	}
	persisted, err := store.GetScanRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != scanner.RunStatusRunning {
		t.Fatalf("run status = %q, want running after rolled-back finalization", persisted.Status)
	}
	directoriesCount, _ := countCatalog(t, store, libraryRecord.ID)
	if directoriesCount != 3 {
		t.Fatalf("directories after rejected cycle = %d, want 3", directoriesCount)
	}
}

func TestCompleteFullScanRejectsCrossLibraryRelationshipBeforeCleanup(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{name: "directory"},
		{name: "asset"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _ := openTestStore(t)
			ctx := context.Background()
			service, err := library.NewService(store)
			if err != nil {
				t.Fatal(err)
			}
			firstLibrary, err := service.Create(ctx, "First", "first")
			if err != nil {
				t.Fatal(err)
			}
			secondLibrary, err := service.Create(ctx, "Second", "second")
			if err != nil {
				t.Fatal(err)
			}

			firstRun, err := store.BeginFullScan(ctx, firstLibrary.ID, scanner.TriggerCreation)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.UpsertCatalogBatch(ctx, firstRun.ID, catalogFixture("album/old.jpg")); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CompleteFullScan(ctx, firstRun.ID, scanner.SkipCounts{}); err != nil {
				t.Fatal(err)
			}
			secondRun, err := store.BeginFullScan(ctx, secondLibrary.ID, scanner.TriggerCreation)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.UpsertCatalogBatch(ctx, secondRun.ID, catalogFixture("album/second.jpg")); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CompleteFullScan(ctx, secondRun.ID, scanner.SkipCounts{}); err != nil {
				t.Fatal(err)
			}

			var staleDirectoryID, secondChildID, secondAssetID int64
			if err := store.db.QueryRowContext(ctx, `
				SELECT id FROM directories WHERE library_id = ? AND relative_path = 'album'`,
				firstLibrary.ID).Scan(&staleDirectoryID); err != nil {
				t.Fatal(err)
			}
			if err := store.db.QueryRowContext(ctx, `
				SELECT id FROM directories WHERE library_id = ? AND relative_path = 'album'`,
				secondLibrary.ID).Scan(&secondChildID); err != nil {
				t.Fatal(err)
			}
			if err := store.db.QueryRowContext(ctx, `
				SELECT id FROM assets WHERE library_id = ?`,
				secondLibrary.ID).Scan(&secondAssetID); err != nil {
				t.Fatal(err)
			}

			connection, err := store.db.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := connection.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
				t.Fatal(err)
			}
			switch test.name {
			case "directory":
				_, err = connection.ExecContext(ctx, `
					UPDATE directories SET parent_id = ? WHERE id = ?`,
					staleDirectoryID, secondChildID)
			case "asset":
				_, err = connection.ExecContext(ctx, `
					UPDATE assets SET directory_id = ? WHERE id = ?`,
					staleDirectoryID, secondAssetID)
			}
			if err != nil {
				t.Fatalf("inject cross-library relationship: %v", err)
			}
			if _, err := connection.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
				t.Fatal(err)
			}
			if err := connection.Close(); err != nil {
				t.Fatal(err)
			}

			rescan, err := store.BeginFullScan(ctx, firstLibrary.ID, scanner.TriggerManual)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.UpsertCatalogBatch(ctx, rescan.ID, []scanner.CatalogEntry{
				{Kind: scanner.CatalogEntryDirectory},
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CompleteFullScan(ctx, rescan.ID, scanner.SkipCounts{}); !errors.Is(err, scanner.ErrInvalidEntry) {
				t.Fatalf("CompleteFullScan() error = %v, want ErrInvalidEntry", err)
			}

			var firstStaleRows, secondChildRows, secondAssetRows int
			if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM directories WHERE id = ?`,
				staleDirectoryID).Scan(&firstStaleRows); err != nil {
				t.Fatal(err)
			}
			if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM directories WHERE id = ?`,
				secondChildID).Scan(&secondChildRows); err != nil {
				t.Fatal(err)
			}
			if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM assets WHERE id = ?`,
				secondAssetID).Scan(&secondAssetRows); err != nil {
				t.Fatal(err)
			}
			if firstStaleRows != 1 || secondChildRows != 1 || secondAssetRows != 1 {
				t.Fatalf("rows after rejected cleanup = stale:%d child:%d asset:%d, want all 1",
					firstStaleRows, secondChildRows, secondAssetRows)
			}
		})
	}
}

func TestCompleteFullScanRejectsCurrentRowsAttachedToStaleDirectory(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{name: "directory"},
		{name: "asset"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _ := openTestStore(t)
			libraryRecord := createTestLibrary(t, store)
			ctx := context.Background()

			firstRun, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.UpsertCatalogBatch(ctx, firstRun.ID, catalogFixture("album/old.jpg")); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CompleteFullScan(ctx, firstRun.ID, scanner.SkipCounts{}); err != nil {
				t.Fatal(err)
			}
			var staleDirectoryID int64
			if err := store.db.QueryRowContext(ctx, `
				SELECT id
				FROM directories
				WHERE library_id = ? AND relative_path = 'album'`,
				libraryRecord.ID,
			).Scan(&staleDirectoryID); err != nil {
				t.Fatal(err)
			}

			secondRun, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
			if err != nil {
				t.Fatal(err)
			}
			entries := []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}}
			switch test.name {
			case "directory":
				entries = append(entries, scanner.CatalogEntry{
					Kind: scanner.CatalogEntryDirectory, RelativePath: "fresh",
					ParentPath: "", Name: "fresh",
				})
			case "asset":
				entries = append(entries, scanner.CatalogEntry{
					Kind: scanner.CatalogEntryAsset, RelativePath: "fresh.jpg",
					ParentPath: "", Name: "fresh.jpg",
					AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
					MIMEType: "image/jpeg", SizeBytes: 1,
				})
			}
			if err := store.UpsertCatalogBatch(ctx, secondRun.ID, entries); err != nil {
				t.Fatal(err)
			}

			switch test.name {
			case "directory":
				_, err = store.db.ExecContext(ctx, `
					UPDATE directories
					SET parent_id = ?
					WHERE library_id = ? AND relative_path = 'fresh'`,
					staleDirectoryID, libraryRecord.ID)
			case "asset":
				_, err = store.db.ExecContext(ctx, `
					UPDATE assets
					SET directory_id = ?
					WHERE library_id = ? AND relative_path = 'fresh.jpg'`,
					staleDirectoryID, libraryRecord.ID)
			}
			if err != nil {
				t.Fatalf("inject current-to-stale relationship: %v", err)
			}

			if _, err := store.CompleteFullScan(ctx, secondRun.ID, scanner.SkipCounts{}); !errors.Is(err, scanner.ErrInvalidEntry) {
				t.Fatalf("CompleteFullScan() error = %v, want ErrInvalidEntry", err)
			}
			persisted, err := store.GetScanRun(ctx, secondRun.ID)
			if err != nil {
				t.Fatal(err)
			}
			if persisted.Status != scanner.RunStatusRunning {
				t.Fatalf("run status = %q, want running after rolled-back finalization", persisted.Status)
			}

			var staleRows, currentRows int
			if err := store.db.QueryRowContext(ctx,
				`SELECT count(*) FROM directories WHERE id = ?`,
				staleDirectoryID,
			).Scan(&staleRows); err != nil {
				t.Fatal(err)
			}
			switch test.name {
			case "directory":
				err = store.db.QueryRowContext(ctx, `
					SELECT count(*)
					FROM directories
					WHERE library_id = ? AND relative_path = 'fresh'`,
					libraryRecord.ID,
				).Scan(&currentRows)
			case "asset":
				err = store.db.QueryRowContext(ctx, `
					SELECT count(*)
					FROM assets
					WHERE library_id = ? AND relative_path = 'fresh.jpg'`,
					libraryRecord.ID,
				).Scan(&currentRows)
			}
			if err != nil {
				t.Fatal(err)
			}
			if staleRows != 1 || currentRows != 1 {
				t.Fatalf("rows after rejected cleanup = stale:%d current:%d, want both 1",
					staleRows, currentRows)
			}
		})
	}
}

func upsertTestEntries(t *testing.T, store *Store, runID int64, entries []scanner.CatalogEntry) {
	t.Helper()
	for len(entries) > 0 {
		batchSize := min(store.maxBatchSize, len(entries))
		if err := store.UpsertCatalogBatch(context.Background(), runID, entries[:batchSize]); err != nil {
			t.Fatalf("UpsertCatalogBatch() error = %v", err)
		}
		entries = entries[batchSize:]
	}
}

func assertDirectoryCounts(
	t *testing.T,
	store *Store,
	libraryID int64,
	relativePath string,
	wantDirect int,
	wantRecursive int,
) {
	t.Helper()
	var direct, recursive int
	if err := store.db.QueryRowContext(context.Background(), `
		SELECT direct_asset_count, recursive_asset_count
		FROM directories
		WHERE library_id = ? AND relative_path = ?`,
		libraryID, relativePath).Scan(&direct, &recursive); err != nil {
		t.Fatalf("read directory %q counts: %v", relativePath, err)
	}
	if direct != wantDirect || recursive != wantRecursive {
		t.Fatalf("directory %q counts = (%d, %d), want (%d, %d)",
			relativePath, direct, recursive, wantDirect, wantRecursive)
	}
}

func TestOfflineAndInterruptedScansPreserveOldIndex(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	first, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, first.ID, catalogFixture("album/old.jpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(ctx, first.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}

	offlineRun, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerScheduled)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, offlineRun.ID, catalogFixture("")); err != nil {
		t.Fatal(err)
	}
	offlineRun, err = store.OfflineFullScan(ctx, offlineRun.ID, scanner.SkipCounts{Files: 2}, "library_offline")
	if err != nil {
		t.Fatalf("OfflineFullScan() error = %v", err)
	}
	if offlineRun.Status != scanner.RunStatusOffline {
		t.Fatalf("offline run status = %q", offlineRun.Status)
	}
	if offlineRun.SkippedCount != 2 ||
		offlineRun.SkippedDirectories != 0 ||
		offlineRun.SkippedFiles != 2 {
		t.Fatalf("offline skipped counters = %#v", offlineRun)
	}
	_, assets := countCatalog(t, store, libraryRecord.ID)
	if assets != 1 {
		t.Fatalf("assets after offline scan = %d, want 1", assets)
	}

	interruptedRun, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerStartup)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, interruptedRun.ID, catalogFixture("")); err != nil {
		t.Fatal(err)
	}
	count, err := store.InterruptActiveScans(ctx)
	if err != nil {
		t.Fatalf("InterruptActiveScans() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("interrupted scan count = %d, want 1", count)
	}
	interruptedRun, err = store.GetScanRun(ctx, interruptedRun.ID)
	if err != nil {
		t.Fatal(err)
	}
	if interruptedRun.Status != scanner.RunStatusInterrupted {
		t.Fatalf("interrupted run status = %q", interruptedRun.Status)
	}
	_, assets = countCatalog(t, store, libraryRecord.ID)
	if assets != 1 {
		t.Fatalf("assets after interrupted scan = %d, want 1", assets)
	}
	current, err := store.GetLibrary(ctx, libraryRecord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentGeneration != 1 {
		t.Fatalf("current generation after offline/interrupted scans = %d, want 1", current.CurrentGeneration)
	}
}

func TestRestartMarksRunningScanInterruptedWithoutCleanup(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "restart.db")
	firstStore, err := Open(ctx, filename, Options{MaxBatchSize: 16})
	if err != nil {
		t.Fatal(err)
	}
	libraryRecord := createTestLibrary(t, firstStore)

	first, err := firstStore.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	if err := firstStore.UpsertCatalogBatch(ctx, first.ID, catalogFixture("album/old.jpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := firstStore.CompleteFullScan(ctx, first.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}

	interrupted, err := firstStore.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if err := firstStore.UpsertCatalogBatch(ctx, interrupted.ID, catalogFixture("")); err != nil {
		t.Fatal(err)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("close before simulated restart: %v", err)
	}

	reopened, err := Open(ctx, filename, Options{MaxBatchSize: 16})
	if err != nil {
		t.Fatalf("reopen after simulated restart: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	count, err := reopened.InterruptActiveScans(ctx)
	if err != nil {
		t.Fatalf("InterruptActiveScans after reopen: %v", err)
	}
	if count != 1 {
		t.Fatalf("interrupted count after reopen = %d, want 1", count)
	}
	run, err := reopened.GetScanRun(ctx, interrupted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != scanner.RunStatusInterrupted {
		t.Fatalf("run status after reopen = %q, want interrupted", run.Status)
	}
	_, assets := countCatalog(t, reopened, libraryRecord.ID)
	if assets != 1 {
		t.Fatalf("assets after restart recovery = %d, want old row preserved", assets)
	}
	current, err := reopened.GetLibrary(ctx, libraryRecord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentGeneration != 1 {
		t.Fatalf("current generation after restart recovery = %d, want 1", current.CurrentGeneration)
	}
}

func TestCancelledScanPreservesOldAndSafelyCommittedNewRows(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	first, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, first.ID, catalogFixture("album/old.jpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(ctx, first.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}

	second, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, second.ID, catalogFixture("album/new.jpg")); err != nil {
		t.Fatal(err)
	}
	cancelled, err := store.CancelFullScan(ctx, second.ID, scanner.SkipCounts{Files: 1})
	if err != nil {
		t.Fatalf("CancelFullScan() error = %v", err)
	}
	if cancelled.Status != scanner.RunStatusCancelled {
		t.Fatalf("cancelled status = %q", cancelled.Status)
	}
	_, assets := countCatalog(t, store, libraryRecord.ID)
	if assets != 2 {
		t.Fatalf("assets after cancellation = %d, want old and new rows", assets)
	}
	current, err := store.GetLibrary(ctx, libraryRecord.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentGeneration != 1 {
		t.Fatalf("current generation after cancellation = %d, want 1", current.CurrentGeneration)
	}
}

func TestActiveScanConstraintWorksAcrossStoreInstances(t *testing.T) {
	firstStore, filename := openTestStore(t)
	secondStore, err := Open(context.Background(), filename, Options{MaxBatchSize: 16})
	if err != nil {
		t.Fatalf("Open(second store) error = %v", err)
	}
	t.Cleanup(func() { _ = secondStore.Close() })
	libraryRecord := createTestLibrary(t, firstStore)

	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, store := range []*Store{firstStore, secondStore} {
		wait.Add(1)
		go func(candidate *Store) {
			defer wait.Done()
			<-start
			_, err := candidate.BeginFullScan(context.Background(), libraryRecord.ID, scanner.TriggerManual)
			results <- err
		}(store)
	}
	close(start)
	wait.Wait()
	close(results)

	var succeeded, active int
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, scanner.ErrScanActive):
			active++
		default:
			t.Fatalf("BeginFullScan() unexpected error = %v", err)
		}
	}
	if succeeded != 1 || active != 1 {
		t.Fatalf("concurrent results: succeeded=%d active=%d", succeeded, active)
	}
}

func TestCompleteAndCancelHaveOneConsistentWinner(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	first, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, first.ID, catalogFixture("album/old.jpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(ctx, first.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}
	second, err := store.BeginFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(ctx, second.ID, catalogFixture("")); err != nil {
		t.Fatal(err)
	}

	type result struct {
		operation string
		run       scanner.ScanRun
		err       error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	go func() {
		<-start
		run, err := store.CompleteFullScan(ctx, second.ID, scanner.SkipCounts{})
		results <- result{operation: "complete", run: run, err: err}
	}()
	go func() {
		<-start
		run, err := store.CancelFullScan(ctx, second.ID, scanner.SkipCounts{})
		results <- result{operation: "cancel", run: run, err: err}
	}()
	close(start)
	firstResult := <-results
	secondResult := <-results

	var winner result
	switch {
	case firstResult.err == nil && errors.Is(secondResult.err, scanner.ErrScanRunNotActive):
		winner = firstResult
	case secondResult.err == nil && errors.Is(firstResult.err, scanner.ErrScanRunNotActive):
		winner = secondResult
	default:
		t.Fatalf("race results = %#v and %#v; want one success and one ErrScanRunNotActive",
			firstResult, secondResult)
	}

	current, err := store.GetLibrary(ctx, libraryRecord.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, assets := countCatalog(t, store, libraryRecord.ID)
	switch winner.operation {
	case "complete":
		if winner.run.Status != scanner.RunStatusSucceeded || current.CurrentGeneration != 2 || assets != 0 {
			t.Fatalf("complete winner left run=%q generation=%d assets=%d",
				winner.run.Status, current.CurrentGeneration, assets)
		}
	case "cancel":
		if winner.run.Status != scanner.RunStatusCancelled || current.CurrentGeneration != 1 || assets != 1 {
			t.Fatalf("cancel winner left run=%q generation=%d assets=%d",
				winner.run.Status, current.CurrentGeneration, assets)
		}
	default:
		t.Fatalf("unknown winner operation %q", winner.operation)
	}
}
