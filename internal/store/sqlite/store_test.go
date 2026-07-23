package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func openTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "foliopath-test.db")
	store, err := Open(context.Background(), filename, Options{
		BusyTimeout:        2 * time.Second,
		MaxOpenConnections: 4,
		MaxBatchSize:       16,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return store, filename
}

func TestOpenUsesFileDatabaseAndAppliesPragmasToEveryConnection(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()

	connections := make([]interface{ Close() error }, 0, 3)
	for index := 0; index < 3; index++ {
		connection, err := store.db.Conn(ctx)
		if err != nil {
			t.Fatalf("Conn(%d) error = %v", index, err)
		}
		connections = append(connections, connection)

		var version, journalMode string
		var foreignKeys, busyTimeout, synchronous int
		if err := connection.QueryRowContext(ctx, `SELECT sqlite_version()`).Scan(&version); err != nil {
			t.Fatalf("sqlite_version connection %d: %v", index, err)
		}
		if version == "" {
			t.Fatalf("sqlite_version connection %d is empty", index)
		}
		if err := connection.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
			t.Fatalf("journal_mode connection %d: %v", index, err)
		}
		if journalMode != "wal" {
			t.Fatalf("journal_mode connection %d = %q, want wal", index, journalMode)
		}
		if err := connection.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
			t.Fatalf("foreign_keys connection %d: %v", index, err)
		}
		if foreignKeys != 1 {
			t.Fatalf("foreign_keys connection %d = %d, want 1", index, foreignKeys)
		}
		if err := connection.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
			t.Fatalf("busy_timeout connection %d: %v", index, err)
		}
		if busyTimeout != 2000 {
			t.Fatalf("busy_timeout connection %d = %d, want 2000", index, busyTimeout)
		}
		if err := connection.QueryRowContext(ctx, `PRAGMA synchronous`).Scan(&synchronous); err != nil {
			t.Fatalf("synchronous connection %d: %v", index, err)
		}
		if synchronous != 1 {
			t.Fatalf("synchronous connection %d = %d, want NORMAL (1)", index, synchronous)
		}
		if index == 0 {
			t.Logf("SQLite runtime version: %s", version)
		}
	}
	for index := len(connections) - 1; index >= 0; index-- {
		if err := connections[index].Close(); err != nil {
			t.Errorf("close connection %d: %v", index, err)
		}
	}
}

func TestHasWALResetFix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{version: "3.53.3", want: true},
		{version: "3.51.3", want: true},
		{version: "3.51.2", want: false},
		{version: "3.50.7", want: true},
		{version: "3.50.6", want: false},
		{version: "3.44.6", want: true},
		{version: "3.44.5", want: false},
		{version: "3.49.99", want: false},
		{version: "invalid", want: false},
	}
	for _, test := range tests {
		test := test
		t.Run(test.version, func(t *testing.T) {
			t.Parallel()
			if got := hasWALResetFix(test.version); got != test.want {
				t.Fatalf("hasWALResetFix(%q) = %v, want %v", test.version, got, test.want)
			}
		})
	}
}

func TestDatabaseIntegrityForeignKeysAndCheckpoint(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()

	var integrity string
	if err := store.db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatalf("integrity_check error = %v", err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity_check = %q, want ok", integrity)
	}

	rows, err := store.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatalf("foreign_key_check error = %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported a violation")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("foreign_key_check iteration error = %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close foreign_key_check rows: %v", err)
	}

	var busy, logFrames, checkpointedFrames int
	if err := store.db.QueryRowContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`).Scan(
		&busy,
		&logFrames,
		&checkpointedFrames,
	); err != nil {
		t.Fatalf("wal_checkpoint error = %v", err)
	}
	if busy != 0 {
		t.Fatalf("wal_checkpoint busy = %d, want 0 (log=%d checkpointed=%d)",
			busy, logFrames, checkpointedFrames)
	}
}

func TestConcurrentFirstOpenSerializesMigrationInitialization(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "concurrent-open.db")
	start := make(chan struct{})
	stores := make(chan *Store, 4)
	errorsFound := make(chan error, 4)
	var wait sync.WaitGroup
	for range 4 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			store, err := Open(context.Background(), filename, Options{MaxOpenConnections: 2})
			if err != nil {
				errorsFound <- err
				return
			}
			stores <- store
		}()
	}
	close(start)
	wait.Wait()
	close(stores)
	close(errorsFound)
	for err := range errorsFound {
		t.Errorf("concurrent Open error = %v", err)
	}
	opened := 0
	for store := range stores {
		opened++
		if err := store.Close(); err != nil {
			t.Errorf("close concurrent Store: %v", err)
		}
	}
	if opened != 4 {
		t.Fatalf("concurrent stores opened = %d, want 4", opened)
	}
}
