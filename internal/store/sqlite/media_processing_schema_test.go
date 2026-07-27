package sqlite

import (
	"context"
	"testing"
)

func TestMediaProcessingMigrationAddsScopedDerivedState(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	for _, column := range []string{
		"width", "height", "duration_ms", "probe_status",
		"probe_error_code", "playback_status",
	} {
		if _, found := tableColumns(t, store, "assets")[column]; !found {
			t.Fatalf("assets is missing %s", column)
		}
	}
	thumbnailColumns := tableColumns(t, store, "thumbnails")
	for _, column := range []string{
		"library_id", "asset_id", "variant", "source_fingerprint",
		"transform_version", "cache_rel_path", "status", "error_code",
		"width", "height", "byte_size", "created_at_ms", "last_accessed_at_ms",
	} {
		if _, found := thumbnailColumns[column]; !found {
			t.Fatalf("thumbnails is missing %s", column)
		}
	}
	for _, table := range []string{
		"media_jobs", "media_job_library_state",
		"media_job_queue_state", "cache_deletions",
	} {
		var name string
		if err := store.db.QueryRowContext(context.Background(), `
            SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
			table,
		).Scan(&name); err != nil {
			t.Fatalf("missing %s: %v", table, err)
		}
	}

	var assetID int64
	var fingerprint, probeStatus, playbackStatus string
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id, source_fingerprint, probe_status, playback_status
        FROM assets WHERE library_id = ? ORDER BY id LIMIT 1`,
		libraryID,
	).Scan(&assetID, &fingerprint, &probeStatus, &playbackStatus); err != nil {
		t.Fatal(err)
	}
	if probeStatus != "pending" || playbackStatus != "unknown" {
		t.Fatalf("backfilled state = %q, %q", probeStatus, playbackStatus)
	}
	var queuedJobs, transformVersion int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT count(*), min(transform_version) FROM media_jobs
        WHERE library_id = ? AND status = 'queued'`,
		libraryID,
	).Scan(&queuedJobs, &transformVersion); err != nil {
		t.Fatal(err)
	}
	if queuedJobs != 4 || transformVersion != 1 {
		t.Fatalf("queued media jobs = %d at transform %d", queuedJobs, transformVersion)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO thumbnails(
            library_id, asset_id, variant, source_fingerprint,
            transform_version, cache_rel_path, status,
            width, height, byte_size, created_at_ms, last_accessed_at_ms
        ) VALUES (?, ?, 'grid', ?, 1, ?, 'ready', 48, 32, 123, 1, 1)`,
		libraryID, assetID, fingerprint,
		"libraries/lib_1/aa/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.webp",
	); err != nil {
		t.Fatalf("insert ready thumbnail: %v", err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO thumbnails(
            library_id, asset_id, variant, source_fingerprint,
            transform_version, status
        ) VALUES (?, ?, 'grid', ?, 1, 'ready')`,
		libraryID, assetID, fingerprint,
	); err == nil {
		t.Fatal("incomplete ready thumbnail unexpectedly accepted")
	}
	if _, err := store.db.ExecContext(context.Background(), `
        DELETE FROM assets WHERE id = ?`,
		assetID,
	); err != nil {
		t.Fatal(err)
	}
	var remaining int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT COUNT(*) FROM thumbnails WHERE asset_id = ?`,
		assetID,
	).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("thumbnail rows after asset delete = %d", remaining)
	}
}
