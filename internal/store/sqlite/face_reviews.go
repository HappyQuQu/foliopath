package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

type faceLineage struct {
	ID, Fingerprint     string
	LibraryID, AssetID  int64
	X, Y, Width, Height int64
	Revision            int64
}

func (s *Store) ListCoreClusterFaceIDs(ctx context.Context, clusterID string, expectedRevision int64, limit int) ([]string, error) {
	if !validResourceID(clusterID) || expectedRevision < 1 || limit < 1 || limit > face.MaxGroupReviewFaces+1 {
		return nil, face.ErrInvalidReview
	}
	rows, err := s.db.QueryContext(ctx, `SELECT member.face_id
		FROM face_clusters cluster
		JOIN face_cluster_builds build ON build.id=cluster.build_id AND build.state='active'
		JOIN face_cluster_members member ON member.build_id=cluster.build_id AND member.cluster_id=cluster.id
		WHERE cluster.id=? AND cluster.role='core' AND cluster.revision=? AND member.role='core'
		ORDER BY member.face_id LIMIT ?`, clusterID, expectedRevision, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) < 2 {
		return nil, face.ErrReviewConflict
	}
	return items, nil
}

func (s *Store) CreatePersonFromCluster(ctx context.Context, command face.CreatePersonFromClusterCommand) (face.ReviewResult, error) {
	name, nameErr := face.NormalizePersonName(command.Name)
	if nameErr != nil || !validResourceID(command.EventID) || !face.ValidReviewDigest(command.RequestHash) || !validResourceID(command.PersonID) ||
		!validResourceID(command.ClusterID) || command.ExpectedClusterRevision < 1 || command.CreatedAt.IsZero() ||
		len(command.AnchorIDs) < 2 || len(command.AnchorIDs) > s.maxBatchSize {
		return face.ReviewResult{}, face.ErrInvalidReview
	}
	for _, id := range command.AnchorIDs {
		if !validResourceID(id) {
			return face.ReviewResult{}, face.ErrInvalidReview
		}
	}
	var result face.ReviewResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if revision, replayed, err := replayFaceAudit(ctx, tx, command.EventID, command.RequestHash, "create_person", command.PersonID, command.ClusterID); err != nil {
			return err
		} else if replayed {
			result = reviewResult(command.EventID, "create_person", []string{command.PersonID}, revision, true)
			result.Replayed = true
			return nil
		}
		lineages, err := readCoreClusterLineages(ctx, tx, command.ClusterID, command.ExpectedClusterRevision)
		if err != nil {
			return err
		}
		if len(lineages) != len(command.AnchorIDs) {
			return face.ErrReviewConflict
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO people(id,name,state,revision,created_at_ms,updated_at_ms) VALUES(?,?,'active',1,?,?)`,
			command.PersonID, name, command.CreatedAt.UTC().UnixMilli(), command.CreatedAt.UTC().UnixMilli()); err != nil {
			return err
		}
		for index, lineage := range lineages {
			if err := assignLineage(ctx, tx, command.AnchorIDs[index], command.PersonID, lineage, command.CreatedAt); err != nil {
				return err
			}
		}
		if err := writeFaceAudit(ctx, tx, command.EventID, command.RequestHash, lineages[0].LibraryID, "create_person", command.PersonID, command.ClusterID, nil, 1, command.CreatedAt); err != nil {
			return err
		}
		result = reviewResult(command.EventID, "create_person", []string{command.PersonID}, 1, true)
		return nil
	})
	return result, wrapReview("create person from cluster", err)
}

func (s *Store) AssignFace(ctx context.Context, command face.AssignFaceCommand) (face.ReviewResult, error) {
	if !validAssignFace(command) {
		return face.ReviewResult{}, face.ErrInvalidReview
	}
	var result face.ReviewResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if revision, replayed, err := replayFaceAudit(ctx, tx, command.EventID, command.RequestHash, "assign_face", command.PersonID, command.FaceID); err != nil {
			return err
		} else if replayed {
			result = reviewResult(command.EventID, "assign_face", []string{command.PersonID}, revision, true)
			result.Replayed = true
			return nil
		}
		observation, err := readFaceLineage(ctx, tx, command.FaceID, command.ExpectedFaceRevision)
		if err != nil {
			return err
		}
		if err := requirePersonRevision(ctx, tx, command.PersonID, command.ExpectedPersonRevision); err != nil {
			return err
		}
		undo, err := captureFaceAssignmentUndo(ctx, tx, observation, command.AnchorID)
		if err != nil {
			return err
		}
		if err := assignLineage(ctx, tx, command.AnchorID, command.PersonID, observation, command.CreatedAt); err != nil {
			return err
		}
		if err := bumpPerson(ctx, tx, command.PersonID, command.ExpectedPersonRevision, command.CreatedAt); err != nil {
			return err
		}
		if err := writeFaceAudit(ctx, tx, command.EventID, command.RequestHash, observation.LibraryID, "assign_face", command.PersonID, command.FaceID,
			command.ExpectedPersonRevision, command.ExpectedPersonRevision+1, command.CreatedAt); err != nil {
			return err
		}
		if err := finalizeFaceAssignmentUndo(ctx, tx, &undo); err != nil {
			return err
		}
		if err := writeFaceUndoSnapshot(ctx, tx, command.EventID, faceUndoPayload{Action: "assign_face", PersonID: command.PersonID, PersonRevision: command.ExpectedPersonRevision + 1, Assignments: []faceAssignmentUndo{undo}}, command.CreatedAt); err != nil {
			return err
		}
		result = reviewResult(command.EventID, "assign_face", []string{command.PersonID}, command.ExpectedPersonRevision+1, true)
		return nil
	})
	return result, wrapReview("assign face", err)
}

func (s *Store) AssignCluster(ctx context.Context, command face.AssignClusterCommand) (face.ReviewResult, error) {
	if !validResourceID(command.EventID) || !face.ValidReviewDigest(command.RequestHash) || !validResourceID(command.ClusterID) || !validResourceID(command.PersonID) ||
		command.ExpectedClusterRevision < 1 || command.ExpectedPersonRevision < 1 || command.CreatedAt.IsZero() || len(command.AnchorIDs) < 2 || len(command.AnchorIDs) > s.maxBatchSize {
		return face.ReviewResult{}, face.ErrInvalidReview
	}
	for _, id := range command.AnchorIDs {
		if !validResourceID(id) {
			return face.ReviewResult{}, face.ErrInvalidReview
		}
	}
	var result face.ReviewResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if revision, replayed, err := replayFaceAudit(ctx, tx, command.EventID, command.RequestHash, "assign_cluster", command.PersonID, command.ClusterID); err != nil {
			return err
		} else if replayed {
			result = reviewResult(command.EventID, "assign_cluster", []string{command.PersonID}, revision, true)
			result.Replayed = true
			return nil
		}
		lineages, err := readCoreClusterLineages(ctx, tx, command.ClusterID, command.ExpectedClusterRevision)
		if err != nil {
			return err
		}
		if err := requirePersonRevision(ctx, tx, command.PersonID, command.ExpectedPersonRevision); err != nil {
			return err
		}
		if len(lineages) != len(command.AnchorIDs) {
			return face.ErrReviewConflict
		}
		undos := make([]faceAssignmentUndo, len(lineages))
		for index, lineage := range lineages {
			undos[index], err = captureFaceAssignmentUndo(ctx, tx, lineage, command.AnchorIDs[index])
			if err != nil {
				return err
			}
		}
		for index, lineage := range lineages {
			if err := assignLineage(ctx, tx, command.AnchorIDs[index], command.PersonID, lineage, command.CreatedAt); err != nil {
				return err
			}
		}
		if err := bumpPerson(ctx, tx, command.PersonID, command.ExpectedPersonRevision, command.CreatedAt); err != nil {
			return err
		}
		if err := writeFaceAudit(ctx, tx, command.EventID, command.RequestHash, lineages[0].LibraryID, "assign_cluster", command.PersonID, command.ClusterID,
			command.ExpectedPersonRevision, command.ExpectedPersonRevision+1, command.CreatedAt); err != nil {
			return err
		}
		for index := range undos {
			if err := finalizeFaceAssignmentUndo(ctx, tx, &undos[index]); err != nil {
				return err
			}
		}
		if err := writeFaceUndoSnapshot(ctx, tx, command.EventID, faceUndoPayload{Action: "assign_cluster", PersonID: command.PersonID, PersonRevision: command.ExpectedPersonRevision + 1, Assignments: undos}, command.CreatedAt); err != nil {
			return err
		}
		result = reviewResult(command.EventID, "assign_cluster", []string{command.PersonID}, command.ExpectedPersonRevision+1, true)
		return nil
	})
	return result, wrapReview("assign cluster", err)
}

func readCoreClusterLineages(ctx context.Context, tx *sql.Tx, clusterID string, expectedRevision int64) ([]faceLineage, error) {
	var role string
	var revision int64
	var buildID string
	err := tx.QueryRowContext(ctx, `SELECT cluster.build_id,cluster.role,cluster.revision FROM face_clusters cluster JOIN face_cluster_builds build ON build.id=cluster.build_id AND build.state='active' WHERE cluster.id=?`, clusterID).Scan(&buildID, &role, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, face.ErrReviewConflict
	}
	if err != nil {
		return nil, err
	}
	if role != "core" || revision != expectedRevision {
		return nil, face.ErrReviewConflict
	}
	rows, err := tx.QueryContext(ctx, `SELECT observation.id,observation.library_id,observation.asset_id,observation.source_fingerprint,
		observation.box_x_ppm,observation.box_y_ppm,observation.box_width_ppm,observation.box_height_ppm,observation.revision
		FROM face_cluster_members member JOIN face_observations observation ON observation.id=member.face_id
		WHERE member.build_id=? AND member.cluster_id=? AND member.role='core' ORDER BY observation.id`, buildID, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]faceLineage, 0)
	for rows.Next() {
		var item faceLineage
		if err := rows.Scan(&item.ID, &item.LibraryID, &item.AssetID, &item.Fingerprint, &item.X, &item.Y, &item.Width, &item.Height, &item.Revision); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) < 2 {
		return nil, face.ErrReviewConflict
	}
	return items, nil
}

func (s *Store) ExcludeFace(ctx context.Context, command face.ExcludeFaceCommand) (face.ReviewResult, error) {
	if !validResourceID(command.EventID) || !face.ValidReviewDigest(command.RequestHash) || !validResourceID(command.ExclusionID) || !validResourceID(command.FaceID) || command.ExpectedFaceRevision < 1 || command.CreatedAt.IsZero() {
		return face.ReviewResult{}, face.ErrInvalidReview
	}
	var result face.ReviewResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if revision, replayed, err := replayFaceAudit(ctx, tx, command.EventID, command.RequestHash, "exclude_face", command.FaceID, nil); err != nil {
			return err
		} else if replayed {
			result = reviewResult(command.EventID, "exclude_face", nil, revision, true)
			result.Replayed = true
			return nil
		}
		lineage, err := readFaceLineage(ctx, tx, command.FaceID, command.ExpectedFaceRevision)
		if err != nil {
			return err
		}
		undo, err := captureFaceAssignmentUndo(ctx, tx, lineage, "")
		if err != nil {
			return err
		}
		var personID sql.NullString
		var personRevision sql.NullInt64
		err = tx.QueryRowContext(ctx, `SELECT anchor.person_id,person.revision FROM person_face_anchors anchor JOIN people person ON person.id=anchor.person_id WHERE anchor.current_face_id=?`, command.FaceID).Scan(&personID, &personRevision)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if personID.Valid {
			if _, err := tx.ExecContext(ctx, `DELETE FROM person_face_anchors WHERE current_face_id=?`, command.FaceID); err != nil {
				return err
			}
			if err := bumpPerson(ctx, tx, personID.String, personRevision.Int64, command.CreatedAt); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO face_exclusions(id,library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm,current_face_id,revision,created_at_ms,updated_at_ms)
			VALUES(?,?,?,?,?,?,?,?,?,1,?,?) ON CONFLICT(library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm)
			DO UPDATE SET current_face_id=excluded.current_face_id,revision=face_exclusions.revision+1,
				updated_at_ms=MAX(face_exclusions.created_at_ms,excluded.updated_at_ms)`,
			command.ExclusionID, lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height, lineage.ID, command.CreatedAt.UTC().UnixMilli(), command.CreatedAt.UTC().UnixMilli())
		if err != nil {
			return err
		}
		if err := writeFaceAudit(ctx, tx, command.EventID, command.RequestHash, lineage.LibraryID, "exclude_face", command.FaceID, faceNullableString(personID), command.ExpectedFaceRevision, command.ExpectedFaceRevision+1, command.CreatedAt); err != nil {
			return err
		}
		var afterExclusionID string
		var afterExclusionRevision int64
		if err := tx.QueryRowContext(ctx, `SELECT id,revision FROM face_exclusions WHERE library_id=? AND asset_id=? AND source_fingerprint=? AND box_x_ppm=? AND box_y_ppm=? AND box_width_ppm=? AND box_height_ppm=?`, lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height).Scan(&afterExclusionID, &afterExclusionRevision); err != nil {
			return err
		}
		payload := faceUndoPayload{Action: "exclude_face", FaceID: command.FaceID, AfterExclusionID: afterExclusionID, AfterExclusionRevision: afterExclusionRevision, Assignments: []faceAssignmentUndo{undo}}
		if personID.Valid {
			payload.PersonID = personID.String
			payload.PersonRevision = personRevision.Int64 + 1
		}
		if err := writeFaceUndoSnapshot(ctx, tx, command.EventID, payload, command.CreatedAt); err != nil {
			return err
		}
		affected := []string{}
		if personID.Valid {
			affected = []string{personID.String}
		}
		result = reviewResult(command.EventID, "exclude_face", affected, command.ExpectedFaceRevision+1, true)
		return nil
	})
	return result, wrapReview("exclude face", err)
}

