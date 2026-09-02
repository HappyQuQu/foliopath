package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/face"
	"time"
)

func (s *Store) GetFaceLibrarySettings(ctx context.Context, libraryID int64) (face.LibrarySettings, error) {
	if libraryID < 1 {
		return face.LibrarySettings{}, face.ErrFaceLibraryNotFound
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM libraries WHERE id=?`, libraryID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return face.LibrarySettings{}, face.ErrFaceLibraryNotFound
		}
		return face.LibrarySettings{}, err
	}
	value := face.LibrarySettings{LibraryID: libraryID, State: "disabled", Revision: 1, Coverage: face.FaceCoverage{Revision: 1}}
	var enabled int
	var generation sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT enabled,state,revision,active_generation_id,coverage_revision FROM face_library_settings WHERE library_id=?`, libraryID).Scan(&enabled, &value.State, &value.Revision, &generation, &value.Coverage.Revision)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return face.LibrarySettings{}, err
	}
	if err == nil {
		value.Enabled = enabled == 1
		if generation.Valid {
			value.ActiveGenerationID = generation.String
			progressErr := s.db.QueryRowContext(ctx, `SELECT eligible_count,completed_count,failed_count,stale_count,revision FROM face_library_progress WHERE generation_id=? AND library_id=?`, generation.String, libraryID).Scan(&value.Coverage.Eligible, &value.Coverage.Completed, &value.Coverage.Failed, &value.Coverage.Stale, &value.Coverage.Revision)
			if progressErr != nil && !errors.Is(progressErr, sql.ErrNoRows) {
				return face.LibrarySettings{}, progressErr
			}
		}
	}
	return value, nil
}
func (s *Store) UpdateFaceLibrarySettings(ctx context.Context, libraryID int64, enabled bool, expectedRevision int64, now time.Time) (face.LibrarySettings, error) {
	if libraryID < 1 || expectedRevision < 1 || now.IsZero() {
		return face.LibrarySettings{}, face.ErrFaceSettingsConflict
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var createdAt int64
		if err := tx.QueryRowContext(ctx, `SELECT created_at_ms FROM libraries WHERE id=?`, libraryID).Scan(&createdAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return face.ErrFaceLibraryNotFound
			}
			return err
		}
		var currentEnabled int
		currentRevision := int64(1)
		err := tx.QueryRowContext(ctx, `SELECT enabled,revision FROM face_library_settings WHERE library_id=?`, libraryID).Scan(&currentEnabled, &currentRevision)
		missing := errors.Is(err, sql.ErrNoRows)
		if err != nil && !missing {
			return err
		}
		if currentRevision != expectedRevision {
			return face.ErrFaceSettingsConflict
		}
		if !enabled {
			if err := cancelFaceAnalysisForDisabledLibrary(ctx, tx, libraryID, now); err != nil {
				return err
			}
		}
		state := "disabled"
		var generation any
		if enabled {
			var id string
			if err := tx.QueryRowContext(ctx, `SELECT id FROM face_generations WHERE state='active'`).Scan(&id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return face.ErrFaceModelUnavailable
				}
				return err
			}
			generation = id
			state = "building"
		}
		next := currentRevision + 1
		if missing {
			updatedAt := max(createdAt, now.UnixMilli())
			_, err = tx.ExecContext(ctx, `INSERT INTO face_library_settings(library_id,enabled,state,active_generation_id,revision,coverage_revision,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,1,?,?)`, libraryID, boolInt(enabled), state, generation, next, createdAt, updatedAt)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE face_library_settings SET enabled=?,state=?,active_generation_id=?,revision=?,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND revision=?`, boolInt(enabled), state, generation, next, now.UnixMilli(), libraryID, currentRevision)
		}
		return err
	})
	if err != nil {
		return face.LibrarySettings{}, fmt.Errorf("update face library settings: %w", err)
	}
	return s.GetFaceLibrarySettings(ctx, libraryID)
}

func cancelFaceAnalysisForDisabledLibrary(ctx context.Context, tx *sql.Tx, libraryID int64, now time.Time) error {
	for _, transition := range []struct {
		jobFrom, jobTo, operationFrom, operationTo, phase string
		terminal                                          bool
	}{
		{jobFrom: "queued", jobTo: "cancelled", operationFrom: "queued", operationTo: "cancelled", phase: "completed", terminal: true},
		{jobFrom: "running", jobTo: "cancelling", operationFrom: "running", operationTo: "cancelling", phase: "building"},
	} {
		operationQuery := `UPDATE ai_model_operations SET state=?,phase=?,error_code=?,revision=revision+1,
			updated_at_ms=MAX(created_at_ms,?),finished_at_ms=CASE WHEN ? IS NULL THEN NULL ELSE MAX(created_at_ms,?) END
			WHERE state=? AND id IN(SELECT operation_id FROM face_analysis_jobs WHERE library_id=? AND state=?)`
		var errorCode, finishedAt any
		if transition.terminal {
			errorCode, finishedAt = "cancelled", now.UnixMilli()
		}
		operationResult, err := tx.ExecContext(ctx, operationQuery, transition.operationTo, transition.phase,
			errorCode, now.UnixMilli(), finishedAt, now.UnixMilli(), transition.operationFrom, libraryID, transition.jobFrom)
		if err != nil {
			return err
		}
		jobResult, err := tx.ExecContext(ctx, `UPDATE face_analysis_jobs SET state=?,requested_revision=requested_revision+1,
			error_code=?,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND state=?`, transition.jobTo, errorCode,
			now.UnixMilli(), libraryID, transition.jobFrom)
		if err != nil {
			return err
		}
		operations, _ := operationResult.RowsAffected()
		jobs, _ := jobResult.RowsAffected()
		if operations != jobs {
			return face.ErrFaceJobConflict
		}
	}
	return nil
}

var _ face.LibrarySettingsRepository = (*Store)(nil)
