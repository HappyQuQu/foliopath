package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestLibraryContractMigrationUpgradesAuthenticationSchema(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-two.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatalf("open version-two database: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatalf("create migration provider: %v", err)
	}
	if _, err := provider.UpTo(ctx, 2); err != nil {
		t.Fatalf("apply version-two schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close version-two database: %v", err)
	}

	store, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatalf("upgrade version-two database: %v", err)
	}
	defer store.Close()

	if columns := tableColumns(t, store, "libraries"); columns["revision"] != "INTEGER" {
		t.Fatalf("libraries.revision type = %q, want INTEGER", columns["revision"])
	}
	for _, table := range []string{"library_removals", "idempotency_records"} {
		if columns := tableColumns(t, store, table); len(columns) == 0 {
			t.Errorf("upgraded database has no %s columns", table)
		}
	}
	var version int64
	if err := store.db.QueryRowContext(
		ctx,
		`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1`,
	).Scan(&version); err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != 11 {
		t.Fatalf("migration version = %d, want 11", version)
	}
}

func TestLibraryCreationContractCommitsLibraryScanAndIdempotencyTogether(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()

	insert := func(tx *sql.Tx) (int64, error) {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO libraries(
				name, name_key, root_rel_path, status,
				current_generation, created_at_ms, updated_at_ms
			) VALUES ('Family', 'family', 'family', 'pending', 0, 1000, 1000)`)
		if err != nil {
			return 0, err
		}
		libraryID, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		scanResult, err := tx.ExecContext(ctx, `
			INSERT INTO scan_runs(
				library_id, generation, trigger_kind, status, created_at_ms
			) VALUES (?, 1, 'library_created', 'queued', 1000)`, libraryID)
		if err != nil {
			return 0, err
		}
		scanID, err := scanResult.LastInsertId()
		if err != nil {
			return 0, err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO idempotency_records(
				operation, key_hash, request_hash, result_kind, result_id,
				created_at_ms, expires_at_ms
			) VALUES ('create_library', ?, ?, 'library', ?, 1000, 86401000)`,
			bytes.Repeat([]byte{0x11}, 32),
			bytes.Repeat([]byte{0x22}, 32),
			libraryID,
		)
		if err != nil {
			return 0, err
		}
		return scanID, nil
	}

	rolledBack, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin rollback transaction: %v", err)
	}
	if _, err := insert(rolledBack); err != nil {
		t.Fatalf("insert rollback fixture: %v", err)
	}
	if err := rolledBack.Rollback(); err != nil {
		t.Fatalf("rollback creation transaction: %v", err)
	}
	for _, table := range []string{"libraries", "scan_runs", "idempotency_records"} {
		if count := tableRowCount(t, store, table); count != 0 {
			t.Fatalf("%s rows after rollback = %d, want 0", table, count)
		}
	}

	committed, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin commit transaction: %v", err)
	}
	if _, err := insert(committed); err != nil {
		t.Fatalf("insert committed fixture: %v", err)
	}
	if err := committed.Commit(); err != nil {
		t.Fatalf("commit creation transaction: %v", err)
	}
	for _, table := range []string{"libraries", "scan_runs", "idempotency_records"} {
		if count := tableRowCount(t, store, table); count != 1 {
			t.Fatalf("%s rows after commit = %d, want 1", table, count)
		}
	}

	var libraryID int64
	if err := store.db.QueryRowContext(ctx, `SELECT id FROM libraries`).Scan(&libraryID); err != nil {
		t.Fatalf("read committed library: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO scan_runs(
			library_id, generation, trigger_kind, status, created_at_ms
		) VALUES (?, 2, 'library_created', 'failed', 2000)`, libraryID); err == nil {
		t.Fatal("a second library-created scan was accepted")
	}
}

func TestLibraryRemovalAndIdempotencyConstraints(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()

	result, err := store.db.ExecContext(ctx, `
		INSERT INTO libraries(
			name, name_key, root_rel_path, status,
			current_generation, created_at_ms, updated_at_ms
		) VALUES ('Archive', 'archive', 'archive', 'ready', 1, 1000, 1000)`)
	if err != nil {
		t.Fatalf("insert library: %v", err)
	}
	libraryID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read library ID: %v", err)
	}

	_, err = store.db.ExecContext(ctx, `
		INSERT INTO library_removals(
			id, library_id, library_name, status, created_at_ms
		) VALUES (10, ?, 'Archive', 'queued', 2000)`, libraryID)
	if err != nil {
		t.Fatalf("insert queued removal: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO library_removals(
			id, library_id, library_name, status, created_at_ms
		) VALUES (11, ?, 'Archive', 'queued', 2001)`, libraryID); err == nil {
		t.Fatal("a second active removal was accepted")
	}

	keyHash := bytes.Repeat([]byte{0x33}, 32)
	requestHash := bytes.Repeat([]byte{0x44}, 32)
	_, err = store.db.ExecContext(ctx, `
		INSERT INTO idempotency_records(
			operation, key_hash, request_hash, result_kind, result_id,
			created_at_ms, expires_at_ms
		) VALUES ('remove_library', ?, ?, 'library_removal', 10, 2000, 86402000)`,
		keyHash,
		requestHash,
	)
	if err != nil {
		t.Fatalf("insert idempotency record: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO idempotency_records(
			operation, key_hash, request_hash, result_kind, result_id,
			created_at_ms, expires_at_ms
		) VALUES ('remove_library', ?, ?, 'library_removal', 10, 2001, 86402001)`,
		keyHash,
		requestHash,
	); err == nil {
		t.Fatal("a duplicate operation/idempotency digest was accepted")
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO idempotency_records(
			operation, key_hash, request_hash, result_kind, result_id,
			created_at_ms, expires_at_ms
		) VALUES ('create_library', ?, ?, 'library', ?, 2000, 86402000)`,
		[]byte("short"),
		requestHash,
		libraryID,
	); err == nil {
		t.Fatal("a non-SHA-256 idempotency digest was accepted")
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO idempotency_records(
			operation, key_hash, request_hash, result_kind, result_id,
			created_at_ms, expires_at_ms
		) VALUES ('create_library', ?, ?, 'library', ?, 2000, 86401999)`,
		bytes.Repeat([]byte{0x55}, 32),
		requestHash,
		libraryID,
	); err == nil {
		t.Fatal("an idempotency record retained for less than 24 hours was accepted")
	}

	if _, err := store.db.ExecContext(ctx, `DELETE FROM libraries WHERE id = ?`, libraryID); err != nil {
		t.Fatalf("delete library configuration: %v", err)
	}
	if count := tableRowCount(t, store, "library_removals"); count != 1 {
		t.Fatalf("removal rows after library deletion = %d, want 1", count)
	}
	if count := tableRowCount(t, store, "idempotency_records"); count != 1 {
		t.Fatalf("idempotency rows after library deletion = %d, want 1", count)
	}

	columns := tableColumns(t, store, "idempotency_records")
	for _, forbidden := range []string{"key", "idempotency_key", "request_body"} {
		if _, exists := columns[forbidden]; exists {
			t.Errorf("idempotency_records exposes forbidden plaintext column %q", forbidden)
		}
	}
}

func tableRowCount(t *testing.T, store *Store, table string) int {
	t.Helper()
	var count int
	if err := store.db.QueryRowContext(
		context.Background(),
		`SELECT COUNT(*) FROM `+table,
	).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", table, err)
	}
	return count
}