func (s *Store) CannotLinkFaces(ctx context.Context, command face.CannotLinkCommand) (face.ReviewResult, error) {
	if !validResourceID(command.EventID) || !face.ValidReviewDigest(command.RequestHash) || !validResourceID(command.LeftFaceID) || !validResourceID(command.RightFaceID) || command.LeftFaceID == command.RightFaceID ||
		command.ExpectedLeftRevision < 1 || command.ExpectedRightRevision < 1 || command.CreatedAt.IsZero() {
		return face.ReviewResult{}, face.ErrInvalidReview
	}
	var result face.ReviewResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if revision, replayed, err := replayFaceAudit(ctx, tx, command.EventID, command.RequestHash, "cannot_link", command.LeftFaceID, command.RightFaceID); err != nil {
			return err
		} else if replayed {
			result = reviewResult(command.EventID, "cannot_link", nil, revision, true)
			result.Replayed = true
			return nil
		}
		leftLineage, err := readFaceLineage(ctx, tx, command.LeftFaceID, command.ExpectedLeftRevision)
		if err != nil {
			return err
		}
		rightLineage, err := readFaceLineage(ctx, tx, command.RightFaceID, command.ExpectedRightRevision)
		if err != nil {
			return err
		}
		var leftAnchor, rightAnchor string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM person_face_anchors WHERE current_face_id=? AND state='bound'`, command.LeftFaceID).Scan(&leftAnchor); err != nil {
			return face.ErrReviewConflict
		}
		if err := tx.QueryRowContext(ctx, `SELECT id FROM person_face_anchors WHERE current_face_id=? AND state='bound'`, command.RightFaceID).Scan(&rightAnchor); err != nil {
			return face.ErrReviewConflict
		}
		if leftAnchor > rightAnchor {
			leftAnchor, rightAnchor = rightAnchor, leftAnchor
		}
		inserted, err := tx.ExecContext(ctx, `INSERT INTO face_cannot_links(left_anchor_id,right_anchor_id,revision,created_at_ms) VALUES(?,?,1,?) ON CONFLICT(left_anchor_id,right_anchor_id) DO NOTHING`, leftAnchor, rightAnchor, command.CreatedAt.UTC().UnixMilli())
		if err != nil {
			return err
		}
		if rows, _ := inserted.RowsAffected(); rows != 1 {
			return face.ErrReviewConflict
		}
		var auditLibrary any
		if leftLineage.LibraryID == rightLineage.LibraryID {
			auditLibrary = leftLineage.LibraryID
		}
		if err := writeFaceAudit(ctx, tx, command.EventID, command.RequestHash, auditLibrary, "cannot_link", command.LeftFaceID, command.RightFaceID, nil, 1, command.CreatedAt); err != nil {
			return err
		}
		if err := writeFaceUndoSnapshot(ctx, tx, command.EventID, faceUndoPayload{Action: "cannot_link", LeftAnchorID: leftAnchor, RightAnchorID: rightAnchor, LinkRevision: 1}, command.CreatedAt); err != nil {
			return err
		}
		result = reviewResult(command.EventID, "cannot_link", nil, 1, true)
		return nil
	})
	return result, wrapReview("cannot link faces", err)
}

func (s *Store) MergePeople(ctx context.Context, command face.MergePeopleCommand) (face.ReviewResult, error) {
	if !validResourceID(command.EventID) || !face.ValidReviewDigest(command.RequestHash) || !validResourceID(command.SourcePersonID) || !validResourceID(command.TargetPersonID) || command.SourcePersonID == command.TargetPersonID ||
		command.ExpectedSourceRevision < 1 || command.ExpectedTargetRevision < 1 || !command.ConflictsAcknowledged || command.CreatedAt.IsZero() {
		return face.ReviewResult{}, face.ErrInvalidReview
	}
	var result face.ReviewResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if revision, replayed, err := replayFaceAudit(ctx, tx, command.EventID, command.RequestHash, "merge_people", command.TargetPersonID, command.SourcePersonID); err != nil {
			return err
		} else if replayed {
			result = reviewResult(command.EventID, "merge_people", []string{command.SourcePersonID, command.TargetPersonID}, revision, true)
			result.Replayed = true
			return nil
		}
		if err := requirePersonRevision(ctx, tx, command.SourcePersonID, command.ExpectedSourceRevision); err != nil {
			return err
		}
		if err := requirePersonRevision(ctx, tx, command.TargetPersonID, command.ExpectedTargetRevision); err != nil {
			return err
		}
		anchorRows, err := tx.QueryContext(ctx, `SELECT id,revision FROM person_face_anchors WHERE person_id=? ORDER BY id LIMIT ?`, command.SourcePersonID, s.maxBatchSize+1)
		if err != nil {
			return err
		}
		mergeAnchors := make([]faceMergeAnchorUndo, 0)
		for anchorRows.Next() {
			var item faceMergeAnchorUndo
			if err := anchorRows.Scan(&item.ID, &item.Revision); err != nil {
				anchorRows.Close()
				return err
			}
			mergeAnchors = append(mergeAnchors, item)
		}
		if err := anchorRows.Close(); err != nil {
			return err
		}
		if len(mergeAnchors) > s.maxBatchSize {
			return face.ErrReviewConflict
		}
		var conflicts int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM face_cannot_links link JOIN person_face_anchors left_anchor ON left_anchor.id=link.left_anchor_id JOIN person_face_anchors right_anchor ON right_anchor.id=link.right_anchor_id WHERE (left_anchor.person_id=? AND right_anchor.person_id=?) OR (left_anchor.person_id=? AND right_anchor.person_id=?)`, command.SourcePersonID, command.TargetPersonID, command.TargetPersonID, command.SourcePersonID).Scan(&conflicts); err != nil {
			return err
		}
		if conflicts > 0 {
			return face.ErrReviewConflict
		}
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM person_face_anchors source JOIN person_face_anchors target ON target.person_id=? AND target.library_id=source.library_id AND target.asset_id=source.asset_id AND target.source_fingerprint=source.source_fingerprint AND target.box_x_ppm=source.box_x_ppm AND target.box_y_ppm=source.box_y_ppm AND target.box_width_ppm=source.box_width_ppm AND target.box_height_ppm=source.box_height_ppm WHERE source.person_id=?`, command.TargetPersonID, command.SourcePersonID).Scan(&conflicts); err != nil {
			return err
		}
		if conflicts > 0 {
			return face.ErrReviewConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE person_face_anchors SET person_id=?,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE person_id=?`, command.TargetPersonID, command.CreatedAt.UTC().UnixMilli(), command.SourcePersonID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE people SET state='tombstoned',revision=revision+1,
			updated_at_ms=MAX(created_at_ms,?),tombstoned_at_ms=MAX(created_at_ms,?) WHERE id=? AND revision=?`, command.CreatedAt.UTC().UnixMilli(), command.CreatedAt.UTC().UnixMilli(), command.SourcePersonID, command.ExpectedSourceRevision); err != nil {
			return err
		}
		if err := bumpPerson(ctx, tx, command.TargetPersonID, command.ExpectedTargetRevision, command.CreatedAt); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO person_aliases(source_person_id,target_person_id,created_at_ms) VALUES(?,?,?)`, command.SourcePersonID, command.TargetPersonID, command.CreatedAt.UTC().UnixMilli()); err != nil {
			return err
		}
		if err := writeFaceAudit(ctx, tx, command.EventID, command.RequestHash, nil, "merge_people", command.TargetPersonID, command.SourcePersonID, command.ExpectedTargetRevision, command.ExpectedTargetRevision+1, command.CreatedAt); err != nil {
			return err
		}
		for index := range mergeAnchors {
			mergeAnchors[index].Revision++
		}
		if err := writeFaceUndoSnapshot(ctx, tx, command.EventID, faceUndoPayload{Action: "merge_people", SourcePersonID: command.SourcePersonID, TargetPersonID: command.TargetPersonID, SourcePersonRevision: command.ExpectedSourceRevision + 1, TargetPersonRevision: command.ExpectedTargetRevision + 1, MergeAnchors: mergeAnchors}, command.CreatedAt); err != nil {
			return err
		}
		result = reviewResult(command.EventID, "merge_people", []string{command.SourcePersonID, command.TargetPersonID}, command.ExpectedTargetRevision+1, true)
		return nil
	})
	return result, wrapReview("merge people", err)
}

