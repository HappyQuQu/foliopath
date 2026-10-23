package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
)

func (s *Store) GetFaceDeploymentGenerationRuntime(ctx context.Context, id string) (face.DeploymentGenerationRuntime, error) {
	if len(id) < 8 || len(id) > 128 {
		return face.DeploymentGenerationRuntime{}, face.ErrRuntimeUnavailable
	}
	var value face.DeploymentGenerationRuntime
	err := s.db.QueryRowContext(ctx, `SELECT g.id, g.state, g.embedding_dimension,
        CASE WHEN m.storage_mode='service' AND m.purpose=? AND m.source_identity=m.content_hash
             THEN m.content_hash ELSE '' END, m.state='available'
        FROM face_generations g JOIN ai_models m ON m.id=g.model_id WHERE g.id=?`, aimodel.PurposeFaceDetectionEmbedding, id).
		Scan(&value.ID, &value.State, &value.Dimension, &value.Fingerprint, &value.Available)
	if errors.Is(err, sql.ErrNoRows) {
		return face.DeploymentGenerationRuntime{}, face.ErrRuntimeUnavailable
	}
	return value, err
}

var _ face.DeploymentGenerationRepository = (*Store)(nil)

// Called again inside each publishing transaction. Legacy file generations
// retain their existing contract; service generations fail closed when offline.
func requireFaceServiceAvailable(ctx context.Context, reader faceClusterStateReader, id string) error {
	var unavailable bool
	err := reader.QueryRowContext(ctx, `SELECT CASE WHEN m.storage_mode='service'
        THEN g.state!='active' OR m.state!='available' OR m.purpose!=? OR m.source_identity!=m.content_hash
        ELSE 0 END FROM face_generations g LEFT JOIN ai_models m ON m.id=g.model_id WHERE g.id=?`, aimodel.PurposeFaceDetectionEmbedding, id).Scan(&unavailable)
	if errors.Is(err, sql.ErrNoRows) || unavailable {
		return face.ErrFaceModelUnavailable
	}
	return err
}
