package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/deletion"
)

func readDeletionSettings(ctx context.Context, reader queryRower) (deletion.Settings, error) {
	var result deletion.Settings
	var days sql.NullInt64
	err := reader.QueryRowContext(ctx, `SELECT enabled,recycle_enabled,retention_days,revision,instance_id FROM media_deletion_settings WHERE singleton=1`).Scan(
		&result.Enabled, &result.RecycleBinEnabled, &days, &result.Revision, &result.Instance)
	if days.Valid {
		result.RetentionDays = &days.Int64
	}
	return result, err
}

func (s *Store) GetDeletionSettings(ctx context.Context) (deletion.Settings, error) {
	return readDeletionSettings(ctx, s.db)
}

func (s *Store) UpdateDeletionSettings(ctx context.Context, revision int64, update deletion.SettingsUpdate) (deletion.Settings, error) {
	var result deletion.Settings
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		changed, err := tx.ExecContext(ctx, `UPDATE media_deletion_settings SET enabled=?,recycle_enabled=?,retention_days=?,revision=revision+1 WHERE singleton=1 AND revision=?`, update.Enabled, update.RecycleBinEnabled, update.RetentionDays, revision)
		if err != nil {
			return err
		}
		rows, err := changed.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return deletion.ErrPrecondition
		}
		if err := s.cancelQueuedRetentionTx(ctx, tx); err != nil {
			return err
		}
		if !update.Enabled {
			if _, err := tx.ExecContext(ctx, `UPDATE media_deletion_operations SET stop_requested=1,state=CASE WHEN state='queued' THEN 'stopped' ELSE state END,revision=revision+1,updated_at_ms=? WHERE state IN ('queued','running')`, s.nowMS()); err != nil {
				return err
			}
		}
		result, err = readDeletionSettings(ctx, tx)
		return err
	})
	return result, err
}

func (s *Store) PreviewDeletionRetention(ctx context.Context, days int64) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_trash_entries WHERE state='available' AND deleted_at_ms<=?`, s.nowMS()-days*86_400_000).Scan(&count)
	return count, err
}

func (s *Store) ResolveDeletionTargets(ctx context.Context, targets []deletion.Target) ([]deletion.Selection, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result := make([]deletion.Selection, 0, len(targets))
	for _, target := range targets {
		item := deletion.Selection{Target: target}
		var status, archivePath string
		var trash sql.NullInt64
		if target.Type == "asset" {
			err = tx.QueryRowContext(ctx, `SELECT a.library_id,l.name,l.root_rel_path,a.name,COALESCE(a.restoration_path,a.relative_path),a.size_bytes,a.mtime_ns,a.archive_path,l.status,a.trash_entry_id
                FROM assets a JOIN libraries l ON l.id=a.library_id WHERE a.id=?`, target.ID).Scan(
				&item.LibraryID, &item.LibraryName, &item.Root, &item.Name, &item.RelativePath, &item.Size, &item.ModifiedNS, &archivePath, &status, &trash)
			item.Kind = "file"
			if archivePath != "" {
				item.BlockedReason = "archive_member"
			}
		} else {
			err = tx.QueryRowContext(ctx, `SELECT d.library_id,l.name,l.root_rel_path,d.name,COALESCE(d.restoration_path,d.relative_path),d.mtime_ns,d.archive_path,l.status,d.trash_entry_id
                FROM directories d JOIN libraries l ON l.id=d.library_id WHERE d.id=?`, target.ID).Scan(
				&item.LibraryID, &item.LibraryName, &item.Root, &item.Name, &item.RelativePath, &item.ModifiedNS, &archivePath, &status, &trash)
			item.Kind = "directory"
			if archivePath != "" {
				if archivePath == item.RelativePath {
					item.Kind = "archive"
				} else {
					item.BlockedReason = "archive_member"
				}
			}
		}
		if errors.Is(err, sql.ErrNoRows) {
			item.BlockedReason = "missing"
			result = append(result, item)
			continue
		}
		if err != nil {
			return nil, err
		}
		if item.RelativePath == "" {
			item.BlockedReason = "root"
		}
		if trash.Valid {
			item.BlockedReason = "busy"
		}
		if status == "offline" {
			item.BlockedReason = "offline"
		}
		if item.BlockedReason == "" {
			busy, err := deletionScopeBusy(ctx, tx, item.LibraryID, item.RelativePath, 0)
			if err != nil {
				return nil, err
			}
			if busy {
				item.BlockedReason = "busy"
			}
		}
		result = append(result, item)
	}
	return result, tx.Commit()
}

func deletionScopeBusy(ctx context.Context, reader queryRower, libraryID int64, relative string, exceptOperation int64) (bool, error) {
	var busy bool
	err := reader.QueryRowContext(ctx, `SELECT EXISTS(
        SELECT 1 FROM media_mutation_scopes scope JOIN media_deletion_items item ON item.id=scope.item_id
        JOIN media_deletion_operations op ON op.id=item.operation_id
        WHERE scope.library_id=? AND op.id<>? AND (op.state IN ('queued','running','reconciling') OR scope.active=1)
          AND (scope.relative_path=? OR instr(?,scope.relative_path||'/')=1 OR instr(scope.relative_path,?||'/')=1)
    ) OR EXISTS(SELECT 1 FROM library_removals WHERE library_id=? AND status IN ('queued','running'))`,
		libraryID, exceptOperation, relative, relative, relative, libraryID).Scan(&busy)
	return busy, err
}

func (s *Store) SaveDeletionPreflight(ctx context.Context, value deletion.Preflight) (deletion.Preflight, error) {
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM media_deletion_preflights WHERE expires_at_ms<=?`, s.nowMS()); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_deletion_preflights`).Scan(&count); err != nil {
			return err
		}
		if count >= deletion.MaxActiveOperations {
			return deletion.ErrQueueFull
		}
		if err := tx.QueryRowContext(ctx, `SELECT lower(hex(randomblob(16)))`).Scan(&value.ID); err != nil {
			return err
		}
		payload, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO media_deletion_preflights(id,expires_at_ms,settings_revision,payload) VALUES(?,?,?,?)`, value.ID, value.ExpiresAtMS, value.SettingsRevision, string(payload))
		return err
	})
	return value, err
}

