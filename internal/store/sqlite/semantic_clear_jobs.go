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

const semanticClearAdmissionSelect = `
    SELECT request.idempotency_key_hash, request.request_hash, request.expected_settings_revision,
           job.id, job.library_id, job.operation_id, job.state, job.requested_revision,
           job.claimed_revision, job.attempt_count, job.lease_expires_ms, job.error_code,
           operation.revision, operation.completed_items, operation.total_items,
           job.created_at_ms, job.updated_at_ms
    FROM semantic_clear_requests request
    JOIN semantic_clear_jobs job ON job.id=request.job_id
    JOIN ai_model_operations operation ON operation.id=job.operation_id`

const semanticClearJobSelect = `
    SELECT job.id, job.library_id, job.operation_id, job.state, job.requested_revision,
           job.claimed_revision, job.attempt_count, job.lease_expires_ms, job.error_code,
           operation.revision, operation.completed_items, operation.total_items,
           job.created_at_ms, job.updated_at_ms
    FROM semantic_clear_jobs job
    JOIN ai_model_operations operation ON operation.id=job.operation_id`

func (s *Store) FindSemanticClear(ctx context.Context, keyHash string) (semantic.ClearAdmission, bool, error) {
	value, err := scanSemanticClearAdmission(s.db.QueryRowContext(ctx, semanticClearAdmissionSelect+`
        WHERE request.idempotency_key_hash=?`, keyHash))
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.ClearAdmission{}, false, nil
	}
	return value, err == nil, err
}