func (s *Store) SplitFace(ctx context.Context, command face.SplitFaceCommand) (face.ReviewResult, error) {
	if !validResourceID(command.EventID) || !face.ValidReviewDigest(command.RequestHash) || !validResourceID(command.FaceID) || !validResourceID(command.SourcePersonID) || command.ExpectedFaceRevision < 1 || command.ExpectedSourceRevision < 1 || command.CreatedAt.IsZero() {
		return face.ReviewResult{}, face.ErrInvalidReview
	}
	var result face.ReviewResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if revision, replayed, err := replayFaceAudit(ctx, tx, command.EventID, command.RequestHash, "split_face", command.SourcePersonID, command.FaceID); err != nil {
			return err
		} else if replayed {
			result = reviewResult(command.EventID, "split_face", []string{command.SourcePersonID}, revision, true)
			result.Replayed = true
			return nil
		}
		lineage, err := readFaceLineage(ctx, tx, command.FaceID, command.ExpectedFaceRevision)
		if err != nil {
			return err
		}
		if err := requirePersonRevision(ctx, tx, command.SourcePersonID, command.ExpectedSourceRevision); err != nil {
			return err
		}
		updated, err := tx.ExecContext(ctx, `UPDATE person_face_anchors SET current_face_id=NULL,state='needs_review',revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE person_id=? AND current_face_id=? AND state='bound'`, command.CreatedAt.UTC().UnixMilli(), command.SourcePersonID, command.FaceID)
		if err != nil {
			return err
		}
		if rows, _ := updated.RowsAffected(); rows != 1 {
			return face.ErrReviewConflict
		}
		if err := bumpPerson(ctx, tx, command.SourcePersonID, command.ExpectedSourceRevision, command.CreatedAt); err != nil {
			return err
		}
		if err := writeFaceAudit(ctx, tx, command.EventID, command.RequestHash, lineage.LibraryID, "split_face", command.SourcePersonID, command.FaceID, command.ExpectedSourceRevision, command.ExpectedSourceRevision+1, command.CreatedAt); err != nil {
			return err
		}
		var anchorID string
		var anchorRevision int64
		if err := tx.QueryRowContext(ctx, `SELECT id,revision FROM person_face_anchors WHERE person_id=? AND library_id=? AND asset_id=? AND source_fingerprint=? AND box_x_ppm=? AND box_y_ppm=? AND box_width_ppm=? AND box_height_ppm=?`, command.SourcePersonID, lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height).Scan(&anchorID, &anchorRevision); err != nil {
			return err
		}
		if err := writeFaceUndoSnapshot(ctx, tx, command.EventID, faceUndoPayload{Action: "split_face", PersonID: command.SourcePersonID, FaceID: command.FaceID, AnchorID: anchorID, PersonRevision: command.ExpectedSourceRevision + 1, AnchorRevision: anchorRevision}, command.CreatedAt); err != nil {
			return err
		}
		result = reviewResult(command.EventID, "split_face", []string{command.SourcePersonID}, command.ExpectedSourceRevision+1, true)
		return nil
	})
	return result, wrapReview("split face", err)
}

