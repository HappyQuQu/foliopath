package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestAssetFingerprintMigrationBackfillsVersionFiveCatalog(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-five.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 5); err != nil {
		t.Fatalf("apply version-five schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO libraries(
			id, name, name_key, root_rel_path, status,
			current_generation, created_at_ms, updated_at_ms, name_sort_key
		) VALUES (1, 'Archive', 'archive', 'archive', 'ready', 1, 1, 1, X'01');
		INSERT INTO directories(
			id, library_id, parent_id, relative_path, name, mtime_ns,
			last_seen_generation, direct_asset_count, recursive_asset_count
		) VALUES (1, 1, NULL, '', '', 0, 1, 1, 1);
		INSERT INTO assets(
			id, library_id, directory_id, relative_path, name, kind,
			media_format, mime_type, size_bytes, mtime_ns, last_seen_generation
		) VALUES (
			1, 1, 1, 'photo.jpg', 'photo.jpg', 'image',
			'jpeg', 'image/jpeg', 42, 1700000000000000001, 1
		)`); err != nil {
		t.Fatalf("seed version-five catalog: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatalf("upgrade version-five database: %v", err)
	}
	defer store.Close()

	if _, ok := tableColumns(t, store, "assets")["source_fingerprint"]; !ok {
		t.Fatal("upgraded assets table is missing source_fingerprint")
	}
	var fingerprint string
	if err := store.db.QueryRowContext(
		ctx,
		`SELECT source_fingerprint FROM assets WHERE id = 1`,
	).Scan(&fingerprint); err != nil {
		t.Fatal(err)
	}
	if got, want := fingerprint, "v1:42:1700000000000000001"; got != want {
		t.Fatalf("backfilled fingerprint = %q, want %q", got, want)
	}
	var version int64
	if err := store.db.QueryRowContext(
		ctx,
		`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1`,
	).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 7 {
		t.Fatalf("migration version = %d, want 7", version)
	}
}
