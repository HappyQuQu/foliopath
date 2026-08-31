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

const semanticBackfillSelect = `
    SELECT request.idempotency_key_hash, request.request_hash,
           job.id, job.library_id, job.generation_id, job.operation_id, job.mode, job.state,
           job.checkpoint_id, job.requested_revision, job.claimed_revision, job.attempt_count,
           job.lease_expires_ms, job.error_code, operation.revision,
           operation.completed_items, operation.total_items, job.created_at_ms, job.updated_at_ms
    FROM semantic_job_requests AS request
    JOIN semantic_jobs AS job ON job.id = request.job_id
    JOIN ai_model_operations AS operation ON operation.id = job.operation_id`

func (s *Store) FindSemanticBackfill(ctx context.Context, keyHash string) (semantic.BackfillAdmission, bool, error) {
	value, err := scanSemanticBackfill(s.db.QueryRowContext(ctx, semanticBackfillSelect+`
        WHERE request.idempotency_key_hash = ?`, keyHash))
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.BackfillAdmission{}, false, nil
	}
	return value, err == nil, err
}

func (s *Store) CreateSemanticBackfill(ctx context.Context, value semantic.BackfillAdmission) (semantic.BackfillAdmission, bool, bool, error) {
	if err := semantic.ValidateBackfillAdmission(value); err != nil {
		return semantic.BackfillAdmission{}, false, false, err
	}
	created, coalesced := false, false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var enabled int
		if err := tx.QueryRowContext(ctx, `SELECT enabled FROM ai_library_settings WHERE library_id = ?`, value.Job.LibraryID).Scan(&enabled); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrInvalidSemanticJob
			}
			return err
		}
		var generationState string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM semantic_generations WHERE id = ?`, value.Job.GenerationID).Scan(&generationState); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrSemanticGenerationUnavailable
			}
			return err
		}
		if enabled != 1 || generationState != "active" {
			return semantic.ErrSemanticGenerationUnavailable
		}
		var clearActive int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM semantic_clear_jobs
			WHERE library_id=? AND state IN ('queued','running','cancelling'))`, value.Job.LibraryID).Scan(&clearActive); err != nil {
			return err
		}
		if clearActive == 1 {
			return semantic.ErrSemanticJobConflict
		}

		var activeJobID, activeMode string
		err := tx.QueryRowContext(ctx, `
            SELECT id, mode FROM semantic_jobs
            WHERE library_id = ? AND generation_id = ?
              AND state IN ('queued', 'running', 'cancelling')
			ORDER BY created_at_ms, id LIMIT 1`, value.Job.LibraryID, value.Job.GenerationID).Scan(&activeJobID, &activeMode)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		jobID := value.Job.ID
		if err == nil {
			if activeMode != string(value.Job.Mode) {
				return semantic.ErrSemanticJobConflict
			}
			jobID, coalesced = activeJobID, true
		} else {
			if value.Job.Mode == semantic.JobAll {
				_, err = tx.ExecContext(ctx, `
                    INSERT INTO semantic_library_progress(
                        generation_id, library_id, eligible_count, completed_count, failed_count,
                        stale_count, checkpoint_id, revision, updated_at_ms
                    ) VALUES(?, ?, ?, 0, 0, 0, 0, 1, ?)
                    ON CONFLICT(generation_id, library_id) DO UPDATE SET
                        eligible_count = excluded.eligible_count, completed_count = 0, failed_count = 0,
                        stale_count = 0, checkpoint_id = 0, revision = semantic_library_progress.revision + 1,
                        updated_at_ms = excluded.updated_at_ms`,
					value.Job.GenerationID, value.Job.LibraryID, value.EligibleCount, value.Job.CreatedAt.UnixMilli())
			} else {
				_, err = tx.ExecContext(ctx, `
                    INSERT INTO semantic_library_progress(
                        generation_id, library_id, eligible_count, completed_count, failed_count,
                        stale_count, checkpoint_id, revision, updated_at_ms
                    ) VALUES(?, ?, ?, ?, 0, 0, 0, 1, ?)
                    ON CONFLICT(generation_id, library_id) DO UPDATE SET
                        eligible_count = excluded.eligible_count,
                        completed_count = excluded.completed_count,
                        failed_count = 0,
                        stale_count = 0,
                        checkpoint_id = 0,
                        revision = semantic_library_progress.revision + 1,
                        updated_at_ms = excluded.updated_at_ms`,
					value.Job.GenerationID, value.Job.LibraryID, value.EligibleCount,
					value.EligibleCount-value.Job.TotalItems, value.Job.CreatedAt.UnixMilli())
			}
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
                INSERT INTO ai_model_operations(
                    id, kind, state, phase, library_id, completed_items, total_items,
                    revision, created_at_ms, updated_at_ms
                ) VALUES(?, ?, 'queued', 'queued', ?, 0, ?, 1, ?, ?)`,
				value.Job.OperationID, value.Job.OperationKind(), value.Job.LibraryID, value.Job.TotalItems,
				value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
                INSERT INTO semantic_jobs(
                    id, library_id, generation_id, operation_id, mode, state, checkpoint_id,
                    requested_revision, attempt_count, created_at_ms, updated_at_ms
                ) VALUES(?, ?, ?, ?, ?, 'queued', ?, 1, 0, ?, ?)`,
				jobID, value.Job.LibraryID, value.Job.GenerationID, value.Job.OperationID, value.Job.Mode,
				int64(0), value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			created = true
		}
		_, err = tx.ExecContext(ctx, `
            INSERT INTO semantic_job_requests(idempotency_key_hash, request_hash, job_id, created_at_ms)
            VALUES(?, ?, ?, ?)`, value.IdempotencyKeyHash, value.RequestHash, jobID, value.Job.CreatedAt.UnixMilli())
		return err
	})
	if err != nil {
		if !isUniqueConstraint(err) {
			return semantic.BackfillAdmission{}, false, false, fmt.Errorf("create semantic backfill: %w", err)
		}
		existing, found, findErr := s.FindSemanticBackfill(ctx, value.IdempotencyKeyHash)
		if findErr != nil || !found {
			return semantic.BackfillAdmission{}, false, false, firstErrorIfMissing(found, findErr)
		}
		return existing, false, false, nil
	}
	stored, found, err := s.FindSemanticBackfill(ctx, value.IdempotencyKeyHash)
	return stored, created, coalesced, firstErrorIfMissing(found, err)
}

