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

func TestAVIMigrationPreservesCatalogAndDerivedForeignKeys(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-thirteen.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 13); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO libraries(
			id, name, name_key, root_rel_path, status,
			current_generation, created_at_ms, updated_at_ms
		) VALUES (1, 'Archive', 'archive', 'archive', 'ready', 1, 1, 1);
		INSERT INTO directories(
			id, library_id, parent_id, relative_path, name, mtime_ns,
			direct_asset_count, recursive_asset_count, last_seen_generation,
			natural_name_key, search_name_key
		) VALUES (1, 1, NULL, '', '', 1, 1, 1, 1, X'01', '');
		INSERT INTO assets(
			id, library_id, directory_id, relative_path, name, kind,
			media_format, mime_type, size_bytes, mtime_ns,
			last_seen_generation, source_fingerprint, natural_name_key,
			search_name_key, search_path_key
		) VALUES (
			1, 1, 1, 'photo.jpg', 'photo.jpg', 'image',
			'jpeg', 'image/jpeg', 10, 1, 1, 'v1:10:1', X'01',
			'photo.jpg', 'photo.jpg'
		);
		INSERT INTO thumbnails(
			library_id, asset_id, variant, source_fingerprint,
			transform_version, status
		) VALUES (1, 1, 'grid', 'v1:10:1', 1, 'pending');
		INSERT INTO media_jobs(
			id, library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, created_at_ms
		) VALUES (1, 1, 1, 'grid', 0, 1, 'v1:10:1', 'queued', 1, 1);
	`); err != nil {
		t.Fatalf("seed version-thirteen state: %v", err)
	}

	if _, err := provider.UpTo(ctx, 14); err != nil {
		t.Fatalf("upgrade to AVI schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO assets(
			id, library_id, directory_id, relative_path, name, kind,
			media_format, mime_type, size_bytes, mtime_ns,
			last_seen_generation, source_fingerprint, natural_name_key,
			search_name_key, search_path_key
		) VALUES (
			2, 1, 1, 'legacy.avi', 'legacy.avi', 'video',
			'avi', 'video/x-msvideo', 20, 2, 1, 'v1:20:2', X'02',
			'legacy.avi', 'legacy.avi'
		)
	`); err != nil {
		t.Fatalf("insert AVI after migration: %v", err)
	}
	for _, index := range []string{
		"assets_browse_directory_size",
		"assets_browse_library_size",
		"assets_search_global_size",
	} {
		var found int
		if err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sqlite_schema
			WHERE type = 'index' AND name = ?
		`, index).Scan(&found); err != nil || found != 1 {
			t.Fatalf("size index %q count = %d, error = %v", index, found, err)
		}
	}
	assertNoForeignKeyViolations(t, db)
	for _, table := range []string{"assets", "thumbnails", "media_jobs"} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		want := 1
		if table == "assets" {
			want = 2
		}
		if count != want {
			t.Fatalf("%s row count = %d, want %d", table, count, want)
		}
	}
	var searchRows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM asset_search`).Scan(&searchRows); err != nil {
		t.Fatal(err)
	}
	if searchRows != 2 {
		t.Fatalf("search rows after upgrade = %d, want 2", searchRows)
	}

	if _, err := provider.DownTo(ctx, 13); err != nil {
		t.Fatalf("downgrade AVI schema: %v", err)
	}
	assertNoForeignKeyViolations(t, db)
	var jpeg, avi int
	if err := db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE media_format = 'jpeg'),
			COUNT(*) FILTER (WHERE media_format = 'avi')
		FROM assets
	`).Scan(&jpeg, &avi); err != nil {
		t.Fatal(err)
	}
	if jpeg != 1 || avi != 0 {
		t.Fatalf("downgraded assets jpeg/avi = %d/%d, want 1/0", jpeg, avi)
	}
}

func assertNoForeignKeyViolations(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("database has a foreign-key violation")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}
