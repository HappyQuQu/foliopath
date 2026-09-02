package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"time"
)

const faceJobSelect = `SELECT job.id,job.library_id,job.generation_id,job.operation_id,job.mode,job.state,job.checkpoint_id,job.requested_revision,job.claimed_revision,job.attempt_count,job.lease_expires_ms,job.error_code,operation.revision,operation.completed_items,operation.total_items,job.created_at_ms,job.updated_at_ms FROM face_analysis_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id`
const faceAdmissionSelect = `SELECT request.idempotency_key_hash,request.request_hash,
    job.id,job.library_id,job.generation_id,job.operation_id,job.mode,job.state,
    job.checkpoint_id,job.requested_revision,job.claimed_revision,job.attempt_count,
    job.lease_expires_ms,job.error_code,operation.revision,operation.completed_items,
    operation.total_items,job.created_at_ms,job.updated_at_ms
    FROM face_analysis_job_requests request
    JOIN face_analysis_jobs job ON job.id=request.job_id
    JOIN ai_model_operations operation ON operation.id=job.operation_id`

func (s *Store) FaceJobLibraryState(ctx context.Context, libraryID int64, generationID string) (face.JobLibraryState, error) {
	if libraryID < 1 || len(generationID) < 8 || len(generationID) > 128 {
		return "", face.ErrInvalidFaceJob
	}
	var status string
	if err := s.db.QueryRowContext(ctx, `SELECT status FROM libraries WHERE id=?`, libraryID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", face.ErrFaceLibraryNotFound
		}
		return "", err
	}
	switch status {
	case "offline":
		return face.JobLibraryOffline, nil
	case "ready":
	default:
		return face.JobLibraryNotReady, nil
	}
	var enabled int
	var activeGeneration, settingsState string
	err := s.db.QueryRowContext(ctx, `SELECT enabled,active_generation_id,state FROM face_library_settings WHERE library_id=?`, libraryID).Scan(&enabled, &activeGeneration, &settingsState)
	if errors.Is(err, sql.ErrNoRows) {
		return face.JobLibraryDisabled, nil
	}
	if err != nil {
		return "", err
	}
	if enabled != 1 {
		return face.JobLibraryDisabled, nil
	}
	if settingsState == "awaiting_model" {
		return face.JobLibraryModelUnavailable, nil
	}
	if !faceSettingsRunnableState(settingsState) {
		return face.JobLibraryNotReady, nil
	}
	if activeGeneration != generationID {
		return face.JobLibraryModelUnavailable, nil
	}
	var generationState string
	if err := s.db.QueryRowContext(ctx, `SELECT state FROM face_generations WHERE id=?`, generationID).Scan(&generationState); errors.Is(err, sql.ErrNoRows) {
		return face.JobLibraryModelUnavailable, nil
	} else if err != nil {
		return "", err
	} else if generationState != "active" {
		return face.JobLibraryModelUnavailable, nil
	}
	return face.JobLibraryReady, nil
}

func (s *Store) CountFaceJobCandidates(ctx context.Context, libraryID int64, generationID string, mode face.JobMode) (face.JobCandidateCounts, error) {
	if libraryID < 1 || len(generationID) < 8 || (mode != face.JobMissing && mode != face.JobAll) {
		return face.JobCandidateCounts{}, face.ErrInvalidFaceJob
	}
	var value face.JobCandidateCounts
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assets WHERE library_id=? AND kind IN ('image','animated')`, libraryID).Scan(&value.Eligible); err != nil {
		return value, err
	}
	if mode == face.JobAll {
		value.Pending = value.Eligible
		return value, nil
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assets asset WHERE asset.library_id=? AND asset.kind IN ('image','animated') AND NOT EXISTS(SELECT 1 FROM face_asset_results result WHERE result.generation_id=? AND result.library_id=asset.library_id AND result.asset_id=asset.id AND result.source_fingerprint=asset.source_fingerprint)`, libraryID, generationID).Scan(&value.Pending); err != nil {
		return value, err
	}
	return value, nil
}

