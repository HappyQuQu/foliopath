package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/deletion"
	"github.com/HappyQuQu/foliopath/internal/jobs"
)

func (s *Store) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	var result jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		row, err := tx.ExecContext(ctx, `UPDATE media_deletion_operations SET state=CASE
            WHEN kind='permanent' AND (stop_requested=1 OR attempts>=3) AND EXISTS(SELECT 1 FROM media_deletion_items item JOIN media_mutation_scopes scope ON scope.item_id=item.id WHERE item.operation_id=media_deletion_operations.id AND scope.active=1) THEN 'reconciling'
            WHEN stop_requested=1 THEN 'stopped' WHEN attempts>=3 THEN 'needs_attention' ELSE 'queued' END,
            after_repair_state=CASE WHEN stop_requested=1 THEN 'stopped' WHEN attempts>=3 THEN 'needs_attention' ELSE after_repair_state END,
            lease_expires_at_ms=NULL,revision=revision+1,updated_at_ms=? WHERE state='running' AND lease_expires_at_ms<=?`, s.nowMS(), s.nowMS())
		if err != nil {
			return err
		}
		result.Requeued, err = row.RowsAffected()
		return err
	})
	return result, err
}

func (s *Store) Claim(ctx context.Context, lease time.Duration) (deletion.Claim, bool, error) {
	var result deletion.Claim
	found := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `UPDATE media_deletion_operations SET state='running',attempts=attempts+1,fence=fence+1,lease_expires_at_ms=?,updated_at_ms=?,revision=revision+1 WHERE id=(
   SELECT op.id FROM media_deletion_operations op WHERE state='queued' AND stop_requested=0 AND attempts<3
    AND (SELECT enabled FROM media_deletion_settings WHERE singleton=1)=1
    AND NOT EXISTS(SELECT 1 FROM media_deletion_operations WHERE state IN ('running','reconciling'))
    AND NOT EXISTS(SELECT 1 FROM media_deletion_items item JOIN scan_runs scan ON scan.library_id=item.library_id WHERE item.operation_id=op.id AND scan.status='running')
    AND NOT EXISTS(SELECT 1 FROM media_deletion_items item JOIN catalog_reconcile_jobs scan ON scan.library_id=item.library_id WHERE item.operation_id=op.id AND scan.status='running')
    ORDER BY id LIMIT 1) RETURNING id,fence,kind,(SELECT instance_id FROM media_deletion_settings WHERE singleton=1)`, s.nowMS()+lease.Milliseconds(), s.nowMS()).Scan(&result.ID, &result.Fence, &result.Kind, &result.Instance)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		found = err == nil
		return err
	})
	return result, found, err
}

func (s *Store) RefreshLease(ctx context.Context, claim deletion.Claim, lease time.Duration) (bool, error) {
	stopped := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `UPDATE media_deletion_operations SET lease_expires_at_ms=? WHERE id=? AND fence=? AND state='running' RETURNING stop_requested OR (SELECT enabled=0 FROM media_deletion_settings WHERE singleton=1)`, s.nowMS()+lease.Milliseconds(), claim.ID, claim.Fence).Scan(&stopped)
		if errors.Is(err, sql.ErrNoRows) {
			stopped = true
			return nil
		}
		return err
	})
	return stopped, err
}

func requireDeletionClaim(ctx context.Context, reader queryRower, claim deletion.Claim, allowStop bool) error {
	var stopped bool
	err := reader.QueryRowContext(ctx, `SELECT stop_requested OR (SELECT enabled=0 FROM media_deletion_settings WHERE singleton=1) FROM media_deletion_operations WHERE id=? AND fence=? AND state='running'`, claim.ID, claim.Fence).Scan(&stopped)
	if errors.Is(err, sql.ErrNoRows) {
		return deletion.ErrPrecondition
	}
	if err != nil {
		return err
	}
	if stopped && !allowStop {
		return deletion.ErrDisabled
	}
	return nil
}

