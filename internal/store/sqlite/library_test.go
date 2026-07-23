package sqlite

import (
	"context"
	"errors"
	"testing"

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