func (s *Store) ClaimSemanticBackfill(ctx context.Context, now time.Time, lease time.Duration) (semantic.BackfillJob, bool, error) {
	leaseMS, err := semanticLeaseMilliseconds(lease)
	if err != nil || now.IsZero() {
		return semantic.BackfillJob{}, false, semantic.ErrInvalidSemanticJob
	}
	var claimed semantic.BackfillJob
	found := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanSemanticJob(tx.QueryRowContext(ctx, semanticJobSelect+`
            WHERE job.state = 'queued'
              AND NOT EXISTS (
                SELECT 1 FROM semantic_jobs active
                WHERE active.library_id = job.library_id AND active.state IN ('running', 'cancelling')
              )
            ORDER BY COALESCE((
                SELECT MAX(previous.updated_at_ms) FROM semantic_jobs previous
                WHERE previous.library_id = job.library_id AND previous.state IN ('succeeded','failed','cancelled')
            ), 0), job.created_at_ms, job.id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		claimRevision := job.RequestedRevision + 1
		result, err := tx.ExecContext(ctx, `
            UPDATE semantic_jobs SET state = 'running', claimed_revision = ?, attempt_count = attempt_count + 1,
                lease_expires_ms = ?, error_code = NULL, updated_at_ms = ?
            WHERE id = ? AND state = 'queued' AND requested_revision = ? AND attempt_count < ?`,
			claimRevision, now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.RequestedRevision,
			semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrSemanticJobConflict
		}
		result, err = tx.ExecContext(ctx, `
            UPDATE ai_model_operations SET state = 'running', phase = 'building', lease_expires_ms = ?,
                revision = revision + 1, updated_at_ms = ? WHERE id = ? AND state = 'queued'`,
			now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrSemanticJobConflict
		}
		claimed, err = scanSemanticJob(tx.QueryRowContext(ctx, semanticJobSelect+` WHERE job.id = ?`, job.ID))
		found = err == nil
		return err
	})
	return claimed, found, err
}

func (s *Store) RefreshSemanticBackfillLease(ctx context.Context, job semantic.BackfillJob, now time.Time, lease time.Duration) (bool, error) {
	leaseMS, err := semanticLeaseMilliseconds(lease)
	if err != nil || now.IsZero() || job.ID == "" || job.ClaimedRevision < 1 {
		return false, semantic.ErrInvalidSemanticJob
	}
	cancelRequested := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM semantic_jobs WHERE id = ? AND claimed_revision = ?`, job.ID, job.ClaimedRevision).Scan(&state); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrSemanticJobConflict
			}
			return err
		}
		if state != "running" && state != "cancelling" {
			return semantic.ErrSemanticJobConflict
		}
		cancelRequested = state == "cancelling"
		result, err := tx.ExecContext(ctx, `
            UPDATE semantic_jobs SET lease_expires_ms = ?, updated_at_ms = ?
            WHERE id = ? AND claimed_revision = ? AND state IN ('running','cancelling')`,
			now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrSemanticJobConflict
		}
		result, err = tx.ExecContext(ctx, `
            UPDATE ai_model_operations SET lease_expires_ms = ?, updated_at_ms = ?
            WHERE id = ? AND state IN ('running','cancelling')`,
			now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrSemanticJobConflict
		}
		return nil
	})
	return cancelRequested, err
}