func (s *Store) CreateSemanticClear(ctx context.Context, value semantic.ClearAdmission) (semantic.ClearAdmission, bool, bool, error) {
	if err := semantic.ValidateClearAdmission(value); err != nil {
		return semantic.ClearAdmission{}, false, false, err
	}
	created, coalesced := false, false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var currentRevision int64
		var createdAt int64
		err := tx.QueryRowContext(ctx, `SELECT revision, created_at_ms FROM ai_library_settings WHERE library_id=?`, value.Job.LibraryID).Scan(&currentRevision, &createdAt)
		missing := errors.Is(err, sql.ErrNoRows)
		if err != nil && !missing {
			return err
		}
		if missing {
			if err := tx.QueryRowContext(ctx, `SELECT created_at_ms FROM libraries WHERE id=?`, value.Job.LibraryID).Scan(&createdAt); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return semantic.ErrSemanticLibraryNotFound
				}
				return err
			}
			currentRevision = 1
		}
		if currentRevision != value.ExpectedSettingsRevision {
			return semantic.ErrSemanticRevisionConflict
		}
		var conflicting int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM semantic_jobs WHERE library_id=? AND state IN ('queued','running','cancelling'))`, value.Job.LibraryID).Scan(&conflicting); err != nil {
			return err
		}
		if conflicting == 1 {
			return semantic.ErrSemanticClearConflict
		}
		var activeJobID string
		err = tx.QueryRowContext(ctx, `SELECT id FROM semantic_clear_jobs
			WHERE library_id=? AND state IN ('queued','running','cancelling')`, value.Job.LibraryID).Scan(&activeJobID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		jobID := value.Job.ID
		if err == nil {
			jobID, coalesced = activeJobID, true
		} else {
			var total int64
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM semantic_embeddings WHERE library_id=?`, value.Job.LibraryID).Scan(&total); err != nil {
				return err
			}
			if missing {
				updatedAt := max(createdAt, value.Job.CreatedAt.UnixMilli())
				_, err = tx.ExecContext(ctx, `INSERT INTO ai_library_settings(
					library_id, enabled, state, revision, coverage_revision, created_at_ms, updated_at_ms
				) VALUES(?, 0, 'clearing', 2, 1, ?, ?)`, value.Job.LibraryID, createdAt, updatedAt)
			} else {
				_, err = tx.ExecContext(ctx, `UPDATE ai_library_settings SET enabled=0, state='clearing',
					revision=revision+1, updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND revision=?`,
					value.Job.CreatedAt.UnixMilli(), value.Job.LibraryID, currentRevision)
			}
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO ai_model_operations(
				id, kind, state, phase, library_id, completed_items, total_items,
				revision, created_at_ms, updated_at_ms
			) VALUES(?, 'semantic_clear', 'queued', 'queued', ?, 0, ?, 1, ?, ?)`,
				value.Job.OperationID, value.Job.LibraryID, total, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO semantic_clear_jobs(
				id, library_id, operation_id, state, requested_revision, attempt_count, created_at_ms, updated_at_ms
			) VALUES(?, ?, ?, 'queued', 1, 0, ?, ?)`, value.Job.ID, value.Job.LibraryID, value.Job.OperationID,
				value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			created = true
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO semantic_clear_requests(
			idempotency_key_hash, request_hash, job_id, expected_settings_revision, created_at_ms
		) VALUES(?, ?, ?, ?, ?)`, value.IdempotencyKeyHash, value.RequestHash, jobID,
			value.ExpectedSettingsRevision, value.Job.CreatedAt.UnixMilli())
		return err
	})
	if err != nil {
		if !isUniqueConstraint(err) {
			return semantic.ClearAdmission{}, false, false, fmt.Errorf("create semantic clear: %w", err)
		}
		existing, found, findErr := s.FindSemanticClear(ctx, value.IdempotencyKeyHash)
		if findErr != nil || !found {
			return semantic.ClearAdmission{}, false, false, firstErrorIfMissing(found, findErr)
		}
		return existing, false, false, nil
	}
	stored, found, err := s.FindSemanticClear(ctx, value.IdempotencyKeyHash)
	return stored, created, coalesced, firstErrorIfMissing(found, err)
}

func (s *Store) ClaimSemanticClear(ctx context.Context, now time.Time, lease time.Duration) (semantic.ClearJob, bool, error) {
	leaseMS, err := semanticLeaseMilliseconds(lease)
	if err != nil || now.IsZero() {
		return semantic.ClearJob{}, false, semantic.ErrInvalidSemanticClear
	}
	var claimed semantic.ClearJob
	found := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanSemanticClearJob(tx.QueryRowContext(ctx, semanticClearJobSelect+`
			WHERE job.state='queued' ORDER BY job.created_at_ms, job.id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		claimRevision := job.RequestedRevision + 1
		result, err := tx.ExecContext(ctx, `UPDATE semantic_clear_jobs
			SET state='running', claimed_revision=?, attempt_count=attempt_count+1,
				lease_expires_ms=MAX(created_at_ms+?,?), error_code=NULL, updated_at_ms=MAX(created_at_ms,?)
			WHERE id=? AND state='queued' AND requested_revision=? AND attempt_count<?`,
			claimRevision, leaseMS, now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.RequestedRevision,
			semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrSemanticClearConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations
			SET state='running', phase='clearing', lease_expires_ms=MAX(created_at_ms+?,?), revision=revision+1, updated_at_ms=MAX(created_at_ms,?)
			WHERE id=? AND state='queued'`, leaseMS, now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrSemanticClearConflict
		}
		claimed, err = scanSemanticClearJob(tx.QueryRowContext(ctx, semanticClearJobSelect+` WHERE job.id=?`, job.ID))
		found = err == nil
		return err
	})
	return claimed, found, err
}

func (s *Store) RefreshSemanticClearLease(ctx context.Context, job semantic.ClearJob, now time.Time, lease time.Duration) (bool, error) {
	leaseMS, err := semanticLeaseMilliseconds(lease)
	if err != nil || now.IsZero() || job.ID == "" || job.ClaimedRevision < 1 {
		return false, semantic.ErrInvalidSemanticClear
	}
	cancelling := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM semantic_clear_jobs WHERE id=? AND claimed_revision=?`, job.ID, job.ClaimedRevision).Scan(&state); err != nil {
			return semantic.ErrSemanticClearConflict
		}
		if state != "running" && state != "cancelling" {
			return semantic.ErrSemanticClearConflict
		}
		cancelling = state == "cancelling"
		if _, err := tx.ExecContext(ctx, `UPDATE semantic_clear_jobs SET lease_expires_ms=MAX(created_at_ms+?,?), updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND claimed_revision=?`,
			leaseMS, now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.ClaimedRevision); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET lease_expires_ms=MAX(created_at_ms+?,?), updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state IN ('running','cancelling')`,
			leaseMS, now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		return err
	})
	return cancelling, err
}

func (s *Store) DeleteSemanticClearBatch(ctx context.Context, job semantic.ClearJob, limit int, now time.Time) (int64, bool, error) {
	if job.ID == "" || job.LibraryID < 1 || job.ClaimedRevision < 1 || limit < 1 || limit > 1000 || now.IsZero() {
		return 0, false, semantic.ErrInvalidSemanticClear
	}
	var deleted int64
	done := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM semantic_clear_jobs WHERE id=? AND library_id=? AND claimed_revision=?`,
			job.ID, job.LibraryID, job.ClaimedRevision).Scan(&state); err != nil || state != "running" {
			return semantic.ErrSemanticClearConflict
		}
		result, err := tx.ExecContext(ctx, `DELETE FROM semantic_embeddings WHERE rowid IN (
			SELECT rowid FROM semantic_embeddings WHERE library_id=? ORDER BY generation_id, asset_id LIMIT ?
		)`, job.LibraryID, limit)
		if err != nil {
			return err
		}
		deleted, _ = result.RowsAffected()
		var remaining int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM semantic_embeddings WHERE library_id=?)`, job.LibraryID).Scan(&remaining); err != nil {
			return err
		}
		done = remaining == 0
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations
			SET completed_items=completed_items+?, revision=revision+1, updated_at_ms=MAX(created_at_ms,?)
			WHERE id=? AND state='running'`, deleted, now.UnixMilli(), job.OperationID); err != nil {
			return err
		}
		if done {
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET total_items=completed_items WHERE id=?`, job.OperationID)
		}
		return err
	})
	return deleted, done, err
}

func (s *Store) FinishSemanticClear(ctx context.Context, job semantic.ClearJob, outcome semantic.JobState, errorCode string, now time.Time) (semantic.ClearJob, error) {
	if job.ID == "" || job.OperationID == "" || job.ClaimedRevision < 1 || now.IsZero() ||
		(outcome != semantic.JobSucceeded && outcome != semantic.JobCancelled && outcome != semantic.JobFailed) ||
		(outcome == semantic.JobFailed) != (errorCode != "") || len(errorCode) > 128 {
		return semantic.ClearJob{}, semantic.ErrInvalidSemanticClear
	}
	if outcome == semantic.JobCancelled {
		errorCode = "cancelled"
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		var remaining int
		if err := tx.QueryRowContext(ctx, `SELECT state FROM semantic_clear_jobs WHERE id=? AND claimed_revision=?`, job.ID, job.ClaimedRevision).Scan(&state); err != nil {
			return semantic.ErrSemanticClearConflict
		}
		if outcome == semantic.JobSucceeded {
			if state != "running" {
				return semantic.ErrSemanticClearConflict
			}
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM semantic_embeddings WHERE library_id=?)`, job.LibraryID).Scan(&remaining); err != nil || remaining != 0 {
				return semantic.ErrSemanticClearConflict
			}
		} else if state != "running" && state != "cancelling" {
			return semantic.ErrSemanticClearConflict
		}
		if outcome == semantic.JobSucceeded {
			if _, err := tx.ExecContext(ctx, `DELETE FROM semantic_library_progress WHERE library_id=?`, job.LibraryID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE ai_library_settings SET enabled=0, state='disabled',
				revision=revision+1, coverage_revision=coverage_revision+1, updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND state='clearing'`,
				now.UnixMilli(), job.LibraryID); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `UPDATE ai_library_settings SET enabled=0, state='degraded',
				revision=revision+1, coverage_revision=coverage_revision+1, updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND state='clearing'`,
				now.UnixMilli(), job.LibraryID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE semantic_clear_jobs SET state=?, lease_expires_ms=NULL,
			error_code=?, requested_revision=requested_revision+1, updated_at_ms=MAX(created_at_ms,?)
			WHERE id=? AND claimed_revision=?`, outcome, nullableString(errorCode), now.UnixMilli(), job.ID, job.ClaimedRevision); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state=?, phase='completed', error_code=?,
			lease_expires_ms=NULL, revision=revision+1, updated_at_ms=MAX(created_at_ms,?), finished_at_ms=MAX(created_at_ms,?)
			WHERE id=? AND state IN ('running','cancelling')`, outcome, nullableString(errorCode),
			now.UnixMilli(), now.UnixMilli(), job.OperationID)
		return err
	})
	if err != nil {
		return semantic.ClearJob{}, err
	}
	return scanSemanticClearJob(s.db.QueryRowContext(ctx, semanticClearJobSelect+` WHERE job.id=?`, job.ID))
}

func (s *Store) CancelSemanticClearOperation(ctx context.Context, operationID string, expectedRevision int64, now time.Time) (semantic.ClearJob, error) {
	if operationID == "" || expectedRevision < 1 || now.IsZero() {
		return semantic.ClearJob{}, semantic.ErrInvalidSemanticClear
	}
	var jobID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM semantic_clear_jobs WHERE operation_id=?`, operationID).Scan(&jobID); err != nil {
		return semantic.ClearJob{}, semantic.ErrSemanticClearNotFound
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanSemanticClearJob(tx.QueryRowContext(ctx, semanticClearJobSelect+` WHERE job.id=?`, jobID))
		if err != nil || job.OperationRevision != expectedRevision {
			return semantic.ErrSemanticClearConflict
		}
		if job.State == semantic.JobQueued {
			if _, err := tx.ExecContext(ctx, `UPDATE semantic_clear_jobs SET state='cancelled', requested_revision=requested_revision+1, error_code='cancelled', updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE ai_library_settings SET state='degraded', revision=revision+1, coverage_revision=coverage_revision+1, updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND state='clearing'`, now.UnixMilli(), job.LibraryID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled', phase='completed', error_code='cancelled', revision=revision+1, updated_at_ms=MAX(created_at_ms,?), finished_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued'`, now.UnixMilli(), now.UnixMilli(), operationID)
			return err
		}
		if job.State == semantic.JobRunning {
			if _, err := tx.ExecContext(ctx, `UPDATE semantic_clear_jobs SET state='cancelling', requested_revision=requested_revision+1, updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelling', revision=revision+1, updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='running'`, now.UnixMilli(), operationID)
			return err
		}
		if job.State == semantic.JobCancelling {
			return nil
		}
		return semantic.ErrSemanticClearConflict
	})
	if err != nil {
		return semantic.ClearJob{}, err
	}
	return scanSemanticClearJob(s.db.QueryRowContext(ctx, semanticClearJobSelect+` WHERE job.id=?`, jobID))
}

