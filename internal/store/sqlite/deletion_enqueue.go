package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/deletion"
)

func (s *Store) FindDeletionReplay(ctx context.Context, key, request [32]byte) (deletion.Operation, bool, error) {
	return findDeletionReplay(ctx, s.db, key, request)
}

func findDeletionReplay(ctx context.Context, reader queryRower, key, request [32]byte) (deletion.Operation, bool, error) {
	var stored []byte
	var id int64
	err := reader.QueryRowContext(ctx, `SELECT request_hash,operation_id FROM media_deletion_requests WHERE key_hash=?`, key[:]).Scan(&stored, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return deletion.Operation{}, false, nil
	}
	if err != nil {
		return deletion.Operation{}, false, err
	}
	if !bytes.Equal(stored, request[:]) {
		return deletion.Operation{}, false, deletion.ErrConflict
	}
	result, err := readDeletionOperation(ctx, reader, id)
	return result, true, err
}

func (s *Store) EnqueueDeletion(ctx context.Context, command deletion.EnqueueCommand) (deletion.Operation, error) {
	var result deletion.Operation
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		replay, found, err := findDeletionReplay(ctx, tx, command.KeyHash, command.RequestHash)
		if err != nil {
			return err
		}
		if found {
			result = replay
			return nil
		}
		preflight, err := readDeletionPreflight(ctx, tx, command.Preflight.ID, s.nowMS())
		if err != nil {
			return err
		}
		settings, err := readDeletionSettings(ctx, tx)
		if err != nil {
			return err
		}
		if !settings.Enabled {
			return deletion.ErrDisabled
		}
		if preflight.SettingsRevision != settings.Revision {
			return deletion.ErrPrecondition
		}
		if err := deletionCapacity(ctx, tx); err != nil {
			return err
		}
		id, err := s.insertDeletionOperation(ctx, tx, preflight.Mode, int64(len(preflight.Items)))
		if err != nil {
			return err
		}
		for _, selection := range preflight.Items {
			var currentPath, status string
			table := "assets"
			if selection.Target.Type == "directory" {
				table = "directories"
			}
			err := tx.QueryRowContext(ctx, `SELECT target.relative_path,l.status FROM `+table+` target JOIN libraries l ON l.id=target.library_id WHERE target.id=? AND target.library_id=? AND target.trash_entry_id IS NULL`, selection.Target.ID, selection.LibraryID).Scan(&currentPath, &status)
			if errors.Is(err, sql.ErrNoRows) || currentPath != selection.RelativePath {
				return deletion.ErrSourceChanged
			}
			if err != nil {
				return err
			}
			if status == "offline" {
				return deletion.ErrSourceIO
			}
			busy, err := deletionScopeBusy(ctx, tx, selection.LibraryID, selection.RelativePath, id)
			if err != nil {
				return err
			}
			if busy {
				return deletion.ErrBusy
			}
			item := deletion.OperationItem{LibraryID: selection.LibraryID, Name: selection.Name, RelativePath: selection.RelativePath, Kind: selection.Kind, Target: selection.Target, Identity: selection.Identity, LibraryIdentity: selection.LibraryIdentity}
			if preflight.Mode == "trash" {
				identity, _ := json.Marshal(item.Identity)
				libraryIdentity, _ := json.Marshal(item.LibraryIdentity)
				err = tx.QueryRowContext(ctx, `INSERT INTO media_trash_entries(library_id,token,target_type,target_id,relative_path,name,kind,identity,library_identity,state) VALUES(?,lower(hex(randomblob(16))),?,?,?,?,?,?,?,'moving') RETURNING id,token`, item.LibraryID, item.Target.Type, item.Target.ID, item.RelativePath, item.Name, item.Kind, string(identity), string(libraryIdentity)).Scan(&item.TrashEntryID, &item.TrashToken)
				if err != nil {
					return err
				}
			}
			if err := insertDeletionItem(ctx, tx, id, item, true); err != nil {
				return err
			}
			if err := s.cancelLibraryScansTx(ctx, tx, item.LibraryID); err != nil {
				return err
			}
		}
		if err := recordDeletionRequest(ctx, tx, command.KeyHash, command.RequestHash, id); err != nil {
			return err
		}
		result, err = readDeletionOperation(ctx, tx, id)
		return err
	})
	return result, err
}

