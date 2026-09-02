package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type videoPlanBuilderStub struct {
	plan semantic.VideoEmbeddingPlan
	err  error
}

func (stub videoPlanBuilderStub) BuildPlan(context.Context, string, int64, int64) (semantic.VideoEmbeddingPlan, error) {
	return stub.plan, stub.err
}

func TestVideoEmbeddingRepositoryAcceptsOnlyCurrentCompleteStoryboard(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	generationID := seedEmbeddingGeneration(t, store, 2)
	now := time.Date(2026, 8, 29, 14, 0, 0, 0, time.UTC)
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO thumbnails(
            library_id, asset_id, variant, source_fingerprint, transform_version,
            cache_rel_path, status, width, height, byte_size, created_at_ms, last_accessed_at_ms,
            frame_count, sprite_columns, sprite_rows, cell_width, cell_height
        ) VALUES(1, 1, 'storyboard', 'v1:42:100', 1, 'libraries/lib_1/story.webp',
                 'ready', 1280, 180, 1000, ?, ?, 4, 4, 1, 320, 180)`, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := semantic.StoryboardFingerprint("v1:42:100", 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	vector, err := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	plan := semantic.VideoEmbeddingPlan{GenerationID: generationID, LibraryID: 1, AssetID: 1,
		SourceFingerprint: "v1:42:100", StoryboardFingerprint: fingerprint, TransformVersion: 1,
		PlanSize: 4, CreatedAt: now}
	for ordinal := 0; ordinal < 4; ordinal++ {
		plan.Frames = append(plan.Frames, semantic.VideoFrameEmbedding{Ordinal: ordinal, TimestampMS: int64(ordinal+1) * 1000, Vector: vector})
	}
	if err := store.ReplaceVideoEmbeddingPlan(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM semantic_video_frames`).Scan(&count); err != nil || count != 4 {
		t.Fatalf("frames = %d, %v", count, err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO ai_library_settings(library_id, enabled, state, revision, coverage_revision, created_at_ms, updated_at_ms)
        VALUES(1, 1, 'ready', 1, 1, ?, ?)`, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	matches, err := store.SearchVideoSemanticVectors(context.Background(), semantic.VideoVectorSearchRequest{
		GenerationID: generationID, LibraryID: 1, Query: []float32{1, 0}, Limit: 10,
	})
	if err != nil || len(matches) != 1 || matches[0].AssetID != 1 || matches[0].Ordinal != 0 || matches[0].PlanSize != 4 {
		t.Fatalf("matches = %#v, %v", matches, err)
	}
	if _, err := store.db.ExecContext(context.Background(), `DELETE FROM semantic_video_frames WHERE ordinal = 3`); err != nil {
		t.Fatal(err)
	}
	matches, err = store.SearchVideoSemanticVectors(context.Background(), semantic.VideoVectorSearchRequest{
		GenerationID: generationID, LibraryID: 1, Query: []float32{1, 0}, Limit: 10,
	})
	if err != nil || len(matches) != 0 {
		t.Fatalf("partial matches = %#v, %v", matches, err)
	}
	if err := store.ReplaceVideoEmbeddingPlan(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	plan.Frames = plan.Frames[:3]
	if err := store.ReplaceVideoEmbeddingPlan(context.Background(), plan); err == nil {
		t.Fatal("partial plan stored")
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM semantic_video_frames`).Scan(&count); err != nil || count != 4 {
		t.Fatalf("reliable plan changed after rejection = %d, %v", count, err)
	}
}

