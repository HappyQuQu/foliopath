package cachefs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func TestPublisherAtomicallyStoresOnlyDerivedCachePath(t *testing.T) {
	root := t.TempDir()
	publisher, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	derivation, err := thumbnail.GridDerivation(
		7, 9, media.SourceFingerprint("v1:42:100"),
	)
	if err != nil {
		t.Fatal(err)
	}
	published, err := publisher.Publish(
		context.Background(), derivation, []byte("webp"),
	)
	if err != nil {
		t.Fatal(err)
	}
	value, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(published.CacheRelativePath)))
	if err != nil {
		t.Fatal(err)
	}
	if string(value) != "webp" || published.ByteSize != 4 {
		t.Fatalf("published = %#v, %q", published, value)
	}
	matches, err := filepath.Glob(filepath.Join(
		filepath.Dir(filepath.Join(root, filepath.FromSlash(published.CacheRelativePath))),
		".thumbnail-*.tmp",
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestPublisherReportsSpaceAndRemovesOnlyDerivedRelativePath(t *testing.T) {
	root := t.TempDir()
	publisher, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if available, err := publisher.AvailableBytes(context.Background()); err != nil ||
		available <= 0 {
		t.Fatalf("available bytes = %d, %v", available, err)
	}
	relative := "libraries/lib_1/aa/cache.webp"
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("cache"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Remove(context.Background(), relative); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed file stat error = %v", err)
	}
	if err := publisher.Remove(
		context.Background(), "../outside",
	); !errors.Is(err, thumbnail.ErrInvalidDerivation) {
		t.Fatalf("traversal removal error = %v", err)
	}
}
