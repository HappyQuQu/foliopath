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

const tagReviewClearAdmissionSelect = `
    SELECT request.idempotency_key_hash,request.request_hash,
           job.id,job.library_id,job.operation_id,job.expected_review_revision,job.state,job.deleted_count,
           job.requested_revision,job.claimed_revision,job.attempt_count,job.lease_expires_ms,job.error_code,
           operation.revision,operation.completed_items,operation.total_items,job.created_at_ms,job.updated_at_ms
    FROM ai_tag_review_clear_requests request
    JOIN ai_tag_review_clear_jobs job ON job.id=request.job_id
    JOIN ai_model_operations operation ON operation.id=job.operation_id`

const tagReviewClearJobSelect = `
    SELECT job.id,job.library_id,job.operation_id,job.expected_review_revision,job.state,job.deleted_count,
           job.requested_revision,job.claimed_revision,job.attempt_count,job.lease_expires_ms,job.error_code,
           operation.revision,operation.completed_items,operation.total_items,job.created_at_ms,job.updated_at_ms
    FROM ai_tag_review_clear_jobs job
    JOIN ai_model_operations operation ON operation.id=job.operation_id`

func (s *Store) FindTagReviewClear(ctx context.Context, keyHash string) (semantic.TagReviewClearAdmission, bool, error) {
	value, err := scanTagReviewClearAdmission(s.db.QueryRowContext(ctx, tagReviewClearAdmissionSelect+`
        WHERE request.idempotency_key_hash=?`, keyHash))
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.TagReviewClearAdmission{}, false, nil
	}
	return value, err == nil, err
}

func (s *Store) CreateTagReviewClear(ctx context.Context, value semantic.TagReviewClearAdmission) (semantic.TagReviewClearAdmission, bool, bool, error) {
	if err := semantic.ValidateTagReviewClearAdmission(value); err != nil {
		return semantic.TagReviewClearAdmission{}, false, false, err
	}
	created, coalesced := false, false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var currentRevision int64
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM ai_tag_review_state WHERE library_id=?`, value.Job.LibraryID).Scan(&currentRevision); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrTagReviewClearNotFound
			}
			return err
		}
		if currentRevision != value.Job.ExpectedReviewRevision {
			return semantic.ErrTagReviewClearConflict
		}
		var activeID string
		err := tx.QueryRowContext(ctx, `SELECT id FROM ai_tag_review_clear_jobs
			WHERE library_id=? AND state IN ('queued','running','cancelling')`, value.Job.LibraryID).Scan(&activeID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		jobID := value.Job.ID
		if err == nil {
			jobID, coalesced = activeID, true
		} else {
			var total int64
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_tag_reviews WHERE library_id=?`, value.Job.LibraryID).Scan(&total); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO ai_model_operations(
				id,kind,state,phase,library_id,completed_items,total_items,revision,created_at_ms,updated_at_ms
			) VALUES(?,'tag_review_clear','queued','queued',?,0,?,1,?,?)`, value.Job.OperationID,
				value.Job.LibraryID, total, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO ai_tag_review_clear_jobs(
				id,library_id,operation_id,expected_review_revision,state,deleted_count,requested_revision,attempt_count,created_at_ms,updated_at_ms
			) VALUES(?,?,?,?,'queued',0,1,0,?,?)`, value.Job.ID, value.Job.LibraryID, value.Job.OperationID,
				value.Job.ExpectedReviewRevision, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			created = true
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ai_tag_review_clear_requests(
			idempotency_key_hash,request_hash,job_id,created_at_ms
		) VALUES(?,?,?,?)`, value.IdempotencyKeyHash, value.RequestHash, jobID, value.Job.CreatedAt.UnixMilli())
		return err
	})
	if err != nil {
		if !isUniqueConstraint(err) {
			return semantic.TagReviewClearAdmission{}, false, false, fmt.Errorf("create tag review clear: %w", err)
		}
		existing, found, findErr := s.FindTagReviewClear(ctx, value.IdempotencyKeyHash)
		return existing, false, false, firstErrorIfMissing(found, findErr)
	}
	stored, found, err := s.FindTagReviewClear(ctx, value.IdempotencyKeyHash)
	return stored, created, coalesced, firstErrorIfMissing(found, err)
}

func (s *Store) ClaimTagReviewClear(ctx context.Context, now time.Time, lease time.Duration) (semantic.TagReviewClearJob, bool, error) {
	leaseMS, err := semanticLeaseMilliseconds(lease)
	if err != nil || now.IsZero() {
		return semantic.TagReviewClearJob{}, false, semantic.ErrInvalidTagReviewClear
	}
	var claimed semantic.TagReviewClearJob
	found := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanTagReviewClearJob(tx.QueryRowContext(ctx, tagReviewClearJobSelect+`
			WHERE job.state='queued' ORDER BY job.created_at_ms,job.id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		claimRevision := job.RequestedRevision + 1
		result, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET state='running',claimed_revision=?,
			attempt_count=attempt_count+1,lease_expires_ms=?,error_code=NULL,updated_at_ms=?
			WHERE id=? AND state='queued' AND requested_revision=? AND attempt_count<?`, claimRevision,
			now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.RequestedRevision, semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagReviewClearConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='running',phase='clearing',
			lease_expires_ms=?,revision=revision+1,updated_at_ms=? WHERE id=? AND state='queued'`,
			now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagReviewClearConflict
		}
		claimed, err = scanTagReviewClearJob(tx.QueryRowContext(ctx, tagReviewClearJobSelect+` WHERE job.id=?`, job.ID))
		found = err == nil
		return err
	})
	return claimed, found, err
}

