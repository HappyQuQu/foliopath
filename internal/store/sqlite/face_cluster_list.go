package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/face"
)

// ReadFaceClusterPage streams one consistent SQLite snapshot. Counts cover the
// selected kind before pagination/review filtering; retained memory is bounded by
// one group and limit+1 cards, never the number of observations in the library.
func (s *Store) ReadFaceClusterPage(ctx context.Context, query face.FaceClusterListQuery) (face.FaceClusterReadPage, error) {
	var page face.FaceClusterReadPage
	if query.LibraryID < 1 || query.Limit < 1 || query.Limit > face.MaxFaceClusterPageSize+1 || (query.Role != "" && query.Role != "core" && query.Role != "edge") || (query.Review != "all" && query.Review != "pending") {
		return page, face.ErrInvalidClusterRecord
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return page, err
	}
	defer tx.Rollback()
	value := &page.Snapshot
	value.LibraryID = query.LibraryID
	var state, buildID string
	var libraryRevision, settingsRevision int64
	var published bool
	err = tx.QueryRowContext(ctx, `SELECT settings.state,COALESCE(build.generation_id,settings.active_generation_id,''),settings.coverage_revision,COALESCE(build.id,''),build.id IS NOT NULL,library.revision,settings.revision FROM face_library_settings settings JOIN libraries library ON library.id=settings.library_id AND library.status='ready' LEFT JOIN face_cluster_builds build ON build.id=settings.active_cluster_build_id AND build.state='active' WHERE settings.library_id=? AND settings.enabled=1`, query.LibraryID).Scan(&state, &value.GenerationID, &value.Revision, &buildID, &published, &libraryRevision, &settingsRevision)
	if errors.Is(err, sql.ErrNoRows) {
		return page, face.ErrFaceNotReady
	}
	if err != nil {
		return page, err
	}
	if state != "ready" && state != "degraded" && !(published && (state == "building" || state == "awaiting_model")) {
		return page, face.ErrFaceNotReady
	}
	err = tx.QueryRowContext(ctx, `SELECT eligible_count,completed_count,failed_count,stale_count,revision FROM face_library_progress WHERE generation_id=? AND library_id=?`, value.GenerationID, query.LibraryID).Scan(&value.Coverage.Eligible, &value.Coverage.Completed, &value.Coverage.Failed, &value.Coverage.Stale, &value.Coverage.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		value.Coverage.Revision = 1
	} else if err != nil {
		return page, err
	}
	value.GroupAssignmentAllowed = face.GroupAssignmentAllowed()
	digest := sha256.New()
	fmt.Fprintf(digest, "%q|%q|%d|%d|%d|%q|%+v\n", value.GenerationID, buildID, value.Revision, libraryRevision, settingsRevision, state, value.Coverage)
	args := []any{buildID, query.LibraryID}
	filter := ""
	if query.Role != "" {
		filter = " AND cluster.role=?"
		args = append(args, query.Role)
	}
	rows, err := tx.QueryContext(ctx, `SELECT cluster.id,cluster.role,cluster.revision,observation.id,observation.revision,observation.asset_id,COALESCE(anchor.id,''),COALESCE(anchor.person_id,''),COALESCE(exclusion.id,'')
 FROM face_clusters cluster JOIN face_cluster_members member ON member.build_id=cluster.build_id AND member.cluster_id=cluster.id
 JOIN face_observations observation ON observation.id=member.face_id`+faceCurrentLineageJoins+`
 WHERE cluster.build_id=? AND cluster.library_id=?`+filter+` ORDER BY cluster.role,cluster.id,observation.asset_id,observation.id`, args...)
	if err != nil {
		return page, fmt.Errorf("read face cluster page: %w", err)
	}
	defer rows.Close()
	page.Items = make([]face.FaceClusterView, 0, query.Limit)
	var current face.FaceClusterView
	flush := func() {
		if current.ID == "" {
			return
		}
		page.Counts.TotalGroups++
		page.Counts.TotalFaces += current.MemberCount
		page.Counts.PendingFaces += current.PendingMemberCount
		if current.PendingMemberCount > 0 {
			page.Counts.PendingGroups++
		}
		if len(page.Items) >= query.Limit || (query.Review == "pending" && current.PendingMemberCount == 0) {
			return
		}
		if query.After != nil && (current.Role < query.After.Role || (current.Role == query.After.Role && current.ID <= query.After.ID)) {
			return
		}
		page.Items = append(page.Items, current)
	}
	for rows.Next() {
		var id, role, faceID, anchorID, personID, exclusionID string
		var revision, faceRevision, assetID int64
		if err := rows.Scan(&id, &role, &revision, &faceID, &faceRevision, &assetID, &anchorID, &personID, &exclusionID); err != nil {
			return page, err
		}
		fmt.Fprintf(digest, "%q|%q|%d|%q|%d|%d|%q|%q|%q\n", id, role, revision, faceID, faceRevision, assetID, anchorID, personID, exclusionID)
		if id != current.ID {
			flush()
			current = face.FaceClusterView{ID: id, LibraryID: query.LibraryID, Role: role, Revision: revision, PreviewAssetIDs: []int64{}}
		}
		current.MemberCount++
		if anchorID == "" && exclusionID == "" {
			current.PendingMemberCount++
		}
		if len(current.PreviewAssetIDs) < 4 && (len(current.PreviewAssetIDs) == 0 || current.PreviewAssetIDs[len(current.PreviewAssetIDs)-1] != assetID) {
			current.PreviewAssetIDs = append(current.PreviewAssetIDs, assetID)
		}
	}
	if err := rows.Err(); err != nil {
		return page, err
	}
	flush()
	if err := rows.Close(); err != nil {
		return page, err
	}
	value.Digest = hex.EncodeToString(digest.Sum(nil))
	return page, tx.Commit()
}

var _ face.FaceClusterListRepository = (*Store)(nil)
