package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const tagJobAdmissionSelect = `SELECT request.idempotency_key_hash,request.request_hash,request.eligible_count,
 job.id,job.library_id,job.generation_id,job.vocabulary_snapshot_id,job.operation_id,job.mode,job.state,job.checkpoint_id,
 job.requested_revision,job.claimed_revision,job.attempt_count,job.lease_expires_ms,job.error_code,
 operation.revision,operation.completed_items,operation.total_items,job.created_at_ms,job.updated_at_ms
 FROM semantic_tag_job_requests request JOIN semantic_tag_jobs job ON job.id=request.job_id
 JOIN ai_model_operations operation ON operation.id=job.operation_id`

const tagJobSelect = `SELECT job.id,job.library_id,job.generation_id,job.vocabulary_snapshot_id,job.operation_id,job.mode,job.state,job.checkpoint_id,
 job.requested_revision,job.claimed_revision,job.attempt_count,job.lease_expires_ms,job.error_code,
 operation.revision,operation.completed_items,operation.total_items,job.created_at_ms,job.updated_at_ms
 FROM semantic_tag_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id`

const tagProgressSelect = `SELECT generation_id,library_id,vocabulary_snapshot_id,eligible_count,ready_count,degraded_count,failed_count,
 stale_count,checkpoint_id,revision,updated_at_ms FROM semantic_tag_library_progress WHERE generation_id=? AND library_id=? AND vocabulary_snapshot_id=?`

func (s *Store) FindTagJob(ctx context.Context, key string) (semantic.TagJobAdmission, bool, error) {
	v, err := scanTagJobAdmission(s.db.QueryRowContext(ctx, tagJobAdmissionSelect+` WHERE request.idempotency_key_hash=?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.TagJobAdmission{}, false, nil
	}
	return v, err == nil, err
}

func (s *Store) CreateTagJob(ctx context.Context, v semantic.TagJobAdmission) (semantic.TagJobAdmission, bool, bool, error) {
	if err := semantic.ValidateTagJobAdmission(v); err != nil {
		return semantic.TagJobAdmission{}, false, false, err
	}
	created, coalesced := false, false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := s.requireActiveTagOwnersTx(ctx, tx, v.Job.GenerationID, v.Job.VocabularySnapshotID); err != nil {
			return err
		}
		var active string
		err := tx.QueryRowContext(ctx, `SELECT id FROM semantic_tag_jobs WHERE library_id=? AND state IN ('queued','running','cancelling')`, v.Job.LibraryID).Scan(&active)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		jobID := v.Job.ID
		if err == nil {
			jobID, coalesced = active, true
		} else {
			if _, err := tx.ExecContext(ctx, `INSERT INTO ai_model_operations(id,kind,state,phase,library_id,completed_items,total_items,revision,created_at_ms,updated_at_ms)
			 VALUES(?,?,'queued','queued',?,0,?,1,?,?)`, v.Job.OperationID, v.Job.OperationKind(), v.Job.LibraryID, v.Job.TotalItems, v.Job.CreatedAt.UnixMilli(), v.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO semantic_tag_jobs(id,library_id,generation_id,vocabulary_snapshot_id,operation_id,mode,state,checkpoint_id,requested_revision,attempt_count,created_at_ms,updated_at_ms)
			 VALUES(?,?,?,?,?,?,'queued',0,1,0,?,?)`, v.Job.ID, v.Job.LibraryID, v.Job.GenerationID, v.Job.VocabularySnapshotID, v.Job.OperationID, v.Job.Mode, v.Job.CreatedAt.UnixMilli(), v.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			if v.Job.Mode == semantic.JobAll {
				if _, err := tx.ExecContext(ctx, `DELETE FROM semantic_tag_asset_progress WHERE generation_id=? AND library_id=? AND vocabulary_snapshot_id=?`, v.Job.GenerationID, v.Job.LibraryID, v.Job.VocabularySnapshotID); err != nil {
					return err
				}
			}
			var ready, degraded, failed, stale int64
			if err := tx.QueryRowContext(ctx, `SELECT
				COALESCE(SUM(outcome='ready'),0),COALESCE(SUM(outcome='degraded'),0),
				COALESCE(SUM(outcome='failed'),0),COALESCE(SUM(outcome='stale'),0)
				FROM semantic_tag_asset_progress progress JOIN assets asset ON asset.library_id=progress.library_id AND asset.id=progress.asset_id
				WHERE progress.generation_id=? AND progress.library_id=? AND progress.vocabulary_snapshot_id=?
				AND progress.source_fingerprint=asset.source_fingerprint`, v.Job.GenerationID, v.Job.LibraryID, v.Job.VocabularySnapshotID).Scan(&ready, &degraded, &failed, &stale); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO semantic_tag_library_progress(
				generation_id,library_id,vocabulary_snapshot_id,eligible_count,ready_count,degraded_count,failed_count,stale_count,checkpoint_id,updated_at_ms
			) VALUES(?,?,?,?,?,?,?,?,0,?) ON CONFLICT(generation_id,library_id,vocabulary_snapshot_id) DO UPDATE SET
			 eligible_count=excluded.eligible_count,ready_count=excluded.ready_count,degraded_count=excluded.degraded_count,
			 failed_count=excluded.failed_count,stale_count=excluded.stale_count,checkpoint_id=0,
			 revision=semantic_tag_library_progress.revision+1,updated_at_ms=excluded.updated_at_ms`,
				v.Job.GenerationID, v.Job.LibraryID, v.Job.VocabularySnapshotID, v.EligibleCount, ready, degraded, failed, stale, v.Job.CreatedAt.UnixMilli()); err != nil {
				return err
			}
			created = true
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO semantic_tag_job_requests(idempotency_key_hash,request_hash,job_id,eligible_count,created_at_ms) VALUES(?,?,?,?,?)`, v.IdempotencyKeyHash, v.RequestHash, jobID, v.EligibleCount, v.Job.CreatedAt.UnixMilli())
		return err
	})
	if err != nil {
		if !isUniqueConstraint(err) {
			return semantic.TagJobAdmission{}, false, false, fmt.Errorf("create tag job: %w", err)
		}
		existing, found, findErr := s.FindTagJob(ctx, v.IdempotencyKeyHash)
		return existing, false, false, firstErrorIfMissing(found, findErr)
	}
	stored, found, err := s.FindTagJob(ctx, v.IdempotencyKeyHash)
	return stored, created, coalesced, firstErrorIfMissing(found, err)
}

