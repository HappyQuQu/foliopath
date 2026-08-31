package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestVectorBackends(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite"} {
		report, err := BenchmarkVectors(backend, 100, 16, 2, 5, 42, 2)
		if err != nil {
			t.Fatalf("%s: %v", backend, err)
		}
		if report.QueryP99MS < 0 || report.BuildMS < 0 {
			t.Fatalf("%s returned negative duration", backend)
		}
	}
}

func TestVectorArgumentsFailClosed(t *testing.T) {
	if _, err := BenchmarkVectors("memory", 1, 8, 1, 2, 42, 1); err == nil {
		t.Fatal("expected top-k larger than item count to fail")
	}
}

func TestVectorConcurrencyCancellationAndRestart(t *testing.T) {
	for _, storageFormat := range []string{"float32", "float16"} {
		report, err := BenchmarkVectorConcurrency(
			200, 16, 25, 5, 2, 42, time.Millisecond, storageFormat,
		)
		if err != nil {
			t.Fatal(err)
		}
		if report.RowsAfterCancellation < 100 || report.RowsAfterCancellation >= 200 ||
			report.RowsAfterRestart != 200 || report.DatabaseBytes <= 0 {
			t.Fatalf("concurrency report = %#v", report)
		}
	}
}

func TestSQLiteVectorGenerationStrongKillRecovery(t *testing.T) {
	if os.Getenv("INT001_VECTOR_KILL_HELPER") == "1" {
		runVectorKillHelper(t)
		return
	}
	databasePath := filepath.Join(t.TempDir(), "vectors.db")
	db := openVectorRecoveryDatabase(t, databasePath)
	insertVectorGeneration(t, db, 1, 16)
	if _, err := db.Exec(`INSERT INTO vector_active(singleton, generation) VALUES (1, 1)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	marker := databasePath + ".ready"
	helper := exec.Command(
		os.Args[0],
		"-test.run=^TestSQLiteVectorGenerationStrongKillRecovery$",
	)
	helper.Env = append(os.Environ(),
		"INT001_VECTOR_KILL_HELPER=1",
		"INT001_VECTOR_KILL_DATABASE="+databasePath,
		"INT001_VECTOR_KILL_MARKER="+marker,
	)
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = helper.Process.Kill()
			_, _ = helper.Process.Wait()
			t.Fatal("vector helper did not reach the uncommitted generation boundary")
		}
		time.Sleep(time.Millisecond)
	}
	if err := helper.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := helper.Wait(); err == nil {
		t.Fatal("strong-killed vector helper exited successfully")
	}

	db = openVectorRecoveryDatabase(t, databasePath)
	defer db.Close()
	assertVectorGenerationState(t, db, 1, 16, 2, 0)
	var integrity string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity check = %q, %v", integrity, err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := insertVectorRows(context.Background(), tx, 2, 32); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if _, err := tx.Exec(`UPDATE vector_active SET generation = 2 WHERE singleton = 1`); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if _, err := tx.Exec(`DELETE FROM vector_rows WHERE generation = 1`); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	assertVectorGenerationState(t, db, 2, 32, 1, 0)
}

func runVectorKillHelper(t *testing.T) {
	t.Helper()
	db := openVectorRecoveryDatabase(t, os.Getenv("INT001_VECTOR_KILL_DATABASE"))
	defer db.Close()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := insertVectorRows(context.Background(), tx, 2, 32); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`UPDATE vector_active SET generation = 2 WHERE singleton = 1`); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("INT001_VECTOR_KILL_MARKER"), []byte("ready"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {}
}

func openVectorRecoveryDatabase(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=FULL;
		PRAGMA busy_timeout=5000;
		CREATE TABLE IF NOT EXISTS vector_rows (
			generation INTEGER NOT NULL,
			asset_id INTEGER NOT NULL,
			vector BLOB NOT NULL,
			PRIMARY KEY (generation, asset_id)
		);
		CREATE TABLE IF NOT EXISTS vector_active (
			singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
			generation INTEGER NOT NULL
		);
	`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func insertVectorGeneration(t *testing.T, db *sql.DB, generation, count int) {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := insertVectorRows(context.Background(), tx, generation, count); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func insertVectorRows(ctx context.Context, tx *sql.Tx, generation, count int) error {
	statement, err := tx.PrepareContext(
		ctx,
		`INSERT INTO vector_rows(generation, asset_id, vector) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer statement.Close()
	for assetID := 0; assetID < count; assetID++ {
		vector := []float32{float32(generation), float32(assetID), 1, -1}
		if _, err := statement.ExecContext(ctx, generation, assetID, encodeVector(vector)); err != nil {
			return fmt.Errorf("insert vector %s: %w", strconv.Itoa(assetID), err)
		}
	}
	return nil
}

func assertVectorGenerationState(
	t *testing.T,
	db *sql.DB,
	wantActive, wantActiveRows, otherGeneration, wantOtherRows int,
) {
	t.Helper()
	var active int
	if err := db.QueryRow(`SELECT generation FROM vector_active WHERE singleton = 1`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != wantActive {
		t.Fatalf("active generation = %d, want %d", active, wantActive)
	}
	for generation, want := range map[int]int{
		wantActive:      wantActiveRows,
		otherGeneration: wantOtherRows,
	} {
		var count int
		if err := db.QueryRow(
			`SELECT count(*) FROM vector_rows WHERE generation = ?`, generation,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("generation %d rows = %d, want %d", generation, count, want)
		}
	}
	var malformed int
	if err := db.QueryRow(
		`SELECT count(*) FROM vector_rows WHERE length(vector) != 16`,
	).Scan(&malformed); err != nil {
		t.Fatal(err)
	}
	if malformed != 0 {
		t.Fatalf("found %d malformed vector blobs", malformed)
	}
}
