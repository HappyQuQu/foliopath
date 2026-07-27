package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

type testEnvironment struct {
	allowedPath string
	root        *files.Root
	store       *sqlitestore.Store
	inspector   *sql.DB
	libraries   *library.Service
	scans       *scanner.Service
	walker      scanner.Walker
}

func newTestEnvironment(t *testing.T) *testEnvironment {
	t.Helper()

	base := t.TempDir()
	allowedPath := filepath.Join(base, "library")
	dataPath := filepath.Join(base, "data")
	if err := os.MkdirAll(allowedPath, 0o755); err != nil {
		t.Fatalf("create allowed media root: %v", err)
	}
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		t.Fatalf("create data directory: %v", err)
	}

	root, err := files.OpenRoot(allowedPath)
	if err != nil {
		t.Fatalf("files.OpenRoot() error = %v", err)
	}
	databasePath := filepath.Join(dataPath, "foliopath.db")
	store, err := sqlitestore.Open(context.Background(), databasePath, sqlitestore.Options{
		BusyTimeout:        2 * time.Second,
		MaxOpenConnections: 4,
		MaxBatchSize:       16,
	})
	if err != nil {
		_ = root.Close()
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	inspector, err := sql.Open("sqlite", databasePath)
	if err != nil {
		_ = store.Close()
		_ = root.Close()
		t.Fatalf("open SQLite inspector: %v", err)
	}
	if err := inspector.PingContext(context.Background()); err != nil {
		_ = inspector.Close()
		_ = store.Close()
		_ = root.Close()
		t.Fatalf("ping SQLite inspector: %v", err)
	}
	info, err := os.Stat(databasePath)
	if err != nil {
		t.Fatalf("stat file-backed SQLite database: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("SQLite database mode = %v, want regular file", info.Mode())
	}

	libraryService, err := library.NewService(store)
	if err != nil {
		t.Fatalf("library.NewService() error = %v", err)
	}
	scanService, err := scanner.NewService(store, scanner.Config{
		// A single-entry batch makes the abort tests prove that already-safe
		// batches survive while stale rows are not removed.
		BatchSize:       1,
		FinalizeTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("scanner.NewService() error = %v", err)
	}
	filesystemWalker, err := files.NewScanWalker(root)
	if err != nil {
		t.Fatalf("files.NewScanWalker() error = %v", err)
	}

	t.Cleanup(func() {
		if err := inspector.Close(); err != nil {
			t.Errorf("close SQLite inspector: %v", err)
		}
		if err := store.Close(); err != nil {
			t.Errorf("close SQLite store: %v", err)
		}
		if err := root.Close(); err != nil {
			t.Errorf("close media root: %v", err)
		}
	})

	return &testEnvironment{
		allowedPath: allowedPath,
		root:        root,
		store:       store,
		inspector:   inspector,
		libraries:   libraryService,
		scans:       scanService,
		walker:      filesystemWalker,
	}
}

func (environment *testEnvironment) createLibrary(t *testing.T, name, relativeRoot string) library.Library {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(environment.allowedPath, filepath.FromSlash(relativeRoot)), 0o755); err != nil {
		t.Fatalf("create library root %q: %v", relativeRoot, err)
	}
	created, err := environment.libraries.Create(context.Background(), name, relativeRoot)
	if err != nil {
		t.Fatalf("create library %q: %v", name, err)
	}
	return created
}

func (environment *testEnvironment) runScan(
	t *testing.T,
	libraryRecord library.Library,
	trigger scanner.Trigger,
	walker scanner.Walker,
) scanner.ScanRun {
	t.Helper()
	if walker == nil {
		walker = environment.walker
	}
	run, err := environment.scans.RunFullScan(context.Background(), scanner.FullScanRequest{
		LibraryID: libraryRecord.ID,
		Trigger:   trigger,
		Walker:    walker,
	})
	if err != nil {
		t.Fatalf("RunFullScan(%q) error = %v", libraryRecord.Name, err)
	}
	return run
}

type hookedWalker struct {
	scanner.Walker
	afterVisit  func(scanner.WalkEntry) error
	beforeCheck func() error
}

type abaWalker struct {
	scanner.Walker
	beforeWalk func() error
	afterWalk  func() error
}

func (walker *abaWalker) Walk(
	ctx context.Context,
	relativeRoot string,
	visit func(scanner.WalkEntry) (scanner.WalkDecision, error),
) error {
	if err := walker.beforeWalk(); err != nil {
		return err
	}
	walkErr := walker.Walker.Walk(ctx, relativeRoot, visit)
	return errors.Join(walkErr, walker.afterWalk())
}

func (walker *hookedWalker) Walk(
	ctx context.Context,
	relativeRoot string,
	visit func(scanner.WalkEntry) (scanner.WalkDecision, error),
) error {
	return walker.Walker.Walk(ctx, relativeRoot, func(entry scanner.WalkEntry) (scanner.WalkDecision, error) {
		decision, err := visit(entry)
		if err != nil {
			return decision, err
		}
		if walker.afterVisit != nil {
			if err := walker.afterVisit(entry); err != nil {
				return decision, err
			}
		}
		return decision, nil
	})
}

func (walker *hookedWalker) VerifyRoot(
	ctx context.Context,
	relativeRoot string,
	expected scanner.RootIdentity,
) error {
	if walker.beforeCheck != nil {
		if err := walker.beforeCheck(); err != nil {
			return err
		}
	}
	return walker.Walker.VerifyRoot(ctx, relativeRoot, expected)
}

type directoryCounts struct {
	direct     int64
	recursive  int64
	generation int64
}

type catalogSnapshot struct {
	directories map[string]directoryCounts
	assets      map[string]int64
}

func readCatalog(t *testing.T, database *sql.DB, libraryID int64) catalogSnapshot {
	t.Helper()
	ctx := context.Background()
	snapshot := catalogSnapshot{
		directories: make(map[string]directoryCounts),
		assets:      make(map[string]int64),
	}

	directories, err := database.QueryContext(ctx, `
        SELECT relative_path, direct_asset_count, recursive_asset_count, last_seen_generation
        FROM directories
        WHERE library_id = ?`, libraryID)
	if err != nil {
		t.Fatalf("query directories: %v", err)
	}
	for directories.Next() {
		var path string
		var counts directoryCounts
		if err := directories.Scan(&path, &counts.direct, &counts.recursive, &counts.generation); err != nil {
			_ = directories.Close()
			t.Fatalf("scan directory row: %v", err)
		}
		snapshot.directories[path] = counts
	}
	if err := directories.Err(); err != nil {
		_ = directories.Close()
		t.Fatalf("iterate directory rows: %v", err)
	}
	if err := directories.Close(); err != nil {
		t.Fatalf("close directory rows: %v", err)
	}

	assets, err := database.QueryContext(ctx, `
        SELECT relative_path, last_seen_generation
        FROM assets
        WHERE library_id = ?`, libraryID)
	if err != nil {
		t.Fatalf("query assets: %v", err)
	}
	for assets.Next() {
		var path string
		var generation int64
		if err := assets.Scan(&path, &generation); err != nil {
			_ = assets.Close()
			t.Fatalf("scan asset row: %v", err)
		}
		snapshot.assets[path] = generation
	}
	if err := assets.Err(); err != nil {
		_ = assets.Close()
		t.Fatalf("iterate asset rows: %v", err)
	}
	if err := assets.Close(); err != nil {
		t.Fatalf("close asset rows: %v", err)
	}
	return snapshot
}

func assertAssetPaths(t *testing.T, snapshot catalogSnapshot, expected ...string) {
	t.Helper()
	actual := make([]string, 0, len(snapshot.assets))
	for path := range snapshot.assets {
		actual = append(actual, path)
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("asset paths = %v, want %v", actual, expected)
	}
}

func assertDirectoryCounts(t *testing.T, snapshot catalogSnapshot, path string, direct, recursive int64) {
	t.Helper()
	counts, ok := snapshot.directories[path]
	if !ok {
		t.Fatalf("directory %q was not indexed", path)
	}
	if counts.direct != direct || counts.recursive != recursive {
		t.Fatalf("directory %q counts = (%d direct, %d recursive), want (%d, %d)",
			path, counts.direct, counts.recursive, direct, recursive)
	}
}

func writeSyntheticFile(t *testing.T, filename, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatalf("create fixture parent: %v", err)
	}
	if err := os.WriteFile(filename, []byte(contents), 0o644); err != nil {
		t.Fatalf("write fixture %q: %v", filepath.Base(filename), err)
	}
}

