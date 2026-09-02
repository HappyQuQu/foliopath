package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/jobs"
)

const faceClearAdmissionSelect = `SELECT request.idempotency_key_hash,request.request_hash,job.id,job.library_id,job.operation_id,job.kind,job.expected_settings_revision,job.expected_person_count,job.expected_assignment_count,job.expected_constraint_count,job.state,job.deleted_count,job.requested_revision,job.claimed_revision,job.attempt_count,job.created_at_ms,job.updated_at_ms FROM face_clear_requests request JOIN face_clear_jobs job ON job.id=request.job_id`
const faceClearJobSelect = `SELECT id,library_id,operation_id,kind,expected_settings_revision,expected_person_count,expected_assignment_count,expected_constraint_count,state,deleted_count,requested_revision,claimed_revision,attempt_count,created_at_ms,updated_at_ms FROM face_clear_jobs`

func (s *Store) FindFaceClear(ctx context.Context, keyHash string) (face.ClearAdmission, bool, error) {
	value, err := scanFaceClearAdmission(s.db.QueryRowContext(ctx, faceClearAdmissionSelect+` WHERE request.idempotency_key_hash=?`, keyHash))
	if errors.Is(err, sql.ErrNoRows) {
		return face.ClearAdmission{}, false, nil
	}
	return value, err == nil, err
}
func (s *Store) CreateFaceClear(ctx context.Context, value face.ClearAdmission) (face.ClearAdmission, bool, error) {
	if !validFaceClearAdmission(value) {
		return face.ClearAdmission{}, false, face.ErrInvalidFaceClear
	}
	created := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var createdAt int64
		var currentRevision sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT settings.revision,library.created_at_ms FROM libraries library LEFT JOIN face_library_settings settings ON settings.library_id=library.id WHERE library.id=?`, value.Job.LibraryID).Scan(&currentRevision, &createdAt)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceLibraryNotFound
		}
		if err != nil {
			return err
		}
		missing := !currentRevision.Valid
		settingsRevision := int64(1)
		if currentRevision.Valid {
			settingsRevision = currentRevision.Int64
		}
		if settingsRevision != value.Job.ExpectedSettingsRevision {
			return face.ErrFaceSettingsConflict
		}
		var active int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM face_analysis_jobs WHERE library_id=? AND state IN ('queued','running','cancelling')) OR EXISTS(SELECT 1 FROM face_clear_jobs WHERE library_id=? AND state IN ('queued','running','cancelling'))`, value.Job.LibraryID, value.Job.LibraryID).Scan(&active); err != nil {
			return err
		}
		if active != 0 {
			return face.ErrFaceClearConflict
		}
		if value.Job.Kind == face.ClearManual {
			actual, err := faceManualCounts(ctx, tx, value.Job.LibraryID)
			if err != nil {
				return err
			}
			if value.Job.ExpectedCounts == nil || actual != *value.Job.ExpectedCounts {
				return face.ErrFaceClearCountConflict
			}
		}
		var total int64
		if value.Job.Kind == face.ClearDerived {
			if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM face_cluster_builds WHERE library_id=?)+(SELECT COUNT(*) FROM face_observations WHERE library_id=?)+(SELECT COUNT(*) FROM face_asset_results WHERE library_id=?)`, value.Job.LibraryID, value.Job.LibraryID, value.Job.LibraryID).Scan(&total); err != nil {
				return err
			}
		} else {
			if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM person_face_anchors WHERE library_id=?)+(SELECT COUNT(*) FROM face_exclusions WHERE library_id=?)+(SELECT COUNT(*) FROM face_audit_events WHERE library_id=?)`, value.Job.LibraryID, value.Job.LibraryID, value.Job.LibraryID).Scan(&total); err != nil {
				return err
			}
		}
		if missing {
			updatedAt := max(createdAt, value.Job.CreatedAt.UnixMilli())
			_, err = tx.ExecContext(ctx, `INSERT INTO face_library_settings(library_id,enabled,state,revision,coverage_revision,created_at_ms,updated_at_ms) VALUES(?,0,'clearing',2,1,?,?)`, value.Job.LibraryID, createdAt, updatedAt)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE face_library_settings SET enabled=0,state='clearing',revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND revision=?`, value.Job.CreatedAt.UnixMilli(), value.Job.LibraryID, settingsRevision)
		}
		if err != nil {
			return err
		}
		operationKind := aimodel.OperationFaceDerivedClear
		if value.Job.Kind == face.ClearManual {
			operationKind = aimodel.OperationFaceManualClear
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO ai_model_operations(id,kind,state,phase,library_id,completed_items,total_items,revision,created_at_ms,updated_at_ms) VALUES(?,?,'queued','queued',?,0,?,1,?,?)`, value.Job.OperationID, operationKind, value.Job.LibraryID, total, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
			return err
		}
		var people, assignments, constraints any
		if value.Job.ExpectedCounts != nil {
			people = value.Job.ExpectedCounts.People
			assignments = value.Job.ExpectedCounts.Assignments
			constraints = value.Job.ExpectedCounts.Constraints
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO face_clear_jobs(id,library_id,operation_id,kind,expected_settings_revision,expected_person_count,expected_assignment_count,expected_constraint_count,state,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,?,?,?,'queued',?,?)`, value.Job.ID, value.Job.LibraryID, value.Job.OperationID, value.Job.Kind, value.Job.ExpectedSettingsRevision, people, assignments, constraints, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO face_clear_requests(idempotency_key_hash,request_hash,job_id,created_at_ms) VALUES(?,?,?,?)`, value.IdempotencyKeyHash, value.RequestHash, value.Job.ID, value.Job.CreatedAt.UnixMilli()); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		if isUniqueConstraint(err) {
			if existing, found, findErr := s.FindFaceClear(ctx, value.IdempotencyKeyHash); findErr == nil && found {
				return existing, false, nil
			}
		}
		return face.ClearAdmission{}, false, fmt.Errorf("create face clear: %w", err)
	}
	stored, found, err := s.FindFaceClear(ctx, value.IdempotencyKeyHash)
	if err != nil || !found {
		return face.ClearAdmission{}, false, firstErrorIfMissing(found, err)
	}
	return stored, created, nil
}
func (s *Store) ClaimFaceClear(ctx context.Context, now time.Time, lease time.Duration) (face.ClearJob, bool, error) {
	if now.IsZero() || lease < time.Second || lease > time.Hour {
		return face.ClearJob{}, false, face.ErrInvalidFaceClear
	}
	var claimed face.ClearJob
	found := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanFaceClearJob(tx.QueryRowContext(ctx, faceClearJobSelect+` WHERE state='queued' ORDER BY created_at_ms,id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		claimRevision := job.RequestedRevision + 1
		leaseMS := lease.Milliseconds()
		result, err := tx.ExecContext(ctx, `UPDATE face_clear_jobs SET state='running',claimed_revision=?,attempt_count=attempt_count+1,lease_expires_ms=MAX(created_at_ms+?,?),updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued' AND requested_revision=? AND attempt_count<?`, claimRevision, leaseMS, now.Add(lease).UnixMilli(), now.UnixMilli(), job.ID, job.RequestedRevision, face.MaximumFaceClearAttempts)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceClearConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='running',phase='clearing',lease_expires_ms=MAX(created_at_ms+?,?),revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued'`, leaseMS, now.Add(lease).UnixMilli(), now.UnixMilli(), job.OperationID); err != nil {
			return err
		}
		claimed, err = scanFaceClearJob(tx.QueryRowContext(ctx, faceClearJobSelect+` WHERE id=?`, job.ID))
		found = err == nil
		return err
	})
	return claimed, found, err
}

func (s *Store) RefreshFaceClearLease(ctx context.Context, job face.ClearJob, now time.Time, lease time.Duration) (bool, error) {
	if job.ID == "" || job.OperationID == "" || job.ClaimedRevision < 1 || now.IsZero() || lease < time.Second || lease > time.Hour {
		return false, face.ErrInvalidFaceClear
	}
	cancelled := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		err := tx.QueryRowContext(ctx, `SELECT state FROM face_clear_jobs WHERE id=? AND operation_id=? AND claimed_revision=?`, job.ID, job.OperationID, job.ClaimedRevision).Scan(&state)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceClearConflict
		}
		if err != nil {
			return err
		}
		if state != "running" && state != "cancelling" {
			return face.ErrFaceClearConflict
		}
		cancelled = state == "cancelling"
		deadline := now.Add(lease).UnixMilli()
		result, err := tx.ExecContext(ctx, `UPDATE face_clear_jobs SET lease_expires_ms=MAX(created_at_ms+?,?),updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND claimed_revision=? AND state IN('running','cancelling')`, lease.Milliseconds(), deadline, now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceClearConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET lease_expires_ms=MAX(created_at_ms+?,?),updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state IN('running','cancelling')`, lease.Milliseconds(), deadline, now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceClearConflict
		}
		return nil
	})
	return cancelled, err
}

