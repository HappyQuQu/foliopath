package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestAnchoredChildCannotOpenDirectoryMovedOutsideBoundary(t *testing.T) {
	base := t.TempDir()
	allowedPath := filepath.Join(base, "library")
	insidePath := filepath.Join(allowedPath, "inside")
	outsidePath := filepath.Join(base, "moved-outside")
	if err := os.MkdirAll(insidePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(insidePath, "secret.jpg"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, err := openAnchoredRoot(allowedPath)
	if err != nil {
		t.Fatalf("openAnchoredRoot: %v", err)
	}
	defer root.Close()
	child, err := root.OpenRoot("inside")
	if err != nil {
		t.Fatalf("OpenRoot(inside): %v", err)
	}
	defer child.Close()

	if err := os.Rename(insidePath, outsidePath); err != nil {
		t.Fatalf("move child outside boundary: %v", err)
	}
	file, err := child.OpenFile("secret.jpg", os.O_RDONLY, 0)
	if file != nil {
		_ = file.Close()
		t.Fatal("child opened a file after its directory moved outside the trusted boundary")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("OpenFile after move error = %v, want fs.ErrNotExist", err)
	}
}
