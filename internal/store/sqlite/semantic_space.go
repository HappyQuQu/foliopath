package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) BindSemanticGenerationSpace(ctx context.Context, generationID string, identity semantic.SpaceIdentity) error {
	fingerprint, err := identity.Fingerprint()
	if err != nil {
		return err
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		return bindSemanticGenerationSpace(ctx, tx, generationID, identity, fingerprint)
	})
}

// Shared by standalone building-space binding and the atomic activation owner.
func bindSemanticGenerationSpace(ctx context.Context, tx *sql.Tx, generationID string, identity semantic.SpaceIdentity, fingerprint string) error {
	var dimension int
	var state, bound string
	err := tx.QueryRowContext(ctx, `SELECT g.embedding_dimension,g.state,COALESCE(s.fingerprint,'')
            FROM semantic_generations g LEFT JOIN semantic_generation_spaces s ON s.generation_id=g.id
            WHERE g.id=?`, generationID).Scan(&dimension, &state, &bound)
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.ErrSemanticGenerationUnavailable
	}
	if err != nil {
		return err
	}
	if dimension != identity.Dimension || state != "building" {
		return semantic.ErrIncompatibleSpace
	}
	if bound != "" {
		if bound == fingerprint {
			return nil
		}
		return semantic.ErrIncompatibleSpace
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM semantic_embeddings WHERE generation_id=?)
            OR EXISTS(SELECT 1 FROM semantic_video_frames WHERE generation_id=?)
            OR EXISTS(SELECT 1 FROM semantic_tag_embeddings WHERE generation_id=?)`, generationID, generationID, generationID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return semantic.ErrIncompatibleSpace
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO semantic_generation_spaces(generation_id,fingerprint) VALUES (?,?)`, generationID, fingerprint)
	return err
}

var _ semantic.GenerationSpaceRepository = (*Store)(nil)
