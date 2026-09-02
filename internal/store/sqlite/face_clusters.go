package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func (s *Store) RebuildFaceClusters(ctx context.Context, generationID string, libraryID int64, jobID string, claimedRevision int64, profile face.ClusterProfile, updatedAt time.Time) error {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 || len(jobID) < 8 || len(jobID) > 128 || claimedRevision < 1 || updatedAt.IsZero() {
		return face.ErrInvalidClusteringInput
	}
	dimension, state, err := s.faceGenerationContract(ctx, s.db, generationID)
	if err != nil {
		return err
	}
	if state != "building" && state != "ready" && state != "active" {
		return face.ErrFaceGenerationUnavailable
	}
	if err := s.requireRunnableFaceClustering(ctx, s.db, generationID, libraryID, jobID, claimedRevision); err != nil {
		return err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT observation.id,observation.vector
		FROM face_observations observation
		WHERE observation.generation_id=? AND observation.library_id=?
		AND NOT EXISTS(SELECT 1 FROM person_face_anchors anchor WHERE anchor.current_face_id=observation.id)
		AND NOT EXISTS(SELECT 1 FROM face_exclusions exclusion WHERE exclusion.current_face_id=observation.id)
		ORDER BY observation.id`, generationID, libraryID)
	if err != nil {
		return fmt.Errorf("list cluster candidates: %w", err)
	}
	faces := make([]face.VectorFace, 0)
	for rows.Next() {
		var item face.VectorFace
		var encoded []byte
		if err := rows.Scan(&item.ID, &encoded); err != nil {
			rows.Close()
			return err
		}
		item.Vector, err = face.DecodeEmbedding(encoded, dimension)
		if err != nil {
			rows.Close()
			return err
		}
		faces = append(faces, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	candidate := make(map[string]struct{}, len(faces))
	for _, item := range faces {
		candidate[item.ID] = struct{}{}
	}
	linkRows, err := s.db.QueryContext(ctx, `SELECT left_anchor.current_face_id,right_anchor.current_face_id
		FROM face_cannot_links link
		JOIN person_face_anchors left_anchor ON left_anchor.id=link.left_anchor_id
		JOIN person_face_anchors right_anchor ON right_anchor.id=link.right_anchor_id
		WHERE left_anchor.library_id=? AND right_anchor.library_id=?
		AND left_anchor.current_face_id IS NOT NULL AND right_anchor.current_face_id IS NOT NULL
		ORDER BY left_anchor.current_face_id,right_anchor.current_face_id`, libraryID, libraryID)
	if err != nil {
		return err
	}
	links := make([]face.FacePair, 0)
	for linkRows.Next() {
		var left, right string
		if err := linkRows.Scan(&left, &right); err != nil {
			linkRows.Close()
			return err
		}
		if _, ok := candidate[left]; !ok {
			continue
		}
		if _, ok := candidate[right]; !ok {
			continue
		}
		if left > right {
			left, right = right, left
		}
		links = append(links, face.FacePair{Left: left, Right: right})
	}
	if err := linkRows.Close(); err != nil {
		return err
	}
	clusters, err := face.ClusterFaces(generationID, faces, links, profile)
	if err != nil {
		return err
	}
	return s.replaceFaceClusters(ctx, face.ClusterBatch{GenerationID: generationID, LibraryID: libraryID, Clusters: clusters, UpdatedAt: updatedAt}, true, jobID, claimedRevision)
}

func (s *Store) ReplaceFaceClusters(ctx context.Context, batch face.ClusterBatch) error {
	return s.replaceFaceClusters(ctx, batch, false, "", 0)
}

func (s *Store) replaceFaceClusters(ctx context.Context, batch face.ClusterBatch, requireRunnable bool, jobID string, claimedRevision int64) error {
	if err := face.ValidateClusterBatch(batch, 1_000_000, 1_000_000); err != nil {
		return err
	}
	buildID, err := newFaceClusterBuildID()
	if err != nil {
		return err
	}
	if err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO face_cluster_builds(id,generation_id,library_id,state,created_at_ms) VALUES(?,?,?,'building',?)`, buildID, batch.GenerationID, batch.LibraryID, batch.UpdatedAt.UnixMilli())
		return err
	}); err != nil {
		return fmt.Errorf("create face cluster build: %w", err)
	}
	for start := 0; start < len(batch.Clusters); start += s.maxBatchSize {
		end := min(start+s.maxBatchSize, len(batch.Clusters))
		part := batch
		part.Clusters = batch.Clusters[start:end]
		if err := s.writeFaceClustersToBuild(ctx, buildID, part); err != nil {
			_ = s.deleteFaceClusterBuild(ctx, buildID)
			return err
		}
	}
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		err := tx.QueryRowContext(ctx, `SELECT state FROM face_cluster_builds WHERE id=? AND generation_id=? AND library_id=?`, buildID, batch.GenerationID, batch.LibraryID).Scan(&state)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrInvalidClusterRecord
		}
		if err != nil {
			return err
		}
		if state != "building" {
			return face.ErrInvalidClusterRecord
		}
		if requireRunnable {
			if err := s.requireRunnableFaceClustering(ctx, tx, batch.GenerationID, batch.LibraryID, jobID, claimedRevision); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE face_cluster_builds SET state='stale' WHERE library_id=? AND state='active'`, batch.LibraryID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE face_cluster_builds SET state='active',activated_at_ms=? WHERE id=? AND state='building'`, batch.UpdatedAt.UnixMilli(), buildID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE face_library_settings SET active_cluster_build_id=?,coverage_revision=coverage_revision+1,
			updated_at_ms=MAX(created_at_ms,?) WHERE library_id=?`, buildID, batch.UpdatedAt.UnixMilli(), batch.LibraryID)
		return err
	})
	if err != nil {
		_ = s.deleteFaceClusterBuild(ctx, buildID)
		return fmt.Errorf("activate face cluster build: %w", err)
	}
	if err := s.deleteStaleFaceClusterBuilds(ctx, batch.LibraryID, buildID); err != nil {
		return err
	}
	return nil
}

type faceClusterStateReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) requireRunnableFaceClustering(ctx context.Context, reader faceClusterStateReader, generationID string, libraryID int64, jobID string, claimedRevision int64) error {
	var enabled int
	var activeGeneration, settingsState, libraryStatus, generationState string
	err := reader.QueryRowContext(ctx, `SELECT settings.enabled,settings.active_generation_id,settings.state,library.status,generation.state
		FROM face_library_settings settings
		JOIN libraries library ON library.id=settings.library_id
		JOIN face_generations generation ON generation.id=settings.active_generation_id
		WHERE settings.library_id=?`, libraryID).Scan(&enabled, &activeGeneration, &settingsState, &libraryStatus, &generationState)
	if errors.Is(err, sql.ErrNoRows) {
		return face.ErrFaceModelUnavailable
	}
	if err != nil {
		return err
	}
	if libraryStatus == "offline" {
		return face.ErrFaceLibraryOffline
	}
	if enabled != 1 {
		return face.ErrFaceDisabled
	}
	if libraryStatus != "ready" || !faceSettingsRunnableState(settingsState) {
		return face.ErrFaceNotReady
	}
	if activeGeneration != generationID || generationState != "active" {
		return face.ErrFaceModelUnavailable
	}
	if jobID != "" {
		var jobState, jobGeneration string
		var jobLibraryID, jobClaimedRevision int64
		err := reader.QueryRowContext(ctx, `SELECT state,generation_id,library_id,claimed_revision FROM face_analysis_jobs WHERE id=?`, jobID).Scan(&jobState, &jobGeneration, &jobLibraryID, &jobClaimedRevision)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrFaceJobConflict
		}
		if err != nil {
			return err
		}
		if jobState != "running" || jobGeneration != generationID || jobLibraryID != libraryID || jobClaimedRevision != claimedRevision {
			return face.ErrFaceJobConflict
		}
	}
	return nil
}

func (s *Store) UpsertFaceClusters(ctx context.Context, batch face.ClusterBatch) error {
	if err := face.ValidateClusterBatch(batch, s.maxBatchSize, s.maxBatchSize*face.MaxCandidatesPerAsset); err != nil {
		return err
	}
	var buildID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM face_cluster_builds WHERE generation_id=? AND library_id=? AND state='active'`, batch.GenerationID, batch.LibraryID).Scan(&buildID); errors.Is(err, sql.ErrNoRows) {
		return s.ReplaceFaceClusters(ctx, batch)
	} else if err != nil {
		return err
	}
	return s.writeFaceClustersToBuild(ctx, buildID, batch)
}

