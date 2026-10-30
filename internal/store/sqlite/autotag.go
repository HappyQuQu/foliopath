package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/jobs"
)

const automaticJobSelect = `SELECT id,operation_id,library_id,model_identity,mode,state,checkpoint_id,upper_asset_id,claim_revision,attempt_count FROM autotag_jobs`

func (s *Store) FindAutomaticTagRequest(ctx context.Context, key, request string) (aimodel.Operation, bool, error) {
	var stored, operationID string
	err := s.db.QueryRowContext(ctx, `SELECT request.request_hash,job.operation_id FROM autotag_job_requests request JOIN autotag_jobs job ON job.id=request.job_id WHERE request.key_hash=?`, key).Scan(&stored, &operationID)
	if errors.Is(err, sql.ErrNoRows) {
		return aimodel.Operation{}, false, nil
	}
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	if stored != request {
		return aimodel.Operation{}, false, aimodel.ErrIdempotencyConflict
	}
	value, err := s.GetAIOperation(ctx, operationID)
	return value, err == nil, err
}

func scanAutomaticJob(row aiModelRowScanner) (autotag.Job, error) {
	var j autotag.Job
	err := row.Scan(&j.ID, &j.OperationID, &j.LibraryID, &j.ModelIdentity, &j.Mode, &j.State, &j.CheckpointID, &j.UpperAssetID, &j.ClaimRevision, &j.AttemptCount)
	return j, err
}

func (s *Store) GetAutomaticTagSettings(ctx context.Context, id int64, identity string) (autotag.Settings, error) {
	v := autotag.Settings{LibraryID: id, Revision: 1}
	err := s.db.QueryRowContext(ctx, `SELECT library.status='offline',COALESCE(settings.enabled,0),COALESCE(settings.revision,1),COALESCE(settings.coverage_revision,1)+(SELECT content_revision FROM catalog_search_state WHERE singleton_key=1),COALESCE(settings.blocked_model_identity=?,0) FROM libraries library LEFT JOIN autotag_library_settings settings ON settings.library_id=library.id WHERE library.id=?`, identity, id).Scan(&v.Offline, &v.Enabled, &v.Revision, &v.Coverage.Revision, &v.AutomaticBlocked)
	if errors.Is(err, sql.ErrNoRows) {
		return v, autotag.ErrNotFound
	}
	if err != nil {
		return v, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(run.source_fingerprint=asset.source_fingerprint AND run.model_identity=? AND run.outcome='ready'),0),COALESCE(SUM(run.source_fingerprint=asset.source_fingerprint AND run.model_identity=? AND run.outcome='failed'),0),COALESCE(SUM(run.asset_id IS NOT NULL AND (run.source_fingerprint<>asset.source_fingerprint OR run.model_identity<>? OR run.outcome='stale')),0) FROM visible_assets asset LEFT JOIN autotag_asset_runs run ON run.library_id=asset.library_id AND run.asset_id=asset.id WHERE asset.library_id=? AND asset.kind IN ('image','animated') AND asset.archive_path = ''`, identity, identity, identity, id).Scan(&v.Coverage.Eligible, &v.Coverage.Completed, &v.Coverage.Failed, &v.Coverage.Stale)
	if err != nil {
		return v, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT operation_id FROM autotag_jobs WHERE library_id=? AND state IN ('queued','running','cancelling')`, id).Scan(&v.ActiveOperationID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return v, err
}