func (s *Store) RecoverExpiredSemanticClears(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	if now.IsZero() {
		return jobs.RecoverySummary{}, semantic.ErrInvalidSemanticClear
	}
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE semantic_clear_jobs SET state='cancelled', requested_revision=requested_revision+1,
			claimed_revision=NULL, lease_expires_ms=NULL, error_code='cancelled', updated_at_ms=?
			WHERE state='cancelling' AND lease_expires_ms<=?`, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		cancelled, _ := result.RowsAffected()
		if cancelled > 0 {
			if _, err := tx.ExecContext(ctx, `UPDATE ai_library_settings SET state='degraded', revision=revision+1,
				coverage_revision=coverage_revision+1, updated_at_ms=? WHERE library_id IN (
					SELECT library_id FROM semantic_clear_jobs WHERE state='cancelled' AND updated_at_ms=?) AND state='clearing'`,
				now.UnixMilli(), now.UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled', phase='completed', error_code='cancelled',
				lease_expires_ms=NULL, revision=revision+1, updated_at_ms=?, finished_at_ms=?
				WHERE id IN (SELECT operation_id FROM semantic_clear_jobs WHERE state='cancelled' AND updated_at_ms=?)
				  AND state='cancelling'`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
				return err
			}
		}
		result, err = tx.ExecContext(ctx, `UPDATE semantic_clear_jobs SET state='queued', requested_revision=requested_revision+1,
			claimed_revision=NULL, lease_expires_ms=NULL, error_code=NULL, updated_at_ms=?
			WHERE state='running' AND lease_expires_ms<=? AND attempt_count<?`, now.UnixMilli(), now.UnixMilli(), semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		summary.Requeued, _ = result.RowsAffected()
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued', phase='queued', lease_expires_ms=NULL,
			revision=revision+1, updated_at_ms=? WHERE id IN (SELECT operation_id FROM semantic_clear_jobs WHERE state='queued' AND updated_at_ms=?) AND state='running'`,
			now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		result, err = tx.ExecContext(ctx, `UPDATE semantic_clear_jobs SET state='failed', requested_revision=requested_revision+1,
			claimed_revision=NULL, lease_expires_ms=NULL, error_code='operation_interrupted', updated_at_ms=?
			WHERE state='running' AND lease_expires_ms<=? AND attempt_count>=?`,
			now.UnixMilli(), now.UnixMilli(), semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		summary.Interrupted, _ = result.RowsAffected()
		if _, err := tx.ExecContext(ctx, `UPDATE ai_library_settings SET state='degraded', revision=revision+1,
			coverage_revision=coverage_revision+1, updated_at_ms=? WHERE library_id IN (
				SELECT library_id FROM semantic_clear_jobs WHERE state='failed' AND updated_at_ms=?) AND state='clearing'`,
			now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='failed', phase='completed', error_code='operation_interrupted',
			lease_expires_ms=NULL, revision=revision+1, updated_at_ms=?, finished_at_ms=?
			WHERE id IN (SELECT operation_id FROM semantic_clear_jobs WHERE state='failed' AND updated_at_ms=?)
			  AND state IN ('running','cancelling')`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli())
		return err
	})
	return summary, err
}

