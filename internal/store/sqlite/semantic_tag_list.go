package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) GetTagSuggestionListSnapshot(ctx context.Context, libraryID int64) (semantic.TagSuggestionListSnapshot, error) {
	if libraryID < 1 {
		return semantic.TagSuggestionListSnapshot{}, semantic.ErrInvalidTagSuggestion
	}
	var snapshot semantic.TagSuggestionListSnapshot
	snapshot.LibraryID = libraryID
	var libraryStatus string
	if err := s.db.QueryRowContext(ctx, `SELECT status FROM libraries WHERE id=?`, libraryID).Scan(&libraryStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.TagSuggestionListSnapshot{}, semantic.ErrSemanticLibraryNotFound
		}
		return semantic.TagSuggestionListSnapshot{}, err
	}
	if libraryStatus != "ready" {
		return semantic.TagSuggestionListSnapshot{}, semantic.ErrSemanticLibraryOffline
	}
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM semantic_generations WHERE state='active'`).Scan(&snapshot.GenerationID); errors.Is(err, sql.ErrNoRows) {
		snapshot.GenerationID = "no-active-generation"
	} else if err != nil {
		return semantic.TagSuggestionListSnapshot{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT id,revision FROM ai_tag_vocabulary_snapshots WHERE state='active'`).Scan(
		&snapshot.VocabularyID, &snapshot.VocabularyRevision); err != nil {
		return semantic.TagSuggestionListSnapshot{}, semantic.ErrTagVocabularyUnavailable
	}
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM catalog_search_state WHERE singleton_key=1`).Scan(&snapshot.CatalogRevision); err != nil {
		return semantic.TagSuggestionListSnapshot{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(revision+updated_at_ms),0) FROM ai_tag_suggestions WHERE library_id=?`, libraryID).Scan(&snapshot.SuggestionRevision); err != nil {
		return semantic.TagSuggestionListSnapshot{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM ai_tag_review_state WHERE library_id=?`, libraryID).Scan(&snapshot.ReviewRevision); err != nil {
		return semantic.TagSuggestionListSnapshot{}, err
	}
	var progressRevision int64
	err := s.db.QueryRowContext(ctx, `SELECT eligible_count,ready_count,degraded_count,failed_count,stale_count,revision
		FROM semantic_tag_library_progress WHERE generation_id=? AND library_id=? AND vocabulary_snapshot_id=?`,
		snapshot.GenerationID, libraryID, snapshot.VocabularyID).Scan(&snapshot.Coverage.Eligible, &snapshot.Coverage.Completed,
		&snapshot.Coverage.Degraded, &snapshot.Coverage.Failed, &snapshot.Coverage.Stale, &progressRevision)
	if errors.Is(err, sql.ErrNoRows) {
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assets asset JOIN semantic_embeddings embedding
			ON embedding.library_id=asset.library_id AND embedding.asset_id=asset.id
			AND embedding.source_fingerprint=asset.source_fingerprint
			WHERE asset.library_id=? AND embedding.generation_id=? AND asset.kind IN ('image','animated')`,
			libraryID, snapshot.GenerationID).Scan(&snapshot.Coverage.Eligible); err != nil {
			return semantic.TagSuggestionListSnapshot{}, err
		}
	} else if err != nil {
		return semantic.TagSuggestionListSnapshot{}, err
	}
	snapshot.Coverage.Revision = max(int64(1), snapshot.SuggestionRevision+snapshot.ReviewRevision+snapshot.VocabularyRevision+progressRevision)
	return snapshot, nil
}

func (s *Store) ListTagSuggestionViews(ctx context.Context, query semantic.TagSuggestionListQuery) ([]semantic.TagSuggestionView, error) {
	if query.LibraryID < 1 || query.Limit < 1 || query.Limit > semantic.MaxTagSuggestionPageSize+1 || query.TagID < 0 {
		return nil, semantic.ErrInvalidTagSuggestion
	}
	switch query.Status {
	case semantic.TagSuggestionPending:
		return s.listPendingTagSuggestionViews(ctx, query)
	case semantic.TagSuggestionAccepted, semantic.TagSuggestionDismissed:
		return s.listReviewedTagSuggestionViews(ctx, query)
	default:
		return nil, semantic.ErrInvalidTagSuggestion
	}
}

