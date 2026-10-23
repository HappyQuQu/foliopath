package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) SearchSemanticVectors(ctx context.Context, request semantic.VectorSearchRequest) ([]semantic.VectorMatch, error) {
	if err := semantic.ValidateVectorSearchRequest(request); err != nil {
		return nil, err
	}
	dimension, state, err := s.semanticGenerationContract(ctx, s.db, request.GenerationID)
	if err != nil {
		return nil, err
	}
	if state != "active" || len(request.Query) != dimension {
		return nil, semantic.ErrSemanticGenerationUnavailable
	}
	query, err := semantic.NormalizeEmbedding(request.Query, dimension)
	if err != nil {
		return nil, semantic.ErrInvalidSemanticSearch
	}
	prefix := ""
	if request.DirectoryID > 0 && request.Recursive {
		prefix = `WITH RECURSIVE subtree(id) AS (
            SELECT id FROM directories WHERE library_id=? AND id=?
            UNION ALL
            SELECT child.id FROM directories child JOIN subtree parent ON child.parent_id=parent.id
            WHERE child.library_id=?
        ) `
	}
	statement := prefix + `
        SELECT embedding.library_id, embedding.asset_id, embedding.vector
        FROM semantic_embeddings embedding
        JOIN assets asset ON asset.library_id=embedding.library_id AND asset.id=embedding.asset_id
        JOIN libraries library ON library.id=embedding.library_id AND library.status <> 'offline'
        JOIN ai_library_settings setting ON setting.library_id=embedding.library_id AND setting.enabled=1
        WHERE embedding.generation_id=? AND embedding.source_fingerprint=asset.source_fingerprint`
	arguments := make([]any, 0, 6)
	if prefix != "" {
		arguments = append(arguments, request.LibraryID, request.DirectoryID, request.LibraryID)
	}
	arguments = append(arguments, request.GenerationID)
	if request.LibraryID > 0 {
		statement += ` AND embedding.library_id=?`
		arguments = append(arguments, request.LibraryID)
	}
	if request.DirectoryID > 0 {
		if request.Recursive {
			statement += ` AND asset.directory_id IN (SELECT id FROM subtree)`
		} else {
			statement += ` AND asset.directory_id=?`
			arguments = append(arguments, request.DirectoryID)
		}
	}
	statement += ` ORDER BY embedding.library_id, embedding.asset_id`
	rows, err := s.db.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("search semantic vectors: %w", err)
	}
	defer rows.Close()
	result, err := semantic.BoundedVectorMatches(request.Limit, func(offer func(semantic.VectorMatch) error) error {
		for rows.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}
			var libraryID, assetID int64
			var blob []byte
			if err := rows.Scan(&libraryID, &assetID, &blob); err != nil {
				return err
			}
			vector, err := semantic.DecodeEmbedding(blob, dimension)
			if err != nil {
				return semantic.ErrInvalidEmbeddingRecord
			}
			var score64 float64
			for index, value := range vector {
				score64 += float64(value) * float64(query[index])
			}
			score := float32(score64)
			if math.IsNaN(float64(score)) || math.IsInf(float64(score), 0) {
				return semantic.ErrInvalidEmbeddingRecord
			}
			if request.After != nil && !(score < request.After.Score || score == request.After.Score && assetID > request.After.AssetID) {
				continue
			}
			if err := offer(semantic.VectorMatch{LibraryID: libraryID, AssetID: assetID, Score: score}); err != nil {
				return err
			}
		}
		return rows.Err()
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []semantic.VectorMatch{}, nil
		}
		return nil, fmt.Errorf("search semantic vectors: %w", err)
	}
	return result, nil
}

var _ semantic.VectorSearchRepository = (*Store)(nil)
