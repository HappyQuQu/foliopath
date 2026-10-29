package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/ocr"
)

func (s *Store) GetOCRCoverage(ctx context.Context, id int64) (ocr.Coverage, error) {
	return getOCRCoverage(ctx, s.db, id)
}
func getOCRCoverage(ctx context.Context, q queryRower, id int64) (ocr.Coverage, error) {
	var v ocr.Coverage
	err := q.QueryRowContext(ctx, `SELECT COUNT(a.id),COALESCE(SUM(r.source_fingerprint=a.source_fingerprint AND r.outcome='ready'),0),COALESCE(SUM(r.source_fingerprint=a.source_fingerprint AND r.outcome='ready' AND r.lines_json='[]'),0),COALESCE(SUM(r.source_fingerprint=a.source_fingerprint AND r.outcome='failed'),0),COALESCE(SUM(r.source_fingerprint<>a.source_fingerprint),0)
 FROM libraries l LEFT JOIN ocr_library_settings s ON s.library_id=l.id LEFT JOIN assets a ON a.library_id=l.id AND a.kind IN ('image','animated') AND a.archive_path = '' LEFT JOIN ocr_asset_results r ON r.library_id=a.library_id AND r.asset_id=a.id AND r.fingerprint=s.fingerprint WHERE l.id=? GROUP BY l.id`, id).Scan(&v.Eligible, &v.Ready, &v.Empty, &v.Failed, &v.Stale)
	if errors.Is(err, sql.ErrNoRows) {
		err = ocr.ErrNotFound
	}
	v.Pending = v.Eligible - v.Ready - v.Failed
	return v, err
}
func (s *Store) ListOCRClearingLibraries(ctx context.Context, after int64, limit int) ([]ocr.Settings, error) {
	if after < 0 || limit < 1 || limit > ocr.MaximumActiveJobs {
		return nil, ocr.ErrInvalidPersistence
	}
	rows, err := s.db.QueryContext(ctx, `SELECT library_id,revision,epoch,enabled,clearing,COALESCE(fingerprint,'') FROM ocr_library_settings WHERE clearing=1 AND library_id>? ORDER BY library_id LIMIT ?`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]ocr.Settings, 0, limit)
	for rows.Next() {
		var v ocr.Settings
		if err = rows.Scan(&v.LibraryID, &v.Revision, &v.Epoch, &v.Enabled, &v.Clearing, &v.Fingerprint); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
func (s *Store) DeferOCRJob(ctx context.Context, j ocr.Job, now time.Time, delay time.Duration) error {
	if now.UnixMilli() < 1 || delay < time.Second || delay > 30*time.Second {
		return ocr.ErrInvalidPersistence
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireOCRClaim(ctx, tx, j, now, ocrClaimProgress); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE ocr_jobs SET state='queued',claim_revision=claim_revision+1,attempt_count=attempt_count-1,lease_expires_ms=NULL,next_attempt_ms=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.Add(delay).UnixMilli(), now.UnixMilli(), j.ID)
		return err
	})
}

// Only the generic worker's signal-marked post-processor/post-heartbeat boundary
// invokes this method in production. It preserves the latest durable progress,
// never accepts caller-substituted counters, and cancellation always wins.
func (s *Store) HandoffOCRJob(ctx context.Context, j ocr.Job, now time.Time) error {
	if now.UnixMilli() < 1 {
		return ocr.ErrInvalidPersistence
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := scanOCRJob(tx.QueryRowContext(ctx, ocrJobSelect+` WHERE id=?`, j.ID))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if current.State != "running" && current.State != "cancelling" {
			return nil
		}
		// Heartbeat policy compares immutable ownership and valid lease, while the
		// cancellation purpose additionally allows acknowledging cancelling state.
		purpose := ocrClaimHandoff
		if current.State == "cancelling" {
			purpose = ocrClaimCancellation
		}
		if err = requireOCRClaim(ctx, tx, j, now, purpose); errors.Is(err, ocr.ErrConflict) {
			return nil
		} else if err != nil {
			return err
		}
		if current.State == "cancelling" {
			_, err = tx.ExecContext(ctx, `UPDATE ocr_jobs SET state='cancelled',claim_revision=claim_revision+1,lease_expires_ms=NULL,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), j.ID)
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO ocr_job_handoffs(job_id,claim_revision,attempt_before,checkpoint_id,completed,created_at_ms) VALUES(?,?,?,?,?,?)`, j.ID, j.ClaimRevision, current.Attempts, current.CheckpointID, current.Completed, now.UnixMilli()); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE ocr_jobs SET state='queued',claim_revision=claim_revision+1,attempt_count=attempt_count-1,lease_expires_ms=NULL,next_attempt_ms=0,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), j.ID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM ocr_job_handoffs WHERE job_id=? AND rowid NOT IN(SELECT rowid FROM ocr_job_handoffs WHERE job_id=? ORDER BY claim_revision DESC LIMIT 10)`, j.ID, j.ID)
		return err
	})
}

func (s *Store) GetLatestOCRJob(ctx context.Context, id int64) (ocr.Job, bool, error) {
	return getLatestOCRJob(ctx, s.db, id)
}
func getLatestOCRJob(ctx context.Context, q queryRower, id int64) (ocr.Job, bool, error) {
	if id < 1 {
		return ocr.Job{}, false, ocr.ErrInvalidPersistence
	}
	j, err := scanOCRJob(q.QueryRowContext(ctx, ocrJobSelect+` WHERE library_id=? ORDER BY created_at_ms DESC,rowid DESC LIMIT 1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return j, false, nil
	}
	return j, err == nil, err
}

func (s *Store) GetOCROverview(ctx context.Context, id int64) (ocr.Status, error) {
	var v ocr.Status
	if id < 1 {
		return v, ocr.ErrInvalidPersistence
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return v, err
	}
	defer tx.Rollback()
	v.Settings, err = getOCRSettings(ctx, tx, id)
	if err != nil {
		return v, err
	}
	v.Coverage, err = getOCRCoverage(ctx, tx, id)
	if err != nil {
		return v, err
	}
	latest, found, err := getLatestOCRJob(ctx, tx, id)
	if err != nil {
		return v, err
	}
	if found {
		v.LatestJob = &latest
	}
	return v, tx.Commit()
}
