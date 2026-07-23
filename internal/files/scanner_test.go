package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func TestScanWalkerUsesLibraryRelativePathsAndReportsPolicySkips(t *testing.T) {
	allowedPath, root := newTestRoot(t)
	libraryPath := filepath.Join(allowedPath, "family")
	writeTestFile(t, filepath.Join(libraryPath, "album", "photo.jpg"), "photo")
	writeTestFile(t, filepath.Join(libraryPath, "%2e%2e", "not-addressable.jpg"), "encoded traversal")
	writeTestFile(t, filepath.Join(libraryPath, "z-last.jpg"), "must still be visited")
	writeTestFile(t, filepath.Join(libraryPath, "@eaDir", "derived.jpg"), "derived")
	external := filepath.Join(t.TempDir(), "external")
	writeTestFile(t, filepath.Join(external, "secret.jpg"), "secret")
	if err := os.Symlink(external, filepath.Join(libraryPath, "external-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	walker, err := NewScanWalker(root)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := walker.CaptureRoot(context.Background(), "family")
	if err != nil {
		t.Fatalf("CaptureRoot: %v", err)
	}
	visited := make(map[string]bool)
	skipped := make(map[string]bool)
	err = walker.Walk(context.Background(), "family", func(entry scanner.WalkEntry) (scanner.WalkDecision, error) {
		if entry.Skipped {
			skipped[entry.RelativePath] = true
			return scanner.WalkContinue, nil
		}
		visited[entry.RelativePath] = true
		if entry.RelativePath == "@eaDir" {
			return scanner.WalkSkipDirectory, nil
		}
		return scanner.WalkContinue, nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	for _, relative := range []string{"", "album", "album/photo.jpg", "@eaDir", "z-last.jpg"} {
		if !visited[relative] {
			t.Errorf("library-relative path %q was not visited", relative)
		}
	}
	if visited["family"] || visited["@eaDir/derived.jpg"] {
		t.Fatalf("walker exposed allowed-root paths or ignored SkipDirectory: %v", visited)
	}
	if !skipped["external-link"] {
		t.Fatalf("symlink policy skip was not reported: %v", skipped)
	}
	if !skipped["%2e%2e"] || visited["%2e%2e/not-addressable.jpg"] {
		t.Fatalf("encoded traversal-like directory was not safely skipped: visited=%v skipped=%v", visited, skipped)
	}
	if err := walker.VerifyRoot(context.Background(), "family", identity); err != nil {
		t.Fatalf("VerifyRoot: %v", err)
	}
}

func TestScanWalkerMapsMissingAndReplacedLibraryRoots(t *testing.T) {
	allowedPath, root := newTestRoot(t)
	libraryPath := filepath.Join(allowedPath, "family")
	if err := os.Mkdir(libraryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	walker, err := NewScanWalker(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := walker.CaptureRoot(context.Background(), "missing"); !errors.Is(err, scanner.ErrLibraryOffline) {
		t.Fatalf("CaptureRoot(missing) error = %v, want ErrLibraryOffline", err)
	}

	identity, err := walker.CaptureRoot(context.Background(), "family")
	if err != nil {
		t.Fatal(err)
	}
	moved := libraryPath + "-original"
	if err := os.Rename(libraryPath, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(libraryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := walker.VerifyRoot(context.Background(), "family", identity); !errors.Is(err, scanner.ErrRootIdentityChanged) {
		t.Fatalf("VerifyRoot(replaced) error = %v, want ErrRootIdentityChanged", err)
	}
}

func TestScanWalkerBindsWalkToCapturedRootAcrossABAReplacement(t *testing.T) {
	allowedPath, root := newTestRoot(t)
	libraryPath := filepath.Join(allowedPath, "family")
	writeTestFile(t, filepath.Join(libraryPath, "original.jpg"), "original")
	walker, err := NewScanWalker(root)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := walker.CaptureRoot(context.Background(), "family")
	if err != nil {
		t.Fatal(err)
	}

	originalPath := libraryPath + "-original"
	if err := os.Rename(libraryPath, originalPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(libraryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(libraryPath, "replacement.jpg"), "replacement")
	callbacks := 0
	walkErr := walker.Walk(context.Background(), "family", func(scanner.WalkEntry) (scanner.WalkDecision, error) {
		callbacks++
		return scanner.WalkContinue, nil
	})
	if callbacks != 0 {
		t.Fatalf("replacement root produced %d callbacks before identity rejection", callbacks)
	}
	if !errors.Is(walkErr, scanner.ErrRootIdentityChanged) {
		t.Fatalf("Walk(replacement) error = %v, want ErrRootIdentityChanged", walkErr)
	}
	if err := os.RemoveAll(libraryPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(originalPath, libraryPath); err != nil {
		t.Fatal(err)
	}
	// Restoring A makes a final-only comparison pass; the traversal must still
	// have failed because it attempted to start from B.
	if err := walker.VerifyRoot(context.Background(), "family", identity); err != nil {
		t.Fatalf("VerifyRoot after restoring original A: %v", err)
	}
}
