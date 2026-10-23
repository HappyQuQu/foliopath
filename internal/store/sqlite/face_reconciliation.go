package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func (s *Store) ReconcileFaceAnchors(ctx context.Context, generationID string, libraryID int64, updatedAt time.Time) (face.ReconciliationResult, error) {
	return s.reconcileFaceBindings(ctx, generationID, libraryID, updatedAt, nil)
}

func (s *Store) reconcileFaceBindings(ctx context.Context, generationID string, libraryID int64, updatedAt time.Time, guard func(*sql.Tx) error) (face.ReconciliationResult, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 || updatedAt.IsZero() {
		return face.ReconciliationResult{}, face.ErrInvalidReview
	}
	if _, _, err := s.faceGenerationContract(ctx, s.db, generationID); err != nil {
		return face.ReconciliationResult{}, err
	}
	var result face.ReconciliationResult
	for _, table := range []string{"person_face_anchors", "face_exclusions"} {
		after := ""
		for {
			pageCount := 0
			err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
				if guard != nil {
					if err := guard(tx); err != nil {
						return err
					}
				}
				rows, err := tx.QueryContext(ctx, `SELECT anchor.id,
			(SELECT COUNT(*) FROM face_observations observation WHERE observation.generation_id=? AND observation.library_id=anchor.library_id AND observation.asset_id=anchor.asset_id AND observation.source_fingerprint=anchor.source_fingerprint AND observation.box_x_ppm=anchor.box_x_ppm AND observation.box_y_ppm=anchor.box_y_ppm AND observation.box_width_ppm=anchor.box_width_ppm AND observation.box_height_ppm=anchor.box_height_ppm),
			(SELECT MIN(id) FROM face_observations observation WHERE observation.generation_id=? AND observation.library_id=anchor.library_id AND observation.asset_id=anchor.asset_id AND observation.source_fingerprint=anchor.source_fingerprint AND observation.box_x_ppm=anchor.box_x_ppm AND observation.box_y_ppm=anchor.box_y_ppm AND observation.box_width_ppm=anchor.box_width_ppm AND observation.box_height_ppm=anchor.box_height_ppm)
			FROM `+table+` anchor WHERE anchor.library_id=? AND anchor.id>? ORDER BY anchor.id LIMIT ?`, generationID, generationID, libraryID, after, s.maxBatchSize)
				if err != nil {
					return err
				}
				type match struct {
					id     string
					count  int64
					faceID sql.NullString
				}
				matches := make([]match, 0)
				for rows.Next() {
					var item match
					if err := rows.Scan(&item.id, &item.count, &item.faceID); err != nil {
						rows.Close()
						return err
					}
					matches = append(matches, item)
				}
				if err := rows.Err(); err != nil {
					rows.Close()
					return err
				}
				if err := rows.Close(); err != nil {
					return err
				}
				pageCount = len(matches)
				for _, item := range matches {
					after = item.id
					state := "needs_review"
					var faceID any
					if item.count == 1 && item.faceID.Valid {
						state = "bound"
						faceID = item.faceID.String
						if table == "person_face_anchors" {
							result.Bound++
						}
					} else {
						if table == "person_face_anchors" {
							result.NeedsReview++
						}
					}
					if table == "face_exclusions" {
						_, err = tx.ExecContext(ctx, `UPDATE face_exclusions SET current_face_id=?,revision=revision+1,updated_at_ms=MAX(updated_at_ms,?) WHERE id=? AND current_face_id IS NOT ?`, faceID, updatedAt.UnixMilli(), item.id, faceID)
					} else {
						_, err = tx.ExecContext(ctx, `UPDATE person_face_anchors SET current_face_id=?,state=CASE WHEN state='excluded' THEN state ELSE ? END,revision=revision+1,updated_at_ms=MAX(updated_at_ms,?) WHERE id=? AND (current_face_id IS NOT ? OR (state!='excluded' AND state!=?))`, faceID, state, updatedAt.UTC().UnixMilli(), item.id, faceID, state)
					}
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return face.ReconciliationResult{}, fmt.Errorf("reconcile face anchors: %w", err)
			}
			if pageCount < s.maxBatchSize {
				break
			}
		}
	}
	return result, nil
}

var _ face.ReconciliationRepository = (*Store)(nil)
