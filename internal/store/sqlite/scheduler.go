package sqlite

import (
	"context"
	"fmt"
)

func (s *Store) GetScheduledScanIntervalHours(ctx context.Context) (*int64, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return settings.ScheduledScanIntervalHours, nil
}

func (s *Store) ListDueLibraryIDs(
	ctx context.Context,
	dueBeforeMS int64,
	afterID int64,
	limit int,
) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT libraries.id
		FROM libraries
		WHERE libraries.id > ?
		  AND NOT EXISTS (
		      SELECT 1 FROM library_removals
		      WHERE library_removals.library_id = libraries.id
		        AND library_removals.status IN ('queued', 'running')
		  )
		  AND COALESCE((
		      SELECT MAX(scan_runs.created_at_ms)
		      FROM scan_runs
		      WHERE scan_runs.library_id = libraries.id
		  ), 0) <= ?
		ORDER BY libraries.id
		LIMIT ?`,
		afterID, dueBeforeMS, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list due scan libraries: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan due library ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due library IDs: %w", err)
	}
	return ids, nil
}
