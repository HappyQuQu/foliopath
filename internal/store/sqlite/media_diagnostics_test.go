package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func TestMediaDiagnosticsListsAndRequeuesEveryMissingOrFailedResult(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
        UPDATE media_jobs
        SET status = 'failed', last_error_code =
              CASE WHEN asset_id = (SELECT MIN(id) FROM assets)
                   THEN 'invalid_media' ELSE 'media_processing_timeout' END,
            attempt_count = 3, finished_at_ms = 100
		WHERE asset_id IN (SELECT id FROM assets ORDER BY id LIMIT 2)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        UPDATE media_jobs
        SET status = 'succeeded', finished_at_ms = 100
        WHERE asset_id = (SELECT id FROM assets ORDER BY id LIMIT 1 OFFSET 2)`); err != nil {
		t.Fatal(err)
	}

	failures, err := store.ListMediaFailures(ctx, thumbnail.FailureQuery{
		LibraryID: libraryID, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 2 || failures[0].RelativePath == "" ||
		failures[0].LibraryName == "" {
		t.Fatalf("failures = %#v", failures)
	}

	summary, err := store.RequeueMediaProcessing(
		ctx, libraryID, thumbnail.RequeueMissing, 256,
	)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Requeued != 3 || summary.RemainingEligible != 0 ||
		summary.PermanentFailures != 0 {
		t.Fatalf("retry summary = %#v", summary)
	}
	remaining, err := store.ListMediaFailures(ctx, thumbnail.FailureQuery{
		LibraryID: libraryID, Limit: 10,
	})
	if err != nil || len(remaining) != 0 {
		t.Fatalf("remaining failures = %#v, %v", remaining, err)
	}
}

func TestMediaDiagnosticsRebuildsAllCompletedAndFailedResults(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
        UPDATE media_jobs
        SET status = CASE WHEN id = (SELECT MIN(id) FROM media_jobs)
                          THEN 'failed' ELSE 'succeeded' END,
            last_error_code = CASE WHEN id = (SELECT MIN(id) FROM media_jobs)
                                   THEN 'invalid_media' ELSE NULL END,
            finished_at_ms = 100`); err != nil {
		t.Fatal(err)
	}
	var total int64
	if err := store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM media_jobs WHERE library_id = ?`, libraryID,
	).Scan(&total); err != nil {
		t.Fatal(err)
	}
	summary, err := store.RequeueMediaProcessing(ctx, libraryID, thumbnail.RequeueAll, 1)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Requeued != total || summary.RemainingEligible != 0 ||
		summary.PermanentFailures != 0 {
		t.Fatalf("rebuild summary = %#v, total = %d", summary, total)
	}
	var queued int64
	if err := store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM media_jobs WHERE library_id = ? AND status = 'queued'`,
		libraryID,
	).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != total {
		t.Fatalf("queued = %d, want %d", queued, total)
	}
}

func TestMediaDiagnosticsReturnsBoundedStructuredAttemptHistory(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	ctx := context.Background()
	var jobID int64
	if err := store.db.QueryRowContext(ctx, `SELECT id FROM media_jobs ORDER BY id LIMIT 1`).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        UPDATE media_jobs SET status = 'failed', last_error_code = 'media_processing_timeout',
            attempt_count = 3, finished_at_ms = 300 WHERE id = ?`, jobID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO media_job_attempts(
            job_id, attempt_number, outcome, stage, reason_code, tool,
            exit_code, duration_ms, finished_at_ms
        ) VALUES (?, 3, 'permanent_failure', 'frame_extract',
                  'time_limit_exceeded', 'ffmpeg', 124, 45000, 300)`, jobID); err != nil {
		t.Fatal(err)
	}
	failure, err := store.GetMediaFailure(ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if len(failure.AttemptHistory) != 1 || failure.LatestAttempt == nil {
		t.Fatalf("failure = %#v", failure)
	}
	attempt := failure.AttemptHistory[0]
	if attempt.Stage != media.StageFrameExtract || attempt.Reason != media.ReasonTimedOut ||
		attempt.Tool != "ffmpeg" || attempt.ExitCode == nil || *attempt.ExitCode != 124 ||
		attempt.DurationMS != 45000 {
		t.Fatalf("attempt = %#v", attempt)
	}
	revision, found, err := store.LatestMediaFailureRevision(ctx, thumbnail.FailureQuery{})
	if err != nil || !found || revision.JobID != jobID || revision.FinishedAtMS != 300 {
		t.Fatalf("revision = %#v, found %t, error %v", revision, found, err)
	}
	if _, err := store.db.ExecContext(ctx,
		`UPDATE media_jobs SET finished_at_ms = 400 WHERE id = ?`, jobID); err != nil {
		t.Fatal(err)
	}
	retriedRevision, found, err := store.LatestMediaFailureRevision(ctx, thumbnail.FailureQuery{})
	if err != nil || !found || retriedRevision.JobID != jobID || retriedRevision.FinishedAtMS != 400 {
		t.Fatalf("retried revision = %#v, found %t, error %v", retriedRevision, found, err)
	}
}

func TestMediaDiagnosticsRejectsMissingLibrary(t *testing.T) {
	store, _ := openTestStore(t)
	_, err := store.RequeueMediaProcessing(
		context.Background(), 999, thumbnail.RequeueMissing, 256,
	)
	if !errors.Is(err, thumbnail.ErrDiagnosticsLibraryNotFound) {
		t.Fatalf("error = %v", err)
	}
}
