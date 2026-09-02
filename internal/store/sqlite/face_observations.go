package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func (s *Store) ReplaceFaceObservations(ctx context.Context, batch face.ObservationBatch) error {
	if err := face.ValidateObservationBatch(batch, min(s.maxBatchSize, face.MaxCandidatesPerAsset)); err != nil {
		return err
	}
	dimension, state, err := s.faceGenerationContract(ctx, s.db, batch.GenerationID)
	if err != nil {
		return err
	}
	if state != "building" && state != "ready" && state != "active" {
		return face.ErrFaceGenerationUnavailable
	}
	for _, item := range batch.Items {
		if err := face.ValidateEncodedEmbedding(item.Vector, dimension); err != nil {
			return err
		}
	}
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		currentDimension, currentState, err := s.faceGenerationContract(ctx, tx, batch.GenerationID)
		if err != nil {
			return err
		}
		if currentDimension != dimension ||
			(currentState != "building" && currentState != "ready" && currentState != "active") {
			return face.ErrFaceGenerationUnavailable
		}
		return s.replaceFaceObservationsTx(ctx, tx, batch)
	})
	if err != nil {
		return fmt.Errorf("replace face observations: %w", err)
	}
	return nil
}

func (s *Store) replaceFaceObservationsTx(ctx context.Context, tx *sql.Tx, batch face.ObservationBatch) error {
	for _, item := range batch.Items {
		box := quantizeBox(item.Box)
		_, err := tx.ExecContext(ctx, `
                INSERT INTO face_observations(
                    id, generation_id, library_id, asset_id, source_fingerprint,
                    box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm,
                    detection_ppm, quality_ppm, vector, revision, created_at_ms, updated_at_ms
                ) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, MAX(?, ?))
                ON CONFLICT(id) DO UPDATE SET
                    source_fingerprint=excluded.source_fingerprint,
                    box_x_ppm=excluded.box_x_ppm, box_y_ppm=excluded.box_y_ppm,
                    box_width_ppm=excluded.box_width_ppm, box_height_ppm=excluded.box_height_ppm,
                    detection_ppm=excluded.detection_ppm, quality_ppm=excluded.quality_ppm,
                    vector=excluded.vector, revision=face_observations.revision + 1,
                    updated_at_ms=MAX(face_observations.created_at_ms, excluded.updated_at_ms)
                WHERE face_observations.generation_id=excluded.generation_id
                  AND face_observations.library_id=excluded.library_id
                  AND face_observations.asset_id=excluded.asset_id
                  AND (face_observations.source_fingerprint<>excluded.source_fingerprint
                    OR face_observations.box_x_ppm<>excluded.box_x_ppm
                    OR face_observations.box_y_ppm<>excluded.box_y_ppm
                    OR face_observations.box_width_ppm<>excluded.box_width_ppm
                    OR face_observations.box_height_ppm<>excluded.box_height_ppm
                    OR face_observations.detection_ppm<>excluded.detection_ppm
                    OR face_observations.quality_ppm<>excluded.quality_ppm
                    OR face_observations.vector<>excluded.vector)`,
			item.ID, batch.GenerationID, batch.LibraryID, batch.AssetID, item.SourceFingerprint,
			box[0], box[1], box[2], box[3], quantizeUnit(item.Detection), quantizeUnit(item.Quality),
			item.Vector, item.CreatedAt.UTC().UnixMilli(), item.CreatedAt.UTC().UnixMilli(), batch.UpdatedAt.UTC().UnixMilli())
		if err != nil {
			return err
		}
	}
	query := `DELETE FROM face_observations WHERE generation_id=? AND library_id=? AND asset_id=?`
	args := []any{batch.GenerationID, batch.LibraryID, batch.AssetID}
	if len(batch.Items) > 0 {
		placeholders := make([]string, len(batch.Items))
		for index, item := range batch.Items {
			placeholders[index] = "?"
			args = append(args, item.ID)
		}
		query += " AND id NOT IN (" + strings.Join(placeholders, ",") + ")"
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) ListFaceObservations(ctx context.Context, generationID string, libraryID, assetID int64) ([]face.StoredObservation, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 || assetID < 1 {
		return nil, face.ErrInvalidObservation
	}
	dimension, _, err := s.faceGenerationContract(ctx, s.db, generationID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, generation_id, library_id, asset_id, source_fingerprint,
               box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm,
               detection_ppm, quality_ppm, vector, revision, created_at_ms, updated_at_ms
        FROM face_observations
        WHERE generation_id=? AND library_id=? AND asset_id=? ORDER BY id`, generationID, libraryID, assetID)
	if err != nil {
		return nil, fmt.Errorf("list face observations: %w", err)
	}
	defer rows.Close()
	items := make([]face.StoredObservation, 0)
	for rows.Next() {
		var item face.StoredObservation
		var x, y, width, height, detection, quality, createdAt, updatedAt int64
		if err := rows.Scan(&item.ID, &item.GenerationID, &item.LibraryID, &item.AssetID, &item.SourceFingerprint,
			&x, &y, &width, &height, &detection, &quality, &item.Vector, &item.Revision, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan face observation: %w", err)
		}
		if err := face.ValidateEncodedEmbedding(item.Vector, dimension); err != nil {
			return nil, err
		}
		item.Box = face.Box{X: float32(x) / 1e6, Y: float32(y) / 1e6, Width: float32(width) / 1e6, Height: float32(height) / 1e6}
		item.Detection, item.Quality = float32(detection)/1e6, float32(quality)/1e6
		item.CreatedAt, item.UpdatedAt = time.UnixMilli(createdAt).UTC(), time.UnixMilli(updatedAt).UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate face observations: %w", err)
	}
	return items, nil
}

func (s *Store) DeleteFaceObservationsIfSourceChanged(ctx context.Context, generationID string, libraryID, assetID int64, fingerprint string, updatedAt time.Time) (bool, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 || assetID < 1 ||
		len(fingerprint) < 1 || len(fingerprint) > 256 || updatedAt.IsZero() {
		return false, face.ErrInvalidObservation
	}
	deleted := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `DELETE FROM face_observations
            WHERE generation_id=? AND library_id=? AND asset_id=? AND source_fingerprint<>?`,
			generationID, libraryID, assetID, fingerprint)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		deleted = rows > 0
		if err != nil || !deleted {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE person_face_anchors
            SET state='needs_review', revision=revision+1,
                updated_at_ms=MAX(created_at_ms, ?)
            WHERE library_id=? AND asset_id=? AND source_fingerprint<>?`,
			updatedAt.UTC().UnixMilli(), libraryID, assetID, fingerprint)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("delete changed face observations: %w", err)
	}
	return deleted, nil
}

type faceGenerationQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) faceGenerationContract(ctx context.Context, db faceGenerationQuerier, generationID string) (int, string, error) {
	var dimension int
	var state string
	err := db.QueryRowContext(ctx, `SELECT embedding_dimension, state FROM face_generations WHERE id=?`, generationID).Scan(&dimension, &state)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", face.ErrFaceGenerationUnavailable
	}
	if err != nil {
		return 0, "", err
	}
	return dimension, state, nil
}

func quantizeUnit(value float32) int64 { return int64(math.Round(float64(value) * 1e6)) }

func quantizeBox(box face.Box) [4]int64 {
	x, y := quantizeUnit(box.X), quantizeUnit(box.Y)
	width, height := quantizeUnit(box.Width), quantizeUnit(box.Height)
	if x+width > 1e6 {
		width = 1e6 - x
	}
	if y+height > 1e6 {
		height = 1e6 - y
	}
	return [4]int64{x, y, width, height}
}

var _ face.ObservationRepository = (*Store)(nil)
