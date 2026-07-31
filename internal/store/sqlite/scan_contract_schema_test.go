package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/store/sqlite/dbgen"
	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestScanContractMigrationUpgradesVersionThreeSchema(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-three.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 3); err != nil {
		t.Fatalf("apply version-three schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatalf("upgrade version-three database: %v", err)
	}
	defer store.Close()

	for _, column := range []string{
		"revision",
		"phase",
		"processed_assets",
		"skipped_directories",
		"skipped_files",
		"error_count",
		"issues_truncated",
		"cancel_requested_at_ms",
		"heartbeat_at_ms",
		"available_at_ms",
		"lease_expires_at_ms",
		"attempt_count",
	} {
		if _, ok := tableColumns(t, store, "scan_runs")[column]; !ok {
			t.Errorf("scan_runs is missing %s", column)
		}
	}
	for _, table := range []string{"scan_issues", "settings"} {
		if len(tableColumns(t, store, table)) == 0 {
			t.Errorf("upgraded database is missing %s", table)
		}
	}
	var version int64
	if err := store.db.QueryRowContext(
		ctx,
		`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1`,
	).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 16 {
		t.Fatalf("migration version = %d, want 16", version)
	}
}

func TestScanContractPersistsBoundedIssuesAndTypedSchedule(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()

	var interval, quota, revision int64
	var language string
	if err := store.db.QueryRowContext(ctx, `
		SELECT scheduled_scan_interval_hours, thumbnail_cache_quota_bytes, language, revision
		FROM settings
		WHERE singleton_key = 1`).Scan(&interval, &quota, &language, &revision); err != nil {
		t.Fatal(err)
	}
	if interval != 24 || quota != 10737418240 || language != "browser" || revision != 1 {
		t.Fatalf("default settings = %d, %d, %q, %d", interval, quota, language, revision)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE settings
		SET scheduled_scan_interval_hours = NULL, revision = revision + 1
		WHERE singleton_key = 1`); err != nil {
		t.Fatalf("disable scheduled scans: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE settings SET scheduled_scan_interval_hours = 0
		WHERE singleton_key = 1`); err == nil {
		t.Fatal("settings accepted a zero-hour interval")
	}

	result, err := store.db.ExecContext(ctx, `
		INSERT INTO libraries(
			name, name_key, root_rel_path, status,
			current_generation, created_at_ms, updated_at_ms
		) VALUES ('Family', 'family', 'family', 'pending', 0, 1000, 1000)`)
	if err != nil {
		t.Fatal(err)
	}
	libraryID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	result, err = store.db.ExecContext(ctx, `
		INSERT INTO scan_runs(
			library_id, generation, trigger_kind, status, phase,
			created_at_ms, available_at_ms
		) VALUES (?, 1, 'manual', 'queued', 'queued', 1000, 1000)`, libraryID)
	if err != nil {
		t.Fatal(err)
	}
	scanID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 50; index++ {
		if _, err := store.db.ExecContext(ctx, `
			INSERT INTO scan_issues(
				scan_run_id, code, issue_count, sample_rel_path, created_at_ms
			) VALUES (?, 'unsupported_file', 1, ?, 1000)`,
			scanID,
			fmt.Sprintf("sample-%02d", index),
		); err != nil {
			t.Fatalf("insert issue %d: %v", index, err)
		}
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO scan_issues(
			scan_run_id, code, issue_count, sample_rel_path, created_at_ms
		) VALUES (?, 'io_error', 1, 'overflow', 1000)`, scanID); err == nil {
		t.Fatal("scan issue limit accepted a fifty-first row")
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE scan_runs SET attempt_count = 4 WHERE id = ?`, scanID); err == nil {
		t.Fatal("scan run accepted more than three attempts")
	}
}

func TestScanContractQueriesClaimCancelAndRecoverDurably(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	queries := dbgen.New(store.db)

	libraryIDs := make([]int64, 2)
	for index, name := range []string{"First", "Second"} {
		result, err := store.db.ExecContext(ctx, `
			INSERT INTO libraries(
				name, name_key, root_rel_path, status,
				current_generation, created_at_ms, updated_at_ms
			) VALUES (?, ?, ?, 'pending', 0, 1000, 1000)`,
			name,
			name,
			name,
		)
		if err != nil {
			t.Fatal(err)
		}
		libraryIDs[index], err = result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
	}
	first, err := queries.InsertQueuedScan(ctx, dbgen.InsertQueuedScanParams{
		LibraryID:     libraryIDs[0],
		Generation:    1,
		TriggerKind:   "manual",
		CreatedAtMs:   1000,
		AvailableAtMs: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := queries.InsertQueuedScan(ctx, dbgen.InsertQueuedScanParams{
		LibraryID:     libraryIDs[1],
		Generation:    1,
		TriggerKind:   "manual",
		CreatedAtMs:   2000,
		AvailableAtMs: 2000,
	})
	if err != nil {
		t.Fatal(err)
	}

	claimed, err := queries.ClaimNextQueuedScan(ctx, dbgen.ClaimNextQueuedScanParams{
		NowMs:            sql.NullInt64{Int64: 3000, Valid: true},
		LeaseExpiresAtMs: sql.NullInt64{Int64: 123000, Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if claimed.ID != first.ID || claimed.Status != "running" ||
		claimed.Phase != "checking_root" || claimed.AttemptCount != 1 {
		t.Fatalf("first claim = %#v", claimed)
	}
	cancelled, err := queries.CancelQueuedScan(ctx, dbgen.CancelQueuedScanParams{
		NowMs: sql.NullInt64{Int64: 4000, Valid: true},
		ID:    second.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != "cancelled" || cancelled.Phase != "completed" {
		t.Fatalf("queued cancellation = %#v", cancelled)
	}

	for attempt, now := range []int64{124000, 246000, 368000} {
		recovered, err := queries.RecoverNextExpiredScan(ctx, now)
		if err != nil {
			t.Fatalf("recover attempt %d: %v", attempt+1, err)
		}
		if attempt < 2 {
			if recovered.Status != "queued" || recovered.Phase != "queued" {
				t.Fatalf("recovery %d = %#v", attempt+1, recovered)
			}
			claimed, err = queries.ClaimNextQueuedScan(ctx, dbgen.ClaimNextQueuedScanParams{
				NowMs:            sql.NullInt64{Int64: now, Valid: true},
				LeaseExpiresAtMs: sql.NullInt64{Int64: now + 120000, Valid: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			if claimed.AttemptCount != int64(attempt+2) {
				t.Fatalf("claim attempt count = %d", claimed.AttemptCount)
			}
		} else if recovered.Status != "interrupted" ||
			recovered.Phase != "completed" ||
			recovered.ErrorCode.String != "scan_interrupted" {
			t.Fatalf("final recovery = %#v", recovered)
		}
	}
}