func (s *Store) ClaimTagJob(ctx context.Context, now time.Time, lease time.Duration) (semantic.TagJob, bool, error) {
	leaseMS, err := semanticLeaseMilliseconds(lease)
	if err != nil || now.IsZero() {
		return semantic.TagJob{}, false, semantic.ErrInvalidTagJob
	}
	var claimed semantic.TagJob
	found := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanTagJob(tx.QueryRowContext(ctx, tagJobSelect+` WHERE job.state='queued' ORDER BY job.created_at_ms,job.id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		claim := job.RequestedRevision + 1
		result, err := tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET state='running',claimed_revision=?,attempt_count=attempt_count+1,lease_expires_ms=?,error_code=NULL,updated_at_ms=? WHERE id=? AND state='queued' AND requested_revision=? AND attempt_count<?`, claim, now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.RequestedRevision, semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='running',phase='building',lease_expires_ms=?,revision=revision+1,updated_at_ms=? WHERE id=? AND state='queued'`, now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagJobConflict
		}
		claimed, err = scanTagJob(tx.QueryRowContext(ctx, tagJobSelect+` WHERE job.id=?`, job.ID))
		found = err == nil
		return err
	})
	return claimed, found, err
}

func (s *Store) RefreshTagJobLease(ctx context.Context, job semantic.TagJob, now time.Time, lease time.Duration) (bool, error) {
	leaseMS, err := semanticLeaseMilliseconds(lease)
	if err != nil || now.IsZero() || job.ID == "" || job.ClaimedRevision < 1 {
		return false, semantic.ErrInvalidTagJob
	}
	cancelling := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM semantic_tag_jobs WHERE id=? AND claimed_revision=?`, job.ID, job.ClaimedRevision).Scan(&state); err != nil {
			return semantic.ErrTagJobConflict
		}
		if state != "running" && state != "cancelling" {
			return semantic.ErrTagJobConflict
		}
		cancelling = state == "cancelling"
		_, err := tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET lease_expires_ms=?,updated_at_ms=? WHERE id=? AND claimed_revision=?`, now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET lease_expires_ms=?,updated_at_ms=? WHERE id=? AND state IN ('running','cancelling')`, now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		return err
	})
	return cancelling, err
}

func (s *Store) GetTagJobProgress(ctx context.Context, generation string, library int64, snapshot string) (semantic.TagJobProgress, bool, error) {
	v, err := scanTagProgress(s.db.QueryRowContext(ctx, tagProgressSelect, generation, library, snapshot))
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.TagJobProgress{}, false, nil
	}
	return v, err == nil, err
}