func (s *Store) UpdateAutomaticTagSettings(ctx context.Context, id int64, enabled bool, revision int64, now time.Time) error {
	if id < 1 || revision < 1 || now.IsZero() {
		return autotag.ErrInvalid
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var current int64
		err := tx.QueryRowContext(ctx, `SELECT COALESCE(settings.revision,1) FROM libraries library LEFT JOIN autotag_library_settings settings ON settings.library_id=library.id WHERE library.id=?`, id).Scan(&current)
		if errors.Is(err, sql.ErrNoRows) {
			return autotag.ErrNotFound
		}
		if err != nil {
			return err
		}
		if current != revision {
			return aimodel.ErrPreconditionFailed
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO autotag_library_settings(library_id,enabled,revision,updated_at_ms) VALUES(?,?,?,?) ON CONFLICT(library_id) DO UPDATE SET enabled=excluded.enabled,revision=excluded.revision,updated_at_ms=excluded.updated_at_ms`, id, enabled, current+1, now.UnixMilli()); err != nil {
			return err
		}
		if enabled {
			_, err = tx.ExecContext(ctx, `UPDATE autotag_library_settings SET blocked_model_identity=NULL,coverage_revision=coverage_revision+1 WHERE library_id=?`, id)
			return err
		}
		rows, err := tx.QueryContext(ctx, automaticJobSelect+` WHERE library_id=? AND state IN ('queued','running')`, id)
		if err != nil {
			return err
		}
		var active []autotag.Job
		for rows.Next() {
			j, e := scanAutomaticJob(rows)
			if e != nil {
				rows.Close()
				return e
			}
			active = append(active, j)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, j := range active {
			if err = cancelAutomaticJobTx(ctx, tx, j, now); err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Store) ListAutomaticTagLibraries(ctx context.Context, after int64, limit int) ([]int64, error) {
	if after < 0 || limit < 1 || limit > 50 {
		return nil, autotag.ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT library_id FROM autotag_library_settings WHERE enabled=1 AND library_id>? ORDER BY library_id LIMIT ?`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) AdmitAutomaticTagJob(ctx context.Context, a autotag.Admission, now time.Time) (aimodel.Operation, bool, error) {
	j := a.Job
	if j.LibraryID < 1 || len(j.ModelIdentity) != 64 || (j.Mode != autotag.Missing && j.Mode != autotag.All) || len(j.ID) < 8 || len(j.OperationID) < 8 || now.IsZero() || (!a.Automatic && (len(a.KeyHash) != 64 || len(a.RequestHash) != 64)) {
		return aimodel.Operation{}, false, autotag.ErrInvalid
	}
	replay := false
	empty := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if !a.Automatic {
			var hash, op string
			err := tx.QueryRowContext(ctx, `SELECT request.request_hash,job.operation_id FROM autotag_job_requests request JOIN autotag_jobs job ON job.id=request.job_id WHERE request.key_hash=?`, a.KeyHash).Scan(&hash, &op)
			if err == nil {
				if hash != a.RequestHash {
					return aimodel.ErrIdempotencyConflict
				}
				j.OperationID = op
				replay = true
				return nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		var enabled, offline bool
		err := tx.QueryRowContext(ctx, `SELECT COALESCE(settings.enabled,0),library.status='offline' FROM libraries library LEFT JOIN autotag_library_settings settings ON settings.library_id=library.id WHERE library.id=?`, j.LibraryID).Scan(&enabled, &offline)
		if errors.Is(err, sql.ErrNoRows) {
			return autotag.ErrNotFound
		}
		if err != nil {
			return err
		}
		if !enabled {
			return autotag.ErrDisabled
		}
		if offline {
			return autotag.ErrOffline
		}
		var blocked bool
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(blocked_model_identity=?,0) FROM autotag_library_settings WHERE library_id=?`, j.ModelIdentity, j.LibraryID).Scan(&blocked); err != nil {
			return err
		}
		if blocked && j.Mode != autotag.All {
			if a.Automatic {
				empty = true
				return nil
			}
			return autotag.ErrConflict
		}
		active, err := scanAutomaticJob(tx.QueryRowContext(ctx, automaticJobSelect+` WHERE library_id=? AND state IN ('queued','running','cancelling')`, j.LibraryID))
		if err == nil {
			if active.Mode != j.Mode || active.ModelIdentity != j.ModelIdentity || active.State == "cancelling" {
				return autotag.ErrConflict
			}
			j = active
		} else {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			var count int
			if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM autotag_jobs WHERE state IN ('queued','running','cancelling')`).Scan(&count); err != nil {
				return err
			}
			if count >= autotag.MaximumActiveJobs {
				return autotag.ErrConflict
			}
			if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id),0) FROM visible_assets AS assets WHERE library_id=? AND kind IN ('image','animated') AND archive_path = ''`, j.LibraryID).Scan(&j.UpperAssetID); err != nil {
				return err
			}
			if a.Automatic {
				var exists bool
				if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM visible_assets asset WHERE asset.library_id=? AND asset.kind IN ('image','animated') AND asset.archive_path = '' AND NOT EXISTS(SELECT 1 FROM autotag_asset_runs run WHERE run.library_id=asset.library_id AND run.asset_id=asset.id AND run.source_fingerprint=asset.source_fingerprint AND run.model_identity=?))`, j.LibraryID, j.ModelIdentity).Scan(&exists); err != nil {
					return err
				}
				if !exists {
					empty = true
					return nil
				}
			}
			kind := "automatic_tags_missing"
			if j.Mode == autotag.All {
				kind = "automatic_tags_rebuild"
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO ai_model_operations(id,kind,state,phase,library_id,created_at_ms,updated_at_ms) VALUES(?,?,'queued','queued',?,?,?)`, j.OperationID, kind, j.LibraryID, now.UnixMilli(), now.UnixMilli()); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO autotag_jobs(id,operation_id,library_id,model_identity,mode,state,upper_asset_id,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,'queued',?,?,?)`, j.ID, j.OperationID, j.LibraryID, j.ModelIdentity, j.Mode, j.UpperAssetID, now.UnixMilli(), now.UnixMilli()); err != nil {
				return err
			}
		}
		if !a.Automatic {
			_, err = tx.ExecContext(ctx, `INSERT INTO autotag_job_requests(key_hash,request_hash,job_id) VALUES(?,?,?)`, a.KeyHash, a.RequestHash, j.ID)
		}
		if err == nil && j.Mode == autotag.All {
			_, err = tx.ExecContext(ctx, `UPDATE autotag_library_settings SET blocked_model_identity=NULL,coverage_revision=coverage_revision+1 WHERE library_id=?`, j.LibraryID)
		}
		return err
	})
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	if empty {
		return aimodel.Operation{}, false, nil
	}
	op, err := s.GetAIOperation(ctx, j.OperationID)
	return op, replay, err
}