func TestVideoJobQueueHashesCoalescesClaimsCancelsAndRecovers(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, 1)
	now := time.Date(2026, 8, 29, 16, 0, 0, 0, time.UTC)
	ids := []string{"vidjob_first_request", "aio_video_first_request", "vidjob_second_request", "aio_video_second_request"}
	wake := &semanticWakeCounter{}
	service, err := semantic.NewVideoJobService(store, store, wake, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.Request(context.Background(), 1, generationID, semantic.JobMissing, "video-key-001")
	if err != nil || !created.Created || created.Job.TotalItems != 1 || wake.count != 1 {
		t.Fatalf("created=%#v wake=%d err=%v", created, wake.count, err)
	}
	replayed, err := service.Request(context.Background(), 1, generationID, semantic.JobMissing, "video-key-001")
	if err != nil || !replayed.Replayed || replayed.Job.ID != created.Job.ID {
		t.Fatalf("replayed=%#v err=%v", replayed, err)
	}
	coalesced, err := service.Request(context.Background(), 1, generationID, semantic.JobMissing, "video-key-002")
	if err != nil || !coalesced.Coalesced || coalesced.Job.ID != created.Job.ID {
		t.Fatalf("coalesced=%#v err=%v", coalesced, err)
	}
	var plaintext int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_video_job_requests WHERE idempotency_key_hash IN ('video-key-001','video-key-002')`).Scan(&plaintext); err != nil || plaintext != 0 {
		t.Fatalf("plaintext=%d err=%v", plaintext, err)
	}
	claimed, found, err := store.ClaimVideoJob(context.Background(), now.Add(time.Second), time.Second)
	if err != nil || !found || claimed.AttemptCount != 1 || claimed.OperationRevision != 2 {
		t.Fatalf("claimed=%#v found=%v err=%v", claimed, found, err)
	}
	if _, err := service.CancelOperation(context.Background(), claimed.OperationID, 1); !errors.Is(err, semantic.ErrVideoJobConflict) {
		t.Fatalf("stale cancel=%v", err)
	}
	cancelling, err := service.CancelOperation(context.Background(), claimed.OperationID, claimed.OperationRevision)
	if err != nil || cancelling.State != semantic.JobCancelling {
		t.Fatalf("cancelling=%#v err=%v", cancelling, err)
	}
	cancelRequested, err := store.RefreshVideoJobLease(context.Background(), claimed, now.Add(2*time.Second), time.Second)
	if err != nil || !cancelRequested {
		t.Fatalf("heartbeat cancel=%v err=%v", cancelRequested, err)
	}
	finished, err := store.FinishVideoJob(context.Background(), claimed, semantic.JobCancelled, "", now.Add(3*time.Second))
	if err != nil || finished.State != semantic.JobCancelled {
		t.Fatalf("finished=%#v err=%v", finished, err)
	}
}

func TestVideoJobLifecycleClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, 1)
	var settingsUpdatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT updated_at_ms FROM ai_library_settings WHERE library_id=1`).Scan(&settingsUpdatedAt); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(settingsUpdatedAt).Add(-time.Minute)
	ids := []string{"vidjob_clock_rollback", "aio_video_clock_rollback"}
	service, err := semantic.NewVideoJobService(store, store, &semanticWakeCounter{}, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.Request(context.Background(), 1, generationID, semantic.JobMissing, "video-job-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	now = now.Add(-time.Minute)
	claimed, found, err := store.ClaimVideoJob(context.Background(), now, time.Minute)
	if err != nil || !found || claimed.State != semantic.JobRunning {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	if claimed.LeaseExpiresAt == nil || claimed.LeaseExpiresAt.Before(claimed.CreatedAt.Add(time.Minute)) {
		t.Fatalf("claimed created=%v lease=%v", claimed.CreatedAt, claimed.LeaseExpiresAt)
	}
	now = now.Add(-time.Minute)
	cancelling, err := store.CancelVideoJobOperation(context.Background(), claimed.OperationID, claimed.OperationRevision, now)
	if err != nil || cancelling.State != semantic.JobCancelling {
		t.Fatalf("cancelling=%+v err=%v", cancelling, err)
	}
	cancelRequested, err := store.RefreshVideoJobLease(context.Background(), claimed, now.Add(-time.Minute), time.Minute)
	if err != nil || !cancelRequested {
		t.Fatalf("refresh cancel=%v err=%v", cancelRequested, err)
	}
	finished, err := store.FinishVideoJob(context.Background(), claimed, semantic.JobCancelled, "", now.Add(-2*time.Minute))
	if err != nil || finished.State != semantic.JobCancelled {
		t.Fatalf("finished=%+v err=%v", finished, err)
	}
	for _, check := range []struct {
		query string
		id    string
	}{
		{`SELECT 1 FROM semantic_video_jobs WHERE id=? AND updated_at_ms>=created_at_ms`, claimed.ID},
		{`SELECT 1 FROM ai_model_operations WHERE id=? AND updated_at_ms>=created_at_ms AND finished_at_ms>=created_at_ms`, claimed.OperationID},
	} {
		var valid int
		if err := store.db.QueryRowContext(context.Background(), check.query, check.id).Scan(&valid); err != nil || valid != 1 {
			t.Fatalf("time invariant query=%q valid=%d err=%v", check.query, valid, err)
		}
	}
}

func TestVideoJobMigrationPreservesOperationForeignKeys(t *testing.T) {
	store, _ := openTestStore(t)
	rows, err := store.db.Query(`SELECT "table", "from", "to" FROM pragma_foreign_key_list('semantic_jobs') WHERE "from"='operation_id'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("semantic_jobs operation foreign key missing")
	}
	var table, from, to string
	if err := rows.Scan(&table, &from, &to); err != nil {
		t.Fatal(err)
	}
	if table != "ai_model_operations" || from != "operation_id" || to != "id" {
		t.Fatalf("foreign key=%s.%s -> %s", table, from, to)
	}
}

func TestVideoJobProgressCommitsCompletePlanAndCheckpointAtomically(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, 1)
	now := time.Date(2026, 8, 29, 17, 0, 0, 0, time.UTC)
	wake := &semanticWakeCounter{}
	ids := []string{"vidjob_atomic_plan", "aio_video_atomic_plan"}
	service, err := semantic.NewVideoJobService(store, store, wake, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	admitted, err := service.Request(context.Background(), 1, generationID, semantic.JobMissing, "video-key-atomic")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO thumbnails(
        library_id,asset_id,variant,source_fingerprint,transform_version,cache_rel_path,status,width,height,byte_size,
        created_at_ms,last_accessed_at_ms,frame_count,sprite_columns,sprite_rows,cell_width,cell_height)
        VALUES(1,1,'storyboard','v1:42:100',1,'libraries/lib_1/story.webp','ready',1280,180,1000,?,?,4,4,1,320,180)`,
		now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimVideoJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found || claimed.ID != admitted.Job.ID {
		t.Fatalf("claimed=%#v found=%v err=%v", claimed, found, err)
	}
	progress, found, err := store.GetVideoJobProgress(context.Background(), generationID, 1)
	if err != nil || !found {
		t.Fatalf("progress=%#v found=%v err=%v", progress, found, err)
	}
	fingerprint, _ := semantic.StoryboardFingerprint("v1:42:100", 1, 4)
	vector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	plan := semantic.VideoEmbeddingPlan{GenerationID: generationID, LibraryID: 1, AssetID: 1,
		SourceFingerprint: "v1:42:100", StoryboardFingerprint: fingerprint, TransformVersion: 1, PlanSize: 4, CreatedAt: now}
	for ordinal := range 4 {
		plan.Frames = append(plan.Frames, semantic.VideoFrameEmbedding{Ordinal: ordinal, TimestampMS: int64(ordinal+1) * 1000, Vector: vector})
	}
	commit := semantic.VideoJobProgressCommit{JobID: claimed.ID, ClaimedRevision: claimed.ClaimedRevision,
		ExpectedProgressRevision: progress.Revision, ExpectedCheckpointID: 0, NextCheckpointID: 1, Plan: &plan, UpdatedAt: now.Add(2 * time.Second)}
	updated, err := store.CommitVideoJobProgress(context.Background(), commit)
	if err != nil || updated.Ready != 1 || updated.CheckpointID != 1 {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	var frames, completed int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_video_frames`).Scan(&frames); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT completed_items FROM ai_model_operations WHERE id=?`, claimed.OperationID).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if frames != 4 || completed != 1 {
		t.Fatalf("frames=%d completed=%d", frames, completed)
	}
	if _, err := store.CommitVideoJobProgress(context.Background(), commit); !errors.Is(err, semantic.ErrVideoJobConflict) {
		t.Fatalf("duplicate commit=%v", err)
	}
	finished, err := store.FinishVideoJob(context.Background(), claimed, semantic.JobSucceeded, "", now.Add(3*time.Second))
	if err != nil || finished.State != semantic.JobSucceeded {
		t.Fatalf("finished=%#v err=%v", finished, err)
	}
}

func TestVideoJobProcessorRecordsMissingStoryboardAsDegradedWithoutPartialFrames(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, 1)
	now := time.Date(2026, 8, 29, 18, 0, 0, 0, time.UTC)
	wake := &semanticWakeCounter{}
	ids := []string{"vidjob_missing_storyboard", "aio_missing_storyboard"}
	service, err := semantic.NewVideoJobService(store, store, wake, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Request(context.Background(), 1, generationID, semantic.JobMissing, "video-key-degraded"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimVideoJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%#v found=%v err=%v", claimed, found, err)
	}
	processor, err := semantic.NewVideoJobProcessor(store, store, videoPlanBuilderStub{err: semantic.ErrStoryboardNotReady}, func() time.Time {
		return now.Add(2 * time.Second)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	progress, found, err := store.GetVideoJobProgress(context.Background(), generationID, 1)
	if err != nil || !found || progress.Degraded != 1 || progress.Ready != 0 || progress.CheckpointID != 1 {
		t.Fatalf("progress=%#v found=%v err=%v", progress, found, err)
	}
	var frames int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_video_frames`).Scan(&frames); err != nil || frames != 0 {
		t.Fatalf("frames=%d err=%v", frames, err)
	}
	stored, err := scanVideoJob(store.db.QueryRow(videoJobSelect+` WHERE job.id=?`, claimed.ID))
	if err != nil || stored.State != semantic.JobSucceeded || stored.CompletedItems != 1 {
		t.Fatalf("stored=%#v err=%v", stored, err)
	}
}

func TestVideoJobRecoveryRequeuesThenFailsAtAttemptLimit(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, 1)
	now := time.Date(2026, 8, 29, 19, 0, 0, 0, time.UTC)
	wake := &semanticWakeCounter{}
	ids := []string{"vidjob_recovery_test", "aio_video_recovery_test"}
	service, err := semantic.NewVideoJobService(store, store, wake, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Request(context.Background(), 1, generationID, semantic.JobMissing, "video-key-recovery")
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= semantic.MaximumVideoJobAttempts; attempt++ {
		claimAt := now.Add(time.Duration(attempt) * time.Minute)
		claimed, found, err := store.ClaimVideoJob(context.Background(), claimAt, time.Second)
		if err != nil || !found || claimed.AttemptCount != attempt {
			t.Fatalf("attempt %d claimed=%#v found=%v err=%v", attempt, claimed, found, err)
		}
		summary, err := store.RecoverExpiredVideoJobs(context.Background(), claimAt.Add(2*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if attempt < semantic.MaximumVideoJobAttempts && (summary.Requeued != 1 || summary.Interrupted != 0) {
			t.Fatalf("attempt %d summary=%#v", attempt, summary)
		}
		if attempt == semantic.MaximumVideoJobAttempts && (summary.Requeued != 0 || summary.Interrupted != 1) {
			t.Fatalf("final summary=%#v", summary)
		}
	}
	stored, err := scanVideoJob(store.db.QueryRow(videoJobSelect+` WHERE job.id=?`, result.Job.ID))
	if err != nil || stored.State != semantic.JobFailed || stored.ErrorCode != "operation_interrupted" {
		t.Fatalf("stored=%#v err=%v", stored, err)
	}
}
