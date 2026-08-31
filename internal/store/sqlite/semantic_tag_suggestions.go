package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) PutTagEmbeddingBatch(ctx context.Context, batch semantic.TagEmbeddingBatch) error {
	var dimension int
	if err := s.db.QueryRowContext(ctx, `
        SELECT embedding_dimension FROM semantic_generations WHERE id = ? AND state IN ('ready', 'active')`,
		batch.GenerationID,
	).Scan(&dimension); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.ErrSemanticGenerationUnavailable
		}
		return fmt.Errorf("resolve tag embedding generation: %w", err)
	}
	if err := semantic.ValidateTagEmbeddingBatch(batch, dimension, s.maxBatchSize); err != nil {
		return err
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		for _, item := range batch.Items {
			if _, err := tx.ExecContext(ctx, `
                INSERT INTO semantic_tag_embeddings(generation_id, snapshot_id, tag_id, vector, created_at_ms)
                VALUES(?, ?, ?, ?, ?)
                ON CONFLICT(generation_id, snapshot_id, tag_id) DO UPDATE SET
                    vector = excluded.vector,
                    created_at_ms = excluded.created_at_ms`,
				batch.GenerationID, batch.SnapshotID, item.TagID, item.Vector, batch.CreatedAt.UTC().UnixMilli(),
			); err != nil {
				return fmt.Errorf("store controlled tag embedding: %w", err)
			}
		}
		return nil
	})
}

