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

func TestAVIRetryMigrationRequeuesFailedMediaProcessing(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-fourteen.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 14); err != nil {
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
			search_name_key, search_path_key, probe_status,
			probe_error_code, playback_status
		) VALUES (
			1, 1, 1, 'legacy.avi', 'legacy.avi', 'video',
			'avi', 'video/x-msvideo', 20, 2, 1, 'v1:20:2', X'02',
			'legacy.avi', 'legacy.avi', 'failed', 'invalid_media', 'unknown'
		);
		INSERT INTO media_jobs(
			id, library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, attempt_count,
			created_at_ms, finished_at_ms, last_error_code
		) VALUES (
			1, 1, 1, 'grid', 0, 1, 'v1:20:2', 'failed', 1, 3, 1, 2,
			'invalid_media'
		);
	`); err != nil {
		t.Fatalf("seed failed AVI state: %v", err)
	}

	if _, err := provider.UpTo(ctx, 15); err != nil {
		t.Fatalf("upgrade to AVI retry schema: %v", err)
	}

	var probeStatus, playbackStatus, jobStatus string
	var probeError, jobError sql.NullString
	var attempts int
	if err := db.QueryRowContext(ctx, `
		SELECT asset.probe_status, asset.probe_error_code,
		       asset.playback_status, job.status, job.last_error_code,
		       job.attempt_count
		FROM assets AS asset
		JOIN media_jobs AS job ON job.asset_id = asset.id
		WHERE asset.id = 1 AND job.variant = 'grid'
	`).Scan(
		&probeStatus,
		&probeError,
		&playbackStatus,
		&jobStatus,
		&jobError,
		&attempts,
	); err != nil {
		t.Fatal(err)
	}
	if probeStatus != "pending" || probeError.Valid ||
		playbackStatus != "unknown" || jobStatus != "queued" ||
		jobError.Valid || attempts != 0 {
		t.Fatalf(
			"AVI retry state = (%q, %v, %q, %q, %v, %d)",
			probeStatus,
			probeError,
			playbackStatus,
			jobStatus,
			jobError,
			attempts,
		)
	}
}