func deletionCapacity(ctx context.Context, reader queryRower) error {
	var count int
	if err := reader.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_deletion_operations WHERE state IN ('queued','running','reconciling')`).Scan(&count); err != nil {
		return err
	}
	if count >= deletion.MaxActiveOperations {
		return deletion.ErrQueueFull
	}
	return nil
}

func (s *Store) insertDeletionOperation(ctx context.Context, tx *sql.Tx, kind string, total int64) (int64, error) {
	result, err := tx.ExecContext(ctx, `INSERT INTO media_deletion_operations(kind,total_items,created_at_ms,updated_at_ms) VALUES(?,?,?,?)`, kind, total, s.nowMS(), s.nowMS())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func insertDeletionItem(ctx context.Context, tx *sql.Tx, operationID int64, item deletion.OperationItem, scope bool) error {
	identity, err := json.Marshal(item.Identity)
	if err != nil {
		return err
	}
	libraryIdentity, err := json.Marshal(item.LibraryIdentity)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO media_deletion_items(operation_id,library_id,target_type,target_id,relative_path,name,kind,identity,library_identity,trash_entry_id,trash_token) VALUES(?,?,?,?,?,?,?,?,?,NULLIF(?,0),?)`, operationID, item.LibraryID, item.Target.Type, item.Target.ID, item.RelativePath, item.Name, item.Kind, string(identity), string(libraryIdentity), item.TrashEntryID, item.TrashToken)
	if err != nil {
		return err
	}
	if scope {
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO media_mutation_scopes(item_id,library_id,relative_path) VALUES(?,?,?)`, id, item.LibraryID, item.RelativePath)
		return err
	}
	return nil
}

func recordDeletionRequest(ctx context.Context, tx *sql.Tx, key, request [32]byte, id int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO media_deletion_requests(key_hash,request_hash,operation_id) VALUES(?,?,?)`, key[:], request[:], id)
	return err
}

func (s *Store) ChangeDeletionOperation(ctx context.Context, command deletion.ChangeCommand) (deletion.Operation, error) {
	var result deletion.Operation
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		replay, found, err := findDeletionReplay(ctx, tx, command.KeyHash, command.RequestHash)
		if err != nil {
			return err
		}
		if found {
			result = replay
			return nil
		}
		current, err := readDeletionOperation(ctx, tx, command.ID)
		if err != nil {
			return err
		}
		if current.Revision != command.Revision {
			return deletion.ErrPrecondition
		}
		if command.Action == "stop" {
			if !current.CanStop {
				return deletion.ErrConflict
			}
			_, err = tx.ExecContext(ctx, `UPDATE media_deletion_operations SET stop_requested=1,state=CASE WHEN state='queued' THEN 'stopped' ELSE state END,revision=revision+1,updated_at_ms=? WHERE id=?`, s.nowMS(), command.ID)
		} else {
			if !current.CanResume {
				return deletion.ErrConflict
			}
			if err := prepareDeletionResume(ctx, tx, current); err != nil {
				return err
			}
			if err := deletionCapacity(ctx, tx); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE media_deletion_operations SET stop_requested=0,state='queued',attempts=0,policy_revision=NULL,revision=revision+1,updated_at_ms=? WHERE id=?`, s.nowMS(), command.ID)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE media_deletion_items SET state='queued',error_code=NULL WHERE operation_id=? AND state IN ('failed','partial','needs_attention','cancelled')`, command.ID)
			if err != nil {
				return err
			}
			if current.Kind != "purge" {
				_, err = tx.ExecContext(ctx, `INSERT INTO media_mutation_scopes(item_id,library_id,relative_path) SELECT id,library_id,relative_path FROM media_deletion_items WHERE operation_id=? AND state IN ('queued','running') ON CONFLICT(item_id) DO NOTHING`, command.ID)
			}
		}
		if err != nil {
			return err
		}
		if err := recordDeletionRequest(ctx, tx, command.KeyHash, command.RequestHash, command.ID); err != nil {
			return err
		}
		result, err = readDeletionOperation(ctx, tx, command.ID)
		return err
	})
	return result, err
}