func TestFullScanIndexesRecursiveTreeAndCounts(t *testing.T) {
	environment := newTestEnvironment(t)
	libraryRecord := environment.createLibrary(t, "Family", "family")
	root := filepath.Join(environment.allowedPath, "family")

	writeSyntheticFile(t, filepath.Join(root, "cover.webp"), "cover")
	writeSyntheticFile(t, filepath.Join(root, "album", "photo.jpg"), "photo")
	writeSyntheticFile(t, filepath.Join(root, "album", "nested", "clip.MOV"), "video")
	writeSyntheticFile(t, filepath.Join(root, "album", "nested", "notes.txt"), "unsupported")
	writeSyntheticFile(t, filepath.Join(root, ".hidden", "kept.png"), "hidden but supported")
	writeSyntheticFile(t, filepath.Join(root, "@eaDir", "derived.jpg"), "system-derived")
	writeSyntheticFile(t, filepath.Join(root, ".Trash", "removed.gif"), "recycle")
	writeSyntheticFile(t, filepath.Join(root, "vector.svg"), "unsupported")
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatalf("create empty directory: %v", err)
	}

	run := environment.runScan(t, libraryRecord, scanner.TriggerCreation, nil)
	if run.Generation != 1 || run.Status != scanner.RunStatusSucceeded {
		t.Fatalf("first run = generation %d, status %q; want generation 1 succeeded", run.Generation, run.Status)
	}
	if run.DiscoveredDirectories != 5 || run.DiscoveredAssets != 4 || run.SkippedCount != 4 {
		t.Fatalf("first run counts = %d directories, %d assets, %d skipped; want 5, 4, 4",
			run.DiscoveredDirectories, run.DiscoveredAssets, run.SkippedCount)
	}
	if run.SkippedDirectories != 2 || run.SkippedFiles != 2 {
		t.Fatalf(
			"first run skipped = %d directories, %d files; want 2 and 2",
			run.SkippedDirectories,
			run.SkippedFiles,
		)
	}

	snapshot := readCatalog(t, environment.inspector, libraryRecord.ID)
	assertAssetPaths(t, snapshot,
		".hidden/kept.png",
		"album/nested/clip.MOV",
		"album/photo.jpg",
		"cover.webp",
	)
	if len(snapshot.directories) != 5 {
		t.Fatalf("indexed directories = %v, want root plus four readable descendants", snapshot.directories)
	}
	assertDirectoryCounts(t, snapshot, "", 1, 4)
	assertDirectoryCounts(t, snapshot, "album", 1, 2)
	assertDirectoryCounts(t, snapshot, "album/nested", 1, 1)
	assertDirectoryCounts(t, snapshot, ".hidden", 1, 1)
	assertDirectoryCounts(t, snapshot, "empty", 0, 0)

	current, err := environment.libraries.Get(context.Background(), libraryRecord.ID)
	if err != nil {
		t.Fatalf("get scanned library: %v", err)
	}
	if current.Status != library.StatusReady || current.CurrentGeneration != 1 {
		t.Fatalf("library after first scan = status %q generation %d, want ready generation 1",
			current.Status, current.CurrentGeneration)
	}
}

