package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/library"
)

func TestDirectorySourceStreamsOnlyDirectDirectoryCandidates(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	for _, directory := range []string{
		"albums",
		"albums/2026",
		"empty",
		".hidden",
		"nested/child",
	} {
		if err := os.MkdirAll(filepath.Join(rootPath, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(rootPath, "photo.jpg"), "not a directory")
	if err := os.Symlink("albums", filepath.Join(rootPath, "album-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	source, err := NewDirectorySource(root)
	if err != nil {
		t.Fatal(err)
	}
	var candidates []library.DirectoryCandidate
	err = source.EnumerateDirectories(
		context.Background(),
		"",
		func(candidate library.DirectoryCandidate) error {
			candidates = append(candidates, candidate)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]library.DirectoryCandidate, len(candidates))
	for _, candidate := range candidates {
		got[candidate.Name] = candidate
	}
	if len(got) != 5 {
		t.Fatalf("candidates = %#v, want five direct directories", candidates)
	}
	if _, exists := got["photo.jpg"]; exists {
		t.Fatal("ordinary file was exposed by directory enumeration")
	}
	if !got["albums"].HasChildren || !got["nested"].HasChildren {
		t.Fatalf("child hints = albums:%t nested:%t", got["albums"].HasChildren, got["nested"].HasChildren)
	}
	if got["empty"].HasChildren || got[".hidden"].BlockedReason != "" {
		t.Fatalf("empty/hidden candidates = %#v / %#v", got["empty"], got[".hidden"])
	}
	if got["album-link"].BlockedReason != library.SelectionBlockedSymlink {
		t.Fatalf("symlink candidate = %#v", got["album-link"])
	}
}

func TestDirectorySourceAnchorsNestedParentAndMapsUnsafeParents(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	if err := os.MkdirAll(filepath.Join(rootPath, "family", "2026"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("family", filepath.Join(rootPath, "family-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	source, err := NewDirectorySource(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	err = source.EnumerateDirectories(
		context.Background(),
		"family",
		func(candidate library.DirectoryCandidate) error {
			names = append(names, candidate.Name)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "2026" {
		t.Fatalf("nested names = %#v", names)
	}

	tests := []struct {
		parent string
		want   error
	}{
		{parent: "../outside", want: library.ErrInvalidParent},
		{parent: "family-link", want: library.ErrParentSymlink},
		{parent: "missing", want: library.ErrParentUnavailable},
	}
	for _, test := range tests {
		if err := source.EnumerateDirectories(
			context.Background(),
			test.parent,
			func(library.DirectoryCandidate) error { return nil },
		); !errors.Is(err, test.want) {
			t.Fatalf("EnumerateDirectories(%q) error = %v, want %v", test.parent, err, test.want)
		}
	}
}
