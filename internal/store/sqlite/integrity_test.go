package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestIntegrityCheckIsReadOnlyAndSharesLiveWAL(t *testing.T) {
	store, filename := openTestStore(t)
	ctx := context.Background()
	createWorkerLibrary(t, store, "Synthetic", "synthetic")
	var workers sync.WaitGroup
	failures := make(chan error, 2)
	workers.Add(2)
	go func() {
		defer workers.Done()
		for i := 0; i < 30; i++ {
			if _, err := store.RecoverExpiredFullScans(ctx); err != nil {
				failures <- err
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 30; i++ {
			if err := CheckIntegrity(ctx, filename); err != nil {
				failures <- err
				return
			}
		}
	}()
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	db, err := sql.Open("sqlite", integrityDSN(filename))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "CREATE TABLE must_not_write(id INTEGER)"); err == nil {
		t.Fatal("checker DSN allows writes")
	}
	var version int
	if err := store.db.QueryRowContext(ctx, "SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1").Scan(&version); err != nil || version != 47 {
		t.Fatalf("migration changed: %d %v", version, err)
	}
}

func TestIntegrityCheckFailsClosedWithoutCreatingOrDisclosing(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "missing.db")
	if err := CheckIntegrity(context.Background(), filename); err != ErrIntegrityCheck {
		t.Fatalf("missing: %v", err)
	}
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Fatal("missing DB was created")
	}
	if err := os.WriteFile(filename, []byte("synthetic corrupt database with secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CheckIntegrity(context.Background(), filename); err != ErrIntegrityCheck {
		t.Fatalf("corrupt: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := CheckIntegrity(ctx, filename); err != ErrIntegrityCheck {
		t.Fatalf("cancelled: %v", err)
	}
}

func TestIntegrityCheckRejectsForeignKeyViolations(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "synthetic.db")
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE parent(id INTEGER PRIMARY KEY); CREATE TABLE child(id INTEGER REFERENCES parent(id)); INSERT INTO child VALUES(123)"); err != nil {
		t.Fatal(err)
	}
	if err := CheckIntegrity(context.Background(), filename); err != ErrIntegrityCheck {
		t.Fatalf("foreign key violation: %v", err)
	}
}
