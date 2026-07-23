package files

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalkSkipsSymlinksAndCallerSelectedSystemDirectories(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	writeTestFile(t, filepath.Join(rootPath, "regular.txt"), "regular")
	writeTestFile(t, filepath.Join(rootPath, "album", "photo.jpg"), "photo")
	writeTestFile(t, filepath.Join(rootPath, "@eaDir", "hidden.jpg"), "hidden")
	external := filepath.Join(t.TempDir(), "external")
	writeTestFile(t, filepath.Join(external, "secret.jpg"), "secret")

	links := map[string]string{
		"internal-dir-link":  "album",
		"external-dir-link":  external,
		"internal-file-link": "regular.txt",
		"external-file-link": filepath.Join(external, "secret.jpg"),
		"broken-link":        "does-not-exist",
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(rootPath, name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}

	visited := make(map[string]bool)
	reported := make(map[string]error)
	err := root.Walk(context.Background(), "", func(relative string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			reported[relative] = walkErr
			return nil
		}
		visited[relative] = true
		if relative == "@eaDir" {
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	for _, relative := range []string{"", "regular.txt", "album", "album/photo.jpg", "@eaDir"} {
		if !visited[relative] {
			t.Errorf("Walk did not visit %q", relative)
		}
	}
	if visited["@eaDir/hidden.jpg"] {
		t.Error("Walk ignored caller's fs.SkipDir for @eaDir")
	}
	for name := range links {
		if !errors.Is(reported[name], ErrSymlink) {
			t.Errorf("Walk error for %q = %v; want ErrSymlink", name, reported[name])
		}
		if visited[name] {
			t.Errorf("Walk treated symlink %q as a normal entry", name)
		}
		for relative := range visited {
			if strings.HasPrefix(relative, name+"/") {
				t.Errorf("Walk followed symlink %q to %q", name, relative)
			}
		}
	}
}

func TestWalkCancellation(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	for index := 0; index < walkBatchSize+5; index++ {
		writeTestFile(t, filepath.Join(rootPath, "album", strings.Repeat("x", index%5)+string(rune('a'+index%26))+"-photo"+strings.Repeat("0", index/26)+".jpg"), "photo")
	}
	ctx, cancel := context.WithCancel(context.Background())
	callbacks := 0
	err := root.Walk(ctx, "", func(string, fs.DirEntry, error) error {
		callbacks++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Walk error = %v; want context.Canceled", err)
	}
	if callbacks != 1 {
		t.Fatalf("callbacks = %d; want cancellation after root callback", callbacks)
	}
}

func TestWalkRejectsRootReplacementDuringScan(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	writeTestFile(t, filepath.Join(rootPath, "photo.jpg"), "photo")
	moved := rootPath + "-mounted"
	replaced := false
	err := root.Walk(context.Background(), "", func(relative string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if relative == "" && !replaced {
			replaced = true
			if err := os.Rename(rootPath, moved); err != nil {
				t.Fatalf("rename root: %v", err)
			}
			if err := os.Mkdir(rootPath, 0o755); err != nil {
				t.Fatalf("create replacement root: %v", err)
			}
		}
		return nil
	})
	if !errors.Is(err, ErrOffline) || !errors.Is(err, ErrRootChanged) {
		t.Fatalf("Walk after root replacement error = %v; want ErrOffline and ErrRootChanged", err)
	}
}

func TestWalkRejectsInvalidStartWithoutEchoingIt(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	invalid := filepath.Join(rootPath, "outside")
	err := root.Walk(context.Background(), invalid, func(string, fs.DirEntry, error) error { return nil })
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Walk(invalid) error = %v; want ErrInvalidPath", err)
	}
	if strings.Contains(err.Error(), rootPath) || strings.Contains(err.Error(), invalid) {
		t.Fatalf("Walk(invalid) exposed absolute input: %v", err)
	}
}

func TestWalkInvalidCallbackInputsDoNotEchoUncheckedPath(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	unchecked := filepath.Join(rootPath, "private")
	tests := []struct {
		name string
		ctx  context.Context
		fn   fs.WalkDirFunc
	}{
		{name: "nil context", fn: func(string, fs.DirEntry, error) error { return nil }},
		{name: "nil callback", ctx: context.Background()},
	}
	for _, test := range tests {
		err := root.Walk(test.ctx, unchecked, test.fn)
		if err == nil {
			t.Fatalf("%s: Walk unexpectedly succeeded", test.name)
		}
		if strings.Contains(err.Error(), rootPath) || strings.Contains(err.Error(), unchecked) {
			t.Fatalf("%s: Walk error exposed unchecked absolute path: %v", test.name, err)
		}
	}
}
