package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestCatalogBrowseMigrationBackfillsVersionSixRowsAndIndexes(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-six.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 6); err != nil {
		t.Fatalf("apply version-six schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
        INSERT INTO libraries(
            id, name, name_key, root_rel_path, status,
            current_generation, created_at_ms, updated_at_ms, name_sort_key
        ) VALUES (1, 'Archive', 'archive', 'archive', 'ready', 1, 1, 1, X'01');
        INSERT INTO directories(
            id, library_id, parent_id, relative_path, name, mtime_ns,
            last_seen_generation, direct_asset_count, recursive_asset_count
        ) VALUES
            (1, 1, NULL, '', '', 0, 1, 1, 1),
            (2, 1, 1, 'Album 10', 'Album 10', 1, 1, 1, 1);
        INSERT INTO assets(
            id, library_id, directory_id, relative_path, name, kind,
            media_format, mime_type, size_bytes, mtime_ns,
            last_seen_generation, source_fingerprint
        ) VALUES (
            1, 1, 2, 'Album 10/photo-2.jpg', 'photo-2.jpg', 'image',
            'jpeg', 'image/jpeg', 42, 2, 1, 'v1:42:2'
        )`); err != nil {
		t.Fatalf("seed version-six catalog: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatalf("upgrade version-six database: %v", err)
	}
	defer store.Close()

	for _, table := range []string{"directories", "assets"} {
		if _, ok := tableColumns(t, store, table)["natural_name_key"]; !ok {
			t.Fatalf("%s is missing natural_name_key", table)
		}
		var empty int64
		if err := store.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM `+table+` WHERE length(natural_name_key) = 0`,
		).Scan(&empty); err != nil {
			t.Fatal(err)
		}
		if empty != 0 {
			t.Fatalf("%s has %d empty natural-name keys", table, empty)
		}
	}
	var directoryKey, assetKey []byte
	if err := store.db.QueryRowContext(ctx,
		`SELECT natural_name_key FROM directories WHERE id = 2`,
	).Scan(&directoryKey); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx,
		`SELECT natural_name_key FROM assets WHERE id = 1`,
	).Scan(&assetKey); err != nil {
		t.Fatal(err)
	}
	if string(directoryKey) != string(catalog.NaturalNameKey("Album 10")) ||
		string(assetKey) != string(catalog.NaturalNameKey("photo-2.jpg")) {
		t.Fatal("catalog natural-name keys were not canonically backfilled")
	}
	for _, index := range []string{
		"directories_browse_children",
		"assets_browse_directory_name",
		"assets_browse_library_name",
		"assets_browse_directory_modified",
	} {
		var exists int64
		if err := store.db.QueryRowContext(ctx, `
            SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`,
			index,
		).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists != 1 {
			t.Fatalf("missing catalog browse index %q", index)
		}
	}
}