func (s *Store) RefreshTagReviewClearLease(ctx context.Context, job semantic.TagReviewClearJob, now time.Time, lease time.Duration) (bool, error) {
	leaseMS, err := semanticLeaseMilliseconds(lease)
	if err != nil || now.IsZero() || job.ID == "" || job.ClaimedRevision < 1 {
		return false, semantic.ErrInvalidTagReviewClear
	}
	cancelling := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM ai_tag_review_clear_jobs WHERE id=? AND claimed_revision=?`, job.ID, job.ClaimedRevision).Scan(&state); err != nil {
			return semantic.ErrTagReviewClearConflict
		}
		if state != "running" && state != "cancelling" {
			return semantic.ErrTagReviewClearConflict
		}
		cancelling = state == "cancelling"
		if _, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET lease_expires_ms=?,updated_at_ms=? WHERE id=? AND claimed_revision=?`,
			now.UnixMilli()+leaseMS, now.UnixMilli(), job.ID, job.ClaimedRevision); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET lease_expires_ms=?,updated_at_ms=?
			WHERE id=? AND state IN ('running','cancelling')`, now.UnixMilli()+leaseMS, now.UnixMilli(), job.OperationID)
		return err
	})
	return cancelling, err
}

func (s *Store) DeleteTagReviewClearBatch(ctx context.Context, job semantic.TagReviewClearJob, limit int, now time.Time) (int64, bool, error) {
	if job.ID == "" || job.LibraryID < 1 || job.ClaimedRevision < 1 || limit < 1 || limit > 1000 || now.IsZero() {
		return 0, false, semantic.ErrInvalidTagReviewClear
	}
	var deleted int64
	done := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM ai_tag_review_clear_jobs WHERE id=? AND library_id=? AND claimed_revision=?`,
			job.ID, job.LibraryID, job.ClaimedRevision).Scan(&state); err != nil || state != "running" {
			return semantic.ErrTagReviewClearConflict
		}
		// The idempotency ledger contains the same review decision and outcome.
		// Remove every request that touched a review in this bounded batch before
		// deleting the authoritative review rows; mixed-library requests are
		// discarded as a whole so replay can never return a partial item set.
		if _, err := tx.ExecContext(ctx, `DELETE FROM ai_tag_review_requests WHERE idempotency_key_hash IN (
			SELECT DISTINCT item.idempotency_key_hash FROM ai_tag_review_request_items item
			JOIN ai_tag_reviews review ON review.source_suggestion_id=item.suggestion_id
			WHERE review.rowid IN (
				SELECT rowid FROM ai_tag_reviews WHERE library_id=? ORDER BY reviewed_at_ms,asset_id,tag_id LIMIT ?
			)
		)`, job.LibraryID, limit); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `DELETE FROM ai_tag_reviews WHERE rowid IN (
			SELECT rowid FROM ai_tag_reviews WHERE library_id=? ORDER BY reviewed_at_ms,asset_id,tag_id LIMIT ?
		)`, job.LibraryID, limit)
		if err != nil {
			return err
		}
		deleted, _ = result.RowsAffected()
		var remaining int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM ai_tag_reviews WHERE library_id=?)`, job.LibraryID).Scan(&remaining); err != nil {
			return err
		}
		done = remaining == 0
		if _, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET deleted_count=deleted_count+?,updated_at_ms=?
			WHERE id=? AND state='running'`, deleted, now.UnixMilli(), job.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET completed_items=completed_items+?,revision=revision+1,updated_at_ms=?
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