func TestAbortedScansPreserveIndexUntilSuccessfulConvergence(t *testing.T) {
	environment := newTestEnvironment(t)
	libraryRecord := environment.createLibrary(t, "Timeline", "timeline")
	libraryPath := filepath.Join(environment.allowedPath, "timeline")
	albumPath := filepath.Join(libraryPath, "album")
	oldPath := filepath.Join(albumPath, "old.jpg")
	writeSyntheticFile(t, oldPath, "old")

	first := environment.runScan(t, libraryRecord, scanner.TriggerCreation, nil)
	if first.Generation != 1 {
		t.Fatalf("first generation = %d, want 1", first.Generation)
	}
	assertAssetPaths(t, readCatalog(t, environment.inspector, libraryRecord.ID), "album/old.jpg")

	if err := os.Remove(oldPath); err != nil {
		t.Fatalf("remove old fixture: %v", err)
	}
	newPath := filepath.Join(albumPath, "new.jpg")
	writeSyntheticFile(t, newPath, "new")
	injectedFailure := errors.New("injected walker failure")
	failedWalker := &hookedWalker{
		Walker: environment.walker,
		afterVisit: func(entry scanner.WalkEntry) error {
			if entry.RelativePath == "album/new.jpg" {
				return injectedFailure
			}
			return nil
		},
	}
	failed, err := environment.scans.RunFullScan(context.Background(), scanner.FullScanRequest{
		LibraryID: libraryRecord.ID,
		Trigger:   scanner.TriggerManual,
		Walker:    failedWalker,
	})
	if !errors.Is(err, injectedFailure) {
		t.Fatalf("failed scan error = %v, want injected failure", err)
	}
	if failed.Generation != 2 || failed.Status != scanner.RunStatusFailed {
		t.Fatalf("failed run = generation %d status %q, want generation 2 failed", failed.Generation, failed.Status)
	}
	// The new one-entry batch is safe to keep; the old generation is not stale-
	// cleaned because the traversal did not finish.
	assertAssetPaths(t, readCatalog(t, environment.inspector, libraryRecord.ID),
		"album/new.jpg", "album/old.jpg")
	assertCurrentGeneration(t, environment, libraryRecord.ID, 1)

	cancelPath := filepath.Join(albumPath, "cancel.png")
	writeSyntheticFile(t, cancelPath, "cancel")
	ctx, cancel := context.WithCancel(context.Background())
	cancelledWalker := &hookedWalker{
		Walker: environment.walker,
		afterVisit: func(entry scanner.WalkEntry) error {
			if entry.RelativePath == "album/cancel.png" {
				cancel()
			}
			return nil
		},
	}
	cancelled, err := environment.scans.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: libraryRecord.ID,
		Trigger:   scanner.TriggerManual,
		Walker:    cancelledWalker,
	})
	cancel()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled scan error = %v, want context.Canceled", err)
	}
	if cancelled.Generation != 3 || cancelled.Status != scanner.RunStatusCancelled {
		t.Fatalf("cancelled run = generation %d status %q, want generation 3 cancelled",
			cancelled.Generation, cancelled.Status)
	}
	assertAssetPaths(t, readCatalog(t, environment.inspector, libraryRecord.ID),
		"album/cancel.png", "album/new.jpg", "album/old.jpg")
	assertCurrentGeneration(t, environment, libraryRecord.ID, 1)

	offlinePath := libraryPath + "-offline"
	if err := os.Rename(libraryPath, offlinePath); err != nil {
		t.Fatalf("take library offline: %v", err)
	}
	offline, offlineErr := environment.scans.RunFullScan(context.Background(), scanner.FullScanRequest{
		LibraryID: libraryRecord.ID,
		Trigger:   scanner.TriggerScheduled,
		Walker:    environment.walker,
	})
	if err := os.Rename(offlinePath, libraryPath); err != nil {
		t.Fatalf("restore offline library: %v", err)
	}
	if !errors.Is(offlineErr, scanner.ErrLibraryOffline) {
		t.Fatalf("offline scan error = %v, want ErrLibraryOffline", offlineErr)
	}
	if offline.Generation != 4 || offline.Status != scanner.RunStatusOffline {
		t.Fatalf("offline run = generation %d status %q, want generation 4 offline",
			offline.Generation, offline.Status)
	}
	assertAssetPaths(t, readCatalog(t, environment.inspector, libraryRecord.ID),
		"album/cancel.png", "album/new.jpg", "album/old.jpg")
	assertCurrentGeneration(t, environment, libraryRecord.ID, 1)

	originalDuringReplacement := libraryPath + "-original"
	replaced := false
	replacementWalker := &hookedWalker{
		Walker: environment.walker,
		beforeCheck: func() error {
			if replaced {
				return nil
			}
			replaced = true
			if err := os.Rename(libraryPath, originalDuringReplacement); err != nil {
				return fmt.Errorf("rename original library before verification: %w", err)
			}
			if err := os.Mkdir(libraryPath, 0o755); err != nil {
				return fmt.Errorf("create empty replacement library: %w", err)
			}
			return nil
		},
	}
	replacedRun, replacementErr := environment.scans.RunFullScan(context.Background(), scanner.FullScanRequest{
		LibraryID: libraryRecord.ID,
		Trigger:   scanner.TriggerManual,
		Walker:    replacementWalker,
	})
	if removeErr := os.Remove(libraryPath); removeErr != nil {
		t.Fatalf("remove replacement library: %v", removeErr)
	}
	if restoreErr := os.Rename(originalDuringReplacement, libraryPath); restoreErr != nil {
		t.Fatalf("restore original after replacement test: %v", restoreErr)
	}
	if !errors.Is(replacementErr, scanner.ErrRootIdentityChanged) {
		t.Fatalf("replacement scan error = %v, want root identity changed", replacementErr)
	}
	if replacedRun.Generation != 5 || replacedRun.Status != scanner.RunStatusFailed ||
		replacedRun.ErrorCode != "root_identity_changed" {
		t.Fatalf("replacement run = generation %d status %q code %q, want generation 5 failed/root_identity_changed",
			replacedRun.Generation, replacedRun.Status, replacedRun.ErrorCode)
	}
	assertAssetPaths(t, readCatalog(t, environment.inspector, libraryRecord.ID),
		"album/cancel.png", "album/new.jpg", "album/old.jpg")
	assertCurrentGeneration(t, environment, libraryRecord.ID, 1)

	for _, stalePath := range []string{newPath, cancelPath} {
		if err := os.Remove(stalePath); err != nil {
			t.Fatalf("remove stale fixture %q: %v", filepath.Base(stalePath), err)
		}
	}
	writeSyntheticFile(t, filepath.Join(albumPath, "final.gif"), "final")
	final := environment.runScan(t, libraryRecord, scanner.TriggerManual, nil)
	if final.Generation != 6 || final.Status != scanner.RunStatusSucceeded {
		t.Fatalf("converged run = generation %d status %q, want generation 6 succeeded",
			final.Generation, final.Status)
	}
	finalSnapshot := readCatalog(t, environment.inspector, libraryRecord.ID)
	assertAssetPaths(t, finalSnapshot, "album/final.gif")
	assertDirectoryCounts(t, finalSnapshot, "", 0, 1)
	assertDirectoryCounts(t, finalSnapshot, "album", 1, 1)
	assertCurrentGeneration(t, environment, libraryRecord.ID, 6)
}