func (s *Store) CommitTagJobProgress(ctx context.Context, c semantic.TagJobProgressCommit) (semantic.TagJobProgress, error) {
	processed := c.DegradedCount + c.FailedCount + c.StaleCount
	if c.Plan != nil {
		processed++
	}
	if c.JobID == "" || c.ClaimedRevision < 1 || c.ExpectedProgressRevision < 1 || c.NextCheckpointID <= c.ExpectedCheckpointID || processed != 1 || c.UpdatedAt.IsZero() {
		return semantic.TagJobProgress{}, semantic.ErrInvalidTagJob
	}
	var updated semantic.TagJobProgress
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var generation, snapshot, state, operationID, operationState string
		var library, checkpoint, claim, completed int64
		var total sql.NullInt64
		if err := tx.QueryRowContext(ctx, `SELECT job.generation_id,job.library_id,job.vocabulary_snapshot_id,job.state,job.checkpoint_id,job.claimed_revision,job.operation_id,operation.state,operation.completed_items,operation.total_items FROM semantic_tag_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id WHERE job.id=?`, c.JobID).Scan(&generation, &library, &snapshot, &state, &checkpoint, &claim, &operationID, &operationState, &completed, &total); err != nil {
			return semantic.ErrTagJobConflict
		}
		if state != "running" || operationState != "running" || checkpoint != c.ExpectedCheckpointID || claim != c.ClaimedRevision {
			return semantic.ErrTagJobConflict
		}
		current, err := scanTagProgress(tx.QueryRowContext(ctx, tagProgressSelect, generation, library, snapshot))
		if err != nil || current.Revision != c.ExpectedProgressRevision || current.CheckpointID != checkpoint {
			return semantic.ErrTagJobConflict
		}
		outcome := "ready"
		fingerprint := ""
		if c.Plan != nil {
			if c.Plan.GenerationID != generation || c.Plan.VocabularySnapshotID != snapshot || c.Plan.LibraryID != library || c.Plan.AssetID != c.NextCheckpointID || semantic.ValidatePendingTagSuggestions(c.Plan.Suggestions, library, c.Plan.AssetID, generation, snapshot, c.Plan.SourceFingerprint) != nil {
				return semantic.ErrInvalidTagJob
			}
			fingerprint = c.Plan.SourceFingerprint
			if _, err := tx.ExecContext(ctx, `DELETE FROM ai_tag_suggestions WHERE library_id=? AND asset_id=? AND state='pending'`, library, c.Plan.AssetID); err != nil {
				return err
			}
			for _, item := range c.Plan.Suggestions {
				if _, err := tx.ExecContext(ctx, `INSERT INTO ai_tag_suggestions(id,generation_id,library_id,asset_id,vocabulary_snapshot_id,tag_id,source_fingerprint,confidence,state,revision,created_at_ms,updated_at_ms) SELECT ?,?,?,?,?,?,?,?,'pending',1,?,? WHERE NOT EXISTS(SELECT 1 FROM ai_tag_reviews WHERE library_id=? AND asset_id=? AND tag_id=?) AND NOT EXISTS(SELECT 1 FROM asset_tags WHERE library_id=? AND asset_id=? AND tag_id=?)`, item.ID, generation, library, item.AssetID, snapshot, item.TagID, item.SourceFingerprint, item.Confidence, item.CreatedAt.UnixMilli(), item.UpdatedAt.UnixMilli(), library, item.AssetID, item.TagID, library, item.AssetID, item.TagID); err != nil {
					return err
				}
			}
		} else {
			if c.DegradedCount == 1 {
				outcome = "degraded"
			}
			if c.FailedCount == 1 {
				outcome = "failed"
			}
			if c.StaleCount == 1 {
				outcome = "stale"
			}
			if err := tx.QueryRowContext(ctx, `SELECT source_fingerprint FROM assets WHERE library_id=? AND id=?`, library, c.NextCheckpointID).Scan(&fingerprint); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO semantic_tag_asset_progress(generation_id,library_id,asset_id,vocabulary_snapshot_id,source_fingerprint,outcome,updated_at_ms) VALUES(?,?,?,?,?,?,?) ON CONFLICT(generation_id,library_id,asset_id,vocabulary_snapshot_id) DO UPDATE SET source_fingerprint=excluded.source_fingerprint,outcome=excluded.outcome,updated_at_ms=excluded.updated_at_ms`, generation, library, c.NextCheckpointID, snapshot, fingerprint, outcome, c.UpdatedAt.UnixMilli()); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE semantic_tag_library_progress SET ready_count=ready_count+?,degraded_count=degraded_count+?,failed_count=failed_count+?,stale_count=stale_count+?,checkpoint_id=?,revision=revision+1,updated_at_ms=? WHERE generation_id=? AND library_id=? AND vocabulary_snapshot_id=? AND revision=? AND checkpoint_id=?`, boolInt(c.Plan != nil), c.DegradedCount, c.FailedCount, c.StaleCount, c.NextCheckpointID, c.UpdatedAt.UnixMilli(), generation, library, snapshot, c.ExpectedProgressRevision, c.ExpectedCheckpointID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagJobConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET checkpoint_id=?,updated_at_ms=? WHERE id=? AND checkpoint_id=? AND claimed_revision=?`, c.NextCheckpointID, c.UpdatedAt.UnixMilli(), c.JobID, c.ExpectedCheckpointID, c.ClaimedRevision); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET completed_items=completed_items+1,revision=revision+1,updated_at_ms=? WHERE id=? AND state='running'`, c.UpdatedAt.UnixMilli(), operationID); err != nil {
			return err
		}
		updated, err = scanTagProgress(tx.QueryRowContext(ctx, tagProgressSelect, generation, library, snapshot))
		return err
	})
	return updated, err
}