func (s *Store) FinishTagReviewClear(ctx context.Context, job semantic.TagReviewClearJob, outcome semantic.JobState, errorCode string, now time.Time) (semantic.TagReviewClearJob, error) {
	if job.ID == "" || job.OperationID == "" || job.ClaimedRevision < 1 || now.IsZero() ||
		(outcome != semantic.JobSucceeded && outcome != semantic.JobCancelled && outcome != semantic.JobFailed) ||
		(outcome == semantic.JobFailed) != (errorCode != "") || len(errorCode) > 128 {
		return semantic.TagReviewClearJob{}, semantic.ErrInvalidTagReviewClear
	}
	if outcome == semantic.JobCancelled {
		errorCode = "cancelled"
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM ai_tag_review_clear_jobs WHERE id=? AND claimed_revision=?`, job.ID, job.ClaimedRevision).Scan(&state); err != nil {
			return semantic.ErrTagReviewClearConflict
		}
		if outcome == semantic.JobSucceeded {
			var remaining int
			if state != "running" || tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM ai_tag_reviews WHERE library_id=?)`, job.LibraryID).Scan(&remaining) != nil || remaining != 0 {
				return semantic.ErrTagReviewClearConflict
			}
		} else if state != "running" && state != "cancelling" {
			return semantic.ErrTagReviewClearConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET state=?,lease_expires_ms=NULL,error_code=?,
			requested_revision=requested_revision+1,updated_at_ms=? WHERE id=? AND claimed_revision=?`, outcome,
			nullableString(errorCode), now.UnixMilli(), job.ID, job.ClaimedRevision); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state=?,phase='completed',error_code=?,lease_expires_ms=NULL,
			revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id=? AND state IN ('running','cancelling')`, outcome,
			nullableString(errorCode), now.UnixMilli(), now.UnixMilli(), job.OperationID)
		return err
	})
	if err != nil {
		return semantic.TagReviewClearJob{}, err
	}
	return scanTagReviewClearJob(s.db.QueryRowContext(ctx, tagReviewClearJobSelect+` WHERE job.id=?`, job.ID))
}