func (s *Store) NextDeletionItem(ctx context.Context, claim deletion.Claim) (deletion.OperationItem, bool, error) {
	var result deletion.OperationItem
	found := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireDeletionClaim(ctx, tx, claim, false); err != nil {
			return err
		}
		item, err := scanDeletionItem(tx.QueryRowContext(ctx, `SELECT `+deletionItemColumns+` FROM media_deletion_items i JOIN libraries l ON l.id=i.library_id WHERE i.operation_id=? AND i.state IN ('queued','running') ORDER BY i.id LIMIT 1`, claim.ID))
		if errors.Is(err, sql.ErrNoRows) {
			added, fillErr := fillEmptyTrashBatch(ctx, tx, claim.ID)
			if fillErr != nil || !added {
				return fillErr
			}
			item, err = scanDeletionItem(tx.QueryRowContext(ctx, `SELECT `+deletionItemColumns+` FROM media_deletion_items i JOIN libraries l ON l.id=i.library_id WHERE i.operation_id=? AND i.state='queued' ORDER BY i.id LIMIT 1`, claim.ID))
		}
		if err != nil {
			return err
		}
		result = item
		found = true
		return nil
	})
	return result, found, err
}

func fillEmptyTrashBatch(ctx context.Context, tx *sql.Tx, id int64) (bool, error) {
	var upper, libraryID sql.NullInt64
	var after, sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT empty_upper_id,empty_after_id,empty_library_id,COALESCE(empty_sequence,0) FROM media_deletion_operations WHERE id=?`, id).Scan(&upper, &after, &libraryID, &sequence); err != nil {
		return false, err
	}
	if !upper.Valid || upper.Int64 <= after {
		return false, nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT e.id FROM media_trash_entries e WHERE e.id>? AND e.id<=? AND e.available_sequence<=? AND e.state='available' AND (? IS NULL OR e.library_id=?) AND NOT EXISTS(`+trashReservationRows+` AND op.id<?) ORDER BY e.id LIMIT ?`, after, upper.Int64, sequence, libraryID, libraryID, id, deletion.MaxTargets)
	if err != nil {
		return false, err
	}
	ids := make([]int64, 0, deletion.MaxTargets)
	for rows.Next() {
		var entry int64
		if err := rows.Scan(&entry); err != nil {
			rows.Close()
			return false, err
		}
		ids = append(ids, entry)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return false, err
	}
	for _, entry := range ids {
		item, _, _, _, err := readTrashMutationItem(ctx, tx, entry)
		if err != nil {
			return false, err
		}
		if err := insertDeletionItem(ctx, tx, id, item, false); err != nil {
			return false, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE media_trash_entries SET state='purging',revision=revision+1 WHERE id=?`, entry); err != nil {
			return false, err
		}
	}
	after = upper.Int64
	if len(ids) > 0 {
		after = ids[len(ids)-1]
	}
	_, err = tx.ExecContext(ctx, `UPDATE media_deletion_operations SET empty_after_id=? WHERE id=?`, after, id)
	return len(ids) > 0, err
}

func (s *Store) BeginDeletionItem(ctx context.Context, claim deletion.Claim, item deletion.OperationItem) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireDeletionClaim(ctx, tx, claim, false); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE media_deletion_items SET state='running',stage=CASE WHEN stage='queued' THEN 'intent' ELSE stage END,repair_state='queued',repair_scan_id=NULL WHERE id=? AND operation_id=?`, item.ID, claim.ID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE media_mutation_scopes SET active=1 WHERE item_id=?`, item.ID)
		return err
	})
}

func (s *Store) RecordDeletionFilesystem(ctx context.Context, claim deletion.Claim, item deletion.OperationItem, removed int64, done bool, unchanged bool, code string) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireDeletionClaim(ctx, tx, claim, true); err != nil {
			return err
		}
		state := "running"
		if code != "" {
			state = "failed"
			if removed > 0 || item.RemovedEntries > 0 {
				state = "partial"
			}
		}
		_, err := tx.ExecContext(ctx, `UPDATE media_deletion_items SET removed_entries=removed_entries+?,stage=CASE WHEN ? THEN 'filesystem_applied' ELSE stage END,state=CASE WHEN ?<>'' AND removed_entries+?>0 THEN 'partial' ELSE ? END,error_code=NULLIF(?,'') WHERE id=? AND operation_id=?`, removed, done, code, removed, state, code, item.ID, claim.ID)
		if err != nil {
			return err
		}
		if code != "" && unchanged {
			if _, err := tx.ExecContext(ctx, `UPDATE media_mutation_scopes SET active=0 WHERE item_id=?`, item.ID); err != nil {
				return err
			}
			if claim.Kind == "restore" {
				_, err = tx.ExecContext(ctx, `UPDATE media_trash_entries SET state='available',revision=revision+1 WHERE id=?`, item.TrashEntryID)
			}
		}
		return err
	})
}

func (s *Store) FinishDeletionClaim(ctx context.Context, claim deletion.Claim, state string) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireDeletionClaim(ctx, tx, claim, true); err != nil {
			return err
		}
		if state == "stopped" {
			var requested bool
			if err := tx.QueryRowContext(ctx, `SELECT stop_requested OR (SELECT enabled=0 FROM media_deletion_settings WHERE singleton=1) FROM media_deletion_operations WHERE id=?`, claim.ID).Scan(&requested); err != nil {
				return err
			}
			if !requested {
				return nil
			}
		}
		var unfinished, failed, succeeded, removed int64
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FILTER(WHERE state IN ('queued','running')),COUNT(*) FILTER(WHERE state IN ('failed','partial','needs_attention')),COUNT(*) FILTER(WHERE state IN ('trashed','restored','deleted')),COALESCE(SUM(removed_entries),0) FROM media_deletion_items WHERE operation_id=?`, claim.ID).Scan(&unfinished, &failed, &succeeded, &removed); err != nil {
			return err
		}
		if state == "succeeded" && failed > 0 {
			state = "partial"
			if succeeded == 0 && removed == 0 {
				state = "failed"
			}
		}
		if state == "succeeded" && unfinished > 0 {
			state = "needs_attention"
		}
		var active bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_mutation_scopes scope JOIN media_deletion_items item ON item.id=scope.item_id WHERE item.operation_id=? AND scope.active=1)`, claim.ID).Scan(&active); err != nil {
			return err
		}
		afterRepair := state
		if claim.Kind == "permanent" && active {
			state = "reconciling"
		}
		if claim.Kind != "permanent" && active {
			state = "needs_attention"
		}
		_, err := tx.ExecContext(ctx, `UPDATE media_deletion_operations SET state=?,after_repair_state=?,lease_expires_at_ms=NULL,revision=revision+1,updated_at_ms=? WHERE id=? AND fence=?`, state, afterRepair, s.nowMS(), claim.ID, claim.Fence)
		return err
	})
}

func (s *Store) CheckDeletionPermission(ctx context.Context, claim deletion.Claim) error {
	return requireDeletionClaim(ctx, s.db, claim, false)
}

func (s *Store) HandoffDeletion(ctx context.Context, claim deletion.Claim) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE media_deletion_operations SET state='queued',attempts=attempts-1,lease_expires_at_ms=NULL,last_handoff_fence=fence,last_handoff_at_ms=?,updated_at_ms=?,revision=revision+1 WHERE id=? AND fence=? AND state='running' AND attempts>0 AND stop_requested=0 AND (SELECT enabled FROM media_deletion_settings WHERE singleton=1)=1`, s.nowMS(), s.nowMS(), claim.ID, claim.Fence)
		return err
	})
}
