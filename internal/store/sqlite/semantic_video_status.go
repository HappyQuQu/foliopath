package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) GetVideoStatus(ctx context.Context, libraryID int64) (semantic.VideoStatus, error) {
	var result semantic.VideoStatus
	var generation sql.NullString
	var eligible, ready, degraded, failed, stale, revision sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
        SELECT library.id, COALESCE(setting.enabled,0), library.status='offline',
               state.active_generation_id, progress.eligible_count, progress.ready_count,
               progress.degraded_count, progress.failed_count, progress.stale_count, progress.revision
        FROM libraries library
        LEFT JOIN ai_library_settings setting ON setting.library_id=library.id
        LEFT JOIN ai_model_state state ON state.singleton_key=1
        LEFT JOIN semantic_video_progress progress
          ON progress.library_id=library.id AND progress.generation_id=state.active_generation_id
        WHERE library.id=?`, libraryID).Scan(&result.LibraryID, &result.Enabled, &result.Offline,
		&generation, &eligible, &ready, &degraded, &failed, &stale, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.VideoStatus{}, semantic.ErrSemanticLibraryNotFound
	}
	if err != nil {
		return semantic.VideoStatus{}, fmt.Errorf("read video status: %w", err)
	}
	result.GenerationID = generation.String
	if revision.Valid {
		result.Coverage = &semantic.VideoCoverage{Eligible: eligible.Int64, Ready: ready.Int64,
			Degraded: degraded.Int64, Failed: failed.Int64, Stale: stale.Int64, Revision: revision.Int64}
	}
	return result, nil
}
