//go:build linux && mediafull

package cachefs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func TestPublisherCleansTemporaryFileAfterRealDiskFull(t *testing.T) {
	root := os.Getenv("FOLIOPATH_FULL_CACHE_ROOT")
	if root == "" {
		t.Skip("FOLIOPATH_FULL_CACHE_ROOT is not configured")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		t.Fatal(err)
	}
	publisher, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	derivation, err := thumbnail.GridDerivation(
		1, 1, media.SourceFingerprint("v1:1:1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := derivation.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(
		filepath.Dir(filepath.Join(root, filepath.FromSlash(relative))),
		0o750,
	); err != nil {
		t.Fatal(err)
	}
	fillerPath := filepath.Join(root, "filler")
	filler, err := os.Create(fillerPath)
	if err != nil {
		t.Fatal(err)
	}
	block := make([]byte, 128<<10)
	for {
		_, err = filler.Write(block)
		if err != nil {
			break
		}
	}
	if closeErr := filler.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("fill cache filesystem error = %v, want ENOSPC", err)
	}

	if _, err := publisher.Publish(
		context.Background(), derivation, make([]byte, 256<<10),
	); err == nil {
		t.Fatal("publish on full filesystem unexpectedly succeeded")
	}
	matches, err := filepath.Glob(
		filepath.Join(root, "libraries", "lib_1", "*", ".thumbnail-*.tmp"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain after ENOSPC: %v", matches)
	}

	if err := os.Remove(fillerPath); err != nil {
		t.Fatal(err)
	}
	if _, err := publisher.Publish(
		context.Background(), derivation, []byte("recovered"),
	); err != nil {
		t.Fatalf("publish did not recover after freeing space: %v", err)
	}
}
