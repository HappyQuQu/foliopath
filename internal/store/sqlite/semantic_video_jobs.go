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

const videoJobSelect = `
    SELECT job.id, job.library_id, job.generation_id, job.operation_id, job.mode, job.state,
           job.checkpoint_id, job.requested_revision, job.claimed_revision, job.attempt_count,
           job.lease_expires_ms, job.error_code, operation.revision,
           operation.completed_items, operation.total_items, job.created_at_ms, job.updated_at_ms
    FROM semantic_video_jobs job
    JOIN ai_model_operations operation ON operation.id=job.operation_id`

const videoAdmissionSelect = `
    SELECT request.idempotency_key_hash, request.request_hash,
           job.id, job.library_id, job.generation_id, job.operation_id, job.mode, job.state,
           job.checkpoint_id, job.requested_revision, job.claimed_revision, job.attempt_count,
           job.lease_expires_ms, job.error_code, operation.revision,
           operation.completed_items, operation.total_items, job.created_at_ms, job.updated_at_ms
    FROM semantic_video_job_requests request
    JOIN semantic_video_jobs job ON job.id=request.job_id
    JOIN ai_model_operations operation ON operation.id=job.operation_id`

func (s *Store) FindVideoJob(ctx context.Context, keyHash string) (semantic.VideoJobAdmission, bool, error) {
	value, err := scanVideoAdmission(s.db.QueryRowContext(ctx, videoAdmissionSelect+` WHERE request.idempotency_key_hash=?`, keyHash))
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.VideoJobAdmission{}, false, nil
	}
	return value, err == nil, err
}

func (s *Store) CreateVideoJob(ctx context.Context, value semantic.VideoJobAdmission) (semantic.VideoJobAdmission, bool, bool, error) {
	if err := semantic.ValidateVideoJobAdmission(value); err != nil {
		return semantic.VideoJobAdmission{}, false, false, err
	}
	created, coalesced := false, false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var enabled int
		var generationState string
		if err := tx.QueryRowContext(ctx, `SELECT enabled FROM ai_library_settings WHERE library_id=?`, value.Job.LibraryID).Scan(&enabled); err != nil {
			return semantic.ErrInvalidVideoJob
		}
		if err := tx.QueryRowContext(ctx, `SELECT state FROM semantic_generations WHERE id=?`, value.Job.GenerationID).Scan(&generationState); err != nil {
			return semantic.ErrSemanticGenerationUnavailable
		}
		if enabled != 1 || generationState != "active" {
			return semantic.ErrSemanticGenerationUnavailable
		}
		var activeID, activeMode string
		err := tx.QueryRowContext(ctx, `SELECT id, mode FROM semantic_video_jobs
            WHERE library_id=? AND generation_id=? AND state IN ('queued','running','cancelling')
            ORDER BY created_at_ms,id LIMIT 1`, value.Job.LibraryID, value.Job.GenerationID).Scan(&activeID, &activeMode)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		jobID := value.Job.ID
		if err == nil {
			if activeMode != string(value.Job.Mode) {
				return semantic.ErrVideoJobConflict
			}
			jobID, coalesced = activeID, true
		} else {
			ready := value.EligibleCount - value.Job.TotalItems
			if value.Job.Mode == semantic.JobAll {
				ready = 0
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO semantic_video_progress(
                    generation_id,library_id,eligible_count,ready_count,degraded_count,failed_count,stale_count,checkpoint_id,revision,updated_at_ms)
                VALUES(?,?,?, ?,0,0,0,0,1,?) ON CONFLICT(generation_id,library_id) DO UPDATE SET
                    eligible_count=excluded.eligible_count,ready_count=excluded.ready_count,degraded_count=0,failed_count=0,stale_count=0,
                    checkpoint_id=0,revision=semantic_video_progress.revision+1,updated_at_ms=excluded.updated_at_ms`,
				value.Job.GenerationID, value.Job.LibraryID, value.EligibleCount, ready, value.Job.CreatedAt.UnixMilli())
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO ai_model_operations(
                    id,kind,state,phase,library_id,completed_items,total_items,revision,created_at_ms,updated_at_ms)
                VALUES(?,?,'queued','queued',?,0,?,1,?,?)`, value.Job.OperationID, value.Job.OperationKind(), value.Job.LibraryID,
				value.Job.TotalItems, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli())
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO semantic_video_jobs(
                    id,library_id,generation_id,operation_id,mode,state,checkpoint_id,requested_revision,attempt_count,created_at_ms,updated_at_ms)
                VALUES(?,?,?,?,?,'queued',0,1,0,?,?)`, jobID, value.Job.LibraryID, value.Job.GenerationID, value.Job.OperationID,
				value.Job.Mode, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli())
			if err != nil {
				return err
			}
			created = true
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO semantic_video_job_requests(idempotency_key_hash,request_hash,job_id,created_at_ms) VALUES(?,?,?,?)`,
			value.IdempotencyKeyHash, value.RequestHash, jobID, value.Job.CreatedAt.UnixMilli())
		return err
	})
	if err != nil {
		if !isUniqueConstraint(err) {
			return semantic.VideoJobAdmission{}, false, false, fmt.Errorf("create video semantic job: %w", err)
		}
		existing, found, findErr := s.FindVideoJob(ctx, value.IdempotencyKeyHash)
		if findErr != nil || !found {
			return semantic.VideoJobAdmission{}, false, false, firstErrorIfMissing(found, findErr)
		}
		return existing, false, false, nil
	}
	stored, found, err := s.FindVideoJob(ctx, value.IdempotencyKeyHash)
	return stored, created, coalesced, firstErrorIfMissing(found, err)
}