func (s *Store) CancelSemanticBackfill(ctx context.Context, jobID string, expectedOperationRevision int64, now time.Time) (semantic.BackfillJob, error) {
	if jobID == "" || expectedOperationRevision < 1 || now.IsZero() {
		return semantic.BackfillJob{}, semantic.ErrInvalidSemanticJob
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanSemanticJob(tx.QueryRowContext(ctx, semanticJobSelect+` WHERE job.id = ?`, jobID))
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.ErrSemanticJobNotFound
		}
		if err != nil {
			return err
		}
		if job.OperationRevision != expectedOperationRevision {
			return semantic.ErrSemanticJobConflict
		}
		switch job.State {
		case semantic.JobQueued:
			_, err = tx.ExecContext(ctx, `UPDATE semantic_jobs SET state='cancelled', requested_revision=requested_revision+1, updated_at_ms=? WHERE id=?`, now.UnixMilli(), jobID)
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled', phase='completed', error_code='cancelled', revision=revision+1, updated_at_ms=?, finished_at_ms=? WHERE id=?`, now.UnixMilli(), now.UnixMilli(), job.OperationID)
			}
		case semantic.JobRunning:
			_, err = tx.ExecContext(ctx, `UPDATE semantic_jobs SET state='cancelling', requested_revision=requested_revision+1, updated_at_ms=? WHERE id=?`, now.UnixMilli(), jobID)
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelling', revision=revision+1, updated_at_ms=? WHERE id=?`, now.UnixMilli(), job.OperationID)
			}
		case semantic.JobCancelling:
			return nil
		default:
			return semantic.ErrSemanticJobConflict
		}
		return err
	})
	if err != nil {
		return semantic.BackfillJob{}, err
	}
	return scanSemanticJob(s.db.QueryRowContext(ctx, semanticJobSelect+` WHERE job.id = ?`, jobID))
}

