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

func TestAutomaticDiscoverySchemaDefaultsAndStateConstraints(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()

	var enabled int
	if err := store.db.QueryRowContext(ctx, `
		SELECT automatic_discovery_enabled
		FROM settings
		WHERE singleton_key = 1
	`).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Fatalf("automatic_discovery_enabled = %d, want 1", enabled)
	}

	insertAutomaticDiscoveryLibrary(t, store.db, 1)
	var status string
	var errorCode sql.NullString
	var revision int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT automatic_discovery_status,
		       automatic_discovery_error_code,
		       content_revision
		FROM libraries
		WHERE id = 1
	`).Scan(&status, &errorCode, &revision); err != nil {
		t.Fatal(err)
	}
	if status != "disabled" || errorCode.Valid || revision != 1 {
		t.Fatalf(
			"default discovery state = (%q, %v, %d), want (disabled, NULL, 1)",
			status,
			errorCode,
			revision,
		)
	}

	if _, err := store.db.ExecContext(ctx, `
		UPDATE libraries
		SET automatic_discovery_status = 'active',
		    automatic_discovery_error_code = 'internal_error'
		WHERE id = 1
	`); err == nil {
		t.Fatal("active discovery state with an error unexpectedly accepted")
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE libraries
		SET automatic_discovery_status = 'degraded',
		    automatic_discovery_error_code = NULL
		WHERE id = 1
	`); err == nil {
		t.Fatal("degraded discovery state without an error unexpectedly accepted")
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE libraries
		SET automatic_discovery_status = 'degraded',
		    automatic_discovery_error_code = 'watch_overflow'
		WHERE id = 1
	`); err != nil {
		t.Fatalf("valid degraded discovery state: %v", err)
	}
}

func TestAutomaticDiscoveryJobWatermarkAndPathConstraints(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	insertAutomaticDiscoveryLibrary(t, store.db, 1)

	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO catalog_reconcile_jobs(
			library_id, relative_dir_path, available_at_ms,
			created_at_ms, updated_at_ms
		) VALUES (1, 'albums/2026', 10, 1, 1)
	`); err != nil {
		t.Fatalf("insert valid reconciliation job: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE catalog_reconcile_jobs
		SET status = 'running',
		    claimed_revision = requested_revision,
		    lease_expires_at_ms = 100,
		    attempt_count = 1,
		    updated_at_ms = 2
		WHERE library_id = 1 AND relative_dir_path = 'albums/2026'
	`); err != nil {
		t.Fatalf("claim valid reconciliation job: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE catalog_reconcile_jobs
		SET requested_revision = requested_revision + 1,
		    updated_at_ms = 3
		WHERE library_id = 1 AND relative_dir_path = 'albums/2026'
	`); err != nil {
		t.Fatalf("record event during running claim: %v", err)
	}
	var requested, claimed int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT requested_revision, claimed_revision
		FROM catalog_reconcile_jobs
		WHERE library_id = 1 AND relative_dir_path = 'albums/2026'
	`).Scan(&requested, &claimed); err != nil {
		t.Fatal(err)
	}
	if requested != 2 || claimed != 1 {
		t.Fatalf("job watermarks = requested %d claimed %d, want 2/1", requested, claimed)
	}

	for _, path := range []string{"/absolute", "../escape", "a/../escape", "a//b", "trailing/"} {
		if _, err := store.db.ExecContext(ctx, `
			INSERT INTO catalog_reconcile_jobs(
				library_id, relative_dir_path, available_at_ms,
				created_at_ms, updated_at_ms
			) VALUES (1, ?, 10, 1, 1)
		`, path); err == nil {
			t.Errorf("unsafe reconciliation path %q unexpectedly accepted", path)
		}
	}
}

func TestAutomaticDiscoverySerializesFullAndTargetedWork(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	insertAutomaticDiscoveryLibrary(t, store.db, 1)
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO scan_runs(
			library_id, generation, trigger_kind, status, phase,
			created_at_ms, available_at_ms
		) VALUES (1, 1, 'manual', 'queued', 'queued', 1, 1);
		INSERT INTO catalog_reconcile_jobs(
			library_id, relative_dir_path, available_at_ms,
			created_at_ms, updated_at_ms
		) VALUES (1, '', 1, 1, 1);
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE catalog_reconcile_jobs
		SET status = 'running', claimed_revision = requested_revision,
		    lease_expires_at_ms = 100, attempt_count = 1
		WHERE library_id = 1
	`); err == nil {
		t.Fatal("reconciliation unexpectedly claimed while a full scan is queued")
	}

	if _, err := store.db.ExecContext(ctx, `
		UPDATE scan_runs
		SET status = 'cancelled', phase = 'completed', finished_at_ms = 2
		WHERE library_id = 1
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE catalog_reconcile_jobs
		SET status = 'running', claimed_revision = requested_revision,
		    lease_expires_at_ms = 100, attempt_count = 1
		WHERE library_id = 1
	`); err != nil {
		t.Fatalf("claim reconciliation after full scan became terminal: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO scan_runs(
			library_id, generation, trigger_kind, status, phase,
			created_at_ms, available_at_ms
		) VALUES (1, 2, 'scheduled', 'queued', 'queued', 3, 3)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE scan_runs
		SET status = 'running', phase = 'checking_root',
		    started_at_ms = 3, heartbeat_at_ms = 3,
		    lease_expires_at_ms = 100
		WHERE library_id = 1 AND generation = 2
	`); err == nil {
		t.Fatal("full scan unexpectedly claimed while reconciliation is running")
	}
}

func TestAutomaticDiscoveryContentRevisionIsIndependentFromSearchRevision(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	insertAutomaticDiscoveryLibrary(t, store.db, 1)

	var searchBefore, contentBefore int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT revision, content_revision
		FROM catalog_search_state
		WHERE singleton_key = 1
	`).Scan(&searchBefore, &contentBefore); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE libraries
		SET content_revision = content_revision + 1
		WHERE id = 1;
		UPDATE catalog_search_state
		SET content_revision = content_revision + 1
		WHERE singleton_key = 1;
	`); err != nil {
		t.Fatal(err)
	}
	var searchAfter, contentAfter int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT revision, content_revision
		FROM catalog_search_state
		WHERE singleton_key = 1
	`).Scan(&searchAfter, &contentAfter); err != nil {
		t.Fatal(err)
	}
	if searchAfter != searchBefore || contentAfter != contentBefore+1 {
		t.Fatalf(
			"revisions after targeted commit = search %d content %d, want %d/%d",
			searchAfter,
			contentAfter,
			searchBefore,
			contentBefore+1,
		)
	}
}