func (s *Store) FinishTagJob(ctx context.Context, job semantic.TagJob, outcome semantic.JobState, code string, now time.Time) (semantic.TagJob, error) {
	if job.ID == "" || job.ClaimedRevision < 1 || now.IsZero() || (outcome != semantic.JobSucceeded && outcome != semantic.JobFailed && outcome != semantic.JobCancelled) || (outcome == semantic.JobFailed) != (code != "") {
		return semantic.TagJob{}, semantic.ErrInvalidTagJob
	}
	if outcome == semantic.JobCancelled {
		code = "cancelled"
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state, operationState string
		var completed, total int64
		if err := tx.QueryRowContext(ctx, `SELECT job.state,operation.state,operation.completed_items,operation.total_items
			FROM semantic_tag_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id
			WHERE job.id=? AND job.claimed_revision=? AND job.operation_id=?`, job.ID, job.ClaimedRevision, job.OperationID).Scan(&state, &operationState, &completed, &total); err != nil {
			return semantic.ErrTagJobConflict
		}
		if outcome == semantic.JobSucceeded && (state != "running" || operationState != "running" || completed != total) ||
			outcome == semantic.JobCancelled && (state != "cancelling" || operationState != "cancelling") ||
			outcome == semantic.JobFailed && state != "running" && state != "cancelling" {
			return semantic.ErrTagJobConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET state=?,lease_expires_ms=NULL,error_code=?,requested_revision=requested_revision+1,updated_at_ms=? WHERE id=? AND claimed_revision=? AND state IN ('running','cancelling')`, outcome, nullableString(code), now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state=?,phase='completed',error_code=?,lease_expires_ms=NULL,revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id=? AND state IN ('running','cancelling')`, outcome, nullableString(code), now.UnixMilli(), now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagJobConflict
		}
		return nil
	})
	if err != nil {
		return semantic.TagJob{}, err
	}
	return scanTagJob(s.db.QueryRowContext(ctx, tagJobSelect+` WHERE job.id=?`, job.ID))
}

func (s *Store) CancelTagJobOperation(ctx context.Context, operationID string, revision int64, now time.Time) (semantic.TagJob, error) {
	if operationID == "" || revision < 1 || now.IsZero() {
		return semantic.TagJob{}, semantic.ErrInvalidTagJob
	}
	var jobID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM semantic_tag_jobs WHERE operation_id=?`, operationID).Scan(&jobID); err != nil {
		return semantic.TagJob{}, semantic.ErrTagJobNotFound
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanTagJob(tx.QueryRowContext(ctx, tagJobSelect+` WHERE job.id=?`, jobID))
		if err != nil || job.OperationRevision != revision {
			return semantic.ErrTagJobConflict
		}
		switch job.State {
		case semantic.JobQueued:
			if _, err := tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET state='cancelled',requested_revision=requested_revision+1,error_code='cancelled',updated_at_ms=? WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',error_code='cancelled',revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id=? AND state='queued'`, now.UnixMilli(), now.UnixMilli(), operationID)
			return err
		case semantic.JobRunning:
			if _, err := tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET state='cancelling',requested_revision=requested_revision+1,updated_at_ms=? WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelling',revision=revision+1,updated_at_ms=? WHERE id=? AND state='running'`, now.UnixMilli(), operationID)
			return err
		case semantic.JobCancelling:
			return nil
		default:
			return semantic.ErrTagJobConflict
		}
	})
	if err != nil {
		return semantic.TagJob{}, err
	}
	return scanTagJob(s.db.QueryRowContext(ctx, tagJobSelect+` WHERE job.id=?`, jobID))
}

