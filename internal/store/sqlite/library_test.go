package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/library"
)

func TestLibraryNameRootAndRenameInvariants(t *testing.T) {
	store, _ := openTestStore(t)
	service, err := library.NewService(store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	ctx := context.Background()

	family, err := service.Create(ctx, " Family ", "family")
	if err != nil {
		t.Fatalf("Create(family) error = %v", err)
	}
	if family.Name != "Family" || family.RootRelativePath != "family" {
		t.Fatalf("created family = %#v", family)
	}

	if _, err := service.Create(ctx, "family", "other"); !errors.Is(err, library.ErrNameExists) {
		t.Fatalf("duplicate name error = %v, want ErrNameExists", err)
	}
	if _, err := service.Create(ctx, "Archive", "family/2026"); !errors.Is(err, library.ErrRootOverlap) {
		t.Fatalf("overlapping root error = %v, want ErrRootOverlap", err)
	}

	work, err := service.Create(ctx, "Work", "work")
	if err != nil {
		t.Fatalf("Create(work) error = %v", err)
	}
	if _, err := service.Rename(ctx, work.ID, "Family"); !errors.Is(err, library.ErrNameExists) {
		t.Fatalf("duplicate rename error = %v, want ErrNameExists", err)
	}
	renamed, err := service.Rename(ctx, family.ID, "Home")
	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if renamed.Name != "Home" || renamed.RootRelativePath != "family" {
		t.Fatalf("renamed library = %#v", renamed)
	}
	revisionAfterRename := renamed.Revision
	updatedAfterRename := renamed.UpdatedAtMS
	unchangedRename, err := service.Rename(ctx, family.ID, "  Home  ")
	if err != nil {
		t.Fatalf("Rename(no-op) error = %v", err)
	}
	if unchangedRename.Revision != revisionAfterRename ||
		unchangedRename.UpdatedAtMS != updatedAfterRename {
		t.Fatalf(
			"no-op rename changed revision/timestamp: %#v -> %#v",
			renamed,
			unchangedRename,
		)
	}

	if _, err := store.db.ExecContext(ctx, `UPDATE libraries SET root_rel_path = 'moved' WHERE id = ?`, family.ID); err == nil {
		t.Fatal("database trigger allowed an in-place root change")
	}
	unchanged, err := service.Get(ctx, family.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if unchanged.RootRelativePath != "family" {
		t.Fatalf("root changed to %q", unchanged.RootRelativePath)
	}
}

func TestLibraryUnicodeNameKeysRejectCompatibilityAndCaseFoldDuplicates(t *testing.T) {
	t.Parallel()

	store, _ := openTestStore(t)
	service, err := library.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := service.Create(ctx, "Straße", "germany"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, "STRASSE", "other"); !errors.Is(err, library.ErrNameExists) {
		t.Fatalf("full-fold duplicate error = %v, want ErrNameExists", err)
	}
	if _, err := service.Create(ctx, "Ｓｕｍｍｅｒ", "summer"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, "summer", "other-summer"); !errors.Is(err, library.ErrNameExists) {
		t.Fatalf("compatibility duplicate error = %v, want ErrNameExists", err)
	}
	if _, err := service.Create(ctx, "Cafe\u0301", "cafe"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, "CAFÉ", "other-cafe"); !errors.Is(err, library.ErrNameExists) {
		t.Fatalf("canonical duplicate error = %v, want ErrNameExists", err)
	}
}

func TestLibraryOverlapRejectsSameAncestorDescendantAndAllowedRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		firstRoot string
		nextRoot  string
		want      error
	}{
		{name: "same", firstRoot: "family", nextRoot: "family", want: library.ErrRootOverlap},
		{name: "descendant", firstRoot: "family", nextRoot: "family/2026", want: library.ErrRootOverlap},
		{name: "ancestor", firstRoot: "family/2026", nextRoot: "family", want: library.ErrRootOverlap},
		{name: "root first", firstRoot: "", nextRoot: "family", want: library.ErrRootOverlap},
		{name: "root second", firstRoot: "family", nextRoot: "", want: library.ErrRootOverlap},
		{name: "component sibling", firstRoot: "family", nextRoot: "family-archive"},
		{name: "unicode sibling", firstRoot: "旅行", nextRoot: "旅行者"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, _ := openTestStore(t)
			service, err := library.NewService(store)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Create(context.Background(), "First", test.firstRoot); err != nil {
				t.Fatal(err)
			}
			_, err = service.Create(context.Background(), "Second", test.nextRoot)
			if !errors.Is(err, test.want) {
				t.Fatalf("Create(%q after %q) error = %v, want %v",
					test.nextRoot, test.firstRoot, err, test.want)
			}
		})
	}
}

