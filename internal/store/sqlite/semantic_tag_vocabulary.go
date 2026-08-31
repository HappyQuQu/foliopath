package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) GetActiveTagVocabulary(ctx context.Context) (semantic.TagVocabulary, error) {
	return getActiveTagVocabulary(ctx, s.db)
}

func (s *Store) PublishTagVocabulary(ctx context.Context, snapshotID string, expectedRevision int64, tagIDs []int64, now time.Time) (semantic.TagVocabulary, error) {
	if len(snapshotID) < 8 || len(snapshotID) > 128 || expectedRevision < 1 || now.IsZero() || len(tagIDs) > semantic.MaxControlledVocabularyEntries {
		return semantic.TagVocabulary{}, semantic.ErrInvalidTagSuggestion
	}
	for index, id := range tagIDs {
		if id < 1 || index > 0 && tagIDs[index-1] >= id {
			return semantic.TagVocabulary{}, semantic.ErrInvalidTagSuggestion
		}
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var currentID string
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT id,revision FROM ai_tag_vocabulary_snapshots WHERE state='active'`).Scan(&currentID, &revision); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrTagVocabularyUnavailable
			}
			return err
		}
		if revision != expectedRevision {
			return semantic.ErrTagVocabularyConflict
		}
		for _, tagID := range tagIDs {
			var exists int
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tags WHERE id=?)`, tagID).Scan(&exists); err != nil {
				return err
			}
			if exists != 1 {
				return curation.ErrTagNotFound
			}
		}
		result, err := tx.ExecContext(ctx, `UPDATE ai_tag_vocabulary_snapshots SET state='retired' WHERE id=? AND revision=? AND state='active'`, currentID, expectedRevision)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagVocabularyConflict
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO ai_tag_vocabulary_snapshots(id,revision,state,created_at_ms) VALUES(?,?,'active',?)`,
			snapshotID, expectedRevision+1, now.UTC().UnixMilli()); err != nil {
			return err
		}
		for _, tagID := range tagIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO ai_tag_vocabulary_entries(snapshot_id,tag_id) VALUES(?,?)`, snapshotID, tagID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return semantic.TagVocabulary{}, err
	}
	return getActiveTagVocabulary(ctx, s.db)
}

type tagVocabularyQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func getActiveTagVocabulary(ctx context.Context, queryer tagVocabularyQueryer) (semantic.TagVocabulary, error) {
	var value semantic.TagVocabulary
	var published int64
	if err := queryer.QueryRowContext(ctx, `SELECT id,revision,created_at_ms FROM ai_tag_vocabulary_snapshots WHERE state='active'`).Scan(
		&value.ID, &value.Revision, &published); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.TagVocabulary{}, semantic.ErrTagVocabularyUnavailable
		}
		return semantic.TagVocabulary{}, fmt.Errorf("get active tag vocabulary: %w", err)
	}
	rows, err := queryer.QueryContext(ctx, `SELECT entry.tag_id,tag.name FROM ai_tag_vocabulary_entries entry
        JOIN tags tag ON tag.id=entry.tag_id WHERE entry.snapshot_id=? ORDER BY entry.tag_id`, value.ID)
	if err != nil {
		return semantic.TagVocabulary{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var entry semantic.TagVocabularyEntry
		if err := rows.Scan(&entry.TagID, &entry.Name); err != nil {
			return semantic.TagVocabulary{}, err
		}
		value.Entries = append(value.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		return semantic.TagVocabulary{}, err
	}
	value.PublishedAt = time.UnixMilli(published).UTC()
	return value, nil
}

var _ semantic.TagVocabularyRepository = (*Store)(nil)
