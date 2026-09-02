package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func (s *Store) GetFaceClusterSnapshot(ctx context.Context, libraryID int64) (face.FaceClusterSnapshot, error) {
	if libraryID < 1 {
		return face.FaceClusterSnapshot{}, face.ErrInvalidClusterRecord
	}
	var value face.FaceClusterSnapshot
	value.LibraryID = libraryID
	var state string
	err := s.db.QueryRowContext(ctx, `SELECT settings.state,settings.active_generation_id,settings.coverage_revision FROM face_library_settings settings JOIN libraries library ON library.id=settings.library_id AND library.status='ready' WHERE settings.library_id=? AND settings.enabled=1`, libraryID).Scan(&state, &value.GenerationID, &value.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return face.FaceClusterSnapshot{}, face.ErrFaceNotReady
	}
	if err != nil {
		return face.FaceClusterSnapshot{}, err
	}
	if state != "ready" && state != "degraded" {
		return face.FaceClusterSnapshot{}, face.ErrFaceNotReady
	}
	err = s.db.QueryRowContext(ctx, `SELECT eligible_count,completed_count,failed_count,stale_count,revision FROM face_library_progress WHERE generation_id=? AND library_id=?`, value.GenerationID, libraryID).Scan(&value.Coverage.Eligible, &value.Coverage.Completed, &value.Coverage.Failed, &value.Coverage.Stale, &value.Coverage.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		value.Coverage.Revision = 1
	} else if err != nil {
		return face.FaceClusterSnapshot{}, err
	}
	value.GroupAssignmentAllowed = false
	return value, nil
}
func (s *Store) ListFaceClusterViews(ctx context.Context, query face.FaceClusterListQuery) ([]face.FaceClusterView, error) {
	if query.LibraryID < 1 || len(query.GenerationID) < 8 || query.Limit < 1 || query.Limit > face.MaxFaceClusterPageSize+1 {
		return nil, face.ErrInvalidClusterRecord
	}
	args := []any{query.LibraryID, query.GenerationID}
	filter := ""
	if query.Role != "" {
		filter += ` AND cluster.role=?`
		args = append(args, query.Role)
	}
	if query.After != nil {
		filter += ` AND (cluster.role>? OR (cluster.role=? AND cluster.id>?))`
		args = append(args, query.After.Role, query.After.Role, query.After.ID)
	}
	args = append(args, query.Limit)
	rows, err := s.db.QueryContext(ctx, `SELECT cluster.id,cluster.library_id,cluster.role,COUNT(member.face_id),cluster.revision FROM face_clusters cluster JOIN face_cluster_builds build ON build.id=cluster.build_id AND build.state='active' JOIN face_cluster_members member ON member.build_id=cluster.build_id AND member.cluster_id=cluster.id WHERE cluster.library_id=? AND cluster.generation_id=?`+filter+` GROUP BY cluster.build_id,cluster.id ORDER BY cluster.role,cluster.id LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list face cluster views: %w", err)
	}
	defer rows.Close()
	items := make([]face.FaceClusterView, 0, query.Limit)
	for rows.Next() {
		var item face.FaceClusterView
		if err := rows.Scan(&item.ID, &item.LibraryID, &item.Role, &item.MemberCount, &item.Revision); err != nil {
			return nil, err
		}
		previewRows, err := s.db.QueryContext(ctx, `SELECT DISTINCT observation.asset_id FROM face_cluster_members member JOIN face_cluster_builds build ON build.id=member.build_id AND build.state='active' JOIN face_observations observation ON observation.id=member.face_id WHERE member.cluster_id=? ORDER BY observation.asset_id LIMIT 4`, item.ID)
		if err != nil {
			return nil, err
		}
		for previewRows.Next() {
			var id int64
			if err := previewRows.Scan(&id); err != nil {
				previewRows.Close()
				return nil, err
			}
			item.PreviewAssetIDs = append(item.PreviewAssetIDs, id)
		}
		if err := previewRows.Close(); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

var _ face.FaceClusterListRepository = (*Store)(nil)
