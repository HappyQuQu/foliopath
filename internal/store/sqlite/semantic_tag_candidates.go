package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const currentTagAssetProgress = `EXISTS(
    SELECT 1 FROM semantic_tag_asset_progress progress
    WHERE progress.generation_id=? AND progress.library_id=asset.library_id
      AND progress.asset_id=asset.id AND progress.vocabulary_snapshot_id=?
      AND progress.source_fingerprint=asset.source_fingerprint
)`

func (s *Store) CountTagJobCandidates(ctx context.Context, libraryID int64, generationID, snapshotID string, mode semantic.JobMode) (semantic.TagJobCandidateCounts, error) {
	if err := semantic.ValidateTagJobCandidateQuery(libraryID, generationID, snapshotID, mode, 0, 1); err != nil {
		return semantic.TagJobCandidateCounts{}, err
	}
	if err := s.requireActiveTagOwners(ctx, generationID, snapshotID); err != nil {
		return semantic.TagJobCandidateCounts{}, err
	}
	where := "NOT " + currentTagAssetProgress
	args := []any{generationID, snapshotID, libraryID, generationID}
	if mode == semantic.JobAll {
		where, args = "1=1", []any{libraryID, generationID}
	}
	var value semantic.TagJobCandidateCounts
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(CASE WHEN `+where+` THEN 1 ELSE 0 END),0)
        FROM assets asset JOIN semantic_embeddings embedding ON embedding.library_id=asset.library_id
          AND embedding.asset_id=asset.id AND embedding.source_fingerprint=asset.source_fingerprint
        WHERE asset.library_id=? AND embedding.generation_id=? AND asset.kind IN ('image','animated')`, args...).Scan(&value.Eligible, &value.Pending)
	if err != nil {
		return value, fmt.Errorf("count semantic tag candidates: %w", err)
	}
	return value, nil
}

func (s *Store) ListTagJobCandidates(ctx context.Context, libraryID int64, generationID, snapshotID string, mode semantic.JobMode, checkpoint int64, limit int) (semantic.TagJobCandidatePage, error) {
	if err := semantic.ValidateTagJobCandidateQuery(libraryID, generationID, snapshotID, mode, checkpoint, limit); err != nil {
		return semantic.TagJobCandidatePage{}, err
	}
	if err := s.requireActiveTagOwners(ctx, generationID, snapshotID); err != nil {
		return semantic.TagJobCandidatePage{}, err
	}
	where := "NOT " + currentTagAssetProgress
	args := []any{libraryID, generationID, checkpoint, generationID, snapshotID, limit + 1}
	if mode == semantic.JobAll {
		where, args = "1=1", []any{libraryID, generationID, checkpoint, limit + 1}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT asset.id FROM assets asset
        JOIN semantic_embeddings embedding ON embedding.library_id=asset.library_id AND embedding.asset_id=asset.id
          AND embedding.source_fingerprint=asset.source_fingerprint
        WHERE asset.library_id=? AND embedding.generation_id=? AND asset.id>?
          AND asset.kind IN ('image','animated') AND `+where+` ORDER BY asset.id LIMIT ?`, args...)
	if err != nil {
		return semantic.TagJobCandidatePage{}, fmt.Errorf("list semantic tag candidates: %w", err)
	}
	defer rows.Close()
	items := make([]semantic.TagJobCandidate, 0, limit+1)
	for rows.Next() {
		var item semantic.TagJobCandidate
		if err := rows.Scan(&item.AssetID); err != nil {
			return semantic.TagJobCandidatePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return semantic.TagJobCandidatePage{}, err
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	next := checkpoint
	if len(items) > 0 {
		next = items[len(items)-1].AssetID
	}
	return semantic.TagJobCandidatePage{Items: items, Checkpoint: next, HasMore: hasMore}, nil
}

func (s *Store) requireActiveTagOwners(ctx context.Context, generationID, snapshotID string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM semantic_generations generation,ai_tag_vocabulary_snapshots vocabulary
        WHERE generation.id=? AND generation.state='active' AND vocabulary.id=? AND vocabulary.state='active'`, generationID, snapshotID).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return semantic.ErrSemanticGenerationUnavailable
	}
	return nil
}

type tagSuggestionInputs struct {
	Fingerprint string
	AssetVector []byte
	TagVectors  map[int64][]byte
	Dimension   int
}

func (s *Store) LoadTagSuggestionInputs(ctx context.Context, generationID, snapshotID string, libraryID, assetID int64) (string, []float32, map[int64][]float32, error) {
	if libraryID < 1 || assetID < 1 {
		return "", nil, nil, semantic.ErrInvalidTagSuggestion
	}
	if err := s.requireActiveTagOwners(ctx, generationID, snapshotID); err != nil {
		return "", nil, nil, err
	}
	var fingerprint string
	var assetRaw []byte
	var dimension int
	err := s.db.QueryRowContext(ctx, `SELECT asset.source_fingerprint,embedding.vector,generation.embedding_dimension
        FROM assets asset JOIN semantic_embeddings embedding ON embedding.library_id=asset.library_id AND embedding.asset_id=asset.id
        JOIN semantic_generations generation ON generation.id=embedding.generation_id
        WHERE asset.library_id=? AND asset.id=? AND embedding.generation_id=? AND embedding.source_fingerprint=asset.source_fingerprint`, libraryID, assetID, generationID).Scan(&fingerprint, &assetRaw, &dimension)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, nil, semantic.ErrSemanticSourceChanged
	}
	if err != nil {
		return "", nil, nil, err
	}
	assetVector, err := semantic.DecodeEmbedding(assetRaw, dimension)
	if err != nil {
		return "", nil, nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT tag_id,vector FROM semantic_tag_embeddings WHERE generation_id=? AND snapshot_id=? ORDER BY tag_id`, generationID, snapshotID)
	if err != nil {
		return "", nil, nil, err
	}
	defer rows.Close()
	tags := map[int64][]float32{}
	for rows.Next() {
		var id int64
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return "", nil, nil, err
		}
		vector, err := semantic.DecodeEmbedding(raw, dimension)
		if err != nil {
			return "", nil, nil, err
		}
		tags[id] = vector
	}
	if err := rows.Err(); err != nil {
		return "", nil, nil, err
	}
	if len(tags) == 0 {
		return "", nil, nil, semantic.ErrSemanticGenerationUnavailable
	}
	return fingerprint, assetVector, tags, nil
}

var _ semantic.TagJobCatalog = (*Store)(nil)