func TestRootABAReplacementCannotGainCleanupEligibility(t *testing.T) {
	environment := newTestEnvironment(t)
	libraryRecord := environment.createLibrary(t, "ABA", "aba")
	libraryPath := filepath.Join(environment.allowedPath, "aba")
	writeSyntheticFile(t, filepath.Join(libraryPath, "old.jpg"), "old")
	first := environment.runScan(t, libraryRecord, scanner.TriggerCreation, nil)
	if first.Generation != 1 {
		t.Fatalf("first generation = %d, want 1", first.Generation)
	}

	originalPath := libraryPath + "-original"
	walker := &abaWalker{
		Walker: environment.walker,
		beforeWalk: func() error {
			if err := os.Rename(libraryPath, originalPath); err != nil {
				return err
			}
			return os.Mkdir(libraryPath, 0o755)
		},
		afterWalk: func() error {
			if err := os.Remove(libraryPath); err != nil {
				return err
			}
			return os.Rename(originalPath, libraryPath)
		},
	}
	run, err := environment.scans.RunFullScan(context.Background(), scanner.FullScanRequest{
		LibraryID: libraryRecord.ID,
		Trigger:   scanner.TriggerManual,
		Walker:    walker,
	})
	if !errors.Is(err, scanner.ErrRootIdentityChanged) {
		t.Fatalf("ABA scan error = %v, want ErrRootIdentityChanged", err)
	}
	if run.Status != scanner.RunStatusFailed || run.ErrorCode != "root_identity_changed" {
		t.Fatalf("ABA run = %#v", run)
	}
	assertAssetPaths(t, readCatalog(t, environment.inspector, libraryRecord.ID), "old.jpg")
	assertCurrentGeneration(t, environment, libraryRecord.ID, 1)
}