func (s *Store) ListFaceJobCandidates(ctx context.Context, libraryID int64, generationID string, mode face.JobMode, checkpointID int64, limit int) (face.JobCandidatePage, error) {
	if libraryID < 1 || len(generationID) < 8 || len(generationID) > 128 || checkpointID < 0 || limit < 1 || limit > 1000 || (mode != face.JobMissing && mode != face.JobAll) {
		return face.JobCandidatePage{}, face.ErrInvalidFaceJob
	}
	query := `SELECT asset.id,asset.source_fingerprint FROM assets asset
		WHERE asset.library_id=? AND asset.kind IN ('image','animated') AND asset.id>?`
	args := []any{libraryID, checkpointID}
	if mode == face.JobMissing {
		query += ` AND NOT EXISTS(SELECT 1 FROM face_asset_results result
			WHERE result.generation_id=? AND result.library_id=asset.library_id
			AND result.asset_id=asset.id AND result.source_fingerprint=asset.source_fingerprint)`
		args = append(args, generationID)
	}
	query += ` ORDER BY asset.id LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return face.JobCandidatePage{}, err
	}
	defer rows.Close()
	page := face.JobCandidatePage{Items: make([]face.JobCandidate, 0, limit)}
	for rows.Next() {
		var item face.JobCandidate
		if err := rows.Scan(&item.AssetID, &item.SourceFingerprint); err != nil {
			return face.JobCandidatePage{}, err
		}
		if len(page.Items) == limit {
			page.HasMore = true
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, rows.Err()
}
func (s *Store) FindFaceJob(ctx context.Context, keyHash string) (face.JobAdmission, bool, error) {
	value, err := scanFaceAdmission(s.db.QueryRowContext(ctx, faceAdmissionSelect+` WHERE request.idempotency_key_hash=?`, keyHash))
	if errors.Is(err, sql.ErrNoRows) {
		return face.JobAdmission{}, false, nil
	}
	return value, err == nil, err
}
func (s *Store) CreateFaceJob(ctx context.Context, value face.JobAdmission) (face.JobAdmission, bool, bool, error) {
	if !validFaceJobAdmission(value) {
		return face.JobAdmission{}, false, false, face.ErrInvalidFaceJob
	}
	created, coalesced := false, false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var enabled int
		var activeGeneration, state, libraryStatus string
		if err := tx.QueryRowContext(ctx, `SELECT settings.enabled,settings.active_generation_id,settings.state,library.status FROM face_library_settings settings JOIN libraries library ON library.id=settings.library_id WHERE settings.library_id=?`, value.Job.LibraryID).Scan(&enabled, &activeGeneration, &state, &libraryStatus); errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceModelUnavailable
		} else if err != nil {
			return err
		}
		if libraryStatus == "offline" {
			return face.ErrFaceLibraryOffline
		}
		if enabled != 1 {
			return face.ErrFaceDisabled
		}
		if libraryStatus != "ready" || !faceSettingsRunnableState(state) {
			return face.ErrFaceNotReady
		}
		if activeGeneration != value.Job.GenerationID {
			return face.ErrFaceModelUnavailable
		}
		var generationState string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM face_generations WHERE id=?`, value.Job.GenerationID).Scan(&generationState); errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceModelUnavailable
		} else if err != nil {
			return err
		} else if generationState != "active" {
			return face.ErrFaceModelUnavailable
		}
		var clearActive int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM face_clear_jobs WHERE library_id=? AND state IN ('queued','running','cancelling'))`, value.Job.LibraryID).Scan(&clearActive); err != nil {
			return err
		}
		if clearActive != 0 {
			return face.ErrFaceJobConflict
		}
		var activeID, activeMode string
		err := tx.QueryRowContext(ctx, `SELECT id,mode FROM face_analysis_jobs WHERE library_id=? AND state IN ('queued','running','cancelling')`, value.Job.LibraryID).Scan(&activeID, &activeMode)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		jobID := value.Job.ID
		if err == nil {
			if activeMode != string(value.Job.Mode) {
				return face.ErrFaceJobConflict
			}
			jobID = activeID
			coalesced = true
		} else {
			completed := value.EligibleCount - value.Job.TotalItems
			if value.Job.Mode == face.JobAll {
				completed = 0
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO face_library_progress(generation_id,library_id,eligible_count,completed_count,failed_count,stale_count,checkpoint_id,revision,updated_at_ms) VALUES(?,?,?, ?,0,0,0,1,?) ON CONFLICT(generation_id,library_id) DO UPDATE SET eligible_count=excluded.eligible_count,completed_count=excluded.completed_count,failed_count=0,stale_count=0,checkpoint_id=0,revision=face_library_progress.revision+1,updated_at_ms=MAX(face_library_progress.updated_at_ms,excluded.updated_at_ms)`, value.Job.GenerationID, value.Job.LibraryID, value.EligibleCount, completed, value.Job.CreatedAt.UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO ai_model_operations(id,kind,state,phase,library_id,completed_items,total_items,revision,created_at_ms,updated_at_ms) VALUES(?,?,'queued','queued',?,0,?,1,?,?)`, value.Job.OperationID, value.Job.OperationKind(), value.Job.LibraryID, value.Job.TotalItems, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO face_analysis_jobs(id,library_id,generation_id,operation_id,mode,state,checkpoint_id,requested_revision,attempt_count,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,'queued',0,1,0,?,?)`, jobID, value.Job.LibraryID, value.Job.GenerationID, value.Job.OperationID, value.Job.Mode, value.Job.CreatedAt.UnixMilli(), value.Job.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE face_library_settings SET state='building',coverage_revision=coverage_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=?`, value.Job.CreatedAt.UnixMilli(), value.Job.LibraryID); err != nil {
				return err
			}
			created = true
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO face_analysis_job_requests(idempotency_key_hash,request_hash,job_id,created_at_ms) VALUES(?,?,?,?)`, value.IdempotencyKeyHash, value.RequestHash, jobID, value.Job.CreatedAt.UnixMilli())
		return err
	})
	if err != nil {
		if isUniqueConstraint(err) {
			if existing, found, findErr := s.FindFaceJob(ctx, value.IdempotencyKeyHash); findErr == nil && found {
				return existing, false, false, nil
			}
		}
		return face.JobAdmission{}, false, false, fmt.Errorf("create face job: %w", err)
	}
	stored, found, err := s.FindFaceJob(ctx, value.IdempotencyKeyHash)
	if err != nil || !found {
		return face.JobAdmission{}, false, false, firstErrorIfMissing(found, err)
	}
	return stored, created, coalesced, nil
}
func (s *Store) ClaimFaceJob(ctx context.Context, now time.Time, lease time.Duration) (face.AnalysisJob, bool, error) {
	if now.IsZero() || lease < time.Second || lease > time.Hour {
		return face.AnalysisJob{}, false, face.ErrInvalidFaceJob
	}
	var claimed face.AnalysisJob
	found := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanFaceJob(tx.QueryRowContext(ctx, faceJobSelect+` WHERE job.state='queued' ORDER BY job.created_at_ms,job.id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		claim := job.RequestedRevision + 1
		leaseMS := lease.Milliseconds()
		result, err := tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET state='running',claimed_revision=?,attempt_count=attempt_count+1,lease_expires_ms=MAX(created_at_ms+?,?),error_code=NULL,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued' AND requested_revision=? AND attempt_count<?`, claim, leaseMS, now.Add(lease).UnixMilli(), now.UnixMilli(), job.ID, job.RequestedRevision, face.MaximumFaceJobAttempts)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceJobConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='running',phase='building',lease_expires_ms=MAX(created_at_ms+?,?),revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued'`, leaseMS, now.Add(lease).UnixMilli(), now.UnixMilli(), job.OperationID); err != nil {
			return err
		}
		claimed, err = scanFaceJob(tx.QueryRowContext(ctx, faceJobSelect+` WHERE job.id=?`, job.ID))
		found = err == nil
		return err
	})
	return claimed, found, err
}