func (s *Store) CancelSemanticBackfillOperation(ctx context.Context, operationID string, expectedOperationRevision int64, now time.Time) (semantic.BackfillJob, error) {
	var jobID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM semantic_jobs WHERE operation_id=?`, operationID).Scan(&jobID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.BackfillJob{}, semantic.ErrSemanticJobNotFound
		}
		return semantic.BackfillJob{}, err
	}
	return s.CancelSemanticBackfill(ctx, jobID, expectedOperationRevision, now)
}

func (s *Store) FinishSemanticBackfill(ctx context.Context, job semantic.BackfillJob, outcome semantic.JobState, errorCode string, now time.Time) (semantic.BackfillJob, error) {
	if job.ID == "" || job.OperationID == "" || job.ClaimedRevision < 1 || now.IsZero() ||
		(outcome != semantic.JobSucceeded && outcome != semantic.JobCancelled && outcome != semantic.JobFailed) ||
		(outcome == semantic.JobFailed) != (errorCode != "") || len(errorCode) > 128 {
		return semantic.BackfillJob{}, semantic.ErrInvalidSemanticJob
	}
	if outcome == semantic.JobCancelled {
		errorCode = "cancelled"
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var operationState string
		var completed, total int64
		if err := tx.QueryRowContext(ctx, `
            SELECT operation.state, operation.completed_items, operation.total_items
            FROM semantic_jobs AS current
            JOIN ai_model_operations AS operation ON operation.id = current.operation_id
            WHERE current.id = ? AND current.claimed_revision = ? AND current.operation_id = ?
              AND current.state IN ('running','cancelling')`,
			job.ID, job.ClaimedRevision, job.OperationID).Scan(&operationState, &completed, &total); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrSemanticJobConflict
			}
			return err
		}
		if outcome == semantic.JobSucceeded && (operationState != "running" || completed != total) {
			return semantic.ErrSemanticJobConflict
		}
		if outcome == semantic.JobCancelled && operationState != "cancelling" {
			return semantic.ErrSemanticJobConflict
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE semantic_jobs SET state=?, lease_expires_ms=NULL, error_code=?,
                requested_revision=requested_revision+1, updated_at_ms=?
            WHERE id=? AND claimed_revision=? AND state IN ('running','cancelling')`,
			outcome, nullableString(errorCode), now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrSemanticJobConflict
		}
		result, err = tx.ExecContext(ctx, `
            UPDATE ai_model_operations SET state=?, phase='completed', error_code=?, lease_expires_ms=NULL,
                revision=revision+1, updated_at_ms=?, finished_at_ms=?
            WHERE id=? AND state IN ('running','cancelling')`,
			outcome, nullableString(errorCode), now.UnixMilli(), now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrSemanticJobConflict
		}
		return nil
	})
	if err != nil {
		return semantic.BackfillJob{}, err
	}
	return scanSemanticJob(s.db.QueryRowContext(ctx, semanticJobSelect+` WHERE job.id = ?`, job.ID))
}

