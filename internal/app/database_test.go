package app

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/api"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
	_ "modernc.org/sqlite"
)

func TestDatabaseComponentStartsFromEmptyDataRootAndRestarts(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "empty-data")

	for attempt := 0; attempt < 2; attempt++ {
		readiness := newReadinessState()
		component, _ := newDatabaseComponent(dataRoot, readiness)
		if err := component.start(context.Background()); err != nil {
			t.Fatalf("start attempt %d: %v", attempt+1, err)
		}
		if got := readiness.snapshot(); got.Ready || got.ReasonCode != api.ReadinessDatabaseUnavailable {
			t.Fatalf("readiness before lifecycle attempt %d = %#v, want unavailable", attempt+1, got)
		}
		readyComponent := readinessLifecycle(readiness)
		if err := readyComponent.start(context.Background()); err != nil {
			t.Fatalf("mark ready attempt %d: %v", attempt+1, err)
		}
		if got := readiness.snapshot(); !got.Ready || got.ReasonCode != "" {
			t.Fatalf("readiness attempt %d = %#v, want ready", attempt+1, got)
		}
		if err := readyComponent.stop(context.Background()); err != nil {
			t.Fatalf("stop readiness attempt %d: %v", attempt+1, err)
		}
		if err := component.stop(context.Background()); err != nil {
			t.Fatalf("stop attempt %d: %v", attempt+1, err)
		}
	}

	for _, path := range []string{
		dataRoot,
		filepath.Join(dataRoot, "cache"),
		filepath.Join(dataRoot, "tmp"),
		filepath.Join(dataRoot, databaseFilename),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected startup path %q: %v", filepath.Base(path), err)
		}
	}

	inspector := openDatabaseInspector(t, filepath.Join(dataRoot, databaseFilename))
	defer inspector.Close()
	for _, table := range []string{"libraries", "scan_runs", "directories", "assets", "goose_db_version"} {
		var count int
		if err := inspector.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
			table,
		).Scan(&count); err != nil {
			t.Fatalf("inspect table %q: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %q count = %d, want 1", table, count)
		}
	}
	var appliedInitialMigration int
	if err := inspector.QueryRow(
		`SELECT COUNT(*) FROM goose_db_version WHERE version_id = 1 AND is_applied = 1`,
	).Scan(&appliedInitialMigration); err != nil {
		t.Fatalf("inspect applied migration: %v", err)
	}
	if appliedInitialMigration != 1 {
		t.Fatalf("applied initial migration rows = %d, want 1", appliedInitialMigration)
	}
}

func TestDatabaseComponentClassifiesMigrationFailureAndDoesNotBecomeReady(t *testing.T) {
	dataRoot := t.TempDir()
	databasePath := filepath.Join(dataRoot, databaseFilename)
	inspector := openDatabaseInspector(t, databasePath)
	if _, err := inspector.Exec(`CREATE TABLE libraries (conflict INTEGER)`); err != nil {
		t.Fatalf("create incompatible schema: %v", err)
	}
	if err := inspector.Close(); err != nil {
		t.Fatalf("close incompatible database: %v", err)
	}

	readiness := newReadinessState()
	component, service := newDatabaseComponent(dataRoot, readiness)
	err := component.start(context.Background())
	if !errors.Is(err, sqlitestore.ErrMigration) {
		t.Fatalf("start error = %v, want ErrMigration", err)
	}
	if got := readiness.snapshot(); got.Ready || got.ReasonCode != api.ReadinessMigrationFailed {
		t.Fatalf("readiness = %#v, want migration failure", got)
	}
	if service.store != nil {
		t.Fatal("failed migration retained an open application store")
	}
}

func TestDatabaseComponentClassifiesUnavailableApplicationData(t *testing.T) {
	base := t.TempDir()
	dataRoot := filepath.Join(base, "not-a-directory")
	if err := os.WriteFile(dataRoot, []byte("occupied"), 0o600); err != nil {
		t.Fatalf("create blocking file: %v", err)
	}

	readiness := newReadinessState()
	component, _ := newDatabaseComponent(dataRoot, readiness)
	if err := component.start(context.Background()); err == nil {
		t.Fatal("start unexpectedly succeeded with a file as the data root")
	}
	if got := readiness.snapshot(); got.Ready || got.ReasonCode != api.ReadinessApplicationData {
		t.Fatalf("readiness = %#v, want application data unavailable", got)
	}
}

func openDatabaseInspector(t *testing.T, filename string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatalf("open database inspector: %v", err)
	}
	if err := database.Ping(); err != nil {
		_ = database.Close()
		t.Fatalf("ping database inspector: %v", err)
	}
	return database
}
