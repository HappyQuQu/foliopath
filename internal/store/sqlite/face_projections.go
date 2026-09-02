package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func (s *Store) GetPersonAssetSnapshot(ctx context.Context, personID string) (face.PersonAssetSnapshot, error) {
	if len(personID) < 8 || len(personID) > 128 {
		return face.PersonAssetSnapshot{}, face.ErrInvalidFaceProjection
	}
	var value face.PersonAssetSnapshot
	err := s.db.QueryRowContext(ctx, `SELECT revision FROM people WHERE id=? AND state='active'`, personID).Scan(&value.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return value, face.ErrPersonNotFound
	}
	if err != nil {
		return value, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT library.id,library.revision,library.status FROM person_face_anchors anchor JOIN libraries library ON library.id=anchor.library_id WHERE anchor.person_id=? AND anchor.state='bound' AND anchor.current_face_id IS NOT NULL GROUP BY library.id,library.revision,library.status ORDER BY library.id`, personID)
	if err != nil {
		return value, err
	}
	defer rows.Close()
	for rows.Next() {
		var source face.PersonAssetSource
		if err := rows.Scan(&source.LibraryID, &source.Revision, &source.Status); err != nil {
			return value, err
		}
		value.Sources = append(value.Sources, source)
	}
	return value, rows.Err()
}

func (s *Store) ListPersonAssetViews(ctx context.Context, query face.PersonAssetQuery) ([]face.PersonAssetView, error) {
	if len(query.PersonID) < 8 || len(query.PersonID) > 128 || query.Limit < 1 || query.Limit > face.MaxPersonAssetPageSize+1 {
		return nil, face.ErrInvalidFaceProjection
	}
	filter := ""
	args := []any{query.PersonID}
	if query.After != nil {
		if query.After.AssetID < 1 {
			return nil, face.ErrInvalidFaceProjection
		}
		filter = ` AND (asset.mtime_ns<? OR (asset.mtime_ns=? AND asset.id<?))`
		args = append(args, query.After.MTimeNS, query.After.MTimeNS, query.After.AssetID)
	}
	args = append(args, query.Limit, query.PersonID)
	rows, err := s.db.QueryContext(ctx, `WITH candidates AS(
		SELECT asset.library_id,asset.id,asset.mtime_ns FROM person_face_anchors anchor
		JOIN assets asset ON asset.library_id=anchor.library_id AND asset.id=anchor.asset_id
		JOIN libraries library ON library.id=asset.library_id AND library.status='ready'
		WHERE anchor.person_id=? AND anchor.state='bound' AND anchor.current_face_id IS NOT NULL`+filter+`
		GROUP BY asset.library_id,asset.id,asset.mtime_ns ORDER BY asset.mtime_ns DESC,asset.id DESC LIMIT ?)
		SELECT candidate.library_id,candidate.id,candidate.mtime_ns,anchor.current_face_id
		FROM candidates candidate JOIN person_face_anchors anchor ON anchor.library_id=candidate.library_id AND anchor.asset_id=candidate.id
		WHERE anchor.person_id=? AND anchor.state='bound' AND anchor.current_face_id IS NOT NULL
		ORDER BY candidate.mtime_ns DESC,candidate.id DESC,anchor.current_face_id`, args...)
	if err != nil {
		return nil, fmt.Errorf("list person asset views: %w", err)
	}
	defer rows.Close()
	items := make([]face.PersonAssetView, 0, query.Limit)
	for rows.Next() {
		var libraryID, assetID, mtime int64
		var faceID string
		if err := rows.Scan(&libraryID, &assetID, &mtime, &faceID); err != nil {
			return nil, err
		}
		if len(items) == 0 || items[len(items)-1].AssetID != assetID {
			items = append(items, face.PersonAssetView{LibraryID: libraryID, AssetID: assetID, MTimeNS: mtime})
		}
		items[len(items)-1].FaceIDs = append(items[len(items)-1].FaceIDs, faceID)
	}
	return items, rows.Err()
}

func (s *Store) ListAssetFaceViews(ctx context.Context, assetID int64) ([]face.AssetFaceView, error) {
	if assetID < 1 {
		return nil, face.ErrInvalidFaceProjection
	}
	var libraryID int64
	var generation, state, libraryState string
	var enabled int
	err := s.db.QueryRowContext(ctx, `SELECT asset.library_id,COALESCE(settings.active_generation_id,''),COALESCE(settings.state,'disabled'),COALESCE(settings.enabled,0),library.status FROM assets asset JOIN libraries library ON library.id=asset.library_id LEFT JOIN face_library_settings settings ON settings.library_id=asset.library_id WHERE asset.id=?`, assetID).Scan(&libraryID, &generation, &state, &enabled, &libraryState)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, face.ErrInvalidFaceProjection
	}
	if err != nil {
		return nil, err
	}
	if enabled != 1 || generation == "" || libraryState != "ready" || (state != "ready" && state != "degraded") {
		return nil, face.ErrFaceNotReady
	}
	rows, err := s.db.QueryContext(ctx, `SELECT observation.id,observation.asset_id,observation.box_x_ppm,observation.box_y_ppm,observation.box_width_ppm,observation.box_height_ppm,observation.revision,
		COALESCE(anchor.person_id,''),CASE WHEN anchor.id IS NOT NULL THEN 'assigned' WHEN exclusion.id IS NOT NULL THEN 'excluded' WHEN member.role='core' THEN 'anonymous_core' WHEN member.role='edge' THEN 'anonymous_edge' ELSE 'unclustered' END
		FROM face_observations observation
		LEFT JOIN person_face_anchors anchor ON anchor.current_face_id=observation.id AND anchor.state='bound'
		LEFT JOIN face_exclusions exclusion ON exclusion.current_face_id=observation.id
		LEFT JOIN face_cluster_members member ON member.face_id=observation.id
		LEFT JOIN face_cluster_builds build ON build.id=member.build_id AND build.state='active'
		WHERE observation.generation_id=? AND observation.library_id=? AND observation.asset_id=? AND (member.build_id IS NULL OR build.id IS NOT NULL)
		ORDER BY observation.box_y_ppm,observation.box_x_ppm,observation.id`, generation, libraryID, assetID)
	if err != nil {
		return nil, fmt.Errorf("list asset face views: %w", err)
	}
	defer rows.Close()
	items := make([]face.AssetFaceView, 0)
	for rows.Next() {
		var value face.AssetFaceView
		var x, y, width, height int64
		if err := rows.Scan(&value.FaceID, &value.AssetID, &x, &y, &width, &height, &value.Revision, &value.PersonID, &value.State); err != nil {
			return nil, err
		}
		value.Ordinal = len(items) + 1
		value.Region = coarseRegion(x, y, width, height)
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) > face.MaxCandidatesPerAsset {
		return nil, face.ErrInvalidFaceProjection
	}
	return items, nil
}

func coarsePercent(ppm int64) int { return min(100, max(0, int(math.Round(float64(ppm)/10000)))) }

func coarseRegion(x, y, width, height int64) face.CoarseRegion {
	region := face.CoarseRegion{
		XPercent:      coarsePercent(x),
		YPercent:      coarsePercent(y),
		WidthPercent:  max(1, coarsePercent(width)),
		HeightPercent: max(1, coarsePercent(height)),
	}
	region.WidthPercent = min(region.WidthPercent, 100-region.XPercent)
	region.HeightPercent = min(region.HeightPercent, 100-region.YPercent)
	return region
}

func (s *Store) GetFaceClusterDetailSnapshot(ctx context.Context, libraryID int64, clusterID string) (face.FaceClusterDetailSnapshot, error) {
	if libraryID < 1 || len(clusterID) < 8 || len(clusterID) > 128 {
		return face.FaceClusterDetailSnapshot{}, face.ErrInvalidFaceProjection
	}
	var value face.FaceClusterDetailSnapshot
	value.Cluster.LibraryID = libraryID
	err := s.db.QueryRowContext(ctx, `SELECT build.id,build.generation_id,cluster.id,cluster.role,cluster.revision,COUNT(member.face_id) FROM face_cluster_builds build JOIN face_clusters cluster ON cluster.build_id=build.id JOIN face_cluster_members member ON member.build_id=cluster.build_id AND member.cluster_id=cluster.id WHERE build.library_id=? AND build.state='active' AND cluster.id=? GROUP BY build.id,cluster.id`, libraryID, clusterID).Scan(&value.BuildID, &value.GenerationID, &value.Cluster.ID, &value.Cluster.Role, &value.Cluster.Revision, &value.Cluster.MemberCount)
	if errors.Is(err, sql.ErrNoRows) {
		return value, face.ErrFaceNotReady
	}
	if err != nil {
		return value, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT observation.asset_id FROM face_cluster_members member JOIN face_observations observation ON observation.id=member.face_id WHERE member.build_id=? AND member.cluster_id=? ORDER BY observation.asset_id LIMIT 4`, value.BuildID, clusterID)
	if err != nil {
		return value, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return value, err
		}
		value.Cluster.PreviewAssetIDs = append(value.Cluster.PreviewAssetIDs, id)
	}
	return value, rows.Err()
}