func (s *Store) ListMissingTagEmbeddingInputs(ctx context.Context, generationID, snapshotID string, limit int) ([]semantic.TagEmbeddingInput, error) {
	if len(generationID) < 8 || len(snapshotID) < 8 || limit < 1 || limit > 500 {
		return nil, semantic.ErrInvalidTagSuggestion
	}
	if err := s.requireActiveTagOwners(ctx, generationID, snapshotID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT entry.tag_id,tag.name FROM ai_tag_vocabulary_entries entry
		JOIN tags tag ON tag.id=entry.tag_id
		LEFT JOIN semantic_tag_embeddings embedding ON embedding.generation_id=? AND embedding.snapshot_id=entry.snapshot_id AND embedding.tag_id=entry.tag_id
		WHERE entry.snapshot_id=? AND embedding.tag_id IS NULL ORDER BY entry.tag_id LIMIT ?`, generationID, snapshotID, limit)
	if err != nil {
		return nil, fmt.Errorf("list missing tag embeddings: %w", err)
	}
	defer rows.Close()
	items := make([]semantic.TagEmbeddingInput, 0, limit)
	for rows.Next() {
		var item semantic.TagEmbeddingInput
		if err := rows.Scan(&item.TagID, &item.Text); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ReplacePendingTagSuggestions(
	ctx context.Context,
	libraryID, assetID int64,
	generationID, snapshotID, sourceFingerprint string,
	items []semantic.TagSuggestion,
	now time.Time,
) error {
	if now.IsZero() || semantic.ValidatePendingTagSuggestions(
		items, libraryID, assetID, generationID, snapshotID, sourceFingerprint,
	) != nil {
		return semantic.ErrInvalidTagSuggestion
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var currentFingerprint string
		if err := tx.QueryRowContext(ctx, `
            SELECT source_fingerprint FROM assets WHERE library_id = ? AND id = ?`, libraryID, assetID,
		).Scan(&currentFingerprint); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrInvalidTagSuggestion
			}
			return fmt.Errorf("resolve suggestion asset: %w", err)
		}
		if currentFingerprint != sourceFingerprint {
			return semantic.ErrInvalidTagSuggestion
		}
		var active int
		if err := tx.QueryRowContext(ctx, `
            SELECT COUNT(*) FROM semantic_generations g
            JOIN ai_tag_vocabulary_snapshots v ON v.id = ? AND v.state = 'active'
            WHERE g.id = ? AND g.state = 'active'`, snapshotID, generationID,
		).Scan(&active); err != nil {
			return fmt.Errorf("resolve suggestion owners: %w", err)
		}
		if active != 1 {
			return semantic.ErrSemanticGenerationUnavailable
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE ai_tag_suggestions
            SET state = 'invalidated', revision = revision + 1, updated_at_ms = ?
            WHERE library_id = ? AND asset_id = ? AND state = 'pending'
              AND (generation_id <> ? OR vocabulary_snapshot_id <> ? OR source_fingerprint <> ?)`,
			now.UTC().UnixMilli(), libraryID, assetID, generationID, snapshotID, sourceFingerprint,
		); err != nil {
			return fmt.Errorf("invalidate stale tag suggestions: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            DELETE FROM ai_tag_suggestions
            WHERE library_id = ? AND asset_id = ? AND generation_id = ?
              AND vocabulary_snapshot_id = ? AND state = 'pending'`,
			libraryID, assetID, generationID, snapshotID,
		); err != nil {
			return fmt.Errorf("replace pending tag suggestions: %w", err)
		}
		for _, item := range items {
			result, err := tx.ExecContext(ctx, `
                INSERT INTO ai_tag_suggestions(
                    id, generation_id, library_id, asset_id, vocabulary_snapshot_id,
                    tag_id, source_fingerprint, confidence, state, revision, created_at_ms, updated_at_ms
                )
                SELECT ?, ?, ?, ?, ?, ?, ?, ?, 'pending', 1, ?, ?
                WHERE NOT EXISTS (
                    SELECT 1 FROM ai_tag_reviews r
                    WHERE r.library_id = ? AND r.asset_id = ? AND r.tag_id = ?
                ) AND NOT EXISTS (
                    SELECT 1 FROM asset_tags at
                    WHERE at.library_id = ? AND at.asset_id = ? AND at.tag_id = ?
                )`,
				item.ID, generationID, libraryID, assetID, snapshotID, item.TagID,
				sourceFingerprint, item.Confidence, now.UTC().UnixMilli(), now.UTC().UnixMilli(),
				libraryID, assetID, item.TagID, libraryID, assetID, item.TagID,
			)
			if err != nil {
				return fmt.Errorf("store pending tag suggestion: %w", err)
			}
			if _, err := result.RowsAffected(); err != nil {
				return fmt.Errorf("read pending tag suggestion result: %w", err)
			}
		}
		return nil
	})
}

func (s *Store) GetTagSuggestion(ctx context.Context, suggestionID string) (semantic.TagSuggestion, bool, error) {
	if len(suggestionID) < 8 || len(suggestionID) > 128 {
		return semantic.TagSuggestion{}, false, semantic.ErrInvalidTagSuggestion
	}
	var item semantic.TagSuggestion
	var createdMS, updatedMS int64
	var confidence float64
	err := s.db.QueryRowContext(ctx, `
        SELECT id, generation_id, library_id, asset_id, vocabulary_snapshot_id,
               tag_id, source_fingerprint, confidence, state, revision, created_at_ms, updated_at_ms
        FROM ai_tag_suggestions WHERE id = ?`, suggestionID,
	).Scan(
		&item.ID, &item.GenerationID, &item.LibraryID, &item.AssetID, &item.SnapshotID,
		&item.TagID, &item.SourceFingerprint, &confidence, &item.State, &item.Revision, &createdMS, &updatedMS,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.TagSuggestion{}, false, nil
	}
	if err != nil {
		return semantic.TagSuggestion{}, false, fmt.Errorf("read tag suggestion: %w", err)
	}
	item.Confidence = float32(confidence)
	item.CreatedAt = time.UnixMilli(createdMS).UTC()
	item.UpdatedAt = time.UnixMilli(updatedMS).UTC()
	if semantic.ValidatePendingTagSuggestions(
		[]semantic.TagSuggestion{item}, item.LibraryID, item.AssetID, item.GenerationID, item.SnapshotID, item.SourceFingerprint,
	) != nil && item.State == "pending" {
		return semantic.TagSuggestion{}, false, errors.New("tag suggestion repository returned invalid state")
	}
	return item, true, nil
}

var _ semantic.TagEmbeddingInputRepository = (*Store)(nil)

func (s *Store) CommitTagReview(
	ctx context.Context,
	suggestionID string,
	expectedRevision int64,
	decision semantic.TagReviewDecision,
	acceptedCurationRevision int64,
	now time.Time,
) (semantic.TagReview, error) {
	if len(suggestionID) < 8 || len(suggestionID) > 128 || expectedRevision < 1 || now.IsZero() ||
		(decision != semantic.TagReviewAccept && decision != semantic.TagReviewDismiss) ||
		(decision == semantic.TagReviewAccept) != (acceptedCurationRevision > 0) {
		return semantic.TagReview{}, semantic.ErrInvalidTagSuggestion
	}
	var review semantic.TagReview
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		var sourceGenerationID, sourceSnapshotID string
		var sourceVocabularyRevision int64
		var sourceConfidence float64
		if err := tx.QueryRowContext(ctx, `
            SELECT suggestion.library_id, suggestion.asset_id, suggestion.tag_id, suggestion.state,
                   suggestion.revision, suggestion.generation_id, suggestion.vocabulary_snapshot_id,
                   vocabulary.revision, suggestion.confidence
            FROM ai_tag_suggestions suggestion
            JOIN ai_tag_vocabulary_snapshots vocabulary ON vocabulary.id=suggestion.vocabulary_snapshot_id
            WHERE suggestion.id = ?`, suggestionID,
		).Scan(&review.LibraryID, &review.AssetID, &review.TagID, &state, &review.Revision,
			&sourceGenerationID, &sourceSnapshotID, &sourceVocabularyRevision, &sourceConfidence); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrInvalidTagSuggestion
			}
			return fmt.Errorf("resolve reviewed tag suggestion: %w", err)
		}
		if state != "pending" || review.Revision != expectedRevision {
			return semantic.ErrInvalidTagSuggestion
		}
		var existing int
		if err := tx.QueryRowContext(ctx, `
            SELECT COUNT(*) FROM ai_tag_reviews WHERE library_id = ? AND asset_id = ? AND tag_id = ?`,
			review.LibraryID, review.AssetID, review.TagID,
		).Scan(&existing); err != nil {
			return fmt.Errorf("resolve existing tag review: %w", err)
		}
		if existing != 0 {
			return semantic.ErrInvalidTagSuggestion
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO ai_tag_reviews(
                library_id, asset_id, tag_id, decision, source_suggestion_id,
                accepted_curation_revision, revision, reviewed_at_ms,
                source_generation_id, source_vocabulary_snapshot_id,
                source_vocabulary_revision, source_confidence, source_suggestion_revision
            ) VALUES(?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?)`,
			review.LibraryID, review.AssetID, review.TagID, string(decision), suggestionID,
			nullPositiveInt64(acceptedCurationRevision), now.UTC().UnixMilli(), sourceGenerationID,
			sourceSnapshotID, sourceVocabularyRevision, sourceConfidence, expectedRevision,
		); err != nil {
			return fmt.Errorf("store tag review: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE ai_tag_suggestions
            SET state = 'invalidated', revision = revision + 1, updated_at_ms = ?
            WHERE id = ? AND state = 'pending' AND revision = ?`,
			now.UTC().UnixMilli(), suggestionID, expectedRevision,
		)
		if err != nil {
			return fmt.Errorf("retire reviewed tag suggestion: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil || changed != 1 {
			return semantic.ErrInvalidTagSuggestion
		}
		review.Decision = decision
		review.SourceSuggestionID = suggestionID
		review.AcceptedCurationRevision = acceptedCurationRevision
		review.Revision = 1
		review.ReviewedAt = now.UTC()
		return nil
	})
	return review, err
}

func (s *Store) GetTagReviewBySuggestion(ctx context.Context, suggestionID string) (semantic.TagReview, bool, error) {
	if len(suggestionID) < 8 || len(suggestionID) > 128 {
		return semantic.TagReview{}, false, semantic.ErrInvalidTagSuggestion
	}
	var value semantic.TagReview
	var decision string
	var accepted sql.NullInt64
	var reviewedMS int64
	err := s.db.QueryRowContext(ctx, `SELECT library_id,asset_id,tag_id,decision,source_suggestion_id,
        accepted_curation_revision,revision,reviewed_at_ms FROM ai_tag_reviews WHERE source_suggestion_id=?`, suggestionID).Scan(
		&value.LibraryID, &value.AssetID, &value.TagID, &decision, &value.SourceSuggestionID,
		&accepted, &value.Revision, &reviewedMS)
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.TagReview{}, false, nil
	}
	if err != nil {
		return semantic.TagReview{}, false, err
	}
	value.Decision = semantic.TagReviewDecision(decision)
	value.AcceptedCurationRevision = accepted.Int64
	value.ReviewedAt = time.UnixMilli(reviewedMS).UTC()
	return value, true, nil
}

func nullPositiveInt64(value int64) any {
	if value > 0 {
		return value
	}
	return nil
}
