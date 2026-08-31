package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) ReplaceVideoEmbeddingPlan(ctx context.Context, plan semantic.VideoEmbeddingPlan) error {
	var dimension int
	if err := s.db.QueryRowContext(ctx, `
        SELECT embedding_dimension FROM semantic_generations WHERE id = ? AND state = 'active'`,
		plan.GenerationID,
	).Scan(&dimension); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.ErrSemanticGenerationUnavailable
		}
		return fmt.Errorf("resolve video semantic generation: %w", err)
	}
	if err := semantic.ValidateVideoEmbeddingPlan(plan, dimension); err != nil {
		return err
	}
	expectedFingerprint, err := semantic.StoryboardFingerprint(
		plan.SourceFingerprint, plan.TransformVersion, plan.PlanSize,
	)
	if err != nil || expectedFingerprint != plan.StoryboardFingerprint {
		return semantic.ErrInvalidVideoSemantic
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		return replaceVideoEmbeddingPlanTx(ctx, tx, plan)
	})
}

func replaceVideoEmbeddingPlanTx(ctx context.Context, tx *sql.Tx, plan semantic.VideoEmbeddingPlan) error {
	var sourceFingerprint, status string
	var transformVersion, frameCount int
	if err := tx.QueryRowContext(ctx, `
            SELECT a.source_fingerprint, t.status, t.transform_version, t.frame_count
            FROM assets a
            JOIN thumbnails t ON t.library_id = a.library_id AND t.asset_id = a.id
                              AND t.variant = 'storyboard'
            WHERE a.library_id = ? AND a.id = ?`, plan.LibraryID, plan.AssetID,
	).Scan(&sourceFingerprint, &status, &transformVersion, &frameCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return semantic.ErrStoryboardNotReady
		}
		return fmt.Errorf("resolve complete storyboard: %w", err)
	}
	if status != "ready" || sourceFingerprint != plan.SourceFingerprint ||
		transformVersion != plan.TransformVersion || frameCount != plan.PlanSize {
		return semantic.ErrStoryboardSourceChanged
	}
	if _, err := tx.ExecContext(ctx, `
            DELETE FROM semantic_video_frames
            WHERE generation_id = ? AND library_id = ? AND asset_id = ?`,
		plan.GenerationID, plan.LibraryID, plan.AssetID,
	); err != nil {
		return fmt.Errorf("replace video frame embeddings: %w", err)
	}
	for _, frame := range plan.Frames {
		if _, err := tx.ExecContext(ctx, `
                INSERT INTO semantic_video_frames(
                    generation_id, library_id, asset_id, source_fingerprint,
                    storyboard_fingerprint, storyboard_transform_version, plan_size,
                    ordinal, timestamp_ms, vector, created_at_ms
                ) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			plan.GenerationID, plan.LibraryID, plan.AssetID, plan.SourceFingerprint,
			plan.StoryboardFingerprint, plan.TransformVersion, plan.PlanSize,
			frame.Ordinal, frame.TimestampMS, frame.Vector, plan.CreatedAt.UTC().UnixMilli(),
		); err != nil {
			return fmt.Errorf("store video frame embedding: %w", err)
		}
	}
	return nil
}