func (s *Store) writeFaceClustersToBuild(ctx context.Context, buildID string, batch face.ClusterBatch) error {
	if err := face.ValidateClusterBatch(batch, s.maxBatchSize, s.maxBatchSize*face.MaxCandidatesPerAsset); err != nil {
		return err
	}
	_, state, err := s.faceGenerationContract(ctx, s.db, batch.GenerationID)
	if err != nil {
		return err
	}
	if state != "building" && state != "ready" && state != "active" {
		return face.ErrFaceGenerationUnavailable
	}
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, currentState, err := s.faceGenerationContract(ctx, tx, batch.GenerationID)
		if err != nil {
			return err
		}
		if currentState != state {
			return face.ErrFaceGenerationUnavailable
		}
		var buildState string
		err = tx.QueryRowContext(ctx, `SELECT state FROM face_cluster_builds WHERE id=? AND generation_id=? AND library_id=?`, buildID, batch.GenerationID, batch.LibraryID).Scan(&buildState)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrInvalidClusterRecord
		}
		if err != nil {
			return err
		}
		if buildState != "building" && buildState != "active" {
			return face.ErrInvalidClusterRecord
		}
		for _, cluster := range batch.Clusters {
			if _, err := tx.ExecContext(ctx, `INSERT INTO face_clusters(build_id,id,generation_id,library_id,role,revision,created_at_ms,updated_at_ms)
				VALUES(?,?,?,?,?,1,?,?) ON CONFLICT(build_id,id) DO UPDATE SET role=excluded.role,revision=face_clusters.revision+1,updated_at_ms=excluded.updated_at_ms`,
				buildID, cluster.ID, batch.GenerationID, batch.LibraryID, cluster.Role, batch.UpdatedAt.UTC().UnixMilli(), batch.UpdatedAt.UTC().UnixMilli()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM face_cluster_members WHERE build_id=? AND cluster_id=?`, buildID, cluster.ID); err != nil {
				return err
			}
			for _, member := range cluster.Members {
				result, err := tx.ExecContext(ctx, `INSERT INTO face_cluster_members(build_id,cluster_id,face_id,role,confidence_ppm)
					SELECT ?,?,id,?,? FROM face_observations WHERE id=? AND generation_id=? AND library_id=?`,
					buildID, cluster.ID, member.Role, int64(math.Round(float64(member.Confidence)*1e6)), member.FaceID, batch.GenerationID, batch.LibraryID)
				if err != nil {
					return err
				}
				rows, err := result.RowsAffected()
				if err != nil {
					return err
				}
				if rows != 1 {
					return face.ErrInvalidClusterRecord
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("write face clusters: %w", err)
	}
	return nil
}

func newFaceClusterBuildID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "facebuild_" + hex.EncodeToString(value[:]), nil
}
func (s *Store) deleteFaceClusterBuild(ctx context.Context, id string) error {
	for {
		done := false
		err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
			result, err := tx.ExecContext(ctx, `DELETE FROM face_cluster_members WHERE rowid IN(SELECT rowid FROM face_cluster_members WHERE build_id=? LIMIT ?)`, id, s.maxBatchSize)
			if err != nil {
				return err
			}
			rows, _ := result.RowsAffected()
			if rows > 0 {
				return nil
			}
			result, err = tx.ExecContext(ctx, `DELETE FROM face_clusters WHERE rowid IN(SELECT rowid FROM face_clusters WHERE build_id=? LIMIT ?)`, id, s.maxBatchSize)
			if err != nil {
				return err
			}
			rows, _ = result.RowsAffected()
			if rows > 0 {
				return nil
			}
			result, err = tx.ExecContext(ctx, `DELETE FROM face_cluster_builds WHERE id=? AND state<>'active'`, id)
			if err != nil {
				return err
			}
			rows, _ = result.RowsAffected()
			done = rows == 1
			return nil
		})
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		var exists int
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM face_cluster_builds WHERE id=? AND state<>'active')`, id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return nil
		}
	}
}
func (s *Store) deleteStaleFaceClusterBuilds(ctx context.Context, libraryID int64, keep string) error {
	for {
		var id string
		err := s.db.QueryRowContext(ctx, `SELECT id FROM face_cluster_builds WHERE library_id=? AND state='stale' AND id<>? ORDER BY created_at_ms,id LIMIT 1`, libraryID, keep).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err = s.deleteFaceClusterBuild(ctx, id); err != nil {
			return err
		}
	}
}