func (s *Store) RefreshFaceJobLease(ctx context.Context, job face.AnalysisJob, now time.Time, lease time.Duration) (bool, error) {
	if job.ID == "" || job.OperationID == "" || job.ClaimedRevision < 1 || now.IsZero() || lease < time.Second || lease > time.Hour {
		return false, face.ErrInvalidFaceJob
	}
	cancelled := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		err := tx.QueryRowContext(ctx, `SELECT state FROM face_analysis_jobs WHERE id=? AND operation_id=? AND claimed_revision=?`, job.ID, job.OperationID, job.ClaimedRevision).Scan(&state)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceJobConflict
		}
		if err != nil {
			return err
		}
		if state != "running" && state != "cancelling" {
			return face.ErrFaceJobConflict
		}
		cancelled = state == "cancelling"
		deadline := now.Add(lease).UnixMilli()
		result, err := tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET lease_expires_ms=MAX(created_at_ms+?,?),updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND claimed_revision=? AND state IN('running','cancelling')`, lease.Milliseconds(), deadline, now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET lease_expires_ms=MAX(created_at_ms+?,?),updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state IN('running','cancelling')`, lease.Milliseconds(), deadline, now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceJobConflict
		}
		return nil
	})
	return cancelled, err
}

const faceJobProgressSelect = `SELECT generation_id,library_id,eligible_count,completed_count,failed_count,stale_count,checkpoint_id,revision,updated_at_ms FROM face_library_progress WHERE generation_id=? AND library_id=?`