func (s *Store) GetDeletionPreflight(ctx context.Context, id string) (deletion.Preflight, error) {
	return readDeletionPreflight(ctx, s.db, id, s.nowMS())
}

func readDeletionPreflight(ctx context.Context, reader queryRower, id string, now int64) (deletion.Preflight, error) {
	var payload string
	if err := reader.QueryRowContext(ctx, `SELECT payload FROM media_deletion_preflights WHERE id=? AND expires_at_ms>?`, id, now).Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return deletion.Preflight{}, deletion.ErrExpired
		}
		return deletion.Preflight{}, err
	}
	var value deletion.Preflight
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return value, fmt.Errorf("read deletion confirmation: %w", err)
	}
	return value, nil
}

const deletionOperationColumns = `op.id,op.kind,op.state,op.created_at_ms,op.updated_at_ms,op.revision,op.stop_requested,op.empty_upper_id,op.total_items,
    (SELECT COUNT(*) FROM media_deletion_items i WHERE i.operation_id=op.id AND i.state NOT IN ('queued','running')),
    (SELECT COUNT(*) FROM media_deletion_items i WHERE i.operation_id=op.id AND i.state IN ('failed','partial','needs_attention')),
    (SELECT enabled FROM media_deletion_settings WHERE singleton=1)`

func scanDeletionOperation(row rowScanner) (deletion.Operation, error) {
	var result deletion.Operation
	var stopped, enabled bool
	var upper sql.NullInt64
	err := row.Scan(&result.ID, &result.Kind, &result.State, &result.CreatedAtMS, &result.UpdatedAtMS, &result.Revision, &stopped, &upper, &result.TotalItems, &result.CompletedItems, &result.FailedItems, &enabled)
	result.Items = []deletion.OperationItem{}
	result.CanStop = !stopped && (result.State == "queued" || result.State == "running")
	result.CanResume = enabled && (result.State == "stopped" || result.State == "partial" || result.State == "failed" || result.State == "needs_attention")
	if errors.Is(err, sql.ErrNoRows) {
		return result, deletion.ErrNotFound
	}
	return result, err
}

func readDeletionOperation(ctx context.Context, reader queryRower, id int64) (deletion.Operation, error) {
	return scanDeletionOperation(reader.QueryRowContext(ctx, `SELECT `+deletionOperationColumns+` FROM media_deletion_operations op WHERE op.id=?`, id))
}