func (s *Store) CancelTagReviewClearOperation(ctx context.Context, operationID string, expectedRevision int64, now time.Time) (semantic.TagReviewClearJob, error) {
	if operationID == "" || expectedRevision < 1 || now.IsZero() {
		return semantic.TagReviewClearJob{}, semantic.ErrInvalidTagReviewClear
	}
	var jobID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM ai_tag_review_clear_jobs WHERE operation_id=?`, operationID).Scan(&jobID); err != nil {
		return semantic.TagReviewClearJob{}, semantic.ErrTagReviewClearNotFound
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanTagReviewClearJob(tx.QueryRowContext(ctx, tagReviewClearJobSelect+` WHERE job.id=?`, jobID))
		if err != nil || job.OperationRevision != expectedRevision {
			return semantic.ErrTagReviewClearConflict
		}
		switch job.State {
		case semantic.JobQueued:
			if _, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET state='cancelled',requested_revision=requested_revision+1,
				error_code='cancelled',updated_at_ms=? WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',error_code='cancelled',
				revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id=? AND state='queued'`, now.UnixMilli(), now.UnixMilli(), operationID)
			return err
		case semantic.JobRunning:
			if _, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET state='cancelling',requested_revision=requested_revision+1,
				updated_at_ms=? WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelling',revision=revision+1,updated_at_ms=?
				WHERE id=? AND state='running'`, now.UnixMilli(), operationID)
			return err
		case semantic.JobCancelling:
			return nil
		default:
			return semantic.ErrTagReviewClearConflict
		}
	})
	if err != nil {
		return semantic.TagReviewClearJob{}, err
	}
	return scanTagReviewClearJob(s.db.QueryRowContext(ctx, tagReviewClearJobSelect+` WHERE job.id=?`, jobID))
}

func (s *Store) RecoverExpiredTagReviewClears(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	if now.IsZero() {
		return jobs.RecoverySummary{}, semantic.ErrInvalidTagReviewClear
	}
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET state='cancelled',requested_revision=requested_revision+1,
			claimed_revision=NULL,lease_expires_ms=NULL,error_code='cancelled',updated_at_ms=?
			WHERE state='cancelling' AND lease_expires_ms<=?`, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		_, _ = result.RowsAffected()
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',error_code='cancelled',
			lease_expires_ms=NULL,revision=revision+1,updated_at_ms=?,finished_at_ms=?
			WHERE id IN (SELECT operation_id FROM ai_tag_review_clear_jobs WHERE state='cancelled' AND updated_at_ms=?) AND state='cancelling'`,
			now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET state='queued',requested_revision=requested_revision+1,
			claimed_revision=NULL,lease_expires_ms=NULL,error_code=NULL,updated_at_ms=?
			WHERE state='running' AND lease_expires_ms<=? AND attempt_count<?`, now.UnixMilli(), now.UnixMilli(), semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		summary.Requeued, _ = result.RowsAffected()
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued',phase='queued',lease_expires_ms=NULL,
			revision=revision+1,updated_at_ms=? WHERE id IN (SELECT operation_id FROM ai_tag_review_clear_jobs WHERE state='queued' AND updated_at_ms=?) AND state='running'`,
			now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_tag_review_clear_jobs SET state='failed',requested_revision=requested_revision+1,
			claimed_revision=NULL,lease_expires_ms=NULL,error_code='operation_interrupted',updated_at_ms=?
			WHERE state='running' AND lease_expires_ms<=? AND attempt_count>=?`, now.UnixMilli(), now.UnixMilli(), semantic.MaximumSemanticJobAttempts)
		if err != nil {
			return err
		}
		summary.Interrupted, _ = result.RowsAffected()
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='failed',phase='completed',error_code='operation_interrupted',
			lease_expires_ms=NULL,revision=revision+1,updated_at_ms=?,finished_at_ms=?
			WHERE id IN (SELECT operation_id FROM ai_tag_review_clear_jobs WHERE state='failed' AND updated_at_ms=?) AND state='running'`,
			now.UnixMilli(), now.UnixMilli(), now.UnixMilli())
		return err
	})
	return summary, err
}

func scanTagReviewClearAdmission(row interface{ Scan(...any) error }) (semantic.TagReviewClearAdmission, error) {
	var value semantic.TagReviewClearAdmission
	err := scanTagReviewClearFields(row, &value.IdempotencyKeyHash, &value.RequestHash, &value.Job)
	return value, err
}

func scanTagReviewClearJob(row interface{ Scan(...any) error }) (semantic.TagReviewClearJob, error) {
	var value semantic.TagReviewClearJob
	err := scanTagReviewClearFields(row, &value)
	return value, err
}

func scanTagReviewClearFields(row interface{ Scan(...any) error }, prefixAndJob ...any) error {
	job := prefixAndJob[len(prefixAndJob)-1].(*semantic.TagReviewClearJob)
	var state string
	var claimed, lease, total sql.NullInt64
	var code sql.NullString
	var createdAt, updatedAt int64
	targets := append(prefixAndJob[:len(prefixAndJob)-1], &job.ID, &job.LibraryID, &job.OperationID,
		&job.ExpectedReviewRevision, &state, &job.DeletedCount, &job.RequestedRevision, &claimed, &job.AttemptCount,
		&lease, &code, &job.OperationRevision, &job.CompletedItems, &total, &createdAt, &updatedAt)
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

var _ semantic.TagReviewClearQueue = (*Store)(nil)