func (s *Store) GetFaceJobProgress(ctx context.Context, generationID string, libraryID int64) (face.JobProgress, bool, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 {
		return face.JobProgress{}, false, face.ErrInvalidFaceJob
	}
	value, err := scanFaceJobProgress(s.db.QueryRowContext(ctx, faceJobProgressSelect, generationID, libraryID))
	if errors.Is(err, sql.ErrNoRows) {
		return face.JobProgress{}, false, nil
	}
	return value, err == nil, err
}

func (s *Store) CommitFaceJobProgress(ctx context.Context, commit face.JobProgressCommit) (face.JobProgress, error) {
	if commit.JobID == "" || commit.ClaimedRevision < 1 || commit.ExpectedProgressRevision < 1 || commit.ExpectedCheckpointID < 0 || commit.NextCheckpointID <= commit.ExpectedCheckpointID || len(commit.SourceFingerprint) < 1 || len(commit.SourceFingerprint) > 256 || commit.UpdatedAt.IsZero() || commit.FailedCount < 0 || commit.StaleCount < 0 || commit.FailedCount > 1 || commit.StaleCount > 1 || commit.FailedCount+commit.StaleCount > 1 {
		return face.JobProgress{}, face.ErrInvalidFaceJob
	}
	success := commit.FailedCount == 0 && commit.StaleCount == 0
	if success {
		if err := face.ValidateObservationBatch(commit.Batch, min(s.maxBatchSize, face.MaxCandidatesPerAsset)); err != nil {
			return face.JobProgress{}, err
		}
		for _, item := range commit.Batch.Items {
			if item.SourceFingerprint != commit.SourceFingerprint {
				return face.JobProgress{}, face.ErrInvalidFaceJob
			}
		}
	} else if len(commit.Batch.Items) != 0 {
		return face.JobProgress{}, face.ErrInvalidFaceJob
	}
	var updated face.JobProgress
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var generationID, state, operationID, operationState, currentSource string
		var libraryID, checkpoint, claimed, operationCompleted, operationTotal int64
		err := tx.QueryRowContext(ctx, `SELECT job.generation_id,job.library_id,job.state,job.checkpoint_id,job.claimed_revision,job.operation_id,operation.state,operation.completed_items,operation.total_items FROM face_analysis_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id WHERE job.id=?`, commit.JobID).Scan(&generationID, &libraryID, &state, &checkpoint, &claimed, &operationID, &operationState, &operationCompleted, &operationTotal)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceJobConflict
		}
		if err != nil {
			return err
		}
		if claimed != commit.ClaimedRevision || checkpoint != commit.ExpectedCheckpointID || state != "running" || operationState != "running" || operationCompleted+1 > operationTotal {
			return face.ErrFaceJobConflict
		}
		if commit.Batch.GenerationID != generationID || commit.Batch.LibraryID != libraryID || commit.Batch.AssetID != commit.NextCheckpointID {
			return face.ErrInvalidFaceJob
		}
		current, err := scanFaceJobProgress(tx.QueryRowContext(ctx, faceJobProgressSelect, generationID, libraryID))
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceJobConflict
		}
		if err != nil {
			return err
		}
		if current.Revision != commit.ExpectedProgressRevision || current.CheckpointID != commit.ExpectedCheckpointID || current.Completed+current.Failed+current.Stale+1 > current.Eligible {
			return face.ErrFaceJobConflict
		}
		err = tx.QueryRowContext(ctx, `SELECT source_fingerprint FROM assets WHERE library_id=? AND id=? AND kind IN('image','animated')`, libraryID, commit.NextCheckpointID).Scan(&currentSource)
		if errors.Is(err, sql.ErrNoRows) {
			currentSource = ""
		} else if err != nil {
			return err
		}
		if success && currentSource != commit.SourceFingerprint {
			return face.ErrSourceChanged
		}
		if commit.StaleCount == 1 && currentSource == commit.SourceFingerprint {
			return face.ErrFaceJobConflict
		}
		if success {
			dimension, generationState, err := s.faceGenerationContract(ctx, tx, generationID)
			if err != nil {
				return err
			}
			if generationState != "building" && generationState != "ready" && generationState != "active" {
				return face.ErrFaceGenerationUnavailable
			}
			for _, item := range commit.Batch.Items {
				if err := face.ValidateEncodedEmbedding(item.Vector, dimension); err != nil {
					return err
				}
			}
			if err := s.replaceFaceObservationsTx(ctx, tx, commit.Batch); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO face_asset_results(generation_id,library_id,asset_id,source_fingerprint,face_count,revision,updated_at_ms) VALUES(?,?,?,?,?,1,?) ON CONFLICT(generation_id,library_id,asset_id) DO UPDATE SET source_fingerprint=excluded.source_fingerprint,face_count=excluded.face_count,revision=face_asset_results.revision+1,updated_at_ms=excluded.updated_at_ms`, generationID, libraryID, commit.NextCheckpointID, commit.SourceFingerprint, len(commit.Batch.Items), commit.UpdatedAt.UnixMilli()); err != nil {
				return err
			}
		}
		completedIncrement := int64(0)
		if success {
			completedIncrement = 1
		}
		result, err := tx.ExecContext(ctx, `UPDATE face_library_progress SET completed_count=completed_count+?,failed_count=failed_count+?,stale_count=stale_count+?,checkpoint_id=?,revision=revision+1,updated_at_ms=MAX(updated_at_ms,?) WHERE generation_id=? AND library_id=? AND revision=? AND checkpoint_id=?`, completedIncrement, commit.FailedCount, commit.StaleCount, commit.NextCheckpointID, commit.UpdatedAt.UnixMilli(), generationID, libraryID, commit.ExpectedProgressRevision, commit.ExpectedCheckpointID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET checkpoint_id=?,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND checkpoint_id=? AND claimed_revision=? AND state='running'`, commit.NextCheckpointID, commit.UpdatedAt.UnixMilli(), commit.JobID, commit.ExpectedCheckpointID, commit.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET completed_items=completed_items+1,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='running'`, commit.UpdatedAt.UnixMilli(), operationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceJobConflict
		}
		updated, err = scanFaceJobProgress(tx.QueryRowContext(ctx, faceJobProgressSelect, generationID, libraryID))
		return err
	})
	return updated, err
}

func (s *Store) FinishFaceJob(ctx context.Context, job face.AnalysisJob, succeeded bool, errorCode string, now time.Time) (face.AnalysisJob, error) {
	if job.ID == "" || job.OperationID == "" || job.ClaimedRevision < 1 || now.IsZero() || succeeded == (errorCode != "") || len(errorCode) > 128 {
		return face.AnalysisJob{}, face.ErrInvalidFaceJob
	}
	jobState, operationState := "failed", "failed"
	if succeeded {
		jobState, operationState = "succeeded", "succeeded"
	}
	if errorCode == "cancelled" {
		jobState, operationState = "cancelled", "cancelled"
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var currentState, opState, generationID string
		var libraryID, completed, total int64
		err := tx.QueryRowContext(ctx, `SELECT job.state,job.generation_id,job.library_id,operation.state,operation.completed_items,operation.total_items FROM face_analysis_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id WHERE job.id=? AND job.operation_id=? AND job.claimed_revision=?`, job.ID, job.OperationID, job.ClaimedRevision).Scan(&currentState, &generationID, &libraryID, &opState, &completed, &total)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceJobConflict
		}
		if err != nil {
			return err
		}
		if succeeded && (currentState != "running" || opState != "running" || completed != total) {
			return face.ErrFaceJobConflict
		}
		if errorCode == "cancelled" && (currentState != "cancelling" || opState != "cancelling") {
			return face.ErrFaceJobConflict
		}
		if !succeeded && errorCode != "cancelled" && currentState != "running" {
			return face.ErrFaceJobConflict
		}
		if succeeded {
			progress, err := scanFaceJobProgress(tx.QueryRowContext(ctx, faceJobProgressSelect, generationID, libraryID))
			if errors.Is(err, sql.ErrNoRows) {
				return face.ErrFaceJobConflict
			}
			if err != nil {
				return err
			}
			if progress.Completed+progress.Failed+progress.Stale != progress.Eligible {
				return face.ErrFaceJobConflict
			}
		}
		result, err := tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET state=?,lease_expires_ms=NULL,error_code=?,requested_revision=requested_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND claimed_revision=? AND state IN('running','cancelling')`, jobState, nullableString(errorCode), now.UnixMilli(), job.ID, job.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceJobConflict
		}
		result, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state=?,phase='completed',error_code=?,lease_expires_ms=NULL,revision=revision+1,updated_at_ms=MAX(created_at_ms,?),finished_at_ms=MAX(created_at_ms,?) WHERE id=? AND state IN('running','cancelling')`, operationState, nullableString(errorCode), now.UnixMilli(), now.UnixMilli(), job.OperationID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return face.ErrFaceJobConflict
		}
		settingsState := "degraded"
		if succeeded {
			var failed, stale int64
			if err := tx.QueryRowContext(ctx, `SELECT failed_count,stale_count FROM face_library_progress WHERE generation_id=? AND library_id=?`, generationID, libraryID).Scan(&failed, &stale); err != nil {
				return err
			}
			if failed == 0 && stale == 0 {
				settingsState = "ready"
			}
		} else if errorCode == "model_unavailable" {
			settingsState = "awaiting_model"
		}
		_, err = tx.ExecContext(ctx, `UPDATE face_library_settings SET state=?,coverage_revision=coverage_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND enabled=1 AND state='building'`, settingsState, now.UnixMilli(), libraryID)
		return err
	})
	if err != nil {
		return face.AnalysisJob{}, err
	}
	return scanFaceJob(s.db.QueryRowContext(ctx, faceJobSelect+` WHERE job.id=?`, job.ID))
}
func (s *Store) CancelFaceJobOperation(ctx context.Context, operationID string, expectedRevision int64, now time.Time) (face.AnalysisJob, error) {
	if operationID == "" || expectedRevision < 1 || now.IsZero() {
		return face.AnalysisJob{}, face.ErrInvalidFaceJob
	}
	var jobID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM face_analysis_jobs WHERE operation_id=?`, operationID).Scan(&jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return face.AnalysisJob{}, face.ErrFaceJobNotFound
	}
	if err != nil {
		return face.AnalysisJob{}, err
	}
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		job, err := scanFaceJob(tx.QueryRowContext(ctx, faceJobSelect+` WHERE job.id=?`, jobID))
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceJobConflict
		}
		if err != nil {
			return err
		}
		if job.OperationRevision != expectedRevision {
			return face.ErrFaceJobConflict
		}
		switch job.State {
		case "queued":
			if _, err := tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET state='cancelled',requested_revision=requested_revision+1,error_code='cancelled',updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',error_code='cancelled',revision=revision+1,updated_at_ms=MAX(created_at_ms,?),finished_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='queued'`, now.UnixMilli(), now.UnixMilli(), operationID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE face_library_settings SET state='degraded',coverage_revision=coverage_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND enabled=1 AND state='building'`, now.UnixMilli(), job.LibraryID)
			return err
		case "running":
			if _, err := tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET state='cancelling',requested_revision=requested_revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, now.UnixMilli(), jobID); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelling',revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='running'`, now.UnixMilli(), operationID)
			return err
		case "cancelling":
			return nil
		default:
			return face.ErrFaceJobConflict
		}
	})
	if err != nil {
		return face.AnalysisJob{}, err
	}
	return scanFaceJob(s.db.QueryRowContext(ctx, faceJobSelect+` WHERE job.id=?`, jobID))
}
func (s *Store) RecoverExpiredFaceJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	if now.IsZero() {
		return jobs.RecoverySummary{}, face.ErrInvalidFaceJob
	}
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET state='cancelled',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code='cancelled',updated_at_ms=? WHERE state='cancelling' AND lease_expires_ms<=?`, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return err
		}
		cancelled, _ := result.RowsAffected()
		result, err = tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET state='queued',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code=NULL,updated_at_ms=? WHERE state='running' AND lease_expires_ms<=? AND attempt_count<?`, now.UnixMilli(), now.UnixMilli(), face.MaximumFaceJobAttempts)
		if err != nil {
			return err
		}
		summary.Requeued, _ = result.RowsAffected()
		result, err = tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET state='failed',requested_revision=requested_revision+1,claimed_revision=NULL,lease_expires_ms=NULL,error_code='retry_exhausted',updated_at_ms=? WHERE state='running' AND lease_expires_ms<=? AND attempt_count>=?`, now.UnixMilli(), now.UnixMilli(), face.MaximumFaceJobAttempts)
		if err != nil {
			return err
		}
		failed, _ := result.RowsAffected()
		summary.Interrupted = cancelled + failed
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='queued',phase='queued',lease_expires_ms=NULL,revision=revision+1,updated_at_ms=? WHERE id IN(SELECT operation_id FROM face_analysis_jobs WHERE state='queued' AND updated_at_ms=?)`, now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='failed',phase='completed',lease_expires_ms=NULL,error_code='retry_exhausted',revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id IN(SELECT operation_id FROM face_analysis_jobs WHERE state='failed' AND updated_at_ms=?)`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE ai_model_operations SET state='cancelled',phase='completed',lease_expires_ms=NULL,error_code='cancelled',revision=revision+1,updated_at_ms=?,finished_at_ms=? WHERE id IN(SELECT operation_id FROM face_analysis_jobs WHERE state='cancelled' AND updated_at_ms=?)`, now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE face_library_settings SET state='degraded',coverage_revision=coverage_revision+1,updated_at_ms=? WHERE library_id IN(SELECT library_id FROM face_analysis_jobs WHERE state IN('failed','cancelled') AND updated_at_ms=?) AND enabled=1 AND state='building'`, now.UnixMilli(), now.UnixMilli())
		return err
	})
	return summary, err
}

func validFaceJobAdmission(value face.JobAdmission) bool {
	job := value.Job
	return face.ValidReviewDigest(value.IdempotencyKeyHash) && face.ValidReviewDigest(value.RequestHash) && value.EligibleCount >= 0 && len(job.ID) >= 8 && job.LibraryID > 0 && len(job.GenerationID) >= 8 && len(job.OperationID) >= 8 && (job.Mode == face.JobMissing || job.Mode == face.JobAll) && job.State == "queued" && job.CheckpointID == 0 && job.RequestedRevision == 1 && job.ClaimedRevision == 0 && job.AttemptCount == 0 && job.OperationRevision == 1 && job.CompletedItems == 0 && job.TotalItems >= 0 && job.TotalItems <= value.EligibleCount && !job.CreatedAt.IsZero() && job.UpdatedAt.Equal(job.CreatedAt)
}

func faceSettingsRunnableState(state string) bool {
	return state == "building" || state == "ready" || state == "degraded"
}

type faceJobScanner interface{ Scan(...any) error }

func scanFaceJobProgress(row faceJobScanner) (face.JobProgress, error) {
	var value face.JobProgress
	var updated int64
	err := row.Scan(&value.GenerationID, &value.LibraryID, &value.Eligible, &value.Completed, &value.Failed, &value.Stale, &value.CheckpointID, &value.Revision, &updated)
	if err == nil {
		value.UpdatedAt = time.UnixMilli(updated).UTC()
	}
	return value, err
}

func scanFaceAdmission(row faceJobScanner) (face.JobAdmission, error) {
	var value face.JobAdmission
	job, err := scanFaceJobWithPrefix(row, &value.IdempotencyKeyHash, &value.RequestHash)
	value.Job = job
	return value, err
}
func scanFaceJob(row faceJobScanner) (face.AnalysisJob, error) { return scanFaceJobWithPrefix(row) }
func scanFaceJobWithPrefix(row faceJobScanner, prefix ...any) (face.AnalysisJob, error) {
	var value face.AnalysisJob
	var mode, state string
	var claimed, lease sql.NullInt64
	var errorCode sql.NullString
	var total sql.NullInt64
	var created, updated int64
	targets := append(prefix, &value.ID, &value.LibraryID, &value.GenerationID, &value.OperationID, &mode, &state, &value.CheckpointID, &value.RequestedRevision, &claimed, &value.AttemptCount, &lease, &errorCode, &value.OperationRevision, &value.CompletedItems, &total, &created, &updated)
	err := row.Scan(targets...)
	if err != nil {
		return face.AnalysisJob{}, err
	}
	value.Mode = face.JobMode(mode)
	value.State = state
	if claimed.Valid {
		value.ClaimedRevision = claimed.Int64
	}
	if lease.Valid {
		v := time.UnixMilli(lease.Int64).UTC()
		value.LeaseExpiresAt = &v
	}
	if errorCode.Valid {
		value.ErrorCode = errorCode.String
	}
	if total.Valid {
		value.TotalItems = total.Int64
	}
	value.CreatedAt = time.UnixMilli(created).UTC()
	value.UpdatedAt = time.UnixMilli(updated).UTC()
	return value, nil
}

var _ face.JobCatalog = (*Store)(nil)
var _ face.JobQueue = (*Store)(nil)