func (s *Store) ClaimAutomaticTagJob(ctx context.Context, now time.Time, lease time.Duration) (autotag.Job, bool, error) {
	if lease <= 0 || now.IsZero() {
		return autotag.Job{}, false, autotag.ErrInvalid
	}
	var j autotag.Job
	found := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		j, err = scanAutomaticJob(tx.QueryRowContext(ctx, automaticJobSelect+` WHERE state='queued' AND next_attempt_ms<=? ORDER BY created_at_ms,id LIMIT 1`, now.UnixMilli()))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE autotag_jobs SET state='running',claim_revision=claim_revision+1,attempt_count=attempt_count+1,next_attempt_ms=0,lease_expires_ms=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued' AND attempt_count<3`, now.Add(lease).UnixMilli(), now.UnixMilli(), j.ID)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n != 1 {
			return autotag.ErrConflict
		}
		if _, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='running',phase='building',revision=revision+1,lease_expires_ms=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued'`, now.Add(lease).UnixMilli(), now.UnixMilli(), j.OperationID); err != nil {
			return err
		}
		j.State = "running"
		j.ClaimRevision++
		j.AttemptCount++
		found = true
		return nil
	})
	return j, found, err
}
func (s *Store) RefreshAutomaticTagLease(ctx context.Context, j autotag.Job, now time.Time, lease time.Duration) (bool, error) {
	if j.ClaimRevision < 1 || lease <= 0 {
		return false, autotag.ErrInvalid
	}
	cancelled := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM autotag_jobs WHERE id=? AND claim_revision=?`, j.ID, j.ClaimRevision).Scan(&state); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				cancelled = true
				return nil
			}
			return err
		}
		if state != "running" {
			cancelled = true
			return nil
		}
		_, err := tx.ExecContext(ctx, `UPDATE autotag_jobs SET lease_expires_ms=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.Add(lease).UnixMilli(), now.UnixMilli(), j.ID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET lease_expires_ms=? WHERE id=? AND state='running'`, now.Add(lease).UnixMilli(), j.OperationID)
		return err
	})
	return cancelled, err
}
func (s *Store) NextAutomaticTagCandidate(ctx context.Context, j autotag.Job) (autotag.Candidate, bool, error) {
	var v autotag.Candidate
	err := s.db.QueryRowContext(ctx, `SELECT asset.id,asset.library_id,asset.source_fingerprint FROM visible_assets asset WHERE asset.library_id=? AND asset.id>? AND asset.id<=? AND asset.kind IN ('image','animated') AND asset.archive_path = '' AND (?='all' OR NOT EXISTS(SELECT 1 FROM autotag_asset_runs run WHERE run.library_id=asset.library_id AND run.asset_id=asset.id AND run.source_fingerprint=asset.source_fingerprint AND run.model_identity=?)) ORDER BY asset.id LIMIT 1`, j.LibraryID, j.CheckpointID, j.UpperAssetID, j.Mode, j.ModelIdentity).Scan(&v.ID, &v.LibraryID, &v.SourceFingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return v, false, nil
	}
	return v, err == nil, err
}

func (s *Store) CommitAutomaticTagResult(ctx context.Context, c autotag.Commit, now time.Time) error {
	if c.Asset.ID <= c.Job.CheckpointID || c.Asset.LibraryID != c.Job.LibraryID || c.Asset.ID > c.Job.UpperAssetID || (c.Outcome != "ready" && c.Outcome != "failed" && c.Outcome != "stale") || (c.Plan == nil) != (c.Outcome != "ready") {
		return autotag.ErrInvalid
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var permitted bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM autotag_jobs job JOIN autotag_library_settings settings ON settings.library_id=job.library_id JOIN ai_model_operations operation ON operation.id=job.operation_id WHERE job.id=? AND job.state='running' AND operation.state='running' AND job.claim_revision=? AND job.checkpoint_id=? AND job.model_identity=? AND job.library_id=? AND job.operation_id=? AND job.upper_asset_id=? AND job.lease_expires_ms>? AND settings.enabled=1)`, c.Job.ID, c.Job.ClaimRevision, c.Job.CheckpointID, c.Job.ModelIdentity, c.Job.LibraryID, c.Job.OperationID, c.Job.UpperAssetID, now.UnixMilli()).Scan(&permitted); err != nil {
			return err
		}
		if !permitted {
			return autotag.ErrConflict
		}
		outcome := c.Outcome
		if c.Plan != nil {
			if c.Plan.AssetID != c.Asset.ID || c.Plan.LibraryID != c.Job.LibraryID || c.Plan.SourceFingerprint != c.Asset.SourceFingerprint || c.Plan.ModelIdentity != c.Job.ModelIdentity {
				return autotag.ErrInvalid
			}
			if _, err := mergeAutomaticAssetTagsTx(ctx, tx, *c.Plan, now); err != nil {
				if !errors.Is(err, curation.ErrAutomaticTagSourceChanged) {
					return err
				}
				outcome = "stale"
			}
		}
		// The failed/stale attempt retains the attempted fingerprint, not a newer
		// catalog identity; the new source remains eligible on the next bounded pass.
		if _, err := tx.ExecContext(ctx, `INSERT INTO autotag_asset_runs(library_id,asset_id,source_fingerprint,model_identity,outcome,updated_at_ms) SELECT ?,?,?,?,?,? WHERE EXISTS(SELECT 1 FROM visible_assets AS assets WHERE library_id=? AND id=?) ON CONFLICT(library_id,asset_id) DO UPDATE SET source_fingerprint=excluded.source_fingerprint,model_identity=excluded.model_identity,outcome=excluded.outcome,updated_at_ms=excluded.updated_at_ms`, c.Job.LibraryID, c.Asset.ID, c.Asset.SourceFingerprint, c.Job.ModelIdentity, outcome, now.UnixMilli(), c.Job.LibraryID, c.Asset.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE autotag_jobs SET checkpoint_id=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, c.Asset.ID, now.UnixMilli(), c.Job.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE autotag_library_settings SET coverage_revision=coverage_revision+1 WHERE library_id=?`, c.Job.LibraryID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET completed_items=completed_items+1,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), c.Job.OperationID)
		return err
	})
}

func (s *Store) DeferAutomaticTagJob(ctx context.Context, j autotag.Job, now time.Time, delay time.Duration) error {
	if now.IsZero() || delay != autotag.BusyRetryDelay || j.ClaimRevision < 1 {
		return autotag.ErrInvalid
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		// Busy is not a failed inference or an interrupted worker. Fence the old
		// claimant, undo only this live claim's crash-attempt increment, and keep
		// the operation/checkpoint unchanged while releasing the background slot.
		result, err := tx.ExecContext(ctx, `UPDATE autotag_jobs SET state='queued',claim_revision=claim_revision+1,attempt_count=attempt_count-1,next_attempt_ms=?,lease_expires_ms=NULL,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND operation_id=? AND library_id=? AND model_identity=? AND claim_revision=? AND checkpoint_id=? AND upper_asset_id=? AND state='running' AND attempt_count>0 AND lease_expires_ms>? AND EXISTS(SELECT 1 FROM autotag_library_settings WHERE library_id=autotag_jobs.library_id AND enabled=1) AND EXISTS(SELECT 1 FROM ai_model_operations WHERE id=autotag_jobs.operation_id AND state='running')`, now.Add(delay).UnixMilli(), now.UnixMilli(), j.ID, j.OperationID, j.LibraryID, j.ModelIdentity, j.ClaimRevision, j.CheckpointID, j.UpperAssetID, now.UnixMilli())
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n != 1 {
			return autotag.ErrConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued',phase='queued',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), j.OperationID)
		return err
	})
}

func finishAutomaticJobTx(ctx context.Context, tx *sql.Tx, j autotag.Job, state, code string, now time.Time) error {
	if state == "failed" {
		if err := blockAutomaticLibraryTx(ctx, tx, j); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE autotag_jobs SET state=?,lease_expires_ms=NULL,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, state, now.UnixMilli(), j.ID); err != nil {
		return err
	}
	var reason any
	if code != "" {
		reason = code
	}
	_, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state=?,phase='completed',total_items=completed_items,error_code=?,revision=revision+1,lease_expires_ms=NULL,updated_at_ms=MAX(created_at_ms,?),finished_at_ms=MAX(created_at_ms,?) WHERE id=?`, state, reason, now.UnixMilli(), now.UnixMilli(), j.OperationID)
	return err
}
func (s *Store) FinishAutomaticTagJob(ctx context.Context, j autotag.Job, state, code string, now time.Time) error {
	if state != "succeeded" && state != "failed" && state != "cancelled" {
		return autotag.ErrInvalid
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := scanAutomaticJob(tx.QueryRowContext(ctx, automaticJobSelect+` WHERE id=?`, j.ID))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if current.ClaimRevision != j.ClaimRevision {
			return autotag.ErrConflict
		}
		// Shutdown cancellation must not terminalize a durable running job.
		if state == "cancelled" && current.State != "cancelling" {
			return nil
		}
		if current.State == "cancelling" {
			state, code = "cancelled", "cancelled"
		} else if current.State != "running" {
			return autotag.ErrConflict
		}
		return finishAutomaticJobTx(ctx, tx, current, state, code, now)
	})
}
func cancelAutomaticJobTx(ctx context.Context, tx *sql.Tx, j autotag.Job, now time.Time) error {
	if j.State != "queued" && j.State != "running" && j.State != "cancelling" {
		return aimodel.ErrOperationAlreadyFinished
	}
	if err := blockAutomaticLibraryTx(ctx, tx, j); err != nil {
		return err
	}
	if j.State == "queued" {
		return finishAutomaticJobTx(ctx, tx, j, "cancelled", "cancelled", now)
	}
	if j.State == "cancelling" {
		return nil
	}
	if j.State != "running" {
		return aimodel.ErrOperationAlreadyFinished
	}
	if _, err := tx.ExecContext(ctx, `UPDATE autotag_jobs SET state='cancelling',updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), j.ID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelling',revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), j.OperationID)
	return err
}

func blockAutomaticLibraryTx(ctx context.Context, tx *sql.Tx, j autotag.Job) error {
	_, err := tx.ExecContext(ctx, `UPDATE autotag_library_settings SET blocked_model_identity=?,coverage_revision=coverage_revision+1 WHERE library_id=?`, j.ModelIdentity, j.LibraryID)
	return err
}
func (s *Store) CancelAutomaticTagOperation(ctx context.Context, id string, revision int64, now time.Time) (aimodel.Operation, error) {
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		j, err := scanAutomaticJob(tx.QueryRowContext(ctx, automaticJobSelect+` WHERE operation_id=?`, id))
		if errors.Is(err, sql.ErrNoRows) {
			return aimodel.ErrOperationNotFound
		}
		if err != nil {
			return err
		}
		var current int64
		if err = tx.QueryRowContext(ctx, `SELECT revision FROM ai_model_operations WHERE id=?`, id).Scan(&current); err != nil {
			return err
		}
		if current != revision {
			return aimodel.ErrPreconditionFailed
		}
		return cancelAutomaticJobTx(ctx, tx, j, now)
	})
	if err != nil {
		return aimodel.Operation{}, err
	}
	return s.GetAIOperation(ctx, id)
}
func (s *Store) RecoverAutomaticTagJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	var result jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, automaticJobSelect+` WHERE state IN ('running','cancelling') AND lease_expires_ms<=? ORDER BY created_at_ms,id LIMIT 32`, now.UnixMilli())
		if err != nil {
			return err
		}
		var values []autotag.Job
		for rows.Next() {
			j, e := scanAutomaticJob(rows)
			if e != nil {
				rows.Close()
				return e
			}
			values = append(values, j)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, j := range values {
			if j.State == "cancelling" || j.AttemptCount >= 3 {
				state, code := "failed", "operation_interrupted"
				if j.State == "cancelling" {
					state, code = "cancelled", "cancelled"
				}
				if err = finishAutomaticJobTx(ctx, tx, j, state, code, now); err != nil {
					return err
				}
				result.Interrupted++
				continue
			}
			if _, err = tx.ExecContext(ctx, `UPDATE autotag_jobs SET state='queued',lease_expires_ms=NULL,claim_revision=claim_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), j.ID); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued',phase='queued',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), j.OperationID); err != nil {
				return err
			}
			result.Requeued++
		}
		return nil
	})
	return result, err
}