func (s *Store) ListFaceClusterMemberViews(ctx context.Context, query face.FaceClusterMemberQuery) ([]face.FaceClusterMemberView, error) {
	if query.LibraryID < 1 || len(query.ClusterID) < 8 || len(query.BuildID) < 8 || query.Limit < 1 || query.Limit > face.MaxFaceClusterMemberPageSize+1 {
		return nil, face.ErrInvalidFaceProjection
	}
	filter := ""
	args := []any{query.BuildID, query.ClusterID, query.LibraryID}
	if query.After != nil {
		if (query.After.Role != "core" && query.After.Role != "edge") || len(query.After.FaceID) < 8 {
			return nil, face.ErrInvalidFaceProjection
		}
		filter = ` AND (member.role>? OR (member.role=? AND member.face_id>?))`
		args = append(args, query.After.Role, query.After.Role, query.After.FaceID)
	}
	args = append(args, query.Limit)
	rows, err := s.db.QueryContext(ctx, `SELECT member.face_id,observation.asset_id,member.role,observation.box_x_ppm,observation.box_y_ppm,observation.box_width_ppm,observation.box_height_ppm,observation.revision FROM face_cluster_members member JOIN face_cluster_builds build ON build.id=member.build_id AND build.state='active' JOIN face_observations observation ON observation.id=member.face_id WHERE member.build_id=? AND member.cluster_id=? AND build.library_id=?`+filter+` ORDER BY member.role,member.face_id LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list face cluster member views: %w", err)
	}
	defer rows.Close()
	items := make([]face.FaceClusterMemberView, 0, query.Limit)
	for rows.Next() {
		var value face.FaceClusterMemberView
		var x, y, width, height int64
		if err := rows.Scan(&value.FaceID, &value.AssetID, &value.Role, &x, &y, &width, &height, &value.Revision); err != nil {
			return nil, err
		}
		value.Region = coarseRegion(x, y, width, height)
		items = append(items, value)
	}
	return items, rows.Err()
}

var _ face.PersonAssetRepository = (*Store)(nil)
var _ face.AssetFacesRepository = (*Store)(nil)
var _ face.FaceClusterDetailRepository = (*Store)(nil)