func (s *Store) GetDeletionOperation(ctx context.Context, id int64) (deletion.Operation, error) {
	return readDeletionOperation(ctx, s.db, id)
}

func (s *Store) ListDeletionOperations(ctx context.Context, libraryID, before int64, limit int) ([]deletion.Operation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+deletionOperationColumns+` FROM media_deletion_operations op
		WHERE (?=0 OR op.empty_library_id=? OR EXISTS(SELECT 1 FROM media_deletion_items i WHERE i.operation_id=op.id AND i.library_id=?)
		 OR (op.empty_library_id IS NULL AND EXISTS(SELECT 1 FROM media_trash_entries e WHERE e.library_id=? AND e.id<=op.empty_upper_id AND e.available_sequence<=op.empty_sequence)))
		  AND (?=0 OR op.id<?) ORDER BY op.id DESC LIMIT ?`, libraryID, libraryID, libraryID, libraryID, before, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]deletion.Operation, 0, limit)
	for rows.Next() {
		item, err := scanDeletionOperation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

const deletionItemColumns = `i.id,i.library_id,l.name,i.name,i.relative_path,i.kind,i.state,i.removed_entries,COALESCE(i.error_code,''),COALESCE(i.trash_entry_id,0),i.target_type,i.target_id,l.root_rel_path,i.identity,i.library_identity,i.stage,i.trash_token`

func scanDeletionItem(row rowScanner) (deletion.OperationItem, error) {
	var result deletion.OperationItem
	var identity, libraryIdentity string
	err := row.Scan(&result.ID, &result.LibraryID, &result.LibraryName, &result.Name, &result.RelativePath, &result.Kind, &result.State, &result.RemovedEntries, &result.ErrorCode, &result.TrashEntryID, &result.Target.Type, &result.Target.ID, &result.Root, &identity, &libraryIdentity, &result.Stage, &result.TrashToken)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal([]byte(identity), &result.Identity); err != nil {
		return result, err
	}
	if err := json.Unmarshal([]byte(libraryIdentity), &result.LibraryIdentity); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Store) ListDeletionOperationItems(ctx context.Context, id, before int64, limit int) ([]deletion.OperationItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+deletionItemColumns+` FROM media_deletion_items i JOIN libraries l ON l.id=i.library_id
        WHERE i.operation_id=? AND (?=0 OR i.id<?) ORDER BY i.id DESC LIMIT ?`, id, before, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]deletion.OperationItem, 0, limit)
	for rows.Next() {
		item, err := scanDeletionItem(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

const trashReservationRows = `SELECT 1 FROM media_deletion_operations op WHERE op.kind='purge' AND op.state NOT IN ('succeeded','cancelled')
    AND op.empty_upper_id>=e.id AND op.empty_sequence>=e.available_sequence AND (op.empty_library_id IS NULL OR op.empty_library_id=e.library_id)`
const trashReservation = `EXISTS(` + trashReservationRows + `)`

func (s *Store) ListTrashEntries(ctx context.Context, libraryID, before int64, limit int) ([]deletion.TrashEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT e.id,e.library_id,l.name,e.name,e.relative_path,e.kind,e.deleted_at_ms,
        CASE WHEN settings.retention_days IS NULL THEN NULL ELSE e.deleted_at_ms+settings.retention_days*86400000 END,
        CASE WHEN e.state='available' AND `+trashReservation+` THEN 'purging' ELSE e.state END,
        e.revision,settings.enabled AND e.state='available' AND l.status<>'offline' AND NOT `+trashReservation+`
        FROM media_trash_entries e JOIN libraries l ON l.id=e.library_id CROSS JOIN media_deletion_settings settings
        WHERE e.state<>'moving' AND (?=0 OR e.library_id=?) AND (?=0 OR e.id<?)
        ORDER BY e.id DESC LIMIT ?`, libraryID, libraryID, before, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]deletion.TrashEntry, 0, limit)
	for rows.Next() {
		var item deletion.TrashEntry
		var expires sql.NullInt64
		if err := rows.Scan(&item.ID, &item.LibraryID, &item.LibraryName, &item.Name, &item.RelativePath, &item.Kind, &item.DeletedAtMS, &expires, &item.State, &item.Revision, &item.CanRestore); err != nil {
			return nil, err
		}
		if expires.Valid {
			item.ExpiresAtMS = &expires.Int64
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
