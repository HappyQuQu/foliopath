//go:build linux

package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func TestAutomaticDiscoveryComponentPublishesInotifyChanges(t *testing.T) {
	ctx := context.Background()
	rootPath := filepath.Join(t.TempDir(), "library")
	if err := os.MkdirAll(filepath.Join(rootPath, "archive", "album"), 0o755); err != nil {
		t.Fatal(err)
	}

	databaseComponent, database := newDatabaseComponent(
		t.TempDir(),
		newReadinessState(),
	)
	if err := databaseComponent.start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = databaseComponent.stop(context.Background()) })
	mediaRoot, mediaRootComponent, err := newMediaRootService(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := mediaRootComponent.start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mediaRootComponent.stop(context.Background()) })

	libraries, err := library.NewService(database)
	if err != nil {
		t.Fatal(err)
	}
	item, err := libraries.Create(ctx, "Archive", "archive")
	if err != nil {
		t.Fatal(err)
	}
	fullScanner, err := scanner.NewService(database, scanner.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fullScanner.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: item.ID,
		Trigger:   scanner.TriggerManual,
		Walker:    mediaRoot,
	}); err != nil {
		t.Fatal(err)
	}

	scanSignal := jobs.NewSignal()
	configSignal := jobs.NewSignal()
	recoverySignal := jobs.NewSignal()
	reconcileSignal := jobs.NewSignal()
	mediaSignal := jobs.NewSignal()
	scanAdmission, err := scanner.NewAdmissionService(database, scanSignal)
	if err != nil {
		t.Fatal(err)
	}
	reconcileAdmission, err := scanner.NewReconcileAdmission(database, reconcileSignal)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := newAutomaticDiscoveryCoordinator(
		database,
		mediaRoot,
		reconcileAdmission,
		scanAdmission,
		configSignal,
		recoverySignal,
	)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := scanner.NewReconcileProcessor(
		database,
		mediaRoot,
		mediaSignal,
		coordinator,
	)
	if err != nil {
		t.Fatal(err)
	}
	worker, err := jobs.NewWorkerPool(
		reconcileJobQueue{database: database},
		processor,
		reconcileSignal,
		jobs.WorkerOptions{
			Workers:           scanner.MaxConcurrentReconciles,
			HeartbeatInterval: 100 * time.Millisecond,
			LeaseDuration:     time.Second,
			IdlePollInterval:  25 * time.Millisecond,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	automaticComponent, err := newAutomaticDiscoveryComponent(worker, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	if err := automaticComponent.start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = automaticComponent.stop(stopCtx)
	})

	awaitAutomaticDiscoveryState(t, database, item.ID, library.AutomaticDiscoveryActive)
	if err := os.WriteFile(
		filepath.Join(rootPath, "archive", "album", "new.jpg"),
		[]byte("new-media"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	catalogService, err := catalog.NewService(database, nil)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		page, listErr := catalogService.ListAssets(ctx, catalog.AssetRequest{
			LibraryID:    item.ID,
			Recursive:    true,
			RecursiveSet: true,
			Limit:        10,
		})
		if listErr == nil && len(page.Items) == 1 &&
			page.Items[0].RelativePath == "album/new.jpg" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("automatic discovery did not publish asset: page=%#v err=%v", page, listErr)
		}
		time.Sleep(25 * time.Millisecond)
	}

	rootDirectories, err := catalogService.ListDirectories(
		ctx,
		catalog.DirectoryRequest{LibraryID: item.ID, Limit: 10},
	)
	if err != nil || len(rootDirectories.Items) != 1 {
		t.Fatalf("root directories = %#v, err=%v", rootDirectories, err)
	}
	albumID := rootDirectories.Items[0].ID
	if err := os.Mkdir(filepath.Join(rootPath, "archive", "album", "fresh"), 0o755); err != nil {
		t.Fatal(err)
	}
	awaitDirectoryPath(t, catalogService, item.ID, albumID, "album/fresh")
	if err := os.WriteFile(
		filepath.Join(rootPath, "archive", "album", "fresh", "nested.png"),
		[]byte("nested-media"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	awaitAssetPaths(
		t,
		catalogService,
		item.ID,
		[]string{"album/fresh/nested.png", "album/new.jpg"},
	)
}

func awaitAutomaticDiscoveryState(
	t *testing.T,
	database *databaseService,
	libraryID int64,
	want library.AutomaticDiscoveryStatus,
) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		details, err := database.GetLibraryDetails(context.Background(), libraryID)
		if err == nil && details.AutomaticDiscoveryStatus == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"automatic discovery state = %q, want %q (err=%v)",
				details.AutomaticDiscoveryStatus,
				want,
				err,
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func awaitDirectoryPath(
	t *testing.T,
	service *catalog.Service,
	libraryID int64,
	parentID int64,
	want string,
) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		page, err := service.ListDirectories(
			context.Background(),
			catalog.DirectoryRequest{
				LibraryID:         libraryID,
				ParentDirectoryID: parentID,
				Limit:             10,
			},
		)
		if err == nil && len(page.Items) == 1 &&
			page.Items[0].RelativePath == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("directory %q not discovered: page=%#v err=%v", want, page, err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func awaitAssetPaths(
	t *testing.T,
	service *catalog.Service,
	libraryID int64,
	want []string,
) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		page, err := service.ListAssets(context.Background(), catalog.AssetRequest{
			LibraryID:    libraryID,
			Recursive:    true,
			RecursiveSet: true,
			Limit:        10,
		})
		if err == nil && len(page.Items) == len(want) {
			got := make(map[string]struct{}, len(page.Items))
			for _, asset := range page.Items {
				got[asset.RelativePath] = struct{}{}
			}
			allFound := true
			for _, relative := range want {
				if _, exists := got[relative]; !exists {
					allFound = false
				}
			}
			if allFound {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("assets not discovered: page=%#v err=%v", page, err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