func (s *Store) ClaimVideoJob(ctx context.Context, now time.Time, lease time.Duration) (semantic.VideoJob, bool, error) {
	leaseMS, err := videoLeaseMilliseconds(lease)
	if err != nil || now.IsZero() {
		return semantic.VideoJob{}, false, semantic.ErrInvalidVideoJob
	}
	var claimed semantic.VideoJob
	found := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanVideoJob(tx.QueryRowContext(ctx, videoJobSelect+` WHERE job.state='queued'
            AND NOT EXISTS(SELECT 1 FROM semantic_video_jobs active WHERE active.library_id=job.library_id AND active.state IN ('running','cancelling'))
            ORDER BY job.created_at_ms,job.id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		claimRevision := job.RequestedRevision + 1
		result, err := tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET state='running',claimed_revision=?,attempt_count=attempt_count+1,
			lease_expires_ms=MAX(created_at_ms+?,?),error_code=NULL,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued' AND requested_revision=? AND attempt_count<?`,
			claimRevision, leaseMS, now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.RequestedRevision, semantic.MaximumVideoJobAttempts)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrVideoJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='running',phase='building',lease_expires_ms=MAX(created_at_ms+?,?),revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued'`,
			leaseMS, now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrVideoJobConflict
		}
		claimed, err = scanVideoJob(tx.QueryRowContext(ctx, videoJobSelect+` WHERE job.id=?`, job.ID))
		found = err == nil
		return err
	})
	return claimed, found, err
}

