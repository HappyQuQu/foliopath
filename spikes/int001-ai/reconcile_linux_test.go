//go:build linux

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestReconcileModelScanPreservesActiveAndRecoversOrphan(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	catalog, err := ReadModelCatalog("testdata/model-catalog.package.example.json")
	if err != nil {
		t.Fatal(err)
	}
	model := catalog.Models[0]
	writeSyntheticPackage(t, root, model.Directory, false)
	if err := os.Mkdir(filepath.Join(root, "unknown-generation"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unknown-generation", "payload"), []byte("unknown"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := openActivationStore(filepath.Join(t.TempDir(), "activation.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	report, err := ScanModels(root, catalog, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.AcceptedPackages) != 1 || len(report.Rejected) != 1 {
		t.Fatalf("unexpected initial scan: %#v", report)
	}
	if err := reconcileModelScan(ctx, store, report, catalog, "managed"); err != nil {
		t.Fatal(err)
	}
	catalogResult := verifiedCatalog{
		KeyID: "release-2026", Sequence: 1,
		PayloadSHA256: digestText("reconcile-catalog"), authenticated: true,
	}
	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: model.ID, Generation: model.Directory,
		PackageDigest: model.PackageSHA256, Catalog: catalogResult,
	}); err != nil {
		t.Fatal(err)
	}
	assertReconciledAvailability(t, store, true)

	if err := os.RemoveAll(filepath.Join(root, model.Directory)); err != nil {
		t.Fatal(err)
	}
	report, err = ScanModels(root, catalog, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := reconcileModelScan(ctx, store, report, catalog, "managed"); err != nil {
		t.Fatal(err)
	}
	assertReconciledAvailability(t, store, false)

	writeSyntheticPackage(t, root, model.Directory, true)
	report, err = ScanModels(root, catalog, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.AcceptedPackages) != 0 {
		t.Fatal("corrupt restored package was accepted")
	}
	if err := reconcileModelScan(ctx, store, report, catalog, "managed"); err != nil {
		t.Fatal(err)
	}
	assertReconciledAvailability(t, store, false)

	if err := os.RemoveAll(filepath.Join(root, model.Directory)); err != nil {
		t.Fatal(err)
	}
	writeSyntheticPackage(t, root, model.Directory, false)
	report, err = ScanModels(root, catalog, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := reconcileModelScan(ctx, store, report, catalog, "managed"); err != nil {
		t.Fatal(err)
	}
	assertReconciledAvailability(t, store, true)
}

func TestDirectModelLifecycleRequiresReadOnlyMount(t *testing.T) {
	if os.Getenv("INT001_DIRECT_MOUNT_TEST") != "1" {
		t.Skip("set INT001_DIRECT_MOUNT_TEST=1 only in a mount-isolated Linux container")
	}
	ctx := context.Background()
	catalog, err := ReadModelCatalog("testdata/model-catalog.package.example.json")
	if err != nil {
		t.Fatal(err)
	}
	model := catalog.Models[0]
	source := t.TempDir()
	writeSyntheticPackage(t, source, model.Directory, false)
	mountPoint := filepath.Join(t.TempDir(), "models")
	if err := os.Mkdir(mountPoint, 0o700); err != nil {
		t.Fatal(err)
	}
	mounted := false
	mountReadOnly := func() {
		t.Helper()
		if err := unix.Mount(source, mountPoint, "", unix.MS_BIND, ""); err != nil {
			t.Fatal(err)
		}
		mounted = true
		if err := unix.Mount("", mountPoint, "", unix.MS_BIND|unix.MS_REMOUNT|unix.MS_RDONLY, ""); err != nil {
			t.Fatal(err)
		}
	}
	unmount := func() {
		t.Helper()
		if mounted {
			if err := unix.Unmount(mountPoint, 0); err != nil {
				t.Fatal(err)
			}
			mounted = false
		}
	}
	defer func() {
		if mounted {
			_ = unix.Unmount(mountPoint, 0)
		}
	}()
	mountReadOnly()
	report, err := ScanModels(mountPoint, catalog, true)
	if err != nil {
		t.Fatal(err)
	}
	store, err := openActivationStore(filepath.Join(t.TempDir(), "activation.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := reconcileModelScan(ctx, store, report, catalog, "direct"); err != nil {
		t.Fatal(err)
	}
	if err := store.recordPublishedGeneration(ctx, model.ID, model.Directory, model.PackageSHA256); !errors.Is(err, errCatalogEquivocate) {
		t.Fatalf("direct generation was accepted as managed: %v", err)
	}
	catalogResult := verifiedCatalog{
		KeyID: "release-2026", Sequence: 1,
		PayloadSHA256: digestText("direct-catalog"), authenticated: true,
	}
	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: model.ID, Generation: model.Directory,
		PackageDigest: model.PackageSHA256, Catalog: catalogResult,
	}); err != nil {
		t.Fatal(err)
	}
	assertDirectAvailability(t, store, true)

	unmount()
	if err := markCatalogSourceUnavailable(ctx, store, catalog, "direct"); err != nil {
		t.Fatal(err)
	}
	assertDirectAvailability(t, store, false)

	mountReadOnly()
	report, err = ScanModels(mountPoint, catalog, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := reconcileModelScan(ctx, store, report, catalog, "direct"); err != nil {
		t.Fatal(err)
	}
	assertDirectAvailability(t, store, true)
}

func writeSyntheticPackage(t *testing.T, root, directory string, corrupt bool) {
	t.Helper()
	packageRoot := filepath.Join(root, directory)
	if err := os.Mkdir(packageRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{
		"config.json":       "config",
		"model.safetensors": "model",
		"tokenizer.json":    "tokenizer",
	}
	if corrupt {
		contents["model.safetensors"] = "other"
	}
	for filename, content := range contents {
		if err := os.WriteFile(filepath.Join(packageRoot, filename), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func assertReconciledAvailability(t *testing.T, store *activationStore, available bool) {
	t.Helper()
	checkpoint, active, err := store.state(context.Background(), "stable", "synthetic-semantic-package")
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Sequence != 1 || active.Generation != "synthetic-semantic-v1" ||
		active.Available != available || active.SourceKind != "managed" {
		t.Fatalf("unexpected reconciled state: checkpoint=%+v active=%+v", checkpoint, active)
	}
}

func assertDirectAvailability(t *testing.T, store *activationStore, available bool) {
	t.Helper()
	checkpoint, active, err := store.state(context.Background(), "stable", "synthetic-semantic-package")
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Sequence != 1 || active.Generation != "synthetic-semantic-v1" ||
		active.Available != available || active.SourceKind != "direct" {
		t.Fatalf("unexpected direct state: checkpoint=%+v active=%+v", checkpoint, active)
	}
}
