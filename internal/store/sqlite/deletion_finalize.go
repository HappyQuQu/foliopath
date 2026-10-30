package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/deletion"
)

func (s *Store) FinalizeDeletionItem(ctx context.Context, claim deletion.Claim, item deletion.OperationItem) (bool, error) {
	done := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireDeletionClaim(ctx, tx, claim, true); err != nil {
			return err
		}
		var stage string
		if err := tx.QueryRowContext(ctx, `SELECT stage FROM media_deletion_items WHERE id=? AND operation_id=?`, item.ID, claim.ID).Scan(&stage); err != nil {
			return err
		}
		if stage != "filesystem_applied" && stage != "catalog_finalizing" {
			return deletion.ErrPrecondition
		}
		if _, err := tx.ExecContext(ctx, `UPDATE media_deletion_items SET stage='catalog_finalizing' WHERE id=?`, item.ID); err != nil {
			return err
		}
		var changed bool
		var err error
		switch claim.Kind {
		case "trash":
			changed, err = trashCatalogBatch(ctx, tx, item)
		case "restore":
			changed, err = restoreCatalogBatch(ctx, tx, item)
		default:
			changed, err = deleteCatalogBatch(ctx, tx, item, claim.Kind == "purge")
		}
		if err != nil {
			return err
		}
		if changed {
			return nil
		}
		state := "deleted"
		switch claim.Kind {
		case "trash":
			state = "trashed"
			_, err = tx.ExecContext(ctx, `UPDATE media_trash_entries SET state='available',deleted_at_ms=?,revision=revision+1 WHERE id=?`, s.nowMS(), item.TrashEntryID)
		case "restore":
			state = "restored"
			_, err = tx.ExecContext(ctx, `DELETE FROM directories WHERE trash_entry_id=? AND restoration_path IS NULL`, item.TrashEntryID)
			if err == nil {
				_, err = tx.ExecContext(ctx, `DELETE FROM media_trash_entries WHERE id=?`, item.TrashEntryID)
			}
		case "purge":
			_, err = tx.ExecContext(ctx, `DELETE FROM media_trash_entries WHERE id=?`, item.TrashEntryID)
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM media_mutation_scopes WHERE item_id=?`, item.ID); err != nil {
			return err
		}
		if err := refreshDeletionCounts(ctx, tx, item.LibraryID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE media_deletion_items SET state=?,stage='finished',error_code=NULL WHERE id=?`, state, item.ID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE media_deletion_operations SET revision=revision+1,updated_at_ms=? WHERE id=?`, s.nowMS(), claim.ID)
		done = err == nil
		return err
	})
	return done, err
}

func trashCatalogBatch(ctx context.Context, tx *sql.Tx, item deletion.OperationItem) (bool, error) {
	prefix := deletion.TrashDirectory + "/" + strconv.FormatInt(item.TrashEntryID, 10) + "/"
	var holdingID int64
	if item.Target.Type == "asset" {
		err := tx.QueryRowContext(ctx, `INSERT INTO directories(library_id,relative_path,name,last_seen_generation,trash_entry_id) VALUES(?,?,?,1,?) ON CONFLICT(library_id,relative_path) DO UPDATE SET trash_entry_id=excluded.trash_entry_id RETURNING id`, item.LibraryID, prefix+"holding", item.Name, item.TrashEntryID).Scan(&holdingID)
		if err != nil {
			return false, err
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE assets SET restoration_path=relative_path,relative_path=?||relative_path,trash_entry_id=?,directory_id=CASE WHEN ?>0 THEN ? ELSE directory_id END WHERE id IN (
  SELECT id FROM assets WHERE library_id=? AND trash_entry_id IS NULL AND (relative_path=? OR (?='directory' AND instr(relative_path,?||'/')=1)) LIMIT ?)`, prefix, item.TrashEntryID, holdingID, holdingID, item.LibraryID, item.RelativePath, item.Target.Type, item.RelativePath, deletion.BatchSize)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count > 0 {
		return count > 0, err
	}
	if item.Target.Type == "asset" {
		return false, nil
	}
	result, err = tx.ExecContext(ctx, `UPDATE directories SET restoration_path=relative_path,relative_path=?||relative_path,trash_entry_id=?,parent_id=CASE WHEN relative_path=? THEN NULL ELSE parent_id END WHERE id IN (
  SELECT id FROM directories WHERE library_id=? AND trash_entry_id IS NULL AND (relative_path=? OR instr(relative_path,?||'/')=1) LIMIT ?)`, prefix, item.TrashEntryID, item.RelativePath, item.LibraryID, item.RelativePath, item.RelativePath, deletion.BatchSize)
	if err != nil {
		return false, err
	}
	count, err = result.RowsAffected()
	return count > 0, err
}

