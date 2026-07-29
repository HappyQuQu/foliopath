package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

type automaticDiscoveryWaker struct{ count int }

func (waker *automaticDiscoveryWaker) Wake() { waker.count++ }

func TestAutomaticDiscoveryReconcilesOneDirectoryWithoutFullScan(t *testing.T) {
	ctx := context.Background()
	mediaRoot := filepath.Join(t.TempDir(), "library")
	if err := os.MkdirAll(filepath.Join(mediaRoot, "archive", "album"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(mediaRoot, "archive", "album", "old.jpg")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := files.OpenRoot(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	walker, err := files.NewScanWalker(root)
	if err != nil {
		t.Fatal(err)
	}

	now := time.UnixMilli(10_000)
	store, err := sqlitestore.Open(
		ctx,
		filepath.Join(t.TempDir(), "automatic-discovery.db"),
		sqlitestore.Options{Now: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	libraries, err := library.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	item, err := libraries.Create(ctx, "Archive", "archive")
	if err != nil {
		t.Fatal(err)
	}
	fullScanner, err := scanner.NewService(store, scanner.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fullScanner.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: item.ID,
		Trigger:   scanner.TriggerManual,
		Walker:    walker,
	}); err != nil {
		t.Fatal(err)
	}
	catalogService, err := catalog.NewService(store, nil)
	if err != nil {
		t.Fatal(err)
	}
	revisionBefore, err := catalogService.ContentRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(oldPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(mediaRoot, "archive", "album", "new.jpg"),
		[]byte("new-media"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.EnqueueReconcile(
		ctx,
		item.ID,
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
	mediaWaker := &automaticDiscoveryWaker{}
	processor, err := scanner.NewReconcileProcessor(store, walker, mediaWaker, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(ctx, job); err != nil {
		t.Fatal(err)
	}

	page, err := catalogService.ListAssets(ctx, catalog.AssetRequest{
		LibraryID:    item.ID,
		Recursive:    true,
		RecursiveSet: true,
		Limit:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].RelativePath != "album/new.jpg" {
		t.Fatalf("reconciled assets = %#v", page.Items)
	}
	revisionAfter, err := catalogService.ContentRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if revisionAfter != revisionBefore+1 || mediaWaker.count != 1 {
		t.Fatalf(
			"automatic discovery publication = revision %d -> %d, media wakes %d",
			revisionBefore,
			revisionAfter,
			mediaWaker.count,
		)
	}
}
