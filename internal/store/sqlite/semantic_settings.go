package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) GetSemanticLibrarySettings(ctx context.Context, libraryID int64) (semantic.LibrarySettings, error) {
	if libraryID < 1 {
		return semantic.LibrarySettings{}, semantic.ErrSemanticLibraryNotFound
	}
	var value semantic.LibrarySettings
	var generationID sql.NullString
	var modelAvailable bool
	// One statement keeps settings, active generation and progress in one snapshot.
	// Availability gates new work, not access to already recorded progress.
	err := s.db.QueryRowContext(ctx, `
        SELECT library.id, COALESCE(setting.enabled,0), COALESCE(setting.state,'disabled'),
               COALESCE(setting.revision,1), generation.id, COALESCE(model.state='available',0),
               COALESCE(progress.eligible_count,0), COALESCE(progress.completed_count,0),
               COALESCE(progress.failed_count,0), COALESCE(progress.stale_count,0), COALESCE(progress.revision,1), progress.revision IS NOT NULL
        FROM libraries library
        LEFT JOIN ai_library_settings setting ON setting.library_id=library.id
        LEFT JOIN ai_model_state state ON state.singleton_key=1
        LEFT JOIN semantic_generations generation ON generation.id=state.active_generation_id AND generation.state='active'
        LEFT JOIN ai_models model ON model.id=state.active_model_id
        LEFT JOIN semantic_library_progress progress ON progress.generation_id=generation.id AND progress.library_id=library.id
        WHERE library.id=?`, libraryID).Scan(&value.LibraryID, &value.Enabled, &value.State, &value.Revision,
		&generationID, &modelAvailable, &value.Coverage.Eligible, &value.Coverage.Completed, &value.Coverage.Failed,
		&value.Coverage.Stale, &value.Coverage.Revision, &value.CoverageRecorded)
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.LibrarySettings{}, semantic.ErrSemanticLibraryNotFound
	}
	if err != nil {
		return semantic.LibrarySettings{}, fmt.Errorf("read semantic library settings: %w", err)
	}
	// Existing job admission uses this field; do not expose an unavailable generation
	// as actionable merely because its recorded progress is still readable.
	if modelAvailable {
		value.ActiveGenerationID = generationID.String
	}
	return value, nil
}

func (s *Store) UpdateSemanticLibrarySettings(ctx context.Context, libraryID int64, enabled bool, expectedRevision int64, now time.Time) (semantic.LibrarySettings, error) {
	if libraryID < 1 || expectedRevision < 1 || now.IsZero() {
		return semantic.LibrarySettings{}, semantic.ErrSemanticRevisionConflict
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM libraries WHERE id=?`, libraryID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrSemanticLibraryNotFound
			}
			return err
		}
		currentRevision := int64(1)
		var currentEnabled int
		err := tx.QueryRowContext(ctx, `SELECT enabled, revision FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&currentEnabled, &currentRevision)
		missing := errors.Is(err, sql.ErrNoRows)
		if err != nil && !missing {
			return err
		}
		if currentRevision != expectedRevision {
			return semantic.ErrSemanticRevisionConflict
		}
		state := semantic.LibraryDisabled
		if enabled {
			var generationID string
			err = tx.QueryRowContext(ctx, `SELECT state.active_generation_id FROM ai_model_state state
                JOIN semantic_generations generation ON generation.id=state.active_generation_id AND generation.state='active'
                JOIN ai_models model ON model.id=state.active_model_id AND model.state='available'
                WHERE state.singleton_key=1`).Scan(&generationID)
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrSemanticGenerationUnavailable
			}
			if err != nil {
				return err
			}
			state = semantic.LibraryBuilding
		}
		nextRevision := currentRevision + 1
		if missing {
			_, err = tx.ExecContext(ctx, `INSERT INTO ai_library_settings(library_id, enabled, state, revision, coverage_revision, created_at_ms, updated_at_ms)
				SELECT id, ?, ?, ?, 1, MAX(created_at_ms, ?), MAX(created_at_ms, ?)
				FROM libraries WHERE id=?`, boolInt(enabled), state, nextRevision, now.UnixMilli(), now.UnixMilli(), libraryID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE ai_library_settings SET enabled=?, state=?, revision=?, updated_at_ms=MAX(created_at_ms, ?)
				WHERE library_id=? AND revision=?`, boolInt(enabled), state, nextRevision, now.UnixMilli(), libraryID, currentRevision)
		}
		return err
	})
	if err != nil {
		return semantic.LibrarySettings{}, fmt.Errorf("update semantic library settings: %w", err)
	}
	return s.GetSemanticLibrarySettings(ctx, libraryID)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

var _ semantic.LibrarySettingsRepository = (*Store)(nil)
