package face

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidClusterRecord = errors.New("invalid face cluster record")

type ClusterBatch struct {
	GenerationID string
	LibraryID    int64
	Clusters     []Cluster
	UpdatedAt    time.Time
}

type ClusterRepository interface {
	ReplaceFaceClusters(context.Context, ClusterBatch) error
	UpsertFaceClusters(context.Context, ClusterBatch) error
	ListFaceClusters(context.Context, string, int64) ([]Cluster, error)
}

func ValidateClusterBatch(batch ClusterBatch, maximumClusters, maximumMembers int) error {
	if len(batch.GenerationID) < 8 || len(batch.GenerationID) > 128 || batch.LibraryID < 1 ||
		batch.UpdatedAt.IsZero() || maximumClusters < 1 || maximumMembers < 1 || len(batch.Clusters) > maximumClusters {
		return ErrInvalidClusterRecord
	}
	clusterIDs := make(map[string]struct{}, len(batch.Clusters))
	faceIDs := make(map[string]struct{})
	memberCount := 0
	for _, cluster := range batch.Clusters {
		if len(cluster.ID) < 8 || len(cluster.ID) > 128 || (cluster.Role != "core" && cluster.Role != "edge") ||
			len(cluster.Members) < 1 {
			return ErrInvalidClusterRecord
		}
		if _, exists := clusterIDs[cluster.ID]; exists {
			return ErrInvalidClusterRecord
		}
		clusterIDs[cluster.ID] = struct{}{}
		coreCount := 0
		for _, member := range cluster.Members {
			memberCount++
			if memberCount > maximumMembers || len(member.FaceID) < 8 || len(member.FaceID) > 128 ||
				(member.Role != "core" && member.Role != "edge") || member.Confidence < 0 || member.Confidence > 1 {
				return ErrInvalidClusterRecord
			}
			if member.Role == "core" {
				coreCount++
			}
			if _, exists := faceIDs[member.FaceID]; exists {
				return ErrInvalidClusterRecord
			}
			faceIDs[member.FaceID] = struct{}{}
		}
		if cluster.Role == "core" && coreCount < 2 || cluster.Role == "edge" && (coreCount != 0 || len(cluster.Members) != 1) {
			return ErrInvalidClusterRecord
		}
	}
	return nil
}
