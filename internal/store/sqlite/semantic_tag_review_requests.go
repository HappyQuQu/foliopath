package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) BeginTagReviewRequest(ctx context.Context, value semantic.TagReviewRequestRecord) (semantic.TagReviewRequestRecord, bool, error) {
	if err := validateTagReviewRequestRecord(value); err != nil {
		return semantic.TagReviewRequestRecord{}, false, err
	}
	created := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var requestHash string
		err := tx.QueryRowContext(ctx, `SELECT request_hash FROM ai_tag_review_requests WHERE idempotency_key_hash=?`, value.IdempotencyKeyHash).Scan(&requestHash)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO ai_tag_review_requests(idempotency_key_hash,request_hash,state,created_at_ms,updated_at_ms)
            VALUES(?,?,'running',?,?)`, value.IdempotencyKeyHash, value.RequestHash, value.CreatedAt.UnixMilli(), value.UpdatedAt.UnixMilli()); err != nil {
			return err
		}
		for index, state := range value.Items {
			if _, err := tx.ExecContext(ctx, `INSERT INTO ai_tag_review_request_items(
                    idempotency_key_hash,ordinal,suggestion_id,action,expected_suggestion_revision,expected_curation_revision)
                VALUES(?,?,?,?,?,?)`, value.IdempotencyKeyHash, index, state.Item.SuggestionID, state.Item.Action,
				state.Item.ExpectedSuggestionRevision, state.Item.ExpectedCurationRevision); err != nil {
				return err
			}
		}
		created = true
		return nil
	})
	if err != nil {
		return semantic.TagReviewRequestRecord{}, false, err
	}
	stored, err := s.getTagReviewRequest(ctx, value.IdempotencyKeyHash)
	return stored, created, err
}

func (s *Store) CommitTagReviewRequestOutcome(ctx context.Context, keyHash string, ordinal int, outcome semantic.TagReviewOutcome, now time.Time) error {
	if len(keyHash) != 64 || ordinal < 0 || ordinal >= semantic.MaxTagSuggestionReviewBatch || len(outcome.SuggestionID) < 8 ||
		(outcome.Conflict && outcome.Outcome != "") || (!outcome.Conflict && outcome.Outcome != semantic.TagReviewAccept && outcome.Outcome != semantic.TagReviewDismiss) ||
		outcome.Revision < 1 || now.IsZero() {
		return semantic.ErrInvalidTagSuggestion
	}
	state := string(outcome.Outcome)
	if outcome.Conflict {
		state = "conflict"
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_request_items SET outcome=?,outcome_revision=?
            WHERE idempotency_key_hash=? AND ordinal=? AND suggestion_id=? AND outcome IS NULL`, state, outcome.Revision,
			keyHash, ordinal, outcome.SuggestionID)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			var existing string
			var revision int64
			if err := tx.QueryRowContext(ctx, `SELECT outcome,outcome_revision FROM ai_tag_review_request_items
                    WHERE idempotency_key_hash=? AND ordinal=? AND suggestion_id=?`, keyHash, ordinal, outcome.SuggestionID).Scan(&existing, &revision); err != nil || existing != state || revision != outcome.Revision {
				return semantic.ErrTagReviewRequestConflict
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_tag_review_requests SET updated_at_ms=? WHERE idempotency_key_hash=? AND state='running'`, now.UnixMilli(), keyHash)
		return err
	})
}

func (s *Store) CompleteTagReviewRequest(ctx context.Context, keyHash string, now time.Time) error {
	if len(keyHash) != 64 || now.IsZero() {
		return semantic.ErrInvalidTagSuggestion
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var pending int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_tag_review_request_items WHERE idempotency_key_hash=? AND outcome IS NULL`, keyHash).Scan(&pending); err != nil {
			return err
		}
		if pending != 0 {
			return semantic.ErrTagReviewRequestConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE ai_tag_review_requests SET state='completed',updated_at_ms=?
            WHERE idempotency_key_hash=? AND state IN ('running','completed')`, now.UnixMilli(), keyHash)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return semantic.ErrTagReviewRequestConflict
		}
		return nil
	})
}

func (s *Store) getTagReviewRequest(ctx context.Context, keyHash string) (semantic.TagReviewRequestRecord, error) {
	var value semantic.TagReviewRequestRecord
	var created, updated int64
	if err := s.db.QueryRowContext(ctx, `SELECT idempotency_key_hash,request_hash,state,created_at_ms,updated_at_ms
        FROM ai_tag_review_requests WHERE idempotency_key_hash=?`, keyHash).Scan(&value.IdempotencyKeyHash, &value.RequestHash, &value.State, &created, &updated); err != nil {
		return semantic.TagReviewRequestRecord{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT suggestion_id,action,expected_suggestion_revision,expected_curation_revision,outcome,outcome_revision
        FROM ai_tag_review_request_items WHERE idempotency_key_hash=? ORDER BY ordinal`, keyHash)
	if err != nil {
		return semantic.TagReviewRequestRecord{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var state semantic.TagReviewRequestItemState
		var action string
		var outcome sql.NullString
		var revision sql.NullInt64
		if err := rows.Scan(&state.Item.SuggestionID, &action, &state.Item.ExpectedSuggestionRevision,
			&state.Item.ExpectedCurationRevision, &outcome, &revision); err != nil {
			return semantic.TagReviewRequestRecord{}, err
		}
		state.Item.Action = semantic.TagReviewDecision(action)
		if outcome.Valid {
			value := semantic.TagReviewOutcome{SuggestionID: state.Item.SuggestionID, Revision: revision.Int64}
			if outcome.String == "conflict" {
				value.Conflict = true
			} else {
				value.Outcome = semantic.TagReviewDecision(outcome.String)
			}
			state.Outcome = &value
		}
		value.Items = append(value.Items, state)
	}
	value.CreatedAt, value.UpdatedAt = time.UnixMilli(created).UTC(), time.UnixMilli(updated).UTC()
	return value, rows.Err()
}

func validateTagReviewRequestRecord(value semantic.TagReviewRequestRecord) error {
	if len(value.IdempotencyKeyHash) != 64 || len(value.RequestHash) != 64 || value.State != "running" ||
		len(value.Items) < 1 || len(value.Items) > semantic.MaxTagSuggestionReviewBatch || value.CreatedAt.IsZero() || !value.UpdatedAt.Equal(value.CreatedAt) {
		return semantic.ErrInvalidTagSuggestion
	}
	return nil
}

var _ semantic.TagReviewRequestRepository = (*Store)(nil)
