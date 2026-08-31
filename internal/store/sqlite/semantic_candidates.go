package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) CountSemanticBackfillCandidates(ctx context.Context, libraryID int64, generationID string, mode semantic.JobMode) (semantic.BackfillCandidateCounts, error) {
	if err := semantic.ValidateBackfillCandidateQuery(libraryID, generationID, mode, 0, 1); err != nil {
		return semantic.BackfillCandidateCounts{}, err
	}
	where, err := semanticCandidateWhere(mode)
	if err != nil {
		return semantic.BackfillCandidateCounts{}, err
	}
	var generationState string
	if err := s.db.QueryRowContext(ctx, `SELECT state FROM semantic_generations WHERE id = ?`, generationID).Scan(&generationState); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.BackfillCandidateCounts{}, semantic.ErrSemanticGenerationUnavailable
		}
		return semantic.BackfillCandidateCounts{}, err
	}
	if generationState != "active" {
		return semantic.BackfillCandidateCounts{}, semantic.ErrSemanticGenerationUnavailable
	}
	var counts semantic.BackfillCandidateCounts
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN `+where+` THEN 1 ELSE 0 END), 0)
        FROM assets AS asset
        LEFT JOIN semantic_embeddings AS embedding
          ON embedding.generation_id = ? AND embedding.library_id = asset.library_id
         AND embedding.asset_id = asset.id
		WHERE asset.library_id = ? AND asset.kind IN ('image','animated')`,
		generationID, libraryID).Scan(&counts.Eligible, &counts.Pending)
	if err != nil {
		return semantic.BackfillCandidateCounts{}, fmt.Errorf("count semantic candidates: %w", err)
	}
	return counts, nil
}

func (s *Store) ListSemanticBackfillCandidates(
	ctx context.Context,
	libraryID int64,
	generationID string,
	mode semantic.JobMode,
	checkpoint int64,
	limit int,
) (semantic.BackfillCandidatePage, error) {
	if err := semantic.ValidateBackfillCandidateQuery(libraryID, generationID, mode, checkpoint, limit); err != nil {
		return semantic.BackfillCandidatePage{}, err
	}
	where, err := semanticCandidateWhere(mode)
	if err != nil {
		return semantic.BackfillCandidatePage{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT asset.id, asset.source_fingerprint
        FROM assets AS asset
        LEFT JOIN semantic_embeddings AS embedding
          ON embedding.generation_id = ? AND embedding.library_id = asset.library_id
         AND embedding.asset_id = asset.id
        WHERE asset.library_id = ? AND asset.kind IN ('image','animated')
          AND asset.id > ? AND `+where+`
        ORDER BY asset.id ASC LIMIT ?`, generationID, libraryID, checkpoint, limit+1)
	if err != nil {
		return semantic.BackfillCandidatePage{}, fmt.Errorf("list semantic candidates: %w", err)
	}
	defer rows.Close()
	items := make([]semantic.BackfillCandidate, 0, limit)
	for rows.Next() {
		var item semantic.BackfillCandidate
		if err := rows.Scan(&item.AssetID, &item.SourceFingerprint); err != nil {
			return semantic.BackfillCandidatePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return semantic.BackfillCandidatePage{}, err
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	next := checkpoint
	if len(items) > 0 {
		next = items[len(items)-1].AssetID
	}
	return semantic.BackfillCandidatePage{Items: items, Checkpoint: next, HasMore: hasMore}, nil
}

func semanticCandidateWhere(mode semantic.JobMode) (string, error) {
	switch mode {
	case semantic.JobAll:
		return "1 = 1", nil
	case semantic.JobMissing:
		return "(embedding.asset_id IS NULL OR embedding.source_fingerprint <> asset.source_fingerprint)", nil
	default:
		return "", semantic.ErrInvalidSemanticJob
	}
}
