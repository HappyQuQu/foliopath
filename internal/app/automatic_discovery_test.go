package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type automaticDiscoveryWatcherStub struct {
	mu           sync.Mutex
	events       chan scanner.WatchEvent
	watchError   error
	watchCount   int
	unwatchCount int
}

func newAutomaticDiscoveryWatcherStub() *automaticDiscoveryWatcherStub {
	return &automaticDiscoveryWatcherStub{
		events: make(chan scanner.WatchEvent, 8),
	}
}

func (watcher *automaticDiscoveryWatcherStub) WatchLibrary(
	context.Context,
	int64,
	string,
) error {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	watcher.watchCount++
	return watcher.watchError
}

func (*automaticDiscoveryWatcherStub) WatchDirectory(
	context.Context,
	int64,
	string,
	string,
) error {
	return nil
}

func (watcher *automaticDiscoveryWatcherStub) UnwatchLibrary(int64) error {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	watcher.unwatchCount++
	return nil
}

func (watcher *automaticDiscoveryWatcherStub) Events() <-chan scanner.WatchEvent {
	return watcher.events
}

func (*automaticDiscoveryWatcherStub) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (*automaticDiscoveryWatcherStub) Close() error { return nil }

func (watcher *automaticDiscoveryWatcherStub) counts() (int, int) {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	return watcher.watchCount, watcher.unwatchCount
}

type automaticDiscoveryWalkerStub struct{}

func (automaticDiscoveryWalkerStub) CaptureRoot(
	context.Context,
	string,
) (scanner.RootIdentity, error) {
	return scanner.RootIdentity{Device: 1, Inode: 1}, nil
}

func (automaticDiscoveryWalkerStub) Walk(
	ctx context.Context,
	_ string,
	visit func(scanner.WalkEntry) (scanner.WalkDecision, error),
) error {
	_, err := visit(scanner.WalkEntry{IsDirectory: true})
	return err
}

func (automaticDiscoveryWalkerStub) VerifyRoot(
	context.Context,
	string,
	scanner.RootIdentity,
) error {
	return nil
}

func TestAutomaticDiscoveryOverflowRequiresSameLibraryGenerationRecovery(
	t *testing.T,
) {
	ctx := context.Background()
	databaseComponent, database := newDatabaseComponent(
		t.TempDir(),
		newReadinessState(),
	)
	if err := databaseComponent.start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = databaseComponent.stop(context.Background()) })

	libraries, err := library.NewService(database)
	if err != nil {
		t.Fatal(err)
	}
	item, err := libraries.Create(ctx, "Archive", "archive")
	if err != nil {
		t.Fatal(err)
	}
	scanService, err := scanner.NewService(database, scanner.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scanService.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: item.ID,
		Trigger:   scanner.TriggerManual,
		Walker:    automaticDiscoveryWalkerStub{},
	}); err != nil {
		t.Fatal(err)
	}

	configSignal := jobs.NewSignal()
	recoverySignal := jobs.NewSignal()
	reconcileSignal := jobs.NewSignal()
	scanSignal := jobs.NewSignal()
	reconcileAdmission, err := scanner.NewReconcileAdmission(database, reconcileSignal)
	if err != nil {
		t.Fatal(err)
	}
	scanAdmission, err := scanner.NewAdmissionService(database, scanSignal)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := newAutomaticDiscoveryCoordinator(
		database,
		&mediaRootService{},
		reconcileAdmission,
		scanAdmission,
		configSignal,
		recoverySignal,
	)
	if err != nil {
		t.Fatal(err)
	}
	watcher := newAutomaticDiscoveryWatcherStub()
	coordinator.newWatcher = func(
		files.WatcherOptions,
	) (files.LibraryWatcher, error) {
		return watcher, nil
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- coordinator.Run(runCtx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("coordinator run: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("coordinator did not stop")
		}
	})

	awaitWatcherCounts(t, watcher, 1, 0)
	watcher.events <- scanner.WatchEvent{Kind: scanner.WatchEventOverflow}
	awaitDiscoveryDetails(
		t,
		database,
		item.ID,
		library.AutomaticDiscoveryDegraded,
		"watch_overflow",
	)
	run, found, err := database.ClaimNextFullScan(ctx, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim fallback scan found=%t err=%v", found, err)
	}
	time.Sleep(100 * time.Millisecond)
	if watched, _ := watcher.counts(); watched != 1 {
		t.Fatalf("watch registrations before recovery = %d, want 1", watched)
	}
	if _, err := scanService.RunClaimedFullScan(
		ctx,
		run,
		automaticDiscoveryWalkerStub{},
	); err != nil {
		t.Fatal(err)
	}
	recoverySignal.Wake()
	awaitWatcherCounts(t, watcher, 2, 1)
	awaitDiscoveryDetails(
		t,
		database,
		item.ID,
		library.AutomaticDiscoveryActive,
		"",
	)
}

