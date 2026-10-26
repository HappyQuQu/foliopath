package sqlite

import (
	"context"
	"fmt"
	"math"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) SearchVideoSemanticVectors(ctx context.Context, request semantic.VideoVectorSearchRequest) ([]semantic.VideoVectorMatch, error) {
	if err := semantic.ValidateVideoVectorSearchRequest(request); err != nil {
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
		return nil, semantic.ErrInvalidVideoSemantic
	}
	prefix := ""
	arguments := make([]any, 0, 6)
	if request.DirectoryID > 0 && request.Recursive {
		prefix = `WITH RECURSIVE subtree(id) AS (
            SELECT id FROM directories WHERE library_id=? AND id=?
            UNION ALL
            SELECT child.id FROM directories child JOIN subtree parent ON child.parent_id=parent.id
            WHERE child.library_id=?
        ) `
		arguments = append(arguments, request.LibraryID, request.DirectoryID, request.LibraryID)
	}
	statement := prefix + `
        SELECT frame.library_id, frame.asset_id, frame.plan_size, frame.ordinal,
               frame.timestamp_ms, frame.vector
        FROM semantic_video_frames frame
        JOIN assets asset ON asset.library_id=frame.library_id AND asset.id=frame.asset_id
        JOIN libraries library ON library.id=frame.library_id AND library.status <> 'offline'
        JOIN ai_library_settings setting ON setting.library_id=frame.library_id AND setting.enabled=1
        JOIN thumbnails storyboard ON storyboard.library_id=frame.library_id
            AND storyboard.asset_id=frame.asset_id AND storyboard.variant='storyboard'
            AND storyboard.status='ready' AND storyboard.source_fingerprint=asset.source_fingerprint
            AND storyboard.transform_version=frame.storyboard_transform_version
            AND storyboard.frame_count=frame.plan_size
        WHERE frame.generation_id=? AND frame.source_fingerprint=asset.source_fingerprint`
	arguments = append(arguments, request.GenerationID)
	if request.LibraryID > 0 {
		statement += ` AND frame.library_id=?`
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
	statement += ` ORDER BY frame.library_id, frame.asset_id, frame.ordinal`
	rows, err := s.db.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("search video semantic vectors: %w", err)
	}
	defer rows.Close()
	result, err := semantic.BoundedVideoMatches(request.Limit, func(offer func(semantic.VideoVectorMatch) error) error {
		var current semantic.VideoVectorMatch
		seen := 0
		flush := func() error {
			if seen != current.PlanSize {
				return nil
			}
			if request.After != nil && !(current.Score < request.After.Score || current.Score == request.After.Score && current.AssetID > request.After.AssetID) {
				return nil
			}
			return offer(current)
		}
		for rows.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}
			var libraryID, assetID int64
			var planSize, ordinal int
			var timestampMS int64
			var blob []byte
			if err := rows.Scan(&libraryID, &assetID, &planSize, &ordinal, &timestampMS, &blob); err != nil {
				return err
			}
			if seen > 0 && (assetID != current.AssetID || libraryID != current.LibraryID) {
				if err := flush(); err != nil {
					return err
				}
				seen = 0
			}
			vector, err := semantic.DecodeEmbedding(blob, dimension)
			if err != nil {
				return semantic.ErrInvalidVideoSemantic
			}
			var score64 float64
			for index, value := range vector {
				score64 += float64(value) * float64(query[index])
			}
			score := float32(score64)
			if math.IsNaN(float64(score)) || math.IsInf(float64(score), 0) || ordinal != seen ||
				(planSize != 4 && planSize != 10) || ordinal >= planSize {
				return semantic.ErrInvalidVideoSemantic
			}
			if seen == 0 {
				current = semantic.VideoVectorMatch{LibraryID: libraryID, AssetID: assetID, PlanSize: planSize,
					Ordinal: ordinal, TimestampMS: timestampMS, Score: score}
			} else if current.PlanSize != planSize {
				return semantic.ErrInvalidVideoSemantic
			} else if score > current.Score || score == current.Score && ordinal < current.Ordinal {
				current.Ordinal, current.TimestampMS, current.Score = ordinal, timestampMS, score
			}
			seen++
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if seen > 0 {
			return flush()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("search video semantic vectors: %w", err)
	}
	return result, nil
}

var _ semantic.VideoVectorSearchRepository = (*Store)(nil)
