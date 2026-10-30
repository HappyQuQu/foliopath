package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/HappyQuQu/foliopath/internal/deletion"
)

func (s *Store) ListDeletionRepairs(ctx context.Context, limit int) ([]deletion.RepairRequest, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item.operation_id,item.library_id,l.root_rel_path,item.library_identity,COALESCE(item.repair_scan_id,0)
  FROM media_deletion_items item JOIN media_deletion_operations op ON op.id=item.operation_id
  JOIN libraries l ON l.id=item.library_id JOIN media_mutation_scopes scope ON scope.item_id=item.id
  WHERE op.state='reconciling' AND scope.active=1 AND item.repair_state IN ('queued','running')
  GROUP BY item.operation_id,item.library_id,item.library_identity,item.repair_scan_id ORDER BY op.id,item.library_id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]deletion.RepairRequest, 0, limit)
	for rows.Next() {
		var item deletion.RepairRequest
		var identity string
		if err := rows.Scan(&item.OperationID, &item.LibraryID, &item.Root, &identity, &item.ScanID); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(identity), &item.Identity); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) AssignDeletionRepair(ctx context.Context, request deletion.RepairRequest, scanID int64) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE media_deletion_items SET repair_scan_id=?,repair_state='running' WHERE operation_id=? AND library_id=? AND repair_state='queued' AND EXISTS(SELECT 1 FROM media_mutation_scopes WHERE item_id=media_deletion_items.id AND active=1) AND EXISTS(SELECT 1 FROM media_deletion_operations WHERE id=? AND state='reconciling')`, scanID, request.OperationID, request.LibraryID, request.OperationID)
		return err
	})
}

func (s *Store) CompleteDeletionRepair(ctx context.Context, request deletion.RepairRequest, success bool) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var repairing bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_deletion_operations WHERE id=? AND state='reconciling')`, request.OperationID).Scan(&repairing); err != nil {
			return err
		}
		if !repairing {
			return nil
		}
		state := "failed"
		if success {
			state = "succeeded"
		}
		_, err := tx.ExecContext(ctx, `UPDATE media_deletion_items SET repair_state=?,state=CASE WHEN ?=0 THEN 'needs_attention' WHEN state='running' THEN CASE WHEN removed_entries>0 THEN 'partial' ELSE 'cancelled' END ELSE state END WHERE operation_id=? AND library_id=? AND repair_state IN ('queued','running') AND EXISTS(SELECT 1 FROM media_mutation_scopes WHERE item_id=media_deletion_items.id AND active=1) AND COALESCE(repair_scan_id,0)=?`, state, success, request.OperationID, request.LibraryID, request.ScanID)
		if err != nil {
			return err
		}
		if success {
			if _, err := tx.ExecContext(ctx, `DELETE FROM media_mutation_scopes WHERE item_id IN(SELECT id FROM media_deletion_items WHERE operation_id=? AND library_id=? AND repair_state='succeeded')`, request.OperationID, request.LibraryID); err != nil {
				return err
			}
			if err := refreshDeletionCounts(ctx, tx, request.LibraryID); err != nil {
				return err
			}
		}
		var pending, failed bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_deletion_items item JOIN media_mutation_scopes scope ON scope.item_id=item.id WHERE operation_id=? AND scope.active=1 AND repair_state IN ('queued','running')), EXISTS(SELECT 1 FROM media_deletion_items WHERE operation_id=? AND repair_state='failed')`, request.OperationID, request.OperationID).Scan(&pending, &failed); err != nil {
			return err
		}
		if pending {
			return nil
		}
		_, err = tx.ExecContext(ctx, `UPDATE media_deletion_operations SET state=CASE WHEN ? THEN 'needs_attention' ELSE after_repair_state END,updated_at_ms=?,revision=revision+1 WHERE id=?`, failed, s.nowMS(), request.OperationID)
		return err
	})
}