func TestAutomaticDiscoveryMigrationPreservesVersionElevenState(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-eleven.db")
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
	if _, err := provider.UpTo(ctx, 11); err != nil {
		t.Fatalf("apply version-eleven schema: %v", err)
	}
	insertAutomaticDiscoveryLibrary(t, db, 1)
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("upgrade to automatic discovery schema: %v", err)
	}

	var name, status string
	var enabled, revision int64
	if err := db.QueryRowContext(ctx, `
		SELECT l.name, l.automatic_discovery_status,
		       s.automatic_discovery_enabled, l.content_revision
		FROM libraries l
		CROSS JOIN settings s
		WHERE l.id = 1 AND s.singleton_key = 1
	`).Scan(&name, &status, &enabled, &revision); err != nil {
		t.Fatal(err)
	}
	if name != "Archive" || status != "disabled" || enabled != 1 || revision != 1 {
		t.Fatalf(
			"upgraded state = %q/%q/%d/%d",
			name,
			status,
			enabled,
			revision,
		)
	}
	assertSQLiteIntegrity(t, db)
}

func insertAutomaticDiscoveryLibrary(t *testing.T, db *sql.DB, id int64) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO libraries(
			id, name, name_key, root_rel_path, status,
			current_generation, created_at_ms, updated_at_ms
		) VALUES (?, 'Archive', 'archive', 'archive', 'ready', 0, 1, 1)
	`, id); err != nil {
		t.Fatalf("insert automatic discovery library: %v", err)
	}
}
