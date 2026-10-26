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
	faceKillHelperEnv   = "FOLIOPATH_FACE_KILL_HELPER"
	faceKillDatabaseEnv = "FOLIOPATH_FACE_KILL_DATABASE"
)

func TestFaceAnalysisRecoversCheckpointAfterProcessKill(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "face-analysis-kill.db")
	store, err := sqlitestore.Open(ctx, filename, sqlitestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	libraries, err := library.NewService(store)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	item, err := libraries.Create(ctx, "Face evidence", "face-evidence")
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	seedFaceKillFixture(t, filename, item.ID)

	helperContext, cancelHelper := context.WithTimeout(ctx, 5*time.Second)
	defer cancelHelper()
	command := exec.CommandContext(helperContext, os.Args[0], "-test.run=^TestFaceAnalysisClaimKillHelper$")
	command.Env = append(os.Environ(), faceKillHelperEnv+"=1", faceKillDatabaseEnv+"="+filename)
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
		t.Fatalf("read face claim helper: %v", err)
	}
	if strings.TrimSpace(line) != "CLAIMED:face_job_kill" {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("unexpected face claim helper output %q", line)
	}
	if err := command.Process.Kill(); err != nil {
		_ = command.Wait()
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("face claim helper unexpectedly exited successfully")
	}

	time.Sleep(1100 * time.Millisecond)
	restarted, err := sqlitestore.Open(ctx, filename, sqlitestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	summary, err := restarted.RecoverExpiredFaceJobs(ctx, time.Now().UTC())
	if err != nil || summary.Requeued != 1 || summary.Interrupted != 0 {
		t.Fatalf("post-kill recovery = %#v err=%v", summary, err)
	}
	reclaimed, found, err := restarted.ClaimFaceJob(ctx, time.Now().UTC(), time.Minute)
	if err != nil || !found {
		t.Fatalf("post-kill claim = %#v found=%t err=%v", reclaimed, found, err)
	}
	if reclaimed.ID != "face_job_kill" || reclaimed.AttemptCount != 2 || reclaimed.ClaimedRevision != 3 ||
		reclaimed.CheckpointID != 101 || reclaimed.CompletedItems != 1 || reclaimed.TotalItems != 2 {
		t.Fatalf("post-kill state = %#v", reclaimed)
	}
}

func TestFaceAnalysisClaimKillHelper(t *testing.T) {
	if os.Getenv(faceKillHelperEnv) != "1" {
		return
	}
	store, err := sqlitestore.Open(context.Background(), os.Getenv(faceKillDatabaseEnv), sqlitestore.Options{})
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), time.Now().UTC(), time.Second)
	if err != nil || !found {
		fmt.Fprintf(os.Stdout, "ERROR:found=%t err=%v\n", found, err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stdout, "CLAIMED:%s\n", claimed.ID)
	for {
		time.Sleep(time.Hour)
	}
}

func seedFaceKillFixture(t *testing.T, filename string, libraryID int64) {
	t.Helper()
	database, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().UnixMilli()
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
	execute(`INSERT INTO face_generations(
		id,detector_package_id,detector_content_hash,embedder_package_id,embedder_content_hash,
		embedding_dimension,transform_version,threshold_profile,state,created_at_ms,activated_at_ms,updated_at_ms
	) VALUES('face_generation_kill','yunet',?,'sface',?,128,1,'release-v1','active',?,?,?)`,
		strings.Repeat("a", 64), strings.Repeat("b", 64), now, now, now)
	execute(`INSERT INTO face_library_settings(
		library_id,enabled,state,active_generation_id,revision,coverage_revision,created_at_ms,updated_at_ms
	) VALUES(?,1,'building','face_generation_kill',1,2,?,?)`, libraryID, now, now)
	execute(`INSERT INTO face_library_progress(
		generation_id,library_id,eligible_count,completed_count,failed_count,stale_count,checkpoint_id,revision,updated_at_ms
	) VALUES('face_generation_kill',?,2,1,0,0,101,2,?)`, libraryID, now)
	execute(`INSERT INTO ai_model_operations(
		id,kind,state,phase,library_id,completed_items,total_items,revision,created_at_ms,updated_at_ms
	) VALUES('face_operation_kill','face_missing','queued','queued',?,1,2,1,?,?)`, libraryID, now, now)
	execute(`INSERT INTO face_analysis_jobs(
		id,library_id,generation_id,operation_id,mode,state,checkpoint_id,requested_revision,attempt_count,created_at_ms,updated_at_ms
	) VALUES('face_job_kill',?,'face_generation_kill','face_operation_kill','missing','queued',101,1,0,?,?)`,
		libraryID, now, now)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