func (s *Store) CancelFaceClearOperation(ctx context.Context, operationID string, expectedRevision int64, now time.Time) (face.ClearJob, error) {
	if operationID == "" || expectedRevision < 1 || now.IsZero() {
		return face.ClearJob{}, face.ErrInvalidFaceClear
	}
	var jobID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM face_clear_jobs WHERE operation_id=?`, operationID).Scan(&jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return face.ClearJob{}, face.ErrFaceClearConflict
	}
	if err != nil {
		return face.ClearJob{}, err
	}
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		var revision int64
		var libraryID int64
		err := tx.QueryRowContext(ctx, `SELECT job.state,operation.revision,job.library_id FROM face_clear_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id WHERE job.id=?`, jobID).Scan(&state, &revision, &libraryID)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceClearConflict
		}
		if err != nil {
			return err
		}
		if revision != expectedRevision {
			return face.ErrFaceClearConflict
		}
		switch state {
		case "queued":
			if _, err := tx.ExecContext(ctx, `UPDATE face_clear_jobs SET state='cancelled',requested_revision=requested_revision+1,error_code='cancelled',updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',error_code='cancelled',revision=revision+1,updated_at_ms=MAX(created_at_ms,?),finished_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued'`, now.UnixMilli(), now.UnixMilli(), operationID); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `UPDATE face_library_settings SET enabled=0,state='disabled',revision=revision+1,coverage_revision=coverage_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND state='clearing'`, now.UnixMilli(), libraryID)
			return err
		case "running":
			if _, err := tx.ExecContext(ctx, `UPDATE face_clear_jobs SET state='cancelling',requested_revision=requested_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelling',revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='running'`, now.UnixMilli(), operationID)
			return err
		case "cancelling":
			return nil
		default:
			return face.ErrFaceClearConflict
		}
	})
	if err != nil {
		return face.ClearJob{}, err
	}
	return scanFaceClearJob(s.db.QueryRowContext(ctx, faceClearJobSelect+` WHERE id=?`, jobID))
}