func (s *Store) RecoverExpiredSemanticBackfills(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	if now.IsZero() {
		return jobs.RecoverySummary{}, semantic.ErrInvalidSemanticJob
	}
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            UPDATE semantic_jobs SET state='cancelled', requested_revision=requested_revision+1,
                lease_expires_ms=NULL, error_code='cancelled', updated_at_ms=?
            WHERE state='cancelling' AND lease_expires_ms <= ?`, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		cancelled, _ := result.RowsAffected()
		if cancelled > 0 {
			_, err = tx.ExecContext(ctx, `
                UPDATE ai_model_operations SET state='cancelled', phase='completed', error_code='cancelled',
                    lease_expires_ms=NULL, revision=revision+1, updated_at_ms=?, finished_at_ms=?
                WHERE id IN (SELECT operation_id FROM semantic_jobs WHERE state='cancelled' AND updated_at_ms=?)
                  AND state='cancelling'`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli())
			if err != nil {
				return err
			}
		}
		result, err = tx.ExecContext(ctx, `
            UPDATE semantic_jobs SET state='queued', requested_revision=requested_revision+1,
                claimed_revision=NULL, lease_expires_ms=NULL, error_code=NULL, updated_at_ms=?
            WHERE state='running' AND lease_expires_ms <= ? AND attempt_count < ?`,
			now.UnixMilli(), now.UnixMilli(), semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		summary.Requeued, _ = result.RowsAffected()
		_, err = tx.ExecContext(ctx, `
            UPDATE ai_model_operations SET state='queued', phase='queued', lease_expires_ms=NULL,
                revision=revision+1, updated_at_ms=?
            WHERE id IN (SELECT operation_id FROM semantic_jobs WHERE state='queued' AND updated_at_ms=?)
              AND state='running'`, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		result, err = tx.ExecContext(ctx, `
            UPDATE semantic_jobs SET state='failed', requested_revision=requested_revision+1,
                claimed_revision=NULL, lease_expires_ms=NULL, error_code='operation_interrupted', updated_at_ms=?
            WHERE state='running' AND lease_expires_ms <= ? AND attempt_count >= ?`,
			now.UnixMilli(), now.UnixMilli(), semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		summary.Interrupted, _ = result.RowsAffected()
		_, err = tx.ExecContext(ctx, `
            UPDATE ai_model_operations SET state='failed', phase='completed', error_code='operation_interrupted',
                lease_expires_ms=NULL, revision=revision+1, updated_at_ms=?, finished_at_ms=?
            WHERE id IN (SELECT operation_id FROM semantic_jobs WHERE state='failed' AND updated_at_ms=?)
              AND state='running'`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli())
		return err
	})
	return summary, err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func semanticLeaseMilliseconds(value time.Duration) (int64, error) {
	if value < time.Millisecond || value > 10*time.Minute {
		return 0, semantic.ErrInvalidSemanticJob
	}
	return value.Milliseconds(), nil
}

const semanticJobSelect = `
    SELECT job.id, job.library_id, job.generation_id, job.operation_id, job.mode, job.state,
           job.checkpoint_id, job.requested_revision, job.claimed_revision, job.attempt_count,
           job.lease_expires_ms, job.error_code, operation.revision,
           operation.completed_items, operation.total_items, job.created_at_ms, job.updated_at_ms
    FROM semantic_jobs AS job
    JOIN ai_model_operations AS operation ON operation.id = job.operation_id`

type semanticJobRow interface{ Scan(...any) error }

func scanSemanticBackfill(row semanticJobRow) (semantic.BackfillAdmission, error) {
	var value semantic.BackfillAdmission
	err := scanSemanticJobFields(row, &value.IdempotencyKeyHash, &value.RequestHash, &value.Job)
	return value, err
}

func scanSemanticJob(row semanticJobRow) (semantic.BackfillJob, error) {
	var value semantic.BackfillJob
	err := scanSemanticJobFields(row, &value)
	return value, err
}

func scanSemanticJobFields(row semanticJobRow, prefixAndJob ...any) error {
	job := prefixAndJob[len(prefixAndJob)-1].(*semantic.BackfillJob)
	var mode, state string
	var claimedRevision, leaseExpires sql.NullInt64
	var errorCode sql.NullString
	var totalItems sql.NullInt64
	var createdAt, updatedAt int64
	targets := append(prefixAndJob[:len(prefixAndJob)-1],
		&job.ID, &job.LibraryID, &job.GenerationID, &job.OperationID, &mode, &state,
		&job.CheckpointID, &job.RequestedRevision, &claimedRevision, &job.AttemptCount,
		&leaseExpires, &errorCode, &job.OperationRevision, &job.CompletedItems, &totalItems,
		&createdAt, &updatedAt)
	if err := row.Scan(targets...); err != nil {
		return err
	}
	job.Mode, job.State, job.ClaimedRevision, job.ErrorCode = semantic.JobMode(mode), semantic.JobState(state), claimedRevision.Int64, errorCode.String
	job.TotalItems = totalItems.Int64
	job.CreatedAt, job.UpdatedAt = time.UnixMilli(createdAt).UTC(), time.UnixMilli(updatedAt).UTC()
	if leaseExpires.Valid {
		value := time.UnixMilli(leaseExpires.Int64).UTC()
		job.LeaseExpiresAt = &value
	}
	return nil
}

var _ semantic.BackfillQueue = (*Store)(nil)
var _ semantic.BackfillCatalog = (*Store)(nil)