func (s *Store) EnqueueTrashOperation(ctx context.Context, command deletion.TrashCommand) (deletion.Operation, error) {
	var result deletion.Operation
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		replay, found, err := findDeletionReplay(ctx, tx, command.KeyHash, command.RequestHash)
		if err != nil {
			return err
		}
		if found {
			result = replay
			return nil
		}
		settings, err := readDeletionSettings(ctx, tx)
		if err != nil {
			return err
		}
		if !settings.Enabled {
			return deletion.ErrDisabled
		}
		if err := deletionCapacity(ctx, tx); err != nil {
			return err
		}
		id, err := s.insertDeletionOperation(ctx, tx, command.Kind, int64(len(command.Entries)))
		if err != nil {
			return err
		}
		if command.Empty {
			var upper, total int64
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(e.id),0),COUNT(*) FROM media_trash_entries e WHERE e.state='available' AND (?=0 OR e.library_id=?) AND NOT `+trashReservation, command.LibraryID, command.LibraryID).Scan(&upper, &total); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE media_deletion_operations SET empty_upper_id=?,empty_sequence=(SELECT sequence FROM media_trash_clock WHERE singleton=1),empty_library_id=NULLIF(?,0),total_items=? WHERE id=?`, upper, command.LibraryID, total, id)
			if err != nil {
				return err
			}
		} else {
			seen := map[int64]bool{}
			for _, entry := range command.Entries {
				if seen[entry.ID] {
					return deletion.ErrInvalid
				}
				seen[entry.ID] = true
				item, revision, state, reserved, err := readTrashMutationItem(ctx, tx, entry.ID)
				if err != nil {
					return err
				}
				if revision != entry.Revision {
					return deletion.ErrPrecondition
				}
				if state != "available" || reserved {
					return deletion.ErrBusy
				}
				if command.Kind == "restore" {
					busy, err := deletionScopeBusy(ctx, tx, item.LibraryID, item.RelativePath, id)
					if err != nil {
						return err
					}
					if busy {
						return deletion.ErrBusy
					}
					if err := s.cancelLibraryScansTx(ctx, tx, item.LibraryID); err != nil {
						return err
					}
				}
				if err := insertDeletionItem(ctx, tx, id, item, command.Kind == "restore"); err != nil {
					return err
				}
				state = "purging"
				if command.Kind == "restore" {
					state = "restoring"
				}
				if _, err := tx.ExecContext(ctx, `UPDATE media_trash_entries SET state=?,revision=revision+1 WHERE id=?`, state, entry.ID); err != nil {
					return err
				}
			}
		}
		if err := recordDeletionRequest(ctx, tx, command.KeyHash, command.RequestHash, id); err != nil {
			return err
		}
		result, err = readDeletionOperation(ctx, tx, id)
		return err
	})
	return result, err
}

func readTrashMutationItem(ctx context.Context, reader queryRower, id int64) (deletion.OperationItem, int64, string, bool, error) {
	var item deletion.OperationItem
	var identity, libraryIdentity, state string
	var revision int64
	var reserved bool
	err := reader.QueryRowContext(ctx, `SELECT e.library_id,e.relative_path,e.name,e.kind,e.target_type,e.target_id,e.identity,e.library_identity,e.id,e.token,e.revision,e.state,`+trashReservation+` FROM media_trash_entries e WHERE e.id=?`, id).Scan(&item.LibraryID, &item.RelativePath, &item.Name, &item.Kind, &item.Target.Type, &item.Target.ID, &identity, &libraryIdentity, &item.TrashEntryID, &item.TrashToken, &revision, &state, &reserved)
	if errors.Is(err, sql.ErrNoRows) {
		return item, 0, "", false, deletion.ErrNotFound
	}
	if err != nil {
		return item, 0, "", false, err
	}
	if err := json.Unmarshal([]byte(identity), &item.Identity); err != nil {
		return item, 0, "", false, err
	}
	err = json.Unmarshal([]byte(libraryIdentity), &item.LibraryIdentity)
	return item, revision, state, reserved, err
}

func prepareDeletionResume(ctx context.Context, tx *sql.Tx, operation deletion.Operation) error {
	if operation.Kind == "purge" {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+deletionItemColumns+` FROM media_deletion_items i JOIN libraries l ON l.id=i.library_id WHERE operation_id=? AND i.state NOT IN ('trashed','restored','deleted') ORDER BY i.id`, operation.ID)
	if err != nil {
		return err
	}
	items := make([]deletion.OperationItem, 0, deletion.MaxTargets)
	for rows.Next() {
		item, err := scanDeletionItem(rows)
		if err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range items {
		busy, err := deletionScopeBusy(ctx, tx, item.LibraryID, item.RelativePath, operation.ID)
		if err != nil {
			return err
		}
		if busy {
			return deletion.ErrBusy
		}
		if item.TrashEntryID > 0 {
			var other bool
			err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_deletion_items i JOIN media_deletion_operations op ON op.id=i.operation_id WHERE i.trash_entry_id=? AND op.id<>? AND op.state IN ('queued','running','reconciling'))`, item.TrashEntryID, operation.ID).Scan(&other)
			if err != nil {
				return err
			}
			if other {
				return deletion.ErrBusy
			}
		}
		if operation.Kind == "restore" {
			result, err := tx.ExecContext(ctx, `UPDATE media_trash_entries SET state='restoring',revision=revision+1 WHERE id=? AND state IN ('available','restoring','needs_attention')`, item.TrashEntryID)
			if err != nil {
				return err
			}
			count, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if count != 1 {
				return deletion.ErrSourceChanged
			}
		}
	}
	return nil
}