func TestAutomaticDiscoveryWatchLimitDegradesAndAdmitsFallback(t *testing.T) {
	ctx := context.Background()
	databaseComponent, database := newDatabaseComponent(
		t.TempDir(),
		newReadinessState(),
	)
	if err := databaseComponent.start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = databaseComponent.stop(context.Background()) })
	libraries, _ := library.NewService(database)
	item, err := libraries.Create(ctx, "Archive", "archive")
	if err != nil {
		t.Fatal(err)
	}
	scans, _ := scanner.NewService(database, scanner.Config{})
	if _, err := scans.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: item.ID,
		Trigger:   scanner.TriggerManual,
		Walker:    automaticDiscoveryWalkerStub{},
	}); err != nil {
		t.Fatal(err)
	}

	configSignal := jobs.NewSignal()
	recoverySignal := jobs.NewSignal()
	reconcileSignal := jobs.NewSignal()
	scanSignal := jobs.NewSignal()
	reconcileAdmission, _ := scanner.NewReconcileAdmission(database, reconcileSignal)
	scanAdmission, _ := scanner.NewAdmissionService(database, scanSignal)
	coordinator, err := newAutomaticDiscoveryCoordinator(
		database,
		&mediaRootService{},
		reconcileAdmission,
		scanAdmission,
		configSignal,
		recoverySignal,
	)
	if err != nil {
		t.Fatal(err)
	}
	watcher := newAutomaticDiscoveryWatcherStub()
	watcher.watchError = files.ErrWatchResourceLimit
	coordinator.newWatcher = func(
		files.WatcherOptions,
	) (files.LibraryWatcher, error) {
		return watcher, nil
	}
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- coordinator.Run(runCtx) }()

	awaitDiscoveryDetails(
		t,
		database,
		item.ID,
		library.AutomaticDiscoveryDegraded,
		"watch_resource_limit",
	)
	if _, found, err := database.ClaimNextFullScan(ctx, time.Minute); err != nil || !found {
		t.Fatalf("resource fallback scan found=%t err=%v", found, err)
	}
	cancel()
	if err := <-runDone; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func awaitWatcherCounts(
	t *testing.T,
	watcher *automaticDiscoveryWatcherStub,
	wantWatch int,
	wantUnwatch int,
) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		watched, unwatched := watcher.counts()
		if watched == wantWatch && unwatched == wantUnwatch {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"watcher counts = (%d,%d), want (%d,%d)",
				watched,
				unwatched,
				wantWatch,
				wantUnwatch,
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func awaitDiscoveryDetails(
	t *testing.T,
	database *databaseService,
	libraryID int64,
	status library.AutomaticDiscoveryStatus,
	errorCode string,
) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		details, err := database.GetLibraryDetails(context.Background(), libraryID)
		if err == nil &&
			details.AutomaticDiscoveryStatus == status &&
			details.AutomaticDiscoveryErrorCode == errorCode {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"discovery details = %#v, want status=%q error=%q (err=%v)",
				details,
				status,
				errorCode,
				err,
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