func (s *Store) RefreshVideoJobLease(ctx context.Context, job semantic.VideoJob, now time.Time, lease time.Duration) (bool, error) {
	leaseMS, err := videoLeaseMilliseconds(lease)
	if err != nil || now.IsZero() || job.ID == "" || job.ClaimedRevision < 1 {
		return false, semantic.ErrInvalidVideoJob
	}
	cancelled := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM semantic_video_jobs WHERE id=? AND claimed_revision=?`, job.ID, job.ClaimedRevision).Scan(&state); err != nil {
			return semantic.ErrVideoJobConflict
		}
		if state != "running" && state != "cancelling" {
			return semantic.ErrVideoJobConflict
		}
		cancelled = state == "cancelling"
		result, err := tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET lease_expires_ms=MAX(created_at_ms+?,?),updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND claimed_revision=? AND state IN ('running','cancelling')`,
			leaseMS, now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrVideoJobConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET lease_expires_ms=MAX(created_at_ms+?,?),updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state IN ('running','cancelling')`,
			leaseMS, now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		return err
	})
	return cancelled, err
}

func (s *Store) GetVideoJobProgress(ctx context.Context, generationID string, libraryID int64) (semantic.VideoJobProgress, bool, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 {
		return semantic.VideoJobProgress{}, false, semantic.ErrInvalidVideoJob
	}
	value, err := scanVideoJobProgress(s.db.QueryRowContext(ctx, videoJobProgressSelect, generationID, libraryID))
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.VideoJobProgress{}, false, nil
	}
	return value, err == nil, err
}

func (s *Store) CommitVideoJobProgress(ctx context.Context, commit semantic.VideoJobProgressCommit) (semantic.VideoJobProgress, error) {
	generationID, libraryID := "", int64(0)
	if commit.Plan != nil {
		generationID, libraryID = commit.Plan.GenerationID, commit.Plan.LibraryID
	} else {
		if err := s.db.QueryRowContext(ctx, `SELECT generation_id,library_id FROM semantic_video_jobs WHERE id=?`, commit.JobID).Scan(&generationID, &libraryID); err != nil {
			return semantic.VideoJobProgress{}, semantic.ErrVideoJobConflict
		}
	}
	dimension, state, err := s.semanticGenerationContract(ctx, s.db, generationID)
	if err != nil || !writableSemanticGeneration(state) {
		return semantic.VideoJobProgress{}, semantic.ErrSemanticGenerationUnavailable
	}
	if err := semantic.ValidateVideoJobProgressCommit(commit, dimension); err != nil {
		return semantic.VideoJobProgress{}, err
	}
	if commit.Plan != nil {
		expected, err := semantic.StoryboardFingerprint(commit.Plan.SourceFingerprint, commit.Plan.TransformVersion, commit.Plan.PlanSize)
		if err != nil || expected != commit.Plan.StoryboardFingerprint {
			return semantic.VideoJobProgress{}, semantic.ErrInvalidVideoSemantic
		}
	}
	var updated semantic.VideoJobProgress
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var jobGeneration, jobState, operationID, operationState string
		var jobLibrary, checkpoint, claimed, completed int64
		var total sql.NullInt64
		if err := tx.QueryRowContext(ctx, `SELECT job.generation_id,job.library_id,job.state,job.checkpoint_id,job.claimed_revision,
                job.operation_id,operation.state,operation.completed_items,operation.total_items
            FROM semantic_video_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id WHERE job.id=?`, commit.JobID).Scan(
			&jobGeneration, &jobLibrary, &jobState, &checkpoint, &claimed, &operationID, &operationState, &completed, &total); err != nil {
			return semantic.ErrVideoJobConflict
		}
		if jobGeneration != generationID || jobLibrary != libraryID || claimed != commit.ClaimedRevision ||
			checkpoint != commit.ExpectedCheckpointID || (jobState != "running" && jobState != "cancelling") ||
			(operationState != "running" && operationState != "cancelling") || !total.Valid || completed+1 > total.Int64 {
			return semantic.ErrVideoJobConflict
		}
		current, err := scanVideoJobProgress(tx.QueryRowContext(ctx, videoJobProgressSelect, generationID, libraryID))
		if err != nil || current.Revision != commit.ExpectedProgressRevision || current.CheckpointID != commit.ExpectedCheckpointID {
			return semantic.ErrVideoJobConflict
		}
		if current.Ready+current.Degraded+current.Failed+current.Stale+1 > current.Eligible {
			return semantic.ErrInvalidVideoJob
		}
		if commit.Plan != nil {
			currentDimension, currentState, err := s.semanticGenerationContract(ctx, tx, generationID)
			if err != nil || currentDimension != dimension || !writableSemanticGeneration(currentState) {
				return semantic.ErrSemanticGenerationUnavailable
			}
			if err := replaceVideoEmbeddingPlanTx(ctx, tx, *commit.Plan); err != nil {
				return err
			}
		}
		ready := 0
		if commit.Plan != nil {
			ready = 1
		}
		result, err := tx.ExecContext(ctx, `UPDATE semantic_video_progress SET ready_count=ready_count+?,degraded_count=degraded_count+?,
				failed_count=failed_count+?,stale_count=stale_count+?,checkpoint_id=?,revision=revision+1,updated_at_ms=MAX(updated_at_ms,?)
            WHERE generation_id=? AND library_id=? AND revision=? AND checkpoint_id=?`, ready, commit.DegradedCount, commit.FailedCount,
			commit.StaleCount, commit.NextCheckpointID, commit.UpdatedAt.UTC().UnixMilli(), generationID, libraryID,
			commit.ExpectedProgressRevision, commit.ExpectedCheckpointID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrVideoJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET checkpoint_id=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND checkpoint_id=? AND claimed_revision=?`,
			commit.NextCheckpointID, commit.UpdatedAt.UTC().UnixMilli(), commit.JobID, commit.ExpectedCheckpointID, commit.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrVideoJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET completed_items=completed_items+1,revision=revision+1,updated_at_ms=MAX(created_at_ms,?)
            WHERE id=? AND state IN ('running','cancelling')`, commit.UpdatedAt.UTC().UnixMilli(), operationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrVideoJobConflict
		}
		updated, err = scanVideoJobProgress(tx.QueryRowContext(ctx, videoJobProgressSelect, generationID, libraryID))
		return err
	})
	return updated, err
}

func (s *Store) CancelVideoJobOperation(ctx context.Context, operationID string, revision int64, now time.Time) (semantic.VideoJob, error) {
	if operationID == "" || revision < 1 || now.IsZero() {
		return semantic.VideoJob{}, semantic.ErrInvalidVideoJob
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanVideoJob(tx.QueryRowContext(ctx, videoJobSelect+` WHERE job.operation_id=?`, operationID))
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.ErrVideoJobNotFound
		}
		if err != nil {
			return err
		}
		if job.OperationRevision != revision {
			return semantic.ErrVideoJobConflict
		}
		switch job.State {
		case semantic.JobQueued:
			_, err = tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET state='cancelled',requested_revision=requested_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), job.ID)
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',error_code='cancelled',revision=revision+1,updated_at_ms=MAX(created_at_ms,?),finished_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), now.UnixMilli(), operationID)
			}
		case semantic.JobRunning:
			_, err = tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET state='cancelling',requested_revision=requested_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), job.ID)
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelling',revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), operationID)
			}
		case semantic.JobCancelling:
			return nil
		default:
			return semantic.ErrVideoJobConflict
		}
		return err
	})
	if err != nil {
		return semantic.VideoJob{}, err
	}
	return scanVideoJob(s.db.QueryRowContext(ctx, videoJobSelect+` WHERE job.operation_id=?`, operationID))
}

func (s *Store) FinishVideoJob(ctx context.Context, job semantic.VideoJob, outcome semantic.JobState, errorCode string, now time.Time) (semantic.VideoJob, error) {
	if job.ID == "" || job.OperationID == "" || job.ClaimedRevision < 1 || now.IsZero() ||
		(outcome != semantic.JobSucceeded && outcome != semantic.JobCancelled && outcome != semantic.JobFailed) ||
		(outcome == semantic.JobFailed) != (errorCode != "") || len(errorCode) > 128 {
		return semantic.VideoJob{}, semantic.ErrInvalidVideoJob
	}
	if outcome == semantic.JobCancelled {
		errorCode = "cancelled"
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var operationState string
		var completed, total int64
		if err := tx.QueryRowContext(ctx, `SELECT operation.state,operation.completed_items,operation.total_items FROM semantic_video_jobs current
            JOIN ai_model_operations operation ON operation.id=current.operation_id WHERE current.id=? AND current.claimed_revision=? AND current.operation_id=?
            AND current.state IN ('running','cancelling')`, job.ID, job.ClaimedRevision, job.OperationID).Scan(&operationState, &completed, &total); err != nil {
			return semantic.ErrVideoJobConflict
		}
		if outcome == semantic.JobSucceeded && (operationState != "running" || completed != total) {
			return semantic.ErrVideoJobConflict
		}
		if outcome == semantic.JobCancelled && operationState != "cancelling" {
			return semantic.ErrVideoJobConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET state=?,lease_expires_ms=NULL,error_code=?,requested_revision=requested_revision+1,updated_at_ms=MAX(created_at_ms,?)
            WHERE id=? AND claimed_revision=? AND state IN ('running','cancelling')`, outcome, nullableString(errorCode), now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrVideoJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state=?,phase='completed',error_code=?,lease_expires_ms=NULL,revision=revision+1,updated_at_ms=MAX(created_at_ms,?),finished_at_ms=MAX(created_at_ms,?)
            WHERE id=? AND state IN ('running','cancelling')`, outcome, nullableString(errorCode), now.UnixMilli(), now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrVideoJobConflict
		}
		return nil
	})
	if err != nil {
		return semantic.VideoJob{}, err
	}
	return scanVideoJob(s.db.QueryRowContext(ctx, videoJobSelect+` WHERE job.id=?`, job.ID))
}

func (s *Store) RecoverExpiredVideoJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	if now.IsZero() {
		return jobs.RecoverySummary{}, semantic.ErrInvalidVideoJob
	}
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET state='cancelled',requested_revision=requested_revision+1,lease_expires_ms=NULL,error_code='cancelled',updated_at_ms=? WHERE state='cancelling' AND lease_expires_ms<=?`, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',error_code='cancelled',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id IN (SELECT operation_id FROM semantic_video_jobs WHERE state='cancelled' AND updated_at_ms=?) AND state='cancelling'`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET state='queued',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code=NULL,updated_at_ms=? WHERE state='running' AND lease_expires_ms<=? AND attempt_count<?`, now.UnixMilli(), now.UnixMilli(), semantic.MaximumVideoJobAttempts)
		if err != nil {
			return err
		}
		summary.Requeued, _ = result.RowsAffected()
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued',phase='queued',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=? WHERE id IN (SELECT operation_id FROM semantic_video_jobs WHERE state='queued' AND updated_at_ms=?) AND state='running'`, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		result, err = tx.ExecContext(ctx, `UPDATE semantic_video_jobs SET state='failed',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code='operation_interrupted',updated_at_ms=? WHERE state='running' AND lease_expires_ms<=? AND attempt_count>=?`, now.UnixMilli(), now.UnixMilli(), semantic.MaximumVideoJobAttempts)
		if err != nil {
			return err
		}
		summary.Interrupted, _ = result.RowsAffected()
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='failed',phase='completed',error_code='operation_interrupted',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id IN (SELECT operation_id FROM semantic_video_jobs WHERE state='failed' AND updated_at_ms=?) AND state='running'`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli())
		return err
	})
	return summary, err
}

type videoJobRow interface{ Scan(...any) error }

func scanVideoAdmission(row videoJobRow) (semantic.VideoJobAdmission, error) {
	var value semantic.VideoJobAdmission
	err := scanVideoJobFields(row, &value.IdempotencyKeyHash, &value.RequestHash, &value.Job)
	return value, err
}

func scanVideoJob(row videoJobRow) (semantic.VideoJob, error) {
	var value semantic.VideoJob
	err := scanVideoJobFields(row, &value)
	return value, err
}

func scanVideoJobFields(row videoJobRow, prefixAndJob ...any) error {
	job := prefixAndJob[len(prefixAndJob)-1].(*semantic.VideoJob)
	var mode, state string
	var claimed, lease sql.NullInt64
	var errorCode sql.NullString
	var total sql.NullInt64
	var created, updated int64
	targets := append(prefixAndJob[:len(prefixAndJob)-1], &job.ID, &job.LibraryID, &job.GenerationID, &job.OperationID, &mode, &state,
		&job.CheckpointID, &job.RequestedRevision, &claimed, &job.AttemptCount, &lease, &errorCode, &job.OperationRevision,
		&job.CompletedItems, &total, &created, &updated)
	if err := row.Scan(targets...); err != nil {
		return err
	}
	job.Mode, job.State, job.ClaimedRevision, job.ErrorCode = semantic.JobMode(mode), semantic.JobState(state), claimed.Int64, errorCode.String
	job.TotalItems = total.Int64
	job.CreatedAt, job.UpdatedAt = time.UnixMilli(created).UTC(), time.UnixMilli(updated).UTC()
	if lease.Valid {
		value := time.UnixMilli(lease.Int64).UTC()
		job.LeaseExpiresAt = &value
	}
	return nil
}

func videoLeaseMilliseconds(value time.Duration) (int64, error) {
	if value < time.Millisecond || value > 10*time.Minute {
		return 0, semantic.ErrInvalidVideoJob
	}
	return value.Milliseconds(), nil
}

const videoJobProgressSelect = `SELECT generation_id,library_id,eligible_count,ready_count,degraded_count,failed_count,
    stale_count,checkpoint_id,revision,updated_at_ms FROM semantic_video_progress WHERE generation_id=? AND library_id=?`

func scanVideoJobProgress(row videoJobRow) (semantic.VideoJobProgress, error) {
	var value semantic.VideoJobProgress
	var updated int64
	err := row.Scan(&value.GenerationID, &value.LibraryID, &value.Eligible, &value.Ready, &value.Degraded, &value.Failed,
		&value.Stale, &value.CheckpointID, &value.Revision, &updated)
	if err == nil {
		value.UpdatedAt = time.UnixMilli(updated).UTC()
	}
	return value, err
}

var _ semantic.VideoJobQueue = (*Store)(nil)
