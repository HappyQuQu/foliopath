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
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM libraries WHERE id=?`, libraryID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.LibrarySettings{}, semantic.ErrSemanticLibraryNotFound
		}
		return semantic.LibrarySettings{}, err
	}
	value := semantic.LibrarySettings{LibraryID: libraryID, State: semantic.LibraryDisabled, Revision: 1,
		Coverage: semantic.Coverage{Revision: 1}}
	var enabled int
	var state string
	err := s.db.QueryRowContext(ctx, `SELECT enabled, state, revision FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&enabled, &state, &value.Revision)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return semantic.LibrarySettings{}, err
	}
	if err == nil {
		value.Enabled, value.State = enabled == 1, semantic.LibraryState(state)
	}
	var generationID sql.NullString
	_ = s.db.QueryRowContext(ctx, `
        SELECT state.active_generation_id FROM ai_model_state state
        JOIN semantic_generations generation ON generation.id=state.active_generation_id AND generation.state='active'
        JOIN ai_models model ON model.id=state.active_model_id AND model.state='available'
        WHERE state.singleton_key=1`).Scan(&generationID)
	if generationID.Valid {
		value.ActiveGenerationID = generationID.String
		var progressRevision int64
		err = s.db.QueryRowContext(ctx, `SELECT eligible_count, completed_count, failed_count, stale_count, revision
            FROM semantic_library_progress WHERE generation_id=? AND library_id=?`, generationID.String, libraryID).Scan(
			&value.Coverage.Eligible, &value.Coverage.Completed, &value.Coverage.Failed, &value.Coverage.Stale, &progressRevision)
		if err == nil {
			value.Coverage.Revision = progressRevision
		} else if !errors.Is(err, sql.ErrNoRows) {
			return semantic.LibrarySettings{}, err
		}
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
                VALUES(?, ?, ?, ?, 1, ?, ?)`, libraryID, boolInt(enabled), state, nextRevision, now.UnixMilli(), now.UnixMilli())
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE ai_library_settings SET enabled=?, state=?, revision=?, updated_at_ms=?
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