func TestConcurrentStoresAtomicallyEnforceNameAndRootRules(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		params []library.CreateParams
		want   error
	}{
		{
			name: "same folded name",
			params: []library.CreateParams{
				{Name: "Straße", RootRelativePath: "one"},
				{Name: "STRASSE", RootRelativePath: "two"},
			},
			want: library.ErrNameExists,
		},
		{
			name: "overlapping root",
			params: []library.CreateParams{
				{Name: "One", RootRelativePath: "family"},
				{Name: "Two", RootRelativePath: "family/2026"},
			},
			want: library.ErrRootOverlap,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "concurrent-library.db")
			stores := make([]*Store, 2)
			for index := range stores {
				store, err := Open(context.Background(), filename, Options{
					BusyTimeout:        5 * time.Second,
					MaxOpenConnections: 2,
				})
				if err != nil {
					t.Fatal(err)
				}
				stores[index] = store
				t.Cleanup(func() {
					if err := store.Close(); err != nil {
						t.Errorf("close store: %v", err)
					}
				})
			}

			start := make(chan struct{})
			results := make(chan error, 2)
			var wait sync.WaitGroup
			for index := range stores {
				wait.Add(1)
				go func(index int) {
					defer wait.Done()
					<-start
					_, err := stores[index].CreateLibrary(
						context.Background(),
						test.params[index],
					)
					results <- err
				}(index)
			}
			close(start)
			wait.Wait()
			close(results)

			var success, rejected int
			for err := range results {
				switch {
				case err == nil:
					success++
				case errors.Is(err, test.want):
					rejected++
				default:
					t.Errorf("concurrent create error = %v, want nil or %v", err, test.want)
				}
			}
			if success != 1 || rejected != 1 {
				t.Fatalf("concurrent results: success=%d rejected=%d", success, rejected)
			}
		})
	}
}

func TestConcurrentRenameReturnsItsOwnCommittedRepresentation(t *testing.T) {
	t.Parallel()

	store, _ := openTestStore(t)
	created, err := store.CreateLibrary(context.Background(), library.CreateParams{
		Name:             "Initial",
		RootRelativePath: "family",
	})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan library.Library, 2)
	errorsFound := make(chan error, 2)
	var wait sync.WaitGroup
	for _, name := range []string{"Alpha", "Beta"} {
		wait.Add(1)
		go func(name string) {
			defer wait.Done()
			<-start
			renamed, err := store.RenameLibrary(context.Background(), created.ID, name)
			if err != nil {
				errorsFound <- err
				return
			}
			if renamed.Name != name {
				errorsFound <- errors.New("rename returned another operation's representation")
				return
			}
			results <- renamed
		}(name)
	}
	close(start)
	wait.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
	revisions := make(map[int64]struct{})
	for result := range results {
		revisions[result.Revision] = struct{}{}
	}
	if len(revisions) != 2 {
		t.Fatalf("rename revisions = %#v, want two distinct committed revisions", revisions)
	}
}

func TestLibraryFromDatabaseRejectsNoncanonicalState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       int64
		display  string
		root     string
		revision int64
	}{
		{name: "invalid ID", id: 0, display: "Family", root: "family", revision: 1},
		{name: "unnormalized name", id: 1, display: " Family ", root: "family", revision: 1},
		{name: "invalid name", id: 1, display: "Family\nArchive", root: "family", revision: 1},
		{name: "unnormalized root", id: 1, display: "Family", root: "family/./2026", revision: 1},
		{name: "invalid revision", id: 1, display: "Family", root: "family", revision: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := libraryFromDatabase(
				test.id,
				test.display,
				test.root,
				string(library.StatusReady),
				0,
				test.revision,
				1,
				1,
			)
			if err == nil {
				t.Fatal("libraryFromDatabase accepted noncanonical state")
			}
		})
	}
}
