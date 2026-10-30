package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

const ocrJobSelect = `SELECT id,library_id,epoch,fingerprint,mode,state,checkpoint_id,upper_asset_id,claim_revision,attempt_count,completed,failed,error_code FROM ocr_jobs`

func scanOCRJob(row interface{ Scan(...any) error }) (ocr.Job, error) {
	var j ocr.Job
	err := row.Scan(&j.ID, &j.LibraryID, &j.Epoch, &j.Fingerprint, &j.Mode, &j.State, &j.CheckpointID, &j.UpperAssetID, &j.ClaimRevision, &j.Attempts, &j.Completed, &j.Failed, &j.ErrorCode)
	return j, err
}
func (s *Store) GetOCRJob(ctx context.Context, id string) (ocr.Job, error) {
	if !ocr.ValidOpaque(id) {
		return ocr.Job{}, ocr.ErrInvalidPersistence
	}
	j, err := scanOCRJob(s.db.QueryRowContext(ctx, ocrJobSelect+` WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		err = ocr.ErrNotFound
	}
	return j, err
}
func (s *Store) AdmitOCRJob(ctx context.Context, id, revision int64, mode ocr.Mode, key, jobID string, now time.Time) (ocr.Job, bool, error) {
	var j ocr.Job
	replay := false
	if id < 1 || revision < 1 || !ocr.ValidMode(mode) || !ocr.ValidOpaque(key) || !ocr.ValidOpaque(jobID) || now.UnixMilli() < 1 {
		return j, false, ocr.ErrInvalidPersistence
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := getOCRSettings(ctx, tx, id)
		if err != nil {
			return err
		}
		keyHash, requestHash, err := ocr.RequestHashes(id, current.Epoch, current.Fingerprint, mode, key)
		if err != nil {
			return err
		}
		var stored, existing string
		err = tx.QueryRowContext(ctx, `SELECT request_hash,job_id FROM ocr_job_requests WHERE key_hash=?`, keyHash).Scan(&stored, &existing)
		if err == nil {
			if stored != requestHash {
				return ocr.ErrConflict
			}
			j, err = scanOCRJob(tx.QueryRowContext(ctx, ocrJobSelect+` WHERE id=?`, existing))
			replay = true
			return err
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if current.Revision != revision {
			return ocr.ErrRevision
		}
		if current.Clearing {
			return ocr.ErrConflict
		}
		if !current.Enabled {
			return ocr.ErrDisabled
		}
		var status string
		if err = tx.QueryRowContext(ctx, `SELECT status FROM libraries WHERE id=?`, id).Scan(&status); err != nil {
			return err
		}
		if status != "ready" {
			return ocr.ErrOffline
		}
		var active, capacity int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(library_id=?),0) FROM ocr_jobs WHERE state IN ('queued','running','cancelling')`, id).Scan(&capacity, &active); err != nil {
			return err
		}
		if active > 0 || capacity >= ocr.MaximumActiveJobs {
			return ocr.ErrConflict
		}
		var upper int64
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id),0) FROM visible_assets AS assets WHERE library_id=? AND kind IN ('image','animated') AND archive_path = ''`, id).Scan(&upper); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ocr_jobs(id,library_id,epoch,fingerprint,mode,state,upper_asset_id,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,'queued',?,?,?)`, jobID, id, current.Epoch, current.Fingerprint, mode, upper, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO ocr_job_requests VALUES(?,?,?)`, keyHash, requestHash, jobID); err != nil {
			return err
		}
		j, err = scanOCRJob(tx.QueryRowContext(ctx, ocrJobSelect+` WHERE id=?`, jobID))
		return err
	})
	return j, replay, err
}
func validOCRLease(now time.Time, lease time.Duration) bool {
	return now.UnixMilli() > 0 && lease >= time.Second && lease <= jobs.DefaultLeaseDuration
}
func (s *Store) ClaimOCRJob(ctx context.Context, now time.Time, lease time.Duration) (ocr.Job, bool, error) {
	var j ocr.Job
	found := false
	if !validOCRLease(now, lease) {
		return j, false, ocr.ErrInvalidPersistence
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var id string
		err := tx.QueryRowContext(ctx, `SELECT j.id FROM ocr_jobs j JOIN ocr_library_settings s ON s.library_id=j.library_id JOIN libraries l ON l.id=j.library_id WHERE j.state='queued' AND j.next_attempt_ms<=? AND j.attempt_count<3 AND s.enabled=1 AND s.clearing=0 AND s.epoch=j.epoch AND s.fingerprint=j.fingerprint AND l.status='ready' ORDER BY j.created_at_ms,j.id LIMIT 1`, now.UnixMilli()).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE ocr_jobs SET state='running',claim_revision=claim_revision+1,attempt_count=attempt_count+1,lease_expires_ms=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.Add(lease).UnixMilli(), now.UnixMilli(), id); err != nil {
			return err
		}
		j, err = scanOCRJob(tx.QueryRowContext(ctx, ocrJobSelect+` WHERE id=?`, id))
		found = err == nil
		return err
	})
	return j, found, err
}

// One canonical fence is shared by candidate reads, commits, refresh and finish.
// The complete durable token is compared so cross-library/model/checkpoint
// substitutions cannot reuse a valid claim revision.
type ocrClaimPurpose uint8

const (
	ocrClaimProgress ocrClaimPurpose = iota
	ocrClaimHeartbeat
	ocrClaimFailure
	ocrClaimCancellation
	ocrClaimHandoff
)

func requireOCRClaim(ctx context.Context, tx *sql.Tx, j ocr.Job, now time.Time, purpose ocrClaimPurpose) error {
	stored, err := scanOCRJob(tx.QueryRowContext(ctx, ocrJobSelect+` WHERE id=?`, j.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return ocr.ErrConflict
	}
	if err != nil {
		return err
	}
	var expires sql.NullInt64
	if err = tx.QueryRowContext(ctx, `SELECT lease_expires_ms FROM ocr_jobs WHERE id=?`, j.ID).Scan(&expires); err != nil {
		return err
	}
	// A cancellation changes state only; allow acknowledging that exact claim.
	if purpose == ocrClaimCancellation && stored.State == "cancelling" {
		stored.State = j.State
	}
	if purpose == ocrClaimHeartbeat || purpose == ocrClaimCancellation || purpose == ocrClaimHandoff {
		stored.CheckpointID, stored.Completed, stored.Failed = j.CheckpointID, j.Completed, j.Failed
	}
	if stored != j || j.State != "running" || !expires.Valid || expires.Int64 <= now.UnixMilli() {
		return ocr.ErrConflict
	}
	// Terminal failure can release an owned claim when its source goes offline.
	// Configuration changes still win because they change the job to cancelling.
	if purpose == ocrClaimCancellation || purpose == ocrClaimFailure || purpose == ocrClaimHandoff {
		return nil
	}
	var usable bool
	var status string
	err = tx.QueryRowContext(ctx, `SELECT s.enabled=1 AND s.clearing=0 AND s.epoch=? AND s.fingerprint=?,l.status FROM ocr_library_settings s JOIN libraries l ON l.id=s.library_id WHERE s.library_id=?`, j.Epoch, j.Fingerprint, j.LibraryID).Scan(&usable, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ocr.ErrConflict
	}
	if err != nil {
		return err
	}
	if !usable {
		return ocr.ErrConflict
	}
	if purpose != ocrClaimHeartbeat && status != "ready" {
		if status == "scanning" || status == "pending" {
			return ocr.ErrScanning
		}
		return ocr.ErrOffline
	}
	return nil
}
func (s *Store) RefreshOCRLease(ctx context.Context, j ocr.Job, now time.Time, lease time.Duration) error {
	if !validOCRLease(now, lease) {
		return ocr.ErrInvalidPersistence
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireOCRClaim(ctx, tx, j, now, ocrClaimHeartbeat); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE ocr_jobs SET lease_expires_ms=MAX(lease_expires_ms,?),updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.Add(lease).UnixMilli(), now.UnixMilli(), j.ID)
		return err
	})
}
func (s *Store) RecoverOCRJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	var summary jobs.RecoverySummary
	if now.UnixMilli() < 1 {
		return summary, ocr.ErrInvalidPersistence
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT id,state,attempt_count FROM ocr_jobs WHERE state IN ('running','cancelling') AND lease_expires_ms<=? ORDER BY lease_expires_ms,id LIMIT ?`, now.UnixMilli(), ocr.MaximumActiveJobs)
		if err != nil {
			return err
		}
		type expired struct {
			id, state string
			attempts  int
		}
		var batch []expired
		for rows.Next() {
			var e expired
			if err = rows.Scan(&e.id, &e.state, &e.attempts); err != nil {
				_ = rows.Close()
				return err
			}
			batch = append(batch, e)
		}
		if err = rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err = rows.Close(); err != nil {
			return err
		}
		for _, e := range batch {
			state, code := "queued", ""
			switch {
			case e.state == "cancelling":
				state = "cancelled"
				summary.Interrupted++
			case e.attempts >= ocr.MaximumAttempts:
				state, code = "failed", "interrupted"
				summary.Interrupted++
			default:
				summary.Requeued++
			}
			if _, err = tx.ExecContext(ctx, `UPDATE ocr_jobs SET state=?,error_code=?,claim_revision=claim_revision+1,lease_expires_ms=NULL,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, state, code, now.UnixMilli(), e.id); err != nil {
				return err
			}
		}
		return nil
	})
	return summary, err
}
func (s *Store) CancelOCRJob(ctx context.Context, id string, revision int64, now time.Time) error {
	if !ocr.ValidOpaque(id) || revision < 0 || now.UnixMilli() < 1 {
		return ocr.ErrInvalidPersistence
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		j, err := scanOCRJob(tx.QueryRowContext(ctx, ocrJobSelect+` WHERE id=?`, id))
		if errors.Is(err, sql.ErrNoRows) {
			return ocr.ErrNotFound
		}
		if err != nil {
			return err
		}
		if j.ClaimRevision != revision {
			return ocr.ErrRevision
		}
		if j.State != "queued" && j.State != "running" && j.State != "cancelling" {
			return ocr.ErrConflict
		}
		state := "cancelling"
		if j.State == "queued" {
			state = "cancelled"
		}
		_, err = tx.ExecContext(ctx, `UPDATE ocr_jobs SET state=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, state, now.UnixMilli(), id)
		return err
	})
}
func (s *Store) FinishOCRJob(ctx context.Context, j ocr.Job, state, code string, now time.Time) error {
	if now.UnixMilli() < 1 {
		return ocr.ErrInvalidPersistence
	}
	switch state {
	case "succeeded", "cancelled":
		if code != "" {
			return ocr.ErrInvalidPersistence
		}
	case "failed":
		switch code {
		case "model_unavailable", "identity_mismatch", "inference_failed", "output_limit", "preprocess_failed", "source_unavailable":
		default:
			return ocr.ErrInvalidPersistence
		}
	default:
		return ocr.ErrInvalidPersistence
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		purpose := ocrClaimProgress
		if state == "cancelled" {
			purpose = ocrClaimCancellation
		} else if state == "failed" {
			purpose = ocrClaimFailure
		}
		if err := requireOCRClaim(ctx, tx, j, now, purpose); err != nil {
			return err
		}
		if state == "cancelled" {
			var current string
			if err := tx.QueryRowContext(ctx, `SELECT state FROM ocr_jobs WHERE id=?`, j.ID).Scan(&current); err != nil {
				return err
			}
			if current != "cancelling" {
				return ocr.ErrConflict
			}
		}
		if state == "succeeded" {
			_, found, err := nextOCRCandidate(ctx, tx, j)
			if err != nil {
				return err
			}
			if found {
				return ocr.ErrConflict
			}
		}
		_, err := tx.ExecContext(ctx, `UPDATE ocr_jobs SET state=?,error_code=?,claim_revision=claim_revision+1,lease_expires_ms=NULL,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, state, code, now.UnixMilli(), j.ID)
		return err
	})
}
