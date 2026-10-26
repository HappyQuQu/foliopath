package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

// Handoff methods release work, not output: the caller has already reaped the
// processor. They never extend an expired lease or accept an unfenced result.
func (s *Store) HandoffAutomaticTagJob(ctx context.Context, j autotag.Job, now time.Time) (bool, error) {
	return s.handoffAnalysis(ctx, "automatic_tags", j.ID, j.OperationID, j.ClaimRevision, now)
}
func (s *Store) HandoffSemanticBackfill(ctx context.Context, j semantic.BackfillJob, now time.Time) (bool, error) {
	return s.handoffAnalysis(ctx, "semantic", j.ID, j.OperationID, j.ClaimedRevision, now)
}
func (s *Store) HandoffFaceJob(ctx context.Context, j face.AnalysisJob, now time.Time) (bool, error) {
	return s.handoffAnalysis(ctx, "face", j.ID, j.OperationID, j.ClaimedRevision, now)
}

func (s *Store) handoffAnalysis(ctx context.Context, kind, id, operation string, revision int64, now time.Time) (bool, error) {
	var changed bool
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		changed, err = handoffAnalysisTx(ctx, tx, jobs.HandoffClaim{Kind: kind, JobID: id, OperationID: operation, ClaimRevision: revision}, false, now)
		return err
	})
	return changed, err
}

// HandoffStoppedAnalysisJobs is only for the Linux offline maintenance adapter.
// Any stale/mismatching snapshot rolls back the entire bounded batch.
func (s *Store) HandoffStoppedAnalysisJobs(ctx context.Context, claims []jobs.HandoffClaim, now time.Time) error {
	if len(claims) == 0 || len(claims) > 32 || now.IsZero() {
		return jobs.ErrHandoffConflict
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		for _, claim := range claims {
			changed, err := handoffAnalysisTx(ctx, tx, claim, true, now)
			if err != nil {
				return err
			}
			if !changed {
				return jobs.ErrHandoffConflict
			}
		}
		return nil
	})
}

func handoffAnalysisTx(ctx context.Context, tx *sql.Tx, claim jobs.HandoffClaim, exact bool, now time.Time) (bool, error) {
	if claim.JobID == "" || claim.OperationID == "" || claim.ClaimRevision < 1 || now.IsZero() {
		return false, jobs.ErrHandoffConflict
	}
	// All identifiers come from this closed adapter-owned set, never user input.
	var table, fence, transition string
	switch claim.Kind {
	case "automatic_tags":
		table, fence = "autotag_jobs", "claim_revision"
		transition = "claim_revision=claim_revision+1,next_attempt_ms=0,"
	case "semantic":
		table, fence = "semantic_jobs", "claimed_revision"
		transition = "requested_revision=requested_revision+1,claimed_revision=NULL,error_code=NULL,"
	case "face":
		table, fence = "face_analysis_jobs", "claimed_revision"
		transition = "requested_revision=requested_revision+1,claimed_revision=NULL,error_code=NULL,"
	default:
		return false, jobs.ErrHandoffConflict
	}
	var attempt, checkpoint, completed int64
	err := tx.QueryRowContext(ctx, `SELECT j.attempt_count,j.checkpoint_id,o.completed_items
		FROM `+table+` j JOIN ai_model_operations o ON o.id=j.operation_id
		WHERE j.id=? AND j.operation_id=? AND j.`+fence+`=? AND j.state='running' AND o.state='running'`,
		claim.JobID, claim.OperationID, claim.ClaimRevision).Scan(&attempt, &checkpoint, &completed)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if attempt < 1 || attempt > 3 || (exact && (attempt != claim.AttemptCount || checkpoint != claim.CheckpointID || completed != claim.CompletedItems)) {
		return false, jobs.ErrHandoffConflict
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO analysis_job_handoffs(kind,job_id,claim_revision,operation_id,attempt_before,checkpoint_id,completed_items,created_at_ms) VALUES(?,?,?,?,?,?,?,?)`,
		claim.Kind, claim.JobID, claim.ClaimRevision, claim.OperationID, attempt, checkpoint, completed, now.UnixMilli()); err != nil {
		return false, err
	}
	// As with a busy defer, undo only this orderly claim's attempt charge. Past
	// interrupted attempts remain charged; a future unplanned crash still fails
	// at the existing limit. No checkpoint or curation mutation occurs here.
	if _, err = tx.ExecContext(ctx, `UPDATE `+table+` SET state='queued',`+transition+`attempt_count=attempt_count-1,lease_expires_ms=NULL,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND `+fence+`=? AND state='running'`,
		now.UnixMilli(), claim.JobID, claim.ClaimRevision); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued',phase='queued',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='running'`, now.UnixMilli(), claim.OperationID); err != nil {
		return false, err
	}
	// Bound repeated planned-upgrade receipts to the latest ten per operation.
	_, err = tx.ExecContext(ctx, `DELETE FROM analysis_job_handoffs WHERE operation_id=? AND rowid NOT IN (SELECT rowid FROM analysis_job_handoffs WHERE operation_id=? ORDER BY created_at_ms DESC,claim_revision DESC LIMIT 10)`, claim.OperationID, claim.OperationID)
	return err == nil, err
}
