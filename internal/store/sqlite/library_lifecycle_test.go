package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestLibraryLifecycleMigrationBackfillsNaturalSortKey(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-four.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 4); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
        INSERT INTO libraries(
            name, name_key, root_rel_path, status, current_generation,
            revision, created_at_ms, updated_at_ms
        ) VALUES ('Album 10', 'album 10', 'album', 'pending', 0, 1, 1000, 1000)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatalf("upgrade version-four database: %v", err)
	}
	defer store.Close()
	var key []byte
	if err := store.db.QueryRowContext(ctx,
		`SELECT name_sort_key FROM libraries WHERE name = 'Album 10'`,
	).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if len(key) == 0 {
		t.Fatal("version-four library natural sort key was not backfilled")
	}
}

func TestLibraryLifecycleCreateCommitsLibraryScanAndIdempotencyTogether(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	command := library.CreateCommand{
		Name:             "Family",
		NameKey:          "family",
		RootRelativePath: "family",
		KeyHash:          [32]byte{1},
		RequestHash:      [32]byte{2},
		RetentionMS:      86_400_000,
	}

	created, err := store.CreateLibraryWithScan(ctx, command)
	if err != nil {
		t.Fatalf("CreateLibraryWithScan() error = %v", err)
	}
	if created.Replayed || created.Library.ID <= 0 ||
		created.Scan.LibraryID != created.Library.ID ||
		created.Scan.Trigger != "library_created" ||
		created.Scan.Status != "queued" {
		t.Fatalf("created result = %#v", created)
	}

	replayed, err := store.CreateLibraryWithScan(ctx, command)
	if err != nil {
		t.Fatalf("idempotent CreateLibraryWithScan() error = %v", err)
	}
	if !replayed.Replayed ||
		replayed.Library.ID != created.Library.ID ||
		replayed.Scan.ID != created.Scan.ID {
		t.Fatalf("replayed result = %#v, created = %#v", replayed, created)
	}

	conflict := command
	conflict.RequestHash = [32]byte{3}
	if _, err := store.CreateLibraryWithScan(ctx, conflict); !errors.Is(
		err,
		library.ErrIdempotencyConflict,
	) {
		t.Fatalf("different request replay error = %v", err)
	}

	for table, want := range map[string]int{
		"libraries":           1,
		"scan_runs":           1,
		"idempotency_records": 1,
	} {
		var got int
		if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s count = %d, want %d", table, got, want)
		}
	}
}

func TestLibraryLifecycleRenameUsesRevisionAndRemovalIsDerivedOnly(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	created, err := store.CreateLibraryWithScan(ctx, library.CreateCommand{
		Name:             "Family",
		NameKey:          "family",
		RootRelativePath: "family",
		KeyHash:          [32]byte{10},
		RequestHash:      [32]byte{11},
		RetentionMS:      86_400_000,
	})
	if err != nil {
		t.Fatal(err)
	}

	renamed, err := store.RenameLibraryIfRevision(ctx, library.RenameCommand{
		ID:               created.Library.ID,
		ExpectedRevision: created.Library.Revision,
		Name:             "Home",
		NameKey:          "home",
	})
	if err != nil {
		t.Fatalf("RenameLibraryIfRevision() error = %v", err)
	}
	if renamed.Name != "Home" || renamed.Revision != created.Library.Revision+1 {
		t.Fatalf("renamed = %#v", renamed)
	}
	if _, err := store.RenameLibraryIfRevision(ctx, library.RenameCommand{
		ID:               renamed.ID,
		ExpectedRevision: created.Library.Revision,
		Name:             "Stale",
		NameKey:          "stale",
	}); !errors.Is(err, library.ErrPreconditionFailed) {
		t.Fatalf("stale rename error = %v", err)
	}

	removalResult, err := store.RequestLibraryRemoval(ctx, library.RemoveCommand{
		LibraryID:        renamed.ID,
		ExpectedRevision: renamed.Revision,
		KeyHash:          [32]byte{12},
		RequestHash:      [32]byte{13},
		RetentionMS:      86_400_000,
	})
	if err != nil {
		t.Fatalf("RequestLibraryRemoval() error = %v", err)
	}
	var scanStatus string
	if err := store.db.QueryRowContext(ctx,
		`SELECT status FROM scan_runs WHERE id = ?`,
		created.Scan.ID,
	).Scan(&scanStatus); err != nil {
		t.Fatal(err)
	}
	if scanStatus != "cancelled" {
		t.Fatalf("creation scan status = %q, want cancelled", scanStatus)
	}

	claimed, found, err := store.ClaimNextLibraryRemoval(ctx)
	if err != nil || !found || claimed.ID != removalResult.Removal.ID {
		t.Fatalf("ClaimNextLibraryRemoval() = %#v, %t, %v", claimed, found, err)
	}
	for attempts := 0; attempts < 10; attempts++ {
		done, err := store.CleanupLibraryRemovalBatch(ctx, claimed.ID, 1)
		if err != nil {
			t.Fatalf("cleanup attempt %d: %v", attempts, err)
		}
		if done {
			break
		}
	}
	if _, err := store.GetLibraryDetails(ctx, renamed.ID); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("removed library lookup error = %v", err)
	}
	terminal, err := store.GetLibraryRemoval(ctx, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.Status != library.RemovalSucceeded {
		t.Fatalf("terminal removal = %#v", terminal)
	}
	var retained int
	if err := store.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM idempotency_records
        WHERE operation = 'remove_library' AND result_id = ?`,
		claimed.ID,
	).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if retained != 1 {
		t.Fatalf("retained removal idempotency count = %d, want 1", retained)
	}
}

func TestLibraryLifecycleUsesNaturalKeysetOrdering(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	for index, name := range []string{"Album 10", "Album 2", "Album 1"} {
		_, err := store.CreateLibraryWithScan(ctx, library.CreateCommand{
			Name:             name,
			NameKey:          "album-" + string(rune('a'+index)),
			RootRelativePath: "root-" + string(rune('a'+index)),
			KeyHash:          [32]byte{byte(20 + index)},
			RequestHash:      [32]byte{byte(30 + index)},
			RetentionMS:      86_400_000,
		})
		if err != nil {
			t.Fatal(err)
		}
		// Creation scans are durable but are irrelevant to list ordering.
		if _, err := store.db.ExecContext(ctx, `
            UPDATE scan_runs SET status = 'cancelled', phase = 'completed',
                finished_at_ms = created_at_ms
            WHERE library_id = (SELECT id FROM libraries WHERE name = ?)`,
			name,
		); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.ListLibraryPage(ctx, library.ListParams{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].Name != "Album 1" || first[1].Name != "Album 2" {
		t.Fatalf("first natural page = %#v", first)
	}
	position := library.ListPosition{
		NameSortKey: library.NaturalNameSortKey(first[1].Name),
		Name:        first[1].Name,
		ID:          first[1].ID,
	}
	second, err := store.ListLibraryPage(ctx, library.ListParams{After: &position, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].Name != "Album 10" {
		t.Fatalf("second natural page = %#v", second)
	}
}