func (s *Store) listPendingTagSuggestionViews(ctx context.Context, query semantic.TagSuggestionListQuery) ([]semantic.TagSuggestionView, error) {
	args := []any{query.LibraryID}
	filter := ""
	if query.TagID > 0 {
		filter += " AND suggestion.tag_id=?"
		args = append(args, query.TagID)
	}
	if query.After != nil {
		filter += " AND (suggestion.confidence<? OR (suggestion.confidence=? AND suggestion.id>?))"
		args = append(args, query.After.Confidence, query.After.Confidence, query.After.SuggestionID)
	}
	args = append(args, query.Limit)
	rows, err := s.db.QueryContext(ctx, `SELECT suggestion.id,suggestion.library_id,suggestion.asset_id,suggestion.tag_id,tag.name,
            suggestion.confidence,suggestion.generation_id,vocabulary.revision,suggestion.revision
        FROM ai_tag_suggestions suggestion
        JOIN assets asset ON asset.library_id=suggestion.library_id AND asset.id=suggestion.asset_id
		JOIN semantic_embeddings embedding ON embedding.generation_id=suggestion.generation_id
		  AND embedding.library_id=suggestion.library_id AND embedding.asset_id=suggestion.asset_id
		  AND embedding.source_fingerprint=asset.source_fingerprint
        JOIN tags tag ON tag.id=suggestion.tag_id
        JOIN semantic_generations generation ON generation.id=suggestion.generation_id AND generation.state='active'
        JOIN ai_tag_vocabulary_snapshots vocabulary ON vocabulary.id=suggestion.vocabulary_snapshot_id AND vocabulary.state='active'
        WHERE suggestion.library_id=? AND suggestion.state='pending' AND suggestion.source_fingerprint=asset.source_fingerprint`+filter+`
        ORDER BY suggestion.confidence DESC,suggestion.id ASC LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list pending tag suggestions: %w", err)
	}
	defer rows.Close()
	var items []semantic.TagSuggestionView
	for rows.Next() {
		var item semantic.TagSuggestionView
		var confidence float64
		if err := rows.Scan(&item.ID, &item.LibraryID, &item.AssetID, &item.TagID, &item.TagName, &confidence,
			&item.GenerationID, &item.VocabularyRevision, &item.Revision); err != nil {
			return nil, err
		}
		item.Confidence, item.Status = float32(confidence), semantic.TagSuggestionPending
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) listReviewedTagSuggestionViews(ctx context.Context, query semantic.TagSuggestionListQuery) ([]semantic.TagSuggestionView, error) {
	args := []any{query.LibraryID, string(query.Status)}
	filter := ""
	if query.TagID > 0 {
		filter += " AND review.tag_id=?"
		args = append(args, query.TagID)
	}
	if query.After != nil {
		filter += " AND (review.reviewed_at_ms<? OR (review.reviewed_at_ms=? AND review.source_suggestion_id>?))"
		args = append(args, query.After.ReviewedAtMS, query.After.ReviewedAtMS, query.After.SuggestionID)
	}
	args = append(args, query.Limit)
	rows, err := s.db.QueryContext(ctx, `SELECT review.source_suggestion_id,review.library_id,review.asset_id,review.tag_id,tag.name,
            review.source_confidence,review.source_generation_id,review.source_vocabulary_revision,review.revision,review.reviewed_at_ms
        FROM ai_tag_reviews review
        JOIN assets asset ON asset.library_id=review.library_id AND asset.id=review.asset_id
        JOIN tags tag ON tag.id=review.tag_id
        WHERE review.library_id=? AND review.decision=? AND review.source_generation_id IS NOT NULL`+filter+`
        ORDER BY review.reviewed_at_ms DESC,review.source_suggestion_id ASC LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list reviewed tag suggestions: %w", err)
	}
	defer rows.Close()
	var items []semantic.TagSuggestionView
	for rows.Next() {
		var item semantic.TagSuggestionView
		var confidence float64
		var reviewedMS int64
		if err := rows.Scan(&item.ID, &item.LibraryID, &item.AssetID, &item.TagID, &item.TagName, &confidence,
			&item.GenerationID, &item.VocabularyRevision, &item.Revision, &reviewedMS); err != nil {
			return nil, err
		}
		item.Confidence, item.Status = float32(confidence), query.Status
		reviewed := time.UnixMilli(reviewedMS).UTC()
		item.ReviewedAt = &reviewed
		items = append(items, item)
	}
	return items, rows.Err()
}

var _ semantic.TagSuggestionListRepository = (*Store)(nil)
