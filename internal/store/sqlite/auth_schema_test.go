package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestAuthenticationMigrationUpgradesThePreviousSchema(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "version-one.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatalf("open version-one database: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatalf("create migration provider: %v", err)
	}
	if _, err := provider.UpTo(ctx, 1); err != nil {
		t.Fatalf("apply version-one schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close version-one database: %v", err)
	}

	store, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatalf("upgrade version-one database: %v", err)
	}
	defer store.Close()

	for _, table := range []string{"users", "sessions"} {
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

func TestAuthenticationMigrationEnforcesSingleAdministrator(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()

	start := make(chan struct{})
	results := make(chan error, 8)
	var wait sync.WaitGroup
	for candidate := 1; candidate <= 8; candidate++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := store.db.ExecContext(
				ctx,
				`INSERT INTO users (
					id, username, username_key, display_name,
					password_hash, password_scheme, password_parameters,
					created_at_ms, updated_at_ms, password_changed_at_ms
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				candidate,
				fmt.Sprintf("Administrator%d", candidate),
				fmt.Sprintf("administrator%d", candidate),
				fmt.Sprintf("Administrator %d", candidate),
				"encoded-password-verifier",
				"contract-test",
				"version=1",
				int64(1000),
				int64(1000),
				int64(1000),
			)
			results <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent administrator inserts succeeded = %d, want 1", successes)
	}

	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		t.Fatalf("count administrators: %v", err)
	}
	if count != 1 {
		t.Fatalf("administrator rows = %d, want 1", count)
	}
}

func TestAuthenticationMigrationStoresOnlySessionTokenDigests(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()

	_, err := store.db.ExecContext(
		ctx,
		`INSERT INTO users (
			id, username, username_key, display_name,
			password_hash, password_scheme, password_parameters,
			created_at_ms, updated_at_ms, password_changed_at_ms
		) VALUES (1, 'Administrator', 'administrator', 'Administrator',
			'encoded-password-verifier', 'contract-test', 'version=1',
			1000, 1000, 1000)`,
	)
	if err != nil {
		t.Fatalf("insert administrator: %v", err)
	}

	tokenHash := bytes.Repeat([]byte{0x11}, 32)
	csrfTokenHash := bytes.Repeat([]byte{0x22}, 32)
	_, err = store.db.ExecContext(
		ctx,
		`INSERT INTO sessions (
			id, user_id, token_hash, csrf_token_hash, auth_version,
			created_at_ms, last_seen_at_ms, expires_at_ms
		) VALUES (1, 1, ?, ?, 1, 1000, 1000, 2000)`,
		tokenHash,
		csrfTokenHash,
	)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	var storedTokenHash, storedCSRFTokenHash []byte
	if err := store.db.QueryRowContext(
		ctx,
		`SELECT token_hash, csrf_token_hash FROM sessions WHERE id = 1`,
	).Scan(&storedTokenHash, &storedCSRFTokenHash); err != nil {
		t.Fatalf("read session digests: %v", err)
	}
	if !bytes.Equal(storedTokenHash, tokenHash) || !bytes.Equal(storedCSRFTokenHash, csrfTokenHash) {
		t.Fatal("stored session digests differ")
	}

	if _, err := store.db.ExecContext(
		ctx,
		`INSERT INTO sessions (
			id, user_id, token_hash, csrf_token_hash, auth_version,
			created_at_ms, last_seen_at_ms, expires_at_ms
		) VALUES (2, 1, ?, ?, 1, 1000, 1000, 2000)`,
		tokenHash,
		bytes.Repeat([]byte{0x33}, 32),
	); err == nil {
		t.Fatal("duplicate session token digest was accepted")
	}
	if _, err := store.db.ExecContext(
		ctx,
		`INSERT INTO sessions (
			id, user_id, token_hash, csrf_token_hash, auth_version,
			created_at_ms, last_seen_at_ms, expires_at_ms
		) VALUES (3, 1, ?, ?, 1, 1000, 1000, 1000)`,
		bytes.Repeat([]byte{0x44}, 32),
		bytes.Repeat([]byte{0x55}, 32),
	); err == nil {
		t.Fatal("session without a positive lifetime was accepted")
	}

	columns := tableColumns(t, store, "sessions")
	for _, digestColumn := range []string{"token_hash", "csrf_token_hash"} {
		if columns[digestColumn] != "BLOB" {
			t.Errorf("%s type = %q, want BLOB", digestColumn, columns[digestColumn])
		}
	}
	for _, forbidden := range []string{"token", "csrf_token", "password"} {
		if _, exists := columns[forbidden]; exists {
			t.Errorf("sessions table exposes forbidden plaintext column %q", forbidden)
		}
	}
	userColumns := tableColumns(t, store, "users")
	if userColumns["password_hash"] != "TEXT" {
		t.Errorf("password_hash type = %q, want TEXT verifier", userColumns["password_hash"])
	}
	for _, forbidden := range []string{"password", "password_plaintext"} {
		if _, exists := userColumns[forbidden]; exists {
			t.Errorf("users table exposes forbidden plaintext column %q", forbidden)
		}
	}

	if _, err := store.db.ExecContext(ctx, `DELETE FROM users WHERE id = 1`); err != nil {
		t.Fatalf("delete administrator: %v", err)
	}
	var remaining int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&remaining); err != nil {
		t.Fatalf("count sessions after administrator removal: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("sessions after administrator removal = %d, want 0", remaining)
	}
}

func tableColumns(t *testing.T, store *Store, table string) map[string]string {
	t.Helper()
	rows, err := store.db.QueryContext(
		context.Background(),
		`SELECT name, type FROM pragma_table_info(?)`,
		table,
	)
	if err != nil {
		t.Fatalf("inspect table %s: %v", table, err)
	}
	defer rows.Close()

	columns := make(map[string]string)
	for rows.Next() {
		var name, columnType string
		if err := rows.Scan(&name, &columnType); err != nil {
			t.Fatalf("scan table %s column: %v", table, err)
		}
		columns[name] = strings.ToUpper(columnType)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table %s columns: %v", table, err)
	}
	return columns
}
