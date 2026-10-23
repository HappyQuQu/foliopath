package integration

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/library"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
	_ "modernc.org/sqlite"
)

const (
	semanticKillHelperEnv   = "FOLIOPATH_SEMANTIC_KILL_HELPER"
	semanticKillDatabaseEnv = "FOLIOPATH_SEMANTIC_KILL_DATABASE"
)

func TestSemanticBackfillRecoversCheckpointAfterProcessKill(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "semantic-backfill-kill.db")
	store, err := sqlitestore.Open(ctx, filename, sqlitestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	libraries, err := library.NewService(store)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	item, err := libraries.Create(ctx, "Semantic evidence", "semantic-evidence")
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	seedSemanticKillFixture(t, filename, item.ID)

	helperContext, cancelHelper := context.WithTimeout(ctx, 5*time.Second)
	defer cancelHelper()
	command := exec.CommandContext(helperContext, os.Args[0], "-test.run=^TestSemanticBackfillClaimKillHelper$")
	command.Env = append(os.Environ(), semanticKillHelperEnv+"=1", semanticKillDatabaseEnv+"="+filename)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		_ = command.Wait()
		t.Fatalf("read semantic claim helper: %v", err)
	}
	if strings.TrimSpace(line) != "CLAIMED:aij_semantic_kill" {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("unexpected semantic claim helper output %q", line)
	}
	if err := command.Process.Kill(); err != nil {
		_ = command.Wait()
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("semantic claim helper unexpectedly exited successfully")
	}

	time.Sleep(150 * time.Millisecond)
	restarted, err := sqlitestore.Open(ctx, filename, sqlitestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	summary, err := restarted.RecoverExpiredSemanticBackfills(ctx, time.Now().UTC())
	if err != nil || summary.Requeued != 1 || summary.Interrupted != 0 {
		t.Fatalf("post-kill recovery = %#v err=%v", summary, err)
	}
	reclaimed, found, err := restarted.ClaimSemanticBackfill(ctx, time.Now().UTC(), time.Minute)
	if err != nil || !found {
		t.Fatalf("post-kill claim = %#v found=%t err=%v", reclaimed, found, err)
	}
	if reclaimed.ID != "aij_semantic_kill" || reclaimed.AttemptCount != 2 || reclaimed.ClaimedRevision != 3 ||
		reclaimed.CheckpointID != 101 || reclaimed.CompletedItems != 1 || reclaimed.TotalItems != 2 {
		t.Fatalf("post-kill state = %#v", reclaimed)
	}
}

func TestSemanticBackfillClaimKillHelper(t *testing.T) {
	if os.Getenv(semanticKillHelperEnv) != "1" {
		return
	}
	store, err := sqlitestore.Open(context.Background(), os.Getenv(semanticKillDatabaseEnv), sqlitestore.Options{})
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	claimed, found, err := store.ClaimSemanticBackfill(context.Background(), time.Now().UTC(), 100*time.Millisecond)
	if err != nil || !found {
		fmt.Fprintf(os.Stdout, "ERROR:found=%t err=%v\n", found, err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stdout, "CLAIMED:%s\n", claimed.ID)
	for {
		time.Sleep(time.Hour)
	}
}

func seedSemanticKillFixture(t *testing.T, filename string, libraryID int64) {
	t.Helper()
	database, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Now().UTC().UnixMilli()
	if _, err := database.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	execute := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	execute(`
		INSERT INTO ai_models(
			id, purpose, package_id, version, architecture, content_hash, license_id,
			package_size_bytes, storage_mode, state, source_identity,
			availability_revision, created_at_ms, updated_at_ms
		) VALUES(
			'aim_semantic_kill', 'semantic_image_text', 'semantic-kill', '1.0.0', 'arm64',
			'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'Apache-2.0',
			1, 'managed', 'available', 'managed:semantic-kill', 1, ?, ?
		)`, now, now)
	execute(`
		INSERT INTO semantic_generations(
			id, model_id, transform_version, output_schema_version, index_format_version,
			embedding_dimension, state, created_at_ms, activated_at_ms, updated_at_ms
		) VALUES('aig_semantic_kill', 'aim_semantic_kill', 1, 1, 1, 768, 'active', ?, ?, ?)`, now, now, now)
	execute(`
		UPDATE ai_model_state SET active_model_id='aim_semantic_kill', active_generation_id='aig_semantic_kill'
		WHERE singleton_key=1`)
	execute(`
		INSERT INTO ai_library_settings(
			library_id, enabled, state, revision, coverage_revision, created_at_ms, updated_at_ms
		) VALUES(?, 1, 'building', 1, 1, ?, ?)`, libraryID, now, now)
	execute(`
		INSERT INTO semantic_library_progress(
			generation_id, library_id, eligible_count, completed_count, failed_count,
			stale_count, checkpoint_id, revision, updated_at_ms
		) VALUES('aig_semantic_kill', ?, 2, 1, 0, 0, 101, 2, ?)`, libraryID, now)
	execute(`
		INSERT INTO ai_model_operations(
			id, kind, state, phase, library_id, completed_items, total_items,
			revision, created_at_ms, updated_at_ms
		) VALUES('aio_semantic_kill', 'semantic_missing', 'queued', 'queued', ?, 1, 2, 1, ?, ?)`, libraryID, now, now)
	execute(`
		INSERT INTO semantic_jobs(
			id, library_id, generation_id, operation_id, mode, state, checkpoint_id,
			requested_revision, attempt_count, created_at_ms, updated_at_ms
		) VALUES(
			'aij_semantic_kill', ?, 'aig_semantic_kill', 'aio_semantic_kill', 'missing',
			'queued', 101, 1, 0, ?, ?
		)`, libraryID, now, now)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
