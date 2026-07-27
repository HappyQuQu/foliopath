package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/library"
)

func TestMediaRootServiceOwnsOpenAndCloseLifecycle(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(rootPath, "albums"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(rootPath, "albums", "photo.jpg"),
		[]byte("original"),
		0o440,
	); err != nil {
		t.Fatal(err)
	}
	service, lifecycle, err := newMediaRootService(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.EnumerateDirectories(
		context.Background(),
		"",
		func(library.DirectoryCandidate) error { return nil },
	); !errors.Is(err, library.ErrParentUnavailable) {
		t.Fatalf("enumerate before start error = %v, want ErrParentUnavailable", err)
	}
	if err := lifecycle.start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	var candidates []library.DirectoryCandidate
	if err := service.EnumerateDirectories(
		context.Background(),
		"",
		func(candidate library.DirectoryCandidate) error {
			candidates = append(candidates, candidate)
			return nil
		},
	); err != nil {
		t.Fatalf("enumerate after start: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Name != "albums" {
		t.Fatalf("candidates = %#v", candidates)
	}
	asset, err := service.OpenAsset(
		context.Background(), "albums", "photo.jpg",
	)
	if err != nil {
		t.Fatalf("open asset: %v", err)
	}
	if err := asset.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenAsset(
		context.Background(), "albums", "../outside.jpg",
	); err == nil {
		t.Fatal("traversal asset unexpectedly opened")
	}
	if err := lifecycle.stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := service.EnumerateDirectories(
		context.Background(),
		"",
		func(library.DirectoryCandidate) error { return nil },
	); !errors.Is(err, library.ErrParentUnavailable) {
		t.Fatalf("enumerate after stop error = %v, want ErrParentUnavailable", err)
	}
}

func TestMediaRootServiceFailsStartupForUnavailableRoot(t *testing.T) {
	t.Parallel()

	service, lifecycle, err := newMediaRootService(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.start(context.Background()); err == nil {
		t.Fatal("start succeeded for a missing media root")
	}
	if err := service.EnumerateDirectories(
		context.Background(),
		"",
		func(library.DirectoryCandidate) error { return nil },
	); !errors.Is(err, library.ErrParentUnavailable) {
		t.Fatalf("enumerate after failed start error = %v, want ErrParentUnavailable", err)
	}
	if err := lifecycle.stop(context.Background()); err != nil {
		t.Fatalf("stop after failed start: %v", err)
	}
}
