package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

type faceUndoPayload struct {
	Action                 string                `json:"action"`
	PersonID               string                `json:"personId,omitempty"`
	FaceID                 string                `json:"faceId,omitempty"`
	AnchorID               string                `json:"anchorId,omitempty"`
	LeftAnchorID           string                `json:"leftAnchorId,omitempty"`
	RightAnchorID          string                `json:"rightAnchorId,omitempty"`
	PersonRevision         int64                 `json:"personRevision,omitempty"`
	AnchorRevision         int64                 `json:"anchorRevision,omitempty"`
	LinkRevision           int64                 `json:"linkRevision,omitempty"`
	Assignments            []faceAssignmentUndo  `json:"assignments,omitempty"`
	AfterExclusionID       string                `json:"afterExclusionId,omitempty"`
	AfterExclusionRevision int64                 `json:"afterExclusionRevision,omitempty"`
	SourcePersonID         string                `json:"sourcePersonId,omitempty"`
	TargetPersonID         string                `json:"targetPersonId,omitempty"`
	SourcePersonRevision   int64                 `json:"sourcePersonRevision,omitempty"`
	TargetPersonRevision   int64                 `json:"targetPersonRevision,omitempty"`
	MergeAnchors           []faceMergeAnchorUndo `json:"mergeAnchors,omitempty"`
}

type faceMergeAnchorUndo struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
}

type faceAssignmentUndo struct {
	AnchorID              string             `json:"anchorId"`
	AnchorExisted         bool               `json:"anchorExisted"`
	BeforeCurrentFace     string             `json:"beforeCurrentFace,omitempty"`
	AfterCurrentFace      string             `json:"afterCurrentFace"`
	BeforeState           string             `json:"beforeState,omitempty"`
	BeforeAnchorRevision  int64              `json:"beforeAnchorRevision,omitempty"`
	AfterAnchorRevision   int64              `json:"afterAnchorRevision"`
	BeforeAnchorCreatedAt int64              `json:"beforeAnchorCreatedAt,omitempty"`
	Exclusion             *faceExclusionUndo `json:"exclusion,omitempty"`
	LibraryID, AssetID    int64
	Fingerprint           string
	X, Y, Width, Height   int64
}
type faceExclusionUndo struct {
	ID, CurrentFace string
	Revision        int64
	CreatedAt       int64
}

func captureFaceAssignmentUndo(ctx context.Context, tx *sql.Tx, lineage faceLineage, proposedAnchor string) (faceAssignmentUndo, error) {
	value := faceAssignmentUndo{AnchorID: proposedAnchor, AfterCurrentFace: lineage.ID, LibraryID: lineage.LibraryID, AssetID: lineage.AssetID, Fingerprint: lineage.Fingerprint, X: lineage.X, Y: lineage.Y, Width: lineage.Width, Height: lineage.Height}
	var current sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT id,current_face_id,state,revision,created_at_ms FROM person_face_anchors WHERE library_id=? AND asset_id=? AND source_fingerprint=? AND box_x_ppm=? AND box_y_ppm=? AND box_width_ppm=? AND box_height_ppm=?`, lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height).Scan(&value.AnchorID, &current, &value.BeforeState, &value.BeforeAnchorRevision, &value.BeforeAnchorCreatedAt)
	if err == nil {
		value.AnchorExisted = true
		if current.Valid {
			value.BeforeCurrentFace = current.String
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return value, err
	}
	var exclusion faceExclusionUndo
	var exclusionFace sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT id,current_face_id,revision,created_at_ms FROM face_exclusions WHERE library_id=? AND asset_id=? AND source_fingerprint=? AND box_x_ppm=? AND box_y_ppm=? AND box_width_ppm=? AND box_height_ppm=?`, lineage.LibraryID, lineage.AssetID, lineage.Fingerprint, lineage.X, lineage.Y, lineage.Width, lineage.Height).Scan(&exclusion.ID, &exclusionFace, &exclusion.Revision, &exclusion.CreatedAt)
	if err == nil {
		if exclusionFace.Valid {
			exclusion.CurrentFace = exclusionFace.String
		}
		value.Exclusion = &exclusion
	} else if !errors.Is(err, sql.ErrNoRows) {
		return value, err
	}
	return value, nil
}

func finalizeFaceAssignmentUndo(ctx context.Context, tx *sql.Tx, value *faceAssignmentUndo) error {
	return tx.QueryRowContext(ctx, `SELECT revision FROM person_face_anchors WHERE id=?`, value.AnchorID).Scan(&value.AfterAnchorRevision)
}

