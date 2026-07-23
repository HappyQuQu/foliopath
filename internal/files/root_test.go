package files

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestRoot(t *testing.T) (string, *Root) {
	t.Helper()
	rootPath := filepath.Join(t.TempDir(), "media")
	if err := os.Mkdir(rootPath, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRoot(rootPath)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	t.Cleanup(func() {
		_ = root.Close()
	})
	return rootPath, root
}

func writeTestFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityCaptureVerifyAndKey(t *testing.T) {
	t.Parallel()

	_, root := newTestRoot(t)
	identity, err := root.Capture()
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	device, inode, ok := identity.Key()
	if !ok {
		t.Fatal("Identity.Key reported no Unix device/inode identity")
	}
	if inode == 0 {
		t.Fatal("Identity.Key returned a zero inode")
	}
	if device == 0 {
		t.Log("filesystem reports device 0; ok flag still establishes availability")
	}
	if err := root.Verify(identity); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if _, _, ok := (Identity{}).Key(); ok {
		t.Fatal("zero Identity unexpectedly has a platform key")
	}
	if err := root.Verify(Identity{}); !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("Verify(zero) error = %v; want ErrInvalidIdentity", err)
	}
}

func TestCaptureAtAndVerifyAtUseRelativeNoSymlinkBoundary(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	if err := os.MkdirAll(filepath.Join(rootPath, "people", "family"), 0o755); err != nil {
		t.Fatal(err)
	}
	identity, err := root.CaptureAt("people/family")
	if err != nil {
		t.Fatalf("CaptureAt: %v", err)
	}
	if err := root.VerifyAt("people/family", identity); err != nil {
		t.Fatalf("VerifyAt: %v", err)
	}

	external := filepath.Join(t.TempDir(), "external")
	if err := os.MkdirAll(filepath.Join(external, "family"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(rootPath, "external-alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := root.CaptureAt("external-alias/family"); !errors.Is(err, ErrSymlink) {
		t.Fatalf("CaptureAt through external symlink error = %v; want ErrSymlink", err)
	}
	if err := os.Symlink("people", filepath.Join(rootPath, "internal-alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := root.CaptureAt("internal-alias/family"); !errors.Is(err, ErrSymlink) {
		t.Fatalf("CaptureAt through internal symlink error = %v; want ErrSymlink", err)
	}
}

func TestVerifyAtRejectsLibraryRootReplacement(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	libraryPath := filepath.Join(rootPath, "family")
	if err := os.Mkdir(libraryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	identity, err := root.CaptureAt("family")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(libraryPath, libraryPath+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(libraryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	verifyErr := root.VerifyAt("family", identity)
	if !errors.Is(verifyErr, ErrOffline) || !errors.Is(verifyErr, ErrRootChanged) {
		t.Fatalf("VerifyAt after replacement error = %v; want ErrOffline and ErrRootChanged", verifyErr)
	}
	if _, err := root.CaptureAt("missing"); !errors.Is(err, ErrOffline) {
		t.Fatalf("CaptureAt missing root error = %v; want ErrOffline", err)
	}
}

func TestOpenPreservesUnicodeAndLiteralPercentNames(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	relative := "旅行 100%/夏天%GG-%20-photo%2Ejpg"
	writeTestFile(t, filepath.Join(rootPath, filepath.FromSlash(relative)), "raw-name")

	file, err := root.Open(relative)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(contents) != "raw-name" {
		t.Fatalf("contents = %q; raw percent path was likely reinterpreted", contents)
	}
}

func TestOpenAndOpenDirRejectSymlinks(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	writeTestFile(t, filepath.Join(rootPath, "inside", "photo.jpg"), "inside")
	external := filepath.Join(t.TempDir(), "external")
	writeTestFile(t, filepath.Join(external, "secret.jpg"), "outside")

	links := map[string]string{
		"internal-file": filepath.Join("inside", "photo.jpg"),
		"external-file": filepath.Join(external, "secret.jpg"),
		"internal-dir":  "inside",
		"external-dir":  external,
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(rootPath, name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}

	for _, name := range []string{"internal-file", "external-file"} {
		file, err := root.Open(name)
		if file != nil {
			_ = file.Close()
		}
		if !errors.Is(err, ErrSymlink) {
			t.Errorf("Open(%q) error = %v; want ErrSymlink", name, err)
		}
	}
	for _, name := range []string{"internal-dir", "external-dir"} {
		directory, err := root.OpenDir(name)
		if directory != nil {
			_ = directory.Close()
		}
		if !errors.Is(err, ErrSymlink) {
			t.Errorf("OpenDir(%q) error = %v; want ErrSymlink", name, err)
		}
	}
}

func TestOpenRootRejectsFinalSymlink(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	actual := filepath.Join(base, "actual")
	if err := os.Mkdir(actual, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(actual, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	root, err := OpenRoot(link)
	if root != nil {
		_ = root.Close()
	}
	if !errors.Is(err, ErrInvalidRoot) || !errors.Is(err, ErrSymlink) {
		t.Fatalf("OpenRoot(symlink) error = %v; want ErrInvalidRoot and ErrSymlink", err)
	}
}

func TestRootRemovalIsOffline(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	identity, err := root.Capture()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(rootPath); err != nil {
		t.Fatalf("remove root: %v", err)
	}
	if err := root.Verify(identity); !errors.Is(err, ErrOffline) {
		t.Fatalf("Verify after removal error = %v; want ErrOffline", err)
	}
}

func TestRootReplacementIsOfflineAndNeverAppearsEmpty(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	identity, err := root.Capture()
	if err != nil {
		t.Fatal(err)
	}
	moved := rootPath + "-mounted"
	if err := os.Rename(rootPath, moved); err != nil {
		t.Fatalf("rename root: %v", err)
	}
	if err := os.Mkdir(rootPath, 0o755); err != nil {
		t.Fatalf("create replacement root: %v", err)
	}

	verifyErr := root.Verify(identity)
	if !errors.Is(verifyErr, ErrOffline) || !errors.Is(verifyErr, ErrRootChanged) {
		t.Fatalf("Verify after replacement error = %v; want ErrOffline and ErrRootChanged", verifyErr)
	}
	entries, readErr := root.ReadDir("")
	if entries != nil {
		t.Fatalf("ReadDir returned replacement contents: %v", entries)
	}
	if !errors.Is(readErr, ErrOffline) || !errors.Is(readErr, ErrRootChanged) {
		t.Fatalf("ReadDir after replacement error = %v; want ErrOffline and ErrRootChanged", readErr)
	}
}

func TestErrorsDoNotExposeRootOrInvalidAbsoluteInput(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	_, missingErr := root.Open("album/missing.jpg")
	if missingErr == nil {
		t.Fatal("Open(missing) succeeded")
	}
	if strings.Contains(missingErr.Error(), rootPath) {
		t.Fatalf("missing-file error exposed root path: %v", missingErr)
	}

	invalidInput := filepath.Join(rootPath, "secret.jpg")
	_, invalidErr := root.Open(invalidInput)
	if !errors.Is(invalidErr, ErrInvalidPath) {
		t.Fatalf("Open(absolute) error = %v; want ErrInvalidPath", invalidErr)
	}
	if strings.Contains(invalidErr.Error(), invalidInput) || strings.Contains(invalidErr.Error(), rootPath) {
		t.Fatalf("invalid-path error exposed an absolute path: %v", invalidErr)
	}
	var operationErr *Error
	if !errors.As(invalidErr, &operationErr) || operationErr.Path != "" {
		t.Fatalf("invalid-path Error.Path = %q; want empty", operationErr.Path)
	}

	missingRoot := filepath.Join(t.TempDir(), "private-root-name")
	_, rootErr := OpenRoot(missingRoot)
	if rootErr == nil {
		t.Fatal("OpenRoot(missing) succeeded")
	}
	if strings.Contains(rootErr.Error(), missingRoot) {
		t.Fatalf("OpenRoot error exposed configured root: %v", rootErr)
	}
}

func TestReadDirIsSortedAndRequiresDirectory(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	writeTestFile(t, filepath.Join(rootPath, "z.jpg"), "z")
	writeTestFile(t, filepath.Join(rootPath, "a.jpg"), "a")
	entries, err := root.ReadDir("")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 || entries[0].Name() != "a.jpg" || entries[1].Name() != "z.jpg" {
		t.Fatalf("ReadDir entries = %v; want name-sorted entries", entries)
	}
	_, err = root.OpenDir("a.jpg")
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("OpenDir(file) error = %v; want ErrNotDirectory", err)
	}
	_, err = root.Open("")
	if !errors.Is(err, ErrNotRegular) {
		t.Fatalf("Open(root) error = %v; want ErrNotRegular", err)
	}
	if _, err := root.OpenDir("missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("OpenDir(missing) error = %v; want fs.ErrNotExist", err)
	}
}