func (s *Store) RecoverExpiredFaceClears(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	if now.IsZero() {
		return jobs.RecoverySummary{}, face.ErrInvalidFaceClear
	}
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE face_clear_jobs SET state='cancelled',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code='cancelled',updated_at_ms=? WHERE state='cancelling' AND lease_expires_ms<=?`, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		cancelled, _ := result.RowsAffected()
		result, err = tx.ExecContext(ctx, `UPDATE face_clear_jobs SET state='queued',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code=NULL,updated_at_ms=? WHERE state='running' AND lease_expires_ms<=? AND attempt_count<?`, now.UnixMilli(), now.UnixMilli(), face.MaximumFaceClearAttempts)
		if err != nil {
			return err
		}
		summary.Requeued, _ = result.RowsAffected()
		result, err = tx.ExecContext(ctx, `UPDATE face_clear_jobs SET state='failed',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code='retry_exhausted',updated_at_ms=? WHERE state='running' AND lease_expires_ms<=? AND attempt_count>=?`, now.UnixMilli(), now.UnixMilli(), face.MaximumFaceClearAttempts)
		if err != nil {
			return err
		}
		failed, _ := result.RowsAffected()
		summary.Interrupted = cancelled + failed
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued',phase='queued',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=? WHERE id IN(SELECT operation_id FROM face_clear_jobs WHERE state='queued' AND updated_at_ms=?)`, now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='failed',phase='completed',lease_expires_ms=NULL,error_code='retry_exhausted',revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id IN(SELECT operation_id FROM face_clear_jobs WHERE state='failed' AND updated_at_ms=?)`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',lease_expires_ms=NULL,error_code='cancelled',revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id IN(SELECT operation_id FROM face_clear_jobs WHERE state='cancelled' AND updated_at_ms=?)`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE face_library_settings SET enabled=0,state='disabled',revision=revision+1,coverage_revision=coverage_revision+1,updated_at_ms=? WHERE library_id IN(SELECT library_id FROM face_clear_jobs WHERE state IN('failed','cancelled') AND updated_at_ms=?) AND state='clearing'`, now.UnixMilli(), now.UnixMilli())
		return err
	})
	return summary, err
}
func (s *Store) DeleteFaceClearBatch(ctx context.Context, job face.ClearJob, limit int, now time.Time) (int64, bool, error) {
	if job.ID == "" || job.LibraryID < 1 || job.ClaimedRevision < 1 || limit < 1 || limit > 1000 || now.IsZero() {
		return 0, false, face.ErrInvalidFaceClear
	}
	var deleted int64
	done := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state, kind string
		err := tx.QueryRowContext(ctx, `SELECT state,kind FROM face_clear_jobs WHERE id=? AND library_id=? AND claimed_revision=?`, job.ID, job.LibraryID, job.ClaimedRevision).Scan(&state, &kind)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceClearConflict
		}
		if err != nil {
			return err
		}
		if state != "running" {
			return face.ErrFaceClearConflict
		}
		var result sql.Result
		if face.ClearKind(kind) == face.ClearDerived {
			result, err = tx.ExecContext(ctx, `DELETE FROM face_cluster_builds WHERE rowid IN (SELECT rowid FROM face_cluster_builds WHERE library_id=? ORDER BY id LIMIT ?)`, job.LibraryID, limit)
			if err == nil {
				deleted, _ = result.RowsAffected()
			}
			if err == nil && deleted == 0 {
				if _, err = tx.ExecContext(ctx, `UPDATE person_face_anchors SET state='needs_review',revision=revision+1,updated_at_ms=MAX(created_at_ms,?)
				WHERE current_face_id IN (SELECT id FROM face_observations WHERE library_id=? ORDER BY id LIMIT ?)`, now.UnixMilli(), job.LibraryID, limit); err != nil {
					return err
				}
				result, err = tx.ExecContext(ctx, `DELETE FROM face_observations WHERE rowid IN (SELECT rowid FROM face_observations WHERE library_id=? ORDER BY id LIMIT ?)`, job.LibraryID, limit)
				if err == nil {
					deleted, _ = result.RowsAffected()
				}
			}
			if err == nil && deleted == 0 {
				result, err = tx.ExecContext(ctx, `DELETE FROM face_asset_results WHERE rowid IN (SELECT rowid FROM face_asset_results WHERE library_id=? ORDER BY asset_id LIMIT ?)`, job.LibraryID, limit)
				if err == nil {
					deleted, _ = result.RowsAffected()
				}
			}
			if err != nil {
				return err
			}
			var remaining int
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM face_cluster_builds WHERE library_id=?) OR EXISTS(SELECT 1 FROM face_observations WHERE library_id=?) OR EXISTS(SELECT 1 FROM face_asset_results WHERE library_id=?)`, job.LibraryID, job.LibraryID, job.LibraryID).Scan(&remaining); err != nil {
				return err
			}
			done = remaining == 0
		} else {
			result, err = tx.ExecContext(ctx, `DELETE FROM face_exclusions WHERE rowid IN (SELECT rowid FROM face_exclusions WHERE library_id=? ORDER BY id LIMIT ?)`, job.LibraryID, limit)
			if err == nil {
				deleted, _ = result.RowsAffected()
			}
			if err == nil && deleted == 0 {
				result, err = tx.ExecContext(ctx, `DELETE FROM person_face_anchors WHERE rowid IN (SELECT rowid FROM person_face_anchors WHERE library_id=? ORDER BY id LIMIT ?)`, job.LibraryID, limit)
				if err == nil {
					deleted, _ = result.RowsAffected()
				}
			}
			if err == nil && deleted == 0 {
				result, err = tx.ExecContext(ctx, `DELETE FROM face_audit_events WHERE rowid IN (SELECT rowid FROM face_audit_events WHERE library_id=? ORDER BY id LIMIT ?)`, job.LibraryID, limit)
				if err == nil {
					deleted, _ = result.RowsAffected()
				}
			}
			if err != nil {
				return err
			}
			var remaining int
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM face_exclusions WHERE library_id=?) OR EXISTS(SELECT 1 FROM person_face_anchors WHERE library_id=?) OR EXISTS(SELECT 1 FROM face_audit_events WHERE library_id=?)`, job.LibraryID, job.LibraryID, job.LibraryID).Scan(&remaining); err != nil {
				return err
			}
			done = remaining == 0
		}
		if _, err := tx.ExecContext(ctx, `UPDATE face_clear_jobs SET deleted_count=deleted_count+?,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, deleted, now.UnixMilli(), job.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET completed_items=completed_items+?,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='running'`, deleted, now.UnixMilli(), job.OperationID); err != nil {
			return err
		}
		if done {
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET total_items=completed_items WHERE id=?`, job.OperationID)
		}
		return err
	})
	return deleted, done, err
}
func (s *Store) FinishFaceClear(ctx context.Context, job face.ClearJob, succeeded bool, errorCode string, now time.Time) (face.ClearJob, error) {
	if job.ID == "" || job.ClaimedRevision < 1 || now.IsZero() || succeeded && (errorCode != "") || !succeeded && (errorCode == "" || len(errorCode) > 128) {
		return face.ClearJob{}, face.ErrInvalidFaceClear
	}
	outcome := "succeeded"
	if !succeeded {
		outcome = "failed"
	}
	if errorCode == "cancelled" {
		outcome = "cancelled"
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state, kind string
		err := tx.QueryRowContext(ctx, `SELECT state,kind FROM face_clear_jobs WHERE id=? AND claimed_revision=?`, job.ID, job.ClaimedRevision).Scan(&state, &kind)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceClearConflict
		}
		if err != nil {
			return err
		}
		if state != "running" && state != "cancelling" {
			return face.ErrFaceClearConflict
		}
		if succeeded && state != "running" || outcome == "cancelled" && state != "cancelling" || outcome == "failed" && state != "running" {
			return face.ErrFaceClearConflict
		}
		if succeeded {
			var remaining int
			if face.ClearKind(kind) == face.ClearDerived {
				if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM face_cluster_builds WHERE library_id=?) OR EXISTS(SELECT 1 FROM face_observations WHERE library_id=?) OR EXISTS(SELECT 1 FROM face_asset_results WHERE library_id=?)`, job.LibraryID, job.LibraryID, job.LibraryID).Scan(&remaining); err != nil {
					return err
				}
				if _, err := tx.ExecContext(ctx, `DELETE FROM face_library_progress WHERE library_id=?`, job.LibraryID); err != nil {
					return err
				}
			} else {
				if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM face_exclusions WHERE library_id=?) OR EXISTS(SELECT 1 FROM person_face_anchors WHERE library_id=?) OR EXISTS(SELECT 1 FROM face_audit_events WHERE library_id=?)`, job.LibraryID, job.LibraryID, job.LibraryID).Scan(&remaining); err != nil {
					return err
				}
			}
			if remaining != 0 {
				return face.ErrFaceClearConflict
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE face_library_settings SET enabled=0,state='disabled',revision=revision+1,coverage_revision=coverage_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND state='clearing'`, now.UnixMilli(), job.LibraryID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE face_clear_jobs SET state=?,requested_revision=requested_revision+1,lease_expires_ms=NULL,error_code=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND claimed_revision=?`, outcome, nullableString(errorCode), now.UnixMilli(), job.ID, job.ClaimedRevision); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state=?,phase='completed',error_code=?,lease_expires_ms=NULL,revision=revision+1,updated_at_ms=MAX(created_at_ms,?),finished_at_ms=MAX(created_at_ms,?) WHERE id=? AND state IN('running','cancelling')`, outcome, nullableString(errorCode), now.UnixMilli(), now.UnixMilli(), job.OperationID)
		return err
	})
	if err != nil {
		return face.ClearJob{}, err
	}
	return scanFaceClearJob(s.db.QueryRowContext(ctx, faceClearJobSelect+` WHERE id=?`, job.ID))
}

func faceManualCounts(ctx context.Context, tx *sql.Tx, libraryID int64) (face.ManualClearCounts, error) {
	var value face.ManualClearCounts
	err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(DISTINCT person_id) FROM person_face_anchors WHERE library_id=?),(SELECT COUNT(*) FROM person_face_anchors WHERE library_id=?),(SELECT COUNT(*) FROM face_exclusions WHERE library_id=?)+(SELECT COUNT(*) FROM face_cannot_links link WHERE EXISTS(SELECT 1 FROM person_face_anchors anchor WHERE anchor.library_id=? AND anchor.id IN(link.left_anchor_id,link.right_anchor_id)))`, libraryID, libraryID, libraryID, libraryID).Scan(&value.People, &value.Assignments, &value.Constraints)
	return value, err
}
func validFaceClearAdmission(value face.ClearAdmission) bool {
	return face.ValidReviewDigest(value.IdempotencyKeyHash) && face.ValidReviewDigest(value.RequestHash) && len(value.Job.ID) >= 8 && len(value.Job.OperationID) >= 8 && value.Job.LibraryID > 0 && (value.Job.Kind == face.ClearDerived || value.Job.Kind == face.ClearManual) && value.Job.ExpectedSettingsRevision > 0 && value.Job.State == "queued" && value.Job.RequestedRevision == 1 && value.Job.ClaimedRevision == 0 && value.Job.AttemptCount == 0 && !value.Job.CreatedAt.IsZero() && value.Job.UpdatedAt.Equal(value.Job.CreatedAt) && (value.Job.Kind == face.ClearDerived && value.Job.ExpectedCounts == nil || value.Job.Kind == face.ClearManual && value.Job.ExpectedCounts != nil)
}

