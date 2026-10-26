package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestModelDownloadConfigBoundedRead(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, "model-downloads.json")
	if data, err := ReadModelDownloadCatalog(name); err != nil || data != nil {
		t.Fatalf("missing config: %q %v", data, err)
	}
	if err := os.WriteFile(name, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	if data, err := ReadModelDownloadCatalog(name); err != nil || string(data) != "[]" {
		t.Fatalf("read config: %q %v", data, err)
	}
	if err := os.Truncate(name, aimodel.MaxDownloadCatalogBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModelDownloadCatalog(name); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("oversized config: %v", err)
	}
	if _, err := ReadModelDownloadCatalog(root); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("directory config: %v", err)
	}
}