func (s *Store) RecoverExpiredTagJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	if now.IsZero() {
		return jobs.RecoverySummary{}, semantic.ErrInvalidTagJob
	}
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET state='cancelled',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code='cancelled',updated_at_ms=? WHERE state='cancelling' AND lease_expires_ms<=?`, now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',error_code='cancelled',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id IN(SELECT operation_id FROM semantic_tag_jobs WHERE state='cancelled' AND updated_at_ms=?) AND state='cancelling'`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET state='queued',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code=NULL,updated_at_ms=? WHERE state='running' AND lease_expires_ms<=? AND attempt_count<?`, now.UnixMilli(), now.UnixMilli(), semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		summary.Requeued, _ = result.RowsAffected()
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued',phase='queued',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=? WHERE id IN(SELECT operation_id FROM semantic_tag_jobs WHERE state='queued' AND updated_at_ms=?) AND state='running'`, now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		result, err = tx.ExecContext(ctx, `UPDATE semantic_tag_jobs SET state='failed',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code='operation_interrupted',updated_at_ms=? WHERE state='running' AND lease_expires_ms<=? AND attempt_count>=?`, now.UnixMilli(), now.UnixMilli(), semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		summary.Interrupted, _ = result.RowsAffected()
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='failed',phase='completed',error_code='operation_interrupted',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id IN(SELECT operation_id FROM semantic_tag_jobs WHERE state='failed' AND updated_at_ms=?) AND state='running'`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli())
		return err
	})
	return summary, err
}

func (s *Store) requireActiveTagOwnersTx(ctx context.Context, tx *sql.Tx, generation, snapshot string) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM semantic_generations generation,ai_tag_vocabulary_snapshots vocabulary WHERE generation.id=? AND generation.state='active' AND vocabulary.id=? AND vocabulary.state='active'`, generation, snapshot).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return semantic.ErrSemanticGenerationUnavailable
	}
	return nil
}

func scanTagJobAdmission(row interface{ Scan(...any) error }) (semantic.TagJobAdmission, error) {
	var v semantic.TagJobAdmission
	err := scanTagJobFields(row, &v.IdempotencyKeyHash, &v.RequestHash, &v.EligibleCount, &v.Job)
	return v, err
}
func scanTagJob(row interface{ Scan(...any) error }) (semantic.TagJob, error) {
	var v semantic.TagJob
	err := scanTagJobFields(row, &v)
	return v, err
}
func scanTagJobFields(row interface{ Scan(...any) error }, prefixAndJob ...any) error {
	job := prefixAndJob[len(prefixAndJob)-1].(*semantic.TagJob)
	var mode, state string
	var claimed, lease, total sql.NullInt64
	var code sql.NullString
	var created, updated int64
	targets := append(prefixAndJob[:len(prefixAndJob)-1], &job.ID, &job.LibraryID, &job.GenerationID, &job.VocabularySnapshotID, &job.OperationID, &mode, &state, &job.CheckpointID, &job.RequestedRevision, &claimed, &job.AttemptCount, &lease, &code, &job.OperationRevision, &job.CompletedItems, &total, &created, &updated)
	if err := row.Scan(targets...); err != nil {
		return err
	}
	job.Mode, job.State = semantic.JobMode(mode), semantic.JobState(state)
	job.ClaimedRevision, job.ErrorCode = claimed.Int64, code.String
	job.TotalItems = total.Int64
	job.CreatedAt, job.UpdatedAt = time.UnixMilli(created).UTC(), time.UnixMilli(updated).UTC()
	if lease.Valid {
		v := time.UnixMilli(lease.Int64).UTC()
		job.LeaseExpiresAt = &v
	}
	return nil
}
func scanTagProgress(row interface{ Scan(...any) error }) (semantic.TagJobProgress, error) {
	var v semantic.TagJobProgress
	var updated int64
	err := row.Scan(&v.GenerationID, &v.LibraryID, &v.VocabularySnapshotID, &v.Eligible, &v.Ready, &v.Degraded, &v.Failed, &v.Stale, &v.CheckpointID, &v.Revision, &updated)
	v.UpdatedAt = time.UnixMilli(updated).UTC()
	return v, err
}

var _ semantic.TagJobQueue = (*Store)(nil)
