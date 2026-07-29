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

func TestStoryboardSchemaAcceptsValidVariantsAndRejectsInvalidLayout(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	seedStoryboardAsset(t, store.db)

	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO thumbnails(
			library_id, asset_id, variant, source_fingerprint, transform_version,
			cache_rel_path, status, width, height, byte_size, created_at_ms,
			last_accessed_at_ms, frame_count, sprite_columns, sprite_rows,
			cell_width, cell_height
		) VALUES (
			1, 1, 'storyboard', 'v1:42:100', 1,
			'libraries/lib_1/aa/ready.webp', 'ready', 1600, 360, 4096, 10,
			10, 10, 5, 2, 320, 180
		)`); err != nil {
		t.Fatalf("insert ready storyboard: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO media_jobs(
			library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, created_at_ms
		) VALUES (1, 1, 'storyboard', 100, 1, 'v1:42:100', 'queued', 0, 0)
	`); err != nil {
		t.Fatalf("insert storyboard job: %v", err)
	}

	if _, err := store.db.ExecContext(ctx, `
		UPDATE media_jobs SET priority = 0
		WHERE asset_id = 1 AND variant = 'storyboard'
	`); err == nil {
		t.Fatal("storyboard job with grid priority unexpectedly accepted")
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE thumbnails
		SET sprite_columns = 4
		WHERE asset_id = 1 AND variant = 'storyboard'
	`); err == nil {
		t.Fatal("invalid storyboard layout unexpectedly accepted")
	}
}

func TestStoryboardMigrationPreservesVersionTenGridStateAndLease(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-ten.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 10); err != nil {
		t.Fatalf("apply version-ten schema: %v", err)
	}
	seedStoryboardAsset(t, db)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO thumbnails(
			library_id, asset_id, variant, source_fingerprint, transform_version,
			status
		) VALUES (1, 1, 'grid', 'v1:42:100', 1, 'pending');
		INSERT INTO media_jobs(
			id, library_id, asset_id, variant, transform_version,
			source_fingerprint, status, available_at_ms, started_at_ms,
			heartbeat_at_ms, lease_expires_at_ms, attempt_count, created_at_ms
		) VALUES (
			7, 1, 1, 'grid', 1, 'v1:42:100', 'running', 0, 10, 11, 1000, 1, 0
		);
	`); err != nil {
		t.Fatalf("seed version-ten derived state: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("upgrade to storyboard schema: %v", err)
	}

	var variant, status string
	var priority, heartbeat, lease int64
	if err := db.QueryRowContext(ctx, `
		SELECT variant, priority, status, heartbeat_at_ms, lease_expires_at_ms
		FROM media_jobs WHERE id = 7
	`).Scan(&variant, &priority, &status, &heartbeat, &lease); err != nil {
		t.Fatal(err)
	}
	if variant != "grid" || priority != 0 || status != "running" ||
		heartbeat != 11 || lease != 1000 {
		t.Fatalf(
			"upgraded job = variant %q priority %d status %q heartbeat %d lease %d",
			variant,
			priority,
			status,
			heartbeat,
			lease,
		)
	}
	var frameCount sql.NullInt64
	if err := db.QueryRowContext(ctx, `
		SELECT frame_count FROM thumbnails
		WHERE asset_id = 1 AND variant = 'grid'
	`).Scan(&frameCount); err != nil {
		t.Fatal(err)
	}
	if frameCount.Valid {
		t.Fatalf("upgraded grid frame_count = %d, want NULL", frameCount.Int64)
	}
	assertSQLiteIntegrity(t, db)
}

func TestStoryboardMigrationDowngradeIsSafeOrFailsClosed(t *testing.T) {
	t.Run("no storyboard state", func(t *testing.T) {
		store, _ := openTestStore(t)
		provider, err := goose.NewProvider(
			goose.DialectSQLite3,
			store.db,
			migrations.FS,
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := provider.DownTo(context.Background(), 10); err != nil {
			t.Fatalf("downgrade empty storyboard schema: %v", err)
		}
		if _, ok := tableColumns(t, store, "media_jobs")["priority"]; ok {
			t.Fatal("version-ten media_jobs unexpectedly retains priority")
		}
	})

	t.Run("storyboard state exists", func(t *testing.T) {
		store, _ := openTestStore(t)
		seedStoryboardAsset(t, store.db)
		if _, err := store.db.ExecContext(context.Background(), `
			INSERT INTO media_jobs(
				library_id, asset_id, variant, priority, transform_version,
				source_fingerprint, status, available_at_ms, created_at_ms
			) VALUES (
				1, 1, 'storyboard', 100, 1, 'v1:42:100', 'queued', 0, 0
			)
		`); err != nil {
			t.Fatal(err)
		}
		provider, err := goose.NewProvider(
			goose.DialectSQLite3,
			store.db,
			migrations.FS,
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := provider.DownTo(context.Background(), 10); err == nil {
			t.Fatal("downgrade with storyboard state unexpectedly succeeded")
		}
		var jobs int
		if err := store.db.QueryRow(`
			SELECT count(*) FROM media_jobs WHERE variant = 'storyboard'
		`).Scan(&jobs); err != nil {
			t.Fatal(err)
		}
		if jobs != 1 {
			t.Fatalf("storyboard jobs after rejected downgrade = %d, want 1", jobs)
		}
	})
}

func seedStoryboardAsset(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO libraries(
			id, name, name_key, root_rel_path, status,
			current_generation, created_at_ms, updated_at_ms
		) VALUES (1, 'Archive', 'archive', 'archive', 'ready', 1, 1, 1);
		INSERT INTO directories(
			id, library_id, parent_id, relative_path, name, mtime_ns,
			last_seen_generation, direct_asset_count, recursive_asset_count
		) VALUES (1, 1, NULL, '', '', 0, 1, 1, 1);
		INSERT INTO assets(
			id, library_id, directory_id, relative_path, name, kind,
			media_format, mime_type, size_bytes, mtime_ns, last_seen_generation,
			source_fingerprint, width, height, duration_ms, probe_status,
			playback_status
		) VALUES (
			1, 1, 1, 'clip.mp4', 'clip.mp4', 'video',
			'mp4', 'video/mp4', 42, 100, 1,
			'v1:42:100', 1920, 1080, 10000, 'ready', 'playable'
		);
	`); err != nil {
		t.Fatalf("seed storyboard asset: %v", err)
	}
}

func assertSQLiteIntegrity(t *testing.T, db *sql.DB) {
	t.Helper()

	var integrity string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity_check = %q, want ok", integrity)
	}
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported a violation")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}
