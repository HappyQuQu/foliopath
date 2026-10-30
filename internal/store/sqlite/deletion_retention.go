package sqlite

import (
	"context"
	"database/sql"
)

func (s *Store) cancelQueuedRetentionTx(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `UPDATE media_trash_entries SET state='available',revision=revision+1 WHERE id IN(SELECT item.trash_entry_id FROM media_deletion_items item JOIN media_deletion_operations op ON op.id=item.operation_id WHERE op.policy_revision IS NOT NULL AND op.state='queued')`); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE media_deletion_operations SET stop_requested=1,state=CASE WHEN state='queued' THEN 'cancelled' ELSE state END,revision=revision+1,updated_at_ms=? WHERE policy_revision IS NOT NULL AND state IN ('queued','running')`, s.nowMS())
	return err
}
