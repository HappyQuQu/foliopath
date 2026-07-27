package cachefs

import (
	"context"
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
