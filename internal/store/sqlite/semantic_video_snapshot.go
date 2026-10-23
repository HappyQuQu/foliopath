package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) GetVideoSemanticSearchSnapshot(ctx context.Context, scope semantic.SearchScope) (semantic.VideoSearchSnapshot, error) {
	if scope.LibraryID < 0 || scope.DirectoryID < 0 || scope.DirectoryID > 0 && scope.LibraryID == 0 {
		return semantic.VideoSearchSnapshot{}, semantic.ErrInvalidVideoSemantic
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return semantic.VideoSearchSnapshot{}, fmt.Errorf("begin video semantic snapshot: %w", err)
	}
	defer tx.Rollback()
	var snapshot semantic.VideoSearchSnapshot
	err = tx.QueryRowContext(ctx, `
        SELECT state.active_generation_id, catalog.revision
        FROM ai_model_state state
        JOIN semantic_generations generation ON generation.id=state.active_generation_id AND generation.state='active'
        JOIN ai_models model ON model.id=state.active_model_id AND model.state='available'
        CROSS JOIN catalog_search_state catalog
        WHERE state.singleton_key=1`).Scan(&snapshot.GenerationID, &snapshot.CatalogRevision)
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.VideoSearchSnapshot{}, semantic.ErrSemanticGenerationUnavailable
	}
	if err != nil {
		return semantic.VideoSearchSnapshot{}, fmt.Errorf("read video semantic generation: %w", err)
	}
	if scope.LibraryID > 0 {
		var status string
		var enabled int
		err = tx.QueryRowContext(ctx, `
            SELECT library.status, COALESCE(setting.enabled, 0)
            FROM libraries library LEFT JOIN ai_library_settings setting ON setting.library_id=library.id
            WHERE library.id=?`, scope.LibraryID).Scan(&status, &enabled)
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.VideoSearchSnapshot{}, semantic.ErrSemanticLibraryNotFound
		}
		if err != nil {
			return semantic.VideoSearchSnapshot{}, fmt.Errorf("read video semantic library: %w", err)
		}
		if status == "offline" {
			return semantic.VideoSearchSnapshot{}, semantic.ErrSemanticLibraryOffline
		}
		if enabled != 1 {
			return semantic.VideoSearchSnapshot{}, semantic.ErrSemanticDisabled
		}
		if scope.DirectoryID > 0 {
			var exists int
			err = tx.QueryRowContext(ctx, `SELECT 1 FROM directories WHERE library_id=? AND id=?`, scope.LibraryID, scope.DirectoryID).Scan(&exists)
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.VideoSearchSnapshot{}, semantic.ErrSemanticScopeNotFound
			}
			if err != nil {
				return semantic.VideoSearchSnapshot{}, fmt.Errorf("read video semantic directory: %w", err)
			}
		}
	}
	statement := `
        SELECT library.id, library.current_generation, setting.revision,
               COALESCE(progress.eligible_count,0), COALESCE(progress.ready_count,0),
               COALESCE(progress.degraded_count,0), COALESCE(progress.failed_count,0),
               COALESCE(progress.stale_count,0), COALESCE(progress.revision,1)
        FROM libraries library
        JOIN ai_library_settings setting ON setting.library_id=library.id AND setting.enabled=1
        LEFT JOIN semantic_video_progress progress ON progress.library_id=library.id AND progress.generation_id=?
        WHERE library.status <> 'offline'`
	arguments := []any{snapshot.GenerationID}
	if scope.LibraryID > 0 {
		statement += ` AND library.id=?`
		arguments = append(arguments, scope.LibraryID)
	}
	statement += ` ORDER BY library.id`
	rows, err := tx.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return semantic.VideoSearchSnapshot{}, fmt.Errorf("list video semantic members: %w", err)
	}
	for rows.Next() {
		var member semantic.VideoSearchSnapshotMember
		if err := rows.Scan(&member.LibraryID, &member.LibraryGeneration, &member.SettingsRevision,
			&member.Coverage.Eligible, &member.Coverage.Ready, &member.Coverage.Degraded,
			&member.Coverage.Failed, &member.Coverage.Stale, &member.Coverage.Revision); err != nil {
			rows.Close()
			return semantic.VideoSearchSnapshot{}, fmt.Errorf("scan video semantic member: %w", err)
		}
		snapshot.Members = append(snapshot.Members, member)
	}
	if err := rows.Close(); err != nil {
		return semantic.VideoSearchSnapshot{}, fmt.Errorf("close video semantic members: %w", err)
	}
	if err := rows.Err(); err != nil {
		return semantic.VideoSearchSnapshot{}, fmt.Errorf("iterate video semantic members: %w", err)
	}
	if len(snapshot.Members) == 0 {
		return semantic.VideoSearchSnapshot{}, semantic.ErrSemanticDisabled
	}
	if scope.LibraryID == 0 {
		rows, err = tx.QueryContext(ctx, `
            SELECT library.id, COALESCE(setting.revision,1),
                   CASE WHEN library.status='offline' THEN 'offline' ELSE 'disabled' END
            FROM libraries library LEFT JOIN ai_library_settings setting ON setting.library_id=library.id
            WHERE library.status='offline' OR COALESCE(setting.enabled,0)<>1 ORDER BY library.id`)
		if err != nil {
			return semantic.VideoSearchSnapshot{}, fmt.Errorf("list excluded video semantic libraries: %w", err)
		}
		for rows.Next() {
			var excluded semantic.SearchExclusion
			if err := rows.Scan(&excluded.LibraryID, &excluded.SettingsRevision, &excluded.Reason); err != nil {
				rows.Close()
				return semantic.VideoSearchSnapshot{}, fmt.Errorf("scan excluded video semantic library: %w", err)
			}
			snapshot.Excluded = append(snapshot.Excluded, excluded)
		}
		if err := rows.Close(); err != nil {
			return semantic.VideoSearchSnapshot{}, fmt.Errorf("close excluded video semantic libraries: %w", err)
		}
		if err := rows.Err(); err != nil {
			return semantic.VideoSearchSnapshot{}, fmt.Errorf("iterate excluded video semantic libraries: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return semantic.VideoSearchSnapshot{}, fmt.Errorf("commit video semantic snapshot: %w", err)
	}
	return snapshot, nil
}

var _ semantic.VideoSearchSnapshotRepository = (*Store)(nil)