type faceClearScanner interface{ Scan(...any) error }

func scanFaceClearAdmission(row faceClearScanner) (face.ClearAdmission, error) {
	var value face.ClearAdmission
	job, err := scanFaceClearJobWithPrefix(row, &value.IdempotencyKeyHash, &value.RequestHash)
	value.Job = job
	return value, err
}
func scanFaceClearJob(row faceClearScanner) (face.ClearJob, error) {
	return scanFaceClearJobWithPrefix(row)
}
func scanFaceClearJobWithPrefix(row faceClearScanner, prefix ...any) (face.ClearJob, error) {
	var value face.ClearJob
	var kind, state string
	var people, assignments, constraints, claimed sql.NullInt64
	var created, updated int64
	targets := append(prefix, &value.ID, &value.LibraryID, &value.OperationID, &kind, &value.ExpectedSettingsRevision, &people, &assignments, &constraints, &state, &value.DeletedCount, &value.RequestedRevision, &claimed, &value.AttemptCount, &created, &updated)
	err := row.Scan(targets...)
	if err != nil {
		return face.ClearJob{}, err
	}
	value.Kind = face.ClearKind(kind)
	value.State = state
	if claimed.Valid {
		value.ClaimedRevision = claimed.Int64
	}
	if people.Valid {
		value.ExpectedCounts = &face.ManualClearCounts{People: people.Int64, Assignments: assignments.Int64, Constraints: constraints.Int64}
	}
	value.CreatedAt = time.UnixMilli(created).UTC()
	value.UpdatedAt = time.UnixMilli(updated).UTC()
	return value, nil
}

var _ face.ClearQueue = (*Store)(nil)
