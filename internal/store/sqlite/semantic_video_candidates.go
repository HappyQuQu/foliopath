package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const completeCurrentVideoPlan = `
    thumbnail.status = 'ready'
    AND thumbnail.source_fingerprint = asset.source_fingerprint
    AND thumbnail.frame_count IN (4, 10)
    AND (
        SELECT COUNT(*) FROM semantic_video_frames frame
        WHERE frame.generation_id = ?
          AND frame.library_id = asset.library_id
          AND frame.asset_id = asset.id
          AND frame.source_fingerprint = asset.source_fingerprint
          AND frame.storyboard_transform_version = thumbnail.transform_version
          AND frame.plan_size = thumbnail.frame_count
    ) = thumbnail.frame_count`

func (s *Store) CountVideoJobCandidates(ctx context.Context, libraryID int64, generationID string, mode semantic.JobMode) (semantic.VideoJobCandidateCounts, error) {
	if err := semantic.ValidateVideoJobCandidateQuery(libraryID, generationID, mode, 0, 1); err != nil {
		return semantic.VideoJobCandidateCounts{}, err
	}
	if err := s.requireActiveVideoGeneration(ctx, generationID); err != nil {
		return semantic.VideoJobCandidateCounts{}, err
	}
	where := "NOT COALESCE((" + completeCurrentVideoPlan + "), 0)"
	args := []any{generationID, libraryID}
	if mode == semantic.JobAll {
		where = "1 = 1"
		args = []any{libraryID}
	}
	var counts semantic.VideoJobCandidateCounts
	err := s.db.QueryRowContext(ctx, `
        SELECT COUNT(*), COALESCE(SUM(CASE WHEN `+where+` THEN 1 ELSE 0 END), 0)
        FROM assets asset
        LEFT JOIN thumbnails thumbnail ON thumbnail.library_id=asset.library_id
          AND thumbnail.asset_id=asset.id AND thumbnail.variant='storyboard'
        WHERE asset.library_id=? AND asset.kind='video'`, args...).Scan(&counts.Eligible, &counts.Pending)
	if err != nil {
		return semantic.VideoJobCandidateCounts{}, fmt.Errorf("count video semantic candidates: %w", err)
	}
	return counts, nil
}

func (s *Store) ListVideoJobCandidates(ctx context.Context, libraryID int64, generationID string, mode semantic.JobMode, checkpoint int64, limit int) (semantic.VideoJobCandidatePage, error) {
	if err := semantic.ValidateVideoJobCandidateQuery(libraryID, generationID, mode, checkpoint, limit); err != nil {
		return semantic.VideoJobCandidatePage{}, err
	}
	if err := s.requireActiveVideoGeneration(ctx, generationID); err != nil {
		return semantic.VideoJobCandidatePage{}, err
	}
	where := "NOT COALESCE((" + completeCurrentVideoPlan + "), 0)"
	args := []any{libraryID, checkpoint, generationID, limit + 1}
	if mode == semantic.JobAll {
		where = "1 = 1"
		args = []any{libraryID, checkpoint, limit + 1}
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT asset.id
        FROM assets asset
        LEFT JOIN thumbnails thumbnail ON thumbnail.library_id=asset.library_id
          AND thumbnail.asset_id=asset.id AND thumbnail.variant='storyboard'
        WHERE asset.library_id=? AND asset.kind='video' AND asset.id>? AND `+where+`
        ORDER BY asset.id LIMIT ?`, args...)
	if err != nil {
		return semantic.VideoJobCandidatePage{}, fmt.Errorf("list video semantic candidates: %w", err)
	}
	defer rows.Close()
	items := make([]semantic.VideoJobCandidate, 0, limit)
	for rows.Next() {
		var item semantic.VideoJobCandidate
		if err := rows.Scan(&item.AssetID); err != nil {
			return semantic.VideoJobCandidatePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return semantic.VideoJobCandidatePage{}, err
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	next := checkpoint
	if len(items) > 0 {
		next = items[len(items)-1].AssetID
	}
	return semantic.VideoJobCandidatePage{Items: items, Checkpoint: next, HasMore: hasMore}, nil
}

func (s *Store) requireActiveVideoGeneration(ctx context.Context, generationID string) error {
	var state string
	if err := s.db.QueryRowContext(ctx, `SELECT state FROM semantic_generations WHERE id=?`, generationID).Scan(&state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.ErrSemanticGenerationUnavailable
		}
		return err
	}
	if state != "active" {
		return semantic.ErrSemanticGenerationUnavailable
	}
	return nil
}

var _ semantic.VideoJobCatalog = (*Store)(nil)