func writeFaceUndoSnapshot(ctx context.Context, tx *sql.Tx, eventID string, payload faceUndoPayload, at time.Time) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO face_review_undo_snapshots(event_id,before_json,after_json,state,revision,created_at_ms) VALUES(?,?,?,'available',1,?)`, eventID, string(encoded), string(encoded), at.UnixMilli())
	return err
}

func requireFaceReviewState(err error, matches bool) error {
	if errors.Is(err, sql.ErrNoRows) {
		return face.ErrReviewConflict
	}
	if err != nil {
		return err
	}
	if !matches {
		return face.ErrReviewConflict
	}
	return nil
}

func (s *Store) UndoFaceReview(ctx context.Context, command face.UndoReviewCommand) (face.ReviewResult, error) {
	if !validResourceID(command.EventID) || !face.ValidReviewDigest(command.RequestHash) || !validResourceID(command.ReviewID) || command.ExpectedRevision < 1 || command.CreatedAt.IsZero() {
		return face.ReviewResult{}, face.ErrInvalidReview
	}
	var result face.ReviewResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if revision, replayed, err := replayFaceAudit(ctx, tx, command.EventID, command.RequestHash, "undo", command.ReviewID, nil); err != nil {
			return err
		} else if replayed {
			result = reviewResult(command.EventID, "undo", nil, revision, false)
			result.Replayed = true
			return nil
		}
		var action, raw, state string
		var libraryID sql.NullInt64
		var snapshotRevision, auditRevision int64
		err := tx.QueryRowContext(ctx, `SELECT audit.action,audit.library_id,audit.after_revision,snapshot.after_json,snapshot.state,snapshot.revision FROM face_audit_events audit JOIN face_review_undo_snapshots snapshot ON snapshot.event_id=audit.id WHERE audit.id=?`, command.ReviewID).Scan(&action, &libraryID, &auditRevision, &raw, &state, &snapshotRevision)
		if errors.Is(err, sql.ErrNoRows) {
			return face.ErrReviewConflict
		}
		if err != nil {
			return err
		}
		if state != "available" || auditRevision != command.ExpectedRevision {
			return face.ErrReviewConflict
		}
		var payload faceUndoPayload
		if json.Unmarshal([]byte(raw), &payload) != nil || payload.Action != action {
			return face.ErrReviewConflict
		}
		affected := []string{}
		switch action {
		case "assign_face", "assign_cluster":
			var personRevision int64
			err := tx.QueryRowContext(ctx, `SELECT revision FROM people WHERE id=? AND state='active'`, payload.PersonID).Scan(&personRevision)
			if err := requireFaceReviewState(err, personRevision == payload.PersonRevision); err != nil {
				return err
			}
			for _, item := range payload.Assignments {
				var personID, state string
				var current sql.NullString
				var revision int64
				err := tx.QueryRowContext(ctx, `SELECT person_id,current_face_id,state,revision FROM person_face_anchors WHERE id=?`, item.AnchorID).Scan(&personID, &current, &state, &revision)
				if err := requireFaceReviewState(err, personID == payload.PersonID && current.Valid && current.String == item.AfterCurrentFace && state == "bound" && revision == item.AfterAnchorRevision); err != nil {
					return err
				}
				var links int
				err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM face_cannot_links WHERE left_anchor_id=? OR right_anchor_id=?`, item.AnchorID, item.AnchorID).Scan(&links)
				if err := requireFaceReviewState(err, links == 0); err != nil {
					return err
				}
			}
			for index := len(payload.Assignments) - 1; index >= 0; index-- {
				item := payload.Assignments[index]
				if item.AnchorExisted {
					res, err := tx.ExecContext(ctx, `UPDATE person_face_anchors SET current_face_id=?,state=?,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND revision=?`, nullableString(item.BeforeCurrentFace), item.BeforeState, command.CreatedAt.UnixMilli(), item.AnchorID, item.AfterAnchorRevision)
					if err != nil {
						return err
					}
					if rows, _ := res.RowsAffected(); rows != 1 {
						return face.ErrReviewConflict
					}
				} else {
					res, err := tx.ExecContext(ctx, `DELETE FROM person_face_anchors WHERE id=? AND revision=?`, item.AnchorID, item.AfterAnchorRevision)
					if err != nil {
						return err
					}
					if rows, _ := res.RowsAffected(); rows != 1 {
						return face.ErrReviewConflict
					}
				}
				if item.Exclusion != nil {
					_, err := tx.ExecContext(ctx, `INSERT INTO face_exclusions(id,library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm,current_face_id,revision,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,?,?,?,?,?,?,MAX(?,?))`, item.Exclusion.ID, item.LibraryID, item.AssetID, item.Fingerprint, item.X, item.Y, item.Width, item.Height, nullableString(item.Exclusion.CurrentFace), item.Exclusion.Revision+1, item.Exclusion.CreatedAt, item.Exclusion.CreatedAt, command.CreatedAt.UnixMilli())
					if err != nil {
						return face.ErrReviewConflict
					}
				}
			}
			if err := bumpPerson(ctx, tx, payload.PersonID, payload.PersonRevision, command.CreatedAt); err != nil {
				return err
			}
			affected = []string{payload.PersonID}
		case "exclude_face":
			if len(payload.Assignments) != 1 {
				return face.ErrReviewConflict
			}
			item := payload.Assignments[0]
			var exclusionFace sql.NullString
			var exclusionRevision int64
			err := tx.QueryRowContext(ctx, `SELECT current_face_id,revision FROM face_exclusions WHERE id=?`, payload.AfterExclusionID).Scan(&exclusionFace, &exclusionRevision)
			if err := requireFaceReviewState(err, exclusionFace.Valid && exclusionFace.String == payload.FaceID && exclusionRevision == payload.AfterExclusionRevision); err != nil {
				return err
			}
			if payload.PersonID != "" {
				var personRevision int64
				err = tx.QueryRowContext(ctx, `SELECT revision FROM people WHERE id=? AND state='active'`, payload.PersonID).Scan(&personRevision)
				if err := requireFaceReviewState(err, personRevision == payload.PersonRevision); err != nil {
					return err
				}
			}
			res, err := tx.ExecContext(ctx, `DELETE FROM face_exclusions WHERE id=? AND revision=?`, payload.AfterExclusionID, payload.AfterExclusionRevision)
			if err != nil {
				return err
			}
			if rows, _ := res.RowsAffected(); rows != 1 {
				return face.ErrReviewConflict
			}
			if item.Exclusion != nil {
				_, err := tx.ExecContext(ctx, `INSERT INTO face_exclusions(id,library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm,current_face_id,revision,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,?,?,?,?,?,?,MAX(?,?))`, item.Exclusion.ID, item.LibraryID, item.AssetID, item.Fingerprint, item.X, item.Y, item.Width, item.Height, nullableString(item.Exclusion.CurrentFace), item.Exclusion.Revision+1, item.Exclusion.CreatedAt, item.Exclusion.CreatedAt, command.CreatedAt.UnixMilli())
				if err != nil {
					return face.ErrReviewConflict
				}
			}
			if item.AnchorExisted {
				_, err := tx.ExecContext(ctx, `INSERT INTO person_face_anchors(id,person_id,library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm,current_face_id,state,revision,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,MAX(?,?))`, item.AnchorID, payload.PersonID, item.LibraryID, item.AssetID, item.Fingerprint, item.X, item.Y, item.Width, item.Height, nullableString(item.BeforeCurrentFace), item.BeforeState, item.BeforeAnchorRevision+1, item.BeforeAnchorCreatedAt, item.BeforeAnchorCreatedAt, command.CreatedAt.UnixMilli())
				if err != nil {
					return face.ErrReviewConflict
				}
				if err := bumpPerson(ctx, tx, payload.PersonID, payload.PersonRevision, command.CreatedAt); err != nil {
					return err
				}
				affected = []string{payload.PersonID}
			}
		case "split_face":
			var anchorState string
			var currentFace sql.NullString
			var anchorRevision, personRevision int64
			err := tx.QueryRowContext(ctx, `SELECT anchor.state,anchor.current_face_id,anchor.revision,person.revision FROM person_face_anchors anchor JOIN people person ON person.id=anchor.person_id WHERE anchor.id=? AND anchor.person_id=?`, payload.AnchorID, payload.PersonID).Scan(&anchorState, &currentFace, &anchorRevision, &personRevision)
			if err := requireFaceReviewState(err, anchorState == "needs_review" && !currentFace.Valid && anchorRevision == payload.AnchorRevision && personRevision == payload.PersonRevision); err != nil {
				return err
			}
			res, err := tx.ExecContext(ctx, `UPDATE person_face_anchors SET current_face_id=?,state='bound',revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND revision=?`, payload.FaceID, command.CreatedAt.UnixMilli(), payload.AnchorID, payload.AnchorRevision)
			if err != nil {
				return face.ErrReviewConflict
			}
			if rows, _ := res.RowsAffected(); rows != 1 {
				return face.ErrReviewConflict
			}
			if err := bumpPerson(ctx, tx, payload.PersonID, payload.PersonRevision, command.CreatedAt); err != nil {
				return err
			}
			affected = []string{payload.PersonID}
		case "cannot_link":
			res, err := tx.ExecContext(ctx, `DELETE FROM face_cannot_links WHERE left_anchor_id=? AND right_anchor_id=? AND revision=?`, payload.LeftAnchorID, payload.RightAnchorID, payload.LinkRevision)
			if err != nil {
				return err
			}
			if rows, _ := res.RowsAffected(); rows != 1 {
				return face.ErrReviewConflict
			}
		case "merge_people":
			var sourceState, targetState string
			var sourceRevision, targetRevision int64
			err := tx.QueryRowContext(ctx, `SELECT state,revision FROM people WHERE id=?`, payload.SourcePersonID).Scan(&sourceState, &sourceRevision)
			if err := requireFaceReviewState(err, sourceState == "tombstoned" && sourceRevision == payload.SourcePersonRevision); err != nil {
				return err
			}
			err = tx.QueryRowContext(ctx, `SELECT state,revision FROM people WHERE id=?`, payload.TargetPersonID).Scan(&targetState, &targetRevision)
			if err := requireFaceReviewState(err, targetState == "active" && targetRevision == payload.TargetPersonRevision); err != nil {
				return err
			}
			var aliasTarget string
			err = tx.QueryRowContext(ctx, `SELECT target_person_id FROM person_aliases WHERE source_person_id=?`, payload.SourcePersonID).Scan(&aliasTarget)
			if err := requireFaceReviewState(err, aliasTarget == payload.TargetPersonID); err != nil {
				return err
			}
			for _, anchor := range payload.MergeAnchors {
				var personID string
				var revision int64
				err = tx.QueryRowContext(ctx, `SELECT person_id,revision FROM person_face_anchors WHERE id=?`, anchor.ID).Scan(&personID, &revision)
				if err := requireFaceReviewState(err, personID == payload.TargetPersonID && revision == anchor.Revision); err != nil {
					return err
				}
			}
			for _, anchor := range payload.MergeAnchors {
				res, err := tx.ExecContext(ctx, `UPDATE person_face_anchors SET person_id=?,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND person_id=? AND revision=?`, payload.SourcePersonID, command.CreatedAt.UnixMilli(), anchor.ID, payload.TargetPersonID, anchor.Revision)
				if err != nil {
					return err
				}
				if rows, _ := res.RowsAffected(); rows != 1 {
					return face.ErrReviewConflict
				}
			}
			if res, err := tx.ExecContext(ctx, `DELETE FROM person_aliases WHERE source_person_id=? AND target_person_id=?`, payload.SourcePersonID, payload.TargetPersonID); err != nil {
				return err
			} else if rows, _ := res.RowsAffected(); rows != 1 {
				return face.ErrReviewConflict
			}
			res, err := tx.ExecContext(ctx, `UPDATE people SET state='active',revision=revision+1,updated_at_ms=MAX(created_at_ms,?),tombstoned_at_ms=NULL WHERE id=? AND state='tombstoned' AND revision=?`, command.CreatedAt.UnixMilli(), payload.SourcePersonID, payload.SourcePersonRevision)
			if err != nil {
				return err
			}
			if rows, _ := res.RowsAffected(); rows != 1 {
				return face.ErrReviewConflict
			}
			if err := bumpPerson(ctx, tx, payload.TargetPersonID, payload.TargetPersonRevision, command.CreatedAt); err != nil {
				return err
			}
			affected = []string{payload.SourcePersonID, payload.TargetPersonID}
		default:
			return face.ErrReviewConflict
		}
		var library any
		if libraryID.Valid {
			library = libraryID.Int64
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO face_audit_events(id,request_hash,library_id,action,primary_target_id,before_revision,after_revision,undo_of_event_id,created_at_ms) VALUES(?,?,?,'undo',?,?,?,?,?)`, command.EventID, command.RequestHash, library, command.ReviewID, command.ExpectedRevision, command.ExpectedRevision+1, command.ReviewID, command.CreatedAt.UnixMilli())
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE face_review_undo_snapshots SET state='consumed',consumed_by_event_id=?,revision=revision+1 WHERE event_id=? AND state='available' AND revision=?`, command.EventID, command.ReviewID, snapshotRevision)
		if err != nil {
			return err
		}
		if rows, _ := res.RowsAffected(); rows != 1 {
			return face.ErrReviewConflict
		}
		result = reviewResult(command.EventID, "undo", affected, command.ExpectedRevision+1, false)
		return nil
	})
	return result, wrapReview("undo face review", err)
}