func scanSemanticClearAdmission(row interface{ Scan(...any) error }) (semantic.ClearAdmission, error) {
	var value semantic.ClearAdmission
	err := scanSemanticClearFields(row, &value.IdempotencyKeyHash, &value.RequestHash, &value.ExpectedSettingsRevision, &value.Job)
	return value, err
}

func scanSemanticClearJob(row interface{ Scan(...any) error }) (semantic.ClearJob, error) {
	var value semantic.ClearJob
	err := scanSemanticClearFields(row, &value)
	return value, err
}

func scanSemanticClearFields(row interface{ Scan(...any) error }, prefixAndJob ...any) error {
	job := prefixAndJob[len(prefixAndJob)-1].(*semantic.ClearJob)
	var state string
	var claimed, lease sql.NullInt64
	var code sql.NullString
	var total sql.NullInt64
	var createdAt, updatedAt int64
	targets := append(prefixAndJob[:len(prefixAndJob)-1], &job.ID, &job.LibraryID, &job.OperationID,
		&state, &job.RequestedRevision, &claimed, &job.AttemptCount, &lease, &code,
		&job.OperationRevision, &job.CompletedItems, &total, &createdAt, &updatedAt)
	if err := row.Scan(targets...); err != nil {
		return err
	}
	job.State, job.ClaimedRevision, job.ErrorCode = semantic.JobState(state), claimed.Int64, code.String
	job.TotalItems = total.Int64
	job.CreatedAt, job.UpdatedAt = time.UnixMilli(createdAt).UTC(), time.UnixMilli(updatedAt).UTC()
	if lease.Valid {
		value := time.UnixMilli(lease.Int64).UTC()
		job.LeaseExpiresAt = &value
	}
	return nil
}

var _ semantic.ClearQueue = (*Store)(nil)
