package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type fakeWalker struct {
	identity   scanner.RootIdentity
	entries    []scanner.WalkEntry
	captureErr error
	verifyErr  error
	cancel     context.CancelFunc
}

type cancelOnCompleteRepository struct {
	scanner.Repository
	cancel context.CancelFunc
}

func (repository *cancelOnCompleteRepository) CompleteFullScan(
	ctx context.Context,
	runID int64,
	skipped int64,
) (scanner.ScanRun, error) {
	repository.cancel()
	<-ctx.Done()
	return scanner.ScanRun{}, ctx.Err()
}

func (walker *fakeWalker) CaptureRoot(context.Context, string) (scanner.RootIdentity, error) {
	if walker.captureErr != nil {
		return scanner.RootIdentity{}, walker.captureErr
	}
	return walker.identity, nil
}

func (walker *fakeWalker) Walk(
	ctx context.Context,
	_ string,
	visit func(scanner.WalkEntry) (scanner.WalkDecision, error),
) error {
	skippedPrefixes := make([]string, 0)
entries:
	for index, entry := range walker.entries {
		for _, prefix := range skippedPrefixes {
			if strings.HasPrefix(entry.RelativePath, prefix+"/") {
				continue entries
			}
		}
		decision, err := visit(entry)
		if err != nil {
			return err
		}
		if decision == scanner.WalkSkipDirectory {
			skippedPrefixes = append(skippedPrefixes, entry.RelativePath)
		}
		if index == 0 && walker.cancel != nil {
			walker.cancel()
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (walker *fakeWalker) VerifyRoot(context.Context, string, scanner.RootIdentity) error {
	return walker.verifyErr
}

func TestScannerServiceRootIdentityChangeCannotCleanStaleRows(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx := context.Background()

	first := &fakeWalker{
		identity: scanner.RootIdentity{Device: 1, Inode: 10},
		entries: []scanner.WalkEntry{
			{RelativePath: "album", IsDirectory: true},
			{RelativePath: "album/old.jpg", SizeBytes: 1},
		},
	}
	service, err := scanner.NewService(store, scanner.Config{BatchSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: libraryRecord.ID, Trigger: scanner.TriggerCreation, Walker: first,
	}); err != nil {
		t.Fatalf("first RunFullScan() error = %v", err)
	}

	changed := &fakeWalker{
		identity:  scanner.RootIdentity{Device: 1, Inode: 10},
		verifyErr: scanner.ErrRootIdentityChanged,
	}
	run, err := service.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: libraryRecord.ID, Trigger: scanner.TriggerManual, Walker: changed,
	})
	if !errors.Is(err, scanner.ErrRootIdentityChanged) {
		t.Fatalf("changed-root RunFullScan() error = %v", err)
	}
	if run.Status != scanner.RunStatusFailed || run.ErrorCode != "root_identity_changed" {
		t.Fatalf("changed-root run = %#v", run)
	}
	_, assets := countCatalog(t, store, libraryRecord.ID)
	if assets != 1 {
		t.Fatalf("assets after root identity change = %d, want 1", assets)
	}
}

func TestScannerServiceOfflineCaptureRecordsOfflineWithoutCleanup(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	service, err := scanner.NewService(store, scanner.Config{BatchSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	walker := &fakeWalker{captureErr: scanner.ErrLibraryOffline}
	run, err := service.RunFullScan(context.Background(), scanner.FullScanRequest{
		LibraryID: libraryRecord.ID, Trigger: scanner.TriggerStartup, Walker: walker,
	})
	if !errors.Is(err, scanner.ErrLibraryOffline) {
		t.Fatalf("RunFullScan() error = %v, want ErrLibraryOffline", err)
	}
	if run.Status != scanner.RunStatusOffline || run.ErrorCode != "library_offline" {
		t.Fatalf("offline run = %#v", run)
	}
}

func TestScannerServiceRecordsFinalizationFailureWithoutCleanup(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	service, err := scanner.NewService(store, scanner.Config{BatchSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
		CREATE TRIGGER reject_first_generation
		BEFORE UPDATE OF current_generation ON libraries
		WHEN NEW.current_generation = 1
		BEGIN
			SELECT RAISE(ABORT, 'injected service finalization failure');
		END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	walker := &fakeWalker{identity: scanner.RootIdentity{Device: 4, Inode: 40}}
	run, err := service.RunFullScan(context.Background(), scanner.FullScanRequest{
		LibraryID: libraryRecord.ID, Trigger: scanner.TriggerCreation, Walker: walker,
	})
	if err == nil {
		t.Fatal("RunFullScan succeeded despite injected finalization failure")
	}
	if run.Status != scanner.RunStatusFailed || run.ErrorCode != "scan_failed" {
		t.Fatalf("run after finalization failure = %#v", run)
	}
	current, getErr := store.GetLibrary(context.Background(), libraryRecord.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if current.CurrentGeneration != 0 {
		t.Fatalf("current generation after finalization failure = %d, want 0", current.CurrentGeneration)
	}
}

func TestScannerServiceCancellationDuringCompleteIsRecorded(t *testing.T) {
	store, _ := openTestStore(t)
	libraryRecord := createTestLibrary(t, store)
	ctx, cancel := context.WithCancel(context.Background())
	repository := &cancelOnCompleteRepository{Repository: store, cancel: cancel}
	service, err := scanner.NewService(repository, scanner.Config{BatchSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	walker := &fakeWalker{identity: scanner.RootIdentity{Device: 5, Inode: 50}}
	run, err := service.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: libraryRecord.ID, Trigger: scanner.TriggerManual, Walker: walker,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunFullScan error = %v, want context.Canceled", err)
	}
	if run.Status != scanner.RunStatusCancelled {
		t.Fatalf("run status = %q, want cancelled", run.Status)
	}
	persisted, err := store.GetScanRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != scanner.RunStatusCancelled {
		t.Fatalf("persisted run status = %q, want cancelled", persisted.Status)
	}
}

func TestScannerServiceCancellationAndSkipPolicy(t *testing.T) {
	t.Run("cancellation is recorded with independent finalization context", func(t *testing.T) {
		store, _ := openTestStore(t)
		libraryRecord := createTestLibrary(t, store)
		service, err := scanner.NewService(store, scanner.Config{BatchSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		walker := &fakeWalker{
			identity: scanner.RootIdentity{Device: 2, Inode: 20},
			entries:  []scanner.WalkEntry{{RelativePath: "album", IsDirectory: true}},
			cancel:   cancel,
		}
		run, err := service.RunFullScan(ctx, scanner.FullScanRequest{
			LibraryID: libraryRecord.ID, Trigger: scanner.TriggerManual, Walker: walker,
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RunFullScan() error = %v, want context.Canceled", err)
		}
		if run.Status != scanner.RunStatusCancelled {
			t.Fatalf("cancelled run status = %q", run.Status)
		}
	})

	t.Run("hidden items scan while maintained system directories skip", func(t *testing.T) {
		store, _ := openTestStore(t)
		libraryRecord := createTestLibrary(t, store)
		service, err := scanner.NewService(store, scanner.Config{BatchSize: 3})
		if err != nil {
			t.Fatal(err)
		}
		walker := &fakeWalker{
			identity: scanner.RootIdentity{Device: 3, Inode: 30},
			entries: []scanner.WalkEntry{
				{RelativePath: ".hidden", IsDirectory: true},
				{RelativePath: ".hidden/photo.JPG", SizeBytes: 1},
				{RelativePath: "@eaDir", IsDirectory: true},
				{RelativePath: "@eaDir/duplicate.jpg", SizeBytes: 1},
				{RelativePath: "notes.txt", SizeBytes: 1},
			},
		}
		run, err := service.RunFullScan(context.Background(), scanner.FullScanRequest{
			LibraryID: libraryRecord.ID, Trigger: scanner.TriggerCreation, Walker: walker,
		})
		if err != nil {
			t.Fatalf("RunFullScan() error = %v", err)
		}
		if run.SkippedCount != 2 {
			t.Fatalf("skipped count = %d, want system directory and unsupported file", run.SkippedCount)
		}
		directories, assets := countCatalog(t, store, libraryRecord.ID)
		if directories != 2 || assets != 1 {
			t.Fatalf("catalog counts = directories %d assets %d, want 2 and 1", directories, assets)
		}
	})
}