func (s *Store) ListFaceClusters(ctx context.Context, generationID string, libraryID int64) ([]face.Cluster, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 {
		return nil, face.ErrInvalidClusterRecord
	}
	rows, err := s.db.QueryContext(ctx, `SELECT cluster.id,cluster.role,member.face_id,member.role,member.confidence_ppm
		FROM face_clusters cluster JOIN face_cluster_builds build ON build.id=cluster.build_id AND build.state='active' JOIN face_cluster_members member ON member.build_id=cluster.build_id AND member.cluster_id=cluster.id
		WHERE cluster.generation_id=? AND cluster.library_id=? ORDER BY cluster.id,CASE member.role WHEN 'core' THEN 0 ELSE 1 END,member.face_id`, generationID, libraryID)
	if err != nil {
		return nil, fmt.Errorf("list face clusters: %w", err)
	}
	defer rows.Close()
	items := make([]face.Cluster, 0)
	for rows.Next() {
		var clusterID, clusterRole, faceID, memberRole string
		var confidence int64
		if err := rows.Scan(&clusterID, &clusterRole, &faceID, &memberRole, &confidence); err != nil {
			return nil, err
		}
		if len(items) == 0 || items[len(items)-1].ID != clusterID {
			items = append(items, face.Cluster{ID: clusterID, Role: clusterRole})
		}
		last := &items[len(items)-1]
		last.Members = append(last.Members, face.ClusterMember{FaceID: faceID, Role: memberRole, Confidence: float32(confidence) / 1e6})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

var _ face.ClusterRepository = (*Store)(nil)