func restoreCatalogBatch(ctx context.Context, tx *sql.Tx, item deletion.OperationItem) (bool, error) {
	parentPath := path.Dir(item.RelativePath)
	if parentPath == "." {
		parentPath = ""
	}
	var parentID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM directories WHERE library_id=? AND relative_path=? AND trash_entry_id IS NULL`, item.LibraryID, parentPath).Scan(&parentID); err != nil {
		return false, fmt.Errorf("restore parent: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE directories SET relative_path=restoration_path,restoration_path=NULL,trash_entry_id=NULL,parent_id=CASE WHEN restoration_path=? THEN ? ELSE parent_id END WHERE id IN (
  SELECT id FROM directories WHERE trash_entry_id=? AND restoration_path IS NOT NULL LIMIT ?)`, item.RelativePath, parentID, item.TrashEntryID, deletion.BatchSize)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count > 0 {
		return count > 0, err
	}
	result, err = tx.ExecContext(ctx, `UPDATE assets SET relative_path=restoration_path,restoration_path=NULL,trash_entry_id=NULL,directory_id=CASE WHEN ?='asset' THEN ? ELSE directory_id END WHERE id IN (
  SELECT id FROM assets WHERE trash_entry_id=? LIMIT ?)`, item.Target.Type, parentID, item.TrashEntryID, deletion.BatchSize)
	if err != nil {
		return false, err
	}
	count, err = result.RowsAffected()
	return count > 0, err
}

func deleteCatalogBatch(ctx context.Context, tx *sql.Tx, item deletion.OperationItem, trash bool) (bool, error) {
	predicate := `library_id=? AND trash_entry_id IS NULL AND (relative_path=? OR (?='directory' AND instr(relative_path,?||'/')=1))`
	args := []any{item.LibraryID, item.RelativePath, item.Target.Type, item.RelativePath}
	if trash {
		predicate = `trash_entry_id=?`
		args = []any{item.TrashEntryID}
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM assets WHERE id IN (SELECT id FROM assets WHERE `+predicate+` LIMIT ?)`, append(args, deletion.BatchSize)...)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count > 0 {
		return count > 0, err
	}
	if item.Target.Type == "asset" && !trash {
		return false, nil
	}
	result, err = tx.ExecContext(ctx, `DELETE FROM directories WHERE id IN (SELECT id FROM directories WHERE `+predicate+` AND NOT EXISTS(SELECT 1 FROM directories child WHERE child.parent_id=directories.id) LIMIT ?)`, append(args, deletion.BatchSize)...)
	if err != nil {
		return false, err
	}
	count, err = result.RowsAffected()
	return count > 0, err
}

func refreshDeletionCounts(ctx context.Context, tx *sql.Tx, libraryID int64) error {
	var rootID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM directories WHERE library_id=? AND relative_path=''`, libraryID).Scan(&rootID); err != nil {
		return err
	}
	return recalculateDirectoryCountsTx(ctx, tx, libraryID, rootID)
}

func validateDeletionRestore(ctx context.Context, reader queryRower, item deletion.OperationItem) error {
	parent := path.Dir(item.RelativePath)
	if parent == "." {
		parent = ""
	}
	var exists bool
	if err := reader.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM visible_directories WHERE library_id=? AND relative_path=?)`, item.LibraryID, parent).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return deletion.ErrSourceMissing
	}
	var conflict bool
	err := reader.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE library_id=? AND trash_entry_id IS NULL AND (relative_path=? OR instr(relative_path,?||'/')=1)) OR EXISTS(SELECT 1 FROM directories WHERE library_id=? AND trash_entry_id IS NULL AND (relative_path=? OR instr(relative_path,?||'/')=1))`, item.LibraryID, item.RelativePath, item.RelativePath, item.LibraryID, item.RelativePath, item.RelativePath).Scan(&conflict)
	if err != nil {
		return err
	}
	if conflict {
		return deletion.ErrConflict
	}
	return nil
}

func (s *Store) CheckDeletionRestore(ctx context.Context, claim deletion.Claim, item deletion.OperationItem) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireDeletionClaim(ctx, tx, claim, false); err != nil {
			return err
		}
		return validateDeletionRestore(ctx, tx, item)
	})
}