func readFaceLineage(ctx context.Context, tx *sql.Tx, id string, expected int64) (faceLineage, error) {
	var value faceLineage
	err := tx.QueryRowContext(ctx, `SELECT id,library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm,revision FROM face_observations WHERE id=? AND revision=?`, id, expected).Scan(&value.ID, &value.LibraryID, &value.AssetID, &value.Fingerprint, &value.X, &value.Y, &value.Width, &value.Height, &value.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return faceLineage{}, face.ErrReviewConflict
	}
	return value, err
}

func assignLineage(ctx context.Context, tx *sql.Tx, anchorID, personID string, lineage faceLineage, at time.Time) error {
	var existingPerson string
	err := tx.QueryRowContext(ctx, `SELECT person_id FROM person_face_anchors WHERE current_face_id=?`, lineage.ID).Scan(&existingPerson)
	if err == nil {
		if existingPerson == personID {
			return nil
		}
		return face.ErrReviewConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var existingAnchor string
	err = tx.QueryRowContext(ctx, `SELECT id,person_id FROM person_face_anchors
		WHERE library_id=? AND asset_id=? AND source_fingerprint=? AND box_x_ppm=? AND box_y_ppm=? AND box_width_ppm=? AND box_height_ppm=?`,
		lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height).Scan(&existingAnchor, &existingPerson)
	if err == nil {
		if existingPerson != personID {
			return face.ErrReviewConflict
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM face_exclusions WHERE library_id=? AND asset_id=? AND source_fingerprint=? AND box_x_ppm=? AND box_y_ppm=? AND box_width_ppm=? AND box_height_ppm=?`, lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE person_face_anchors SET current_face_id=?,state='bound',revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, lineage.ID, at.UTC().UnixMilli(), existingAnchor)
		return err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM face_exclusions WHERE library_id=? AND asset_id=? AND source_fingerprint=? AND box_x_ppm=? AND box_y_ppm=? AND box_width_ppm=? AND box_height_ppm=?`, lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO person_face_anchors(id,person_id,library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm,current_face_id,state,revision,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,?,?,?,?,?,'bound',1,?,?)`, anchorID, personID, lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height, lineage.ID, at.UTC().UnixMilli(), at.UTC().UnixMilli())
	return err
}

func requirePersonRevision(ctx context.Context, tx *sql.Tx, id string, revision int64) error {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM people WHERE id=? AND state='active' AND revision=?`, id, revision).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return face.ErrReviewConflict
	}
	return nil
}
func bumpPerson(ctx context.Context, tx *sql.Tx, id string, revision int64, at time.Time) error {
	result, err := tx.ExecContext(ctx, `UPDATE people SET revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='active' AND revision=?`, at.UTC().UnixMilli(), id, revision)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return face.ErrReviewConflict
	}
	return nil
}
func writeFaceAudit(ctx context.Context, tx *sql.Tx, id, requestHash string, libraryID any, action, primary string, secondary any, before any, after int64, at time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO face_audit_events(id,request_hash,library_id,action,primary_target_id,secondary_target_id,before_revision,after_revision,created_at_ms) VALUES(?,?,?,?,?,?,?,?,?)`, id, requestHash, libraryID, action, primary, secondary, before, after, at.UTC().UnixMilli())
	return err
}
func replayFaceAudit(ctx context.Context, tx *sql.Tx, id, requestHash, action, primary string, secondary any) (int64, bool, error) {
	var storedHash, storedAction, storedPrimary string
	var storedSecondary sql.NullString
	var revision int64
	err := tx.QueryRowContext(ctx, `SELECT request_hash,action,primary_target_id,secondary_target_id,after_revision FROM face_audit_events WHERE id=?`, id).Scan(&storedHash, &storedAction, &storedPrimary, &storedSecondary, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if storedHash != requestHash || storedAction != action || storedPrimary != primary {
		return 0, false, face.ErrReviewConflict
	}
	return revision, true, nil
}
func reviewResult(id, action string, people []string, revision int64, undoable bool) face.ReviewResult {
	sort.Strings(people)
	return face.ReviewResult{EventID: id, Action: action, AffectedPersonIDs: people, Revision: revision, Undoable: undoable}
}
func faceNullableString(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}
func validResourceID(value string) bool { return len(value) >= 8 && len(value) <= 128 }
func validAssignFace(c face.AssignFaceCommand) bool {
	return validResourceID(c.EventID) && face.ValidReviewDigest(c.RequestHash) && validResourceID(c.AnchorID) && validResourceID(c.FaceID) && validResourceID(c.PersonID) && c.ExpectedFaceRevision > 0 && c.ExpectedPersonRevision > 0 && !c.CreatedAt.IsZero()
}
func wrapReview(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, face.ErrReviewConflict) || errors.Is(err, face.ErrInvalidReview) {
		return err
	}
	return fmt.Errorf("%s: %w", operation, err)
}

var _ face.ReviewRepository = (*Store)(nil)
