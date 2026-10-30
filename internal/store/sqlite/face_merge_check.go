package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/face"
)

// The preview and write transaction share this owner, including detached anchors.
const personMergeCannotLinkFrom = ` FROM person_face_anchors source
 JOIN face_cannot_links link ON (link.left_anchor_id=source.id OR link.right_anchor_id=source.id)
 JOIN person_face_anchors target ON target.id=CASE WHEN link.left_anchor_id=source.id THEN link.right_anchor_id ELSE link.left_anchor_id END `
const personMergePairPredicate = `source.person_id=? AND target.person_id=?`
const personMergeDuplicateFrom = ` FROM person_face_anchors source JOIN person_face_anchors target ON target.person_id=? AND target.library_id=source.library_id AND target.asset_id=source.asset_id AND target.source_fingerprint=source.source_fingerprint AND target.box_x_ppm=source.box_x_ppm AND target.box_y_ppm=source.box_y_ppm AND target.box_width_ppm=source.box_width_ppm AND target.box_height_ppm=source.box_height_ppm WHERE source.person_id=?`

func readPersonMergeConflictCounts(ctx context.Context, tx *sql.Tx, source, target string) (links, duplicates int64, err error) {
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*)`+personMergeCannotLinkFrom+` WHERE `+personMergePairPredicate, source, target).Scan(&links); err != nil {
		return
	}
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*)`+personMergeDuplicateFrom, target, source).Scan(&duplicates)
	return
}
func (s *Store) GetPersonMergeCheck(ctx context.Context, source, target string) (face.PersonMergeCheck, error) {
	if !face.ValidReviewPairGuard(source, 1, target, 1) {
		return face.PersonMergeCheck{}, face.ErrInvalidPerson
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return face.PersonMergeCheck{}, err
	}
	defer tx.Rollback()
	result := face.PersonMergeCheck{SourcePersonID: source, TargetPersonID: target, Examples: []face.MergeConflictExample{}}
	for _, entry := range []struct {
		id       string
		revision *int64
	}{{source, &result.SourceRevision}, {target, &result.TargetRevision}} {
		err = tx.QueryRowContext(ctx, `SELECT revision FROM people WHERE id=? AND state='active'`, entry.id).Scan(entry.revision)
		if errors.Is(err, sql.ErrNoRows) {
			return result, face.ErrPersonNotFound
		}
		if err != nil {
			return result, err
		}
	}
	var anchors int64
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM person_face_anchors WHERE person_id=?`, source).Scan(&anchors); err != nil {
		return result, err
	}
	result.SourceLimitExceeded = anchors > int64(s.maxBatchSize)
	result.CannotLinkCount, result.DuplicateAssignmentCount, err = readPersonMergeConflictCounts(ctx, tx, source, target)
	if err != nil {
		return result, err
	}
	if result.CannotLinkCount > 0 {
		rows, queryErr := tx.QueryContext(ctx, `SELECT DISTINCT source.library_id,source.asset_id,target.library_id,target.asset_id`+personMergeCannotLinkFrom+`
   JOIN visible_assets source_asset ON source_asset.library_id=source.library_id AND source_asset.id=source.asset_id
   JOIN libraries source_library ON source_library.id=source.library_id AND source_library.status='ready'
   JOIN visible_assets target_asset ON target_asset.library_id=target.library_id AND target_asset.id=target.asset_id
   JOIN libraries target_library ON target_library.id=target.library_id AND target_library.status='ready'
   WHERE `+personMergePairPredicate+` AND source.state='bound' AND target.state='bound' AND source.current_face_id IS NOT NULL AND target.current_face_id IS NOT NULL
   ORDER BY source.library_id,source.asset_id,target.library_id,target.asset_id LIMIT ?`, source, target, face.MaxMergeConflictExamples)
		if queryErr != nil {
			return result, queryErr
		}
		for rows.Next() {
			var example face.MergeConflictExample
			if err = rows.Scan(&example.SourceLibraryID, &example.SourceAssetID, &example.TargetLibraryID, &example.TargetAssetID); err != nil {
				rows.Close()
				return result, err
			}
			result.Examples = append(result.Examples, example)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return result, err
		}
		if err = rows.Close(); err != nil {
			return result, err
		}
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

var _ face.PersonMergeCheckRepository = (*Store)(nil)