func assertCurrentGeneration(t *testing.T, environment *testEnvironment, libraryID, expected int64) {
	t.Helper()
	current, err := environment.libraries.Get(context.Background(), libraryID)
	if err != nil {
		t.Fatalf("get current library generation: %v", err)
	}
	if current.CurrentGeneration != expected {
		t.Fatalf("current generation = %d, want %d", current.CurrentGeneration, expected)
	}
}

func TestFullScansAreIsolatedByLibrary(t *testing.T) {
	environment := newTestEnvironment(t)
	firstLibrary := environment.createLibrary(t, "First", "first")
	secondLibrary := environment.createLibrary(t, "Second", "second")
	firstPhoto := filepath.Join(environment.allowedPath, "first", "album", "photo.jpg")
	secondPhoto := filepath.Join(environment.allowedPath, "second", "album", "photo.jpg")
	writeSyntheticFile(t, firstPhoto, "first")
	writeSyntheticFile(t, secondPhoto, "second")

	firstRun := environment.runScan(t, firstLibrary, scanner.TriggerCreation, nil)
	secondRun := environment.runScan(t, secondLibrary, scanner.TriggerCreation, nil)
	if firstRun.Generation != 1 || secondRun.Generation != 1 {
		t.Fatalf("independent first generations = %d and %d, want both 1",
			firstRun.Generation, secondRun.Generation)
	}
	assertAssetPaths(t, readCatalog(t, environment.inspector, firstLibrary.ID), "album/photo.jpg")
	assertAssetPaths(t, readCatalog(t, environment.inspector, secondLibrary.ID), "album/photo.jpg")

	if err := os.Remove(firstPhoto); err != nil {
		t.Fatalf("remove first library photo: %v", err)
	}
	writeSyntheticFile(t, filepath.Join(environment.allowedPath, "first", "album", "replacement.png"), "replacement")
	secondGeneration := environment.runScan(t, firstLibrary, scanner.TriggerManual, nil)
	if secondGeneration.Generation != 2 {
		t.Fatalf("first library next generation = %d, want 2", secondGeneration.Generation)
	}

	firstSnapshot := readCatalog(t, environment.inspector, firstLibrary.ID)
	secondSnapshot := readCatalog(t, environment.inspector, secondLibrary.ID)
	assertAssetPaths(t, firstSnapshot, "album/replacement.png")
	assertAssetPaths(t, secondSnapshot, "album/photo.jpg")
	assertDirectoryCounts(t, firstSnapshot, "album", 1, 1)
	assertDirectoryCounts(t, secondSnapshot, "album", 1, 1)
	assertCurrentGeneration(t, environment, firstLibrary.ID, 2)
	assertCurrentGeneration(t, environment, secondLibrary.ID, 1)
}
