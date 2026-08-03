package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/systemlog"
)

func (s *Store) AppendSystemEvent(
	ctx context.Context,
	event systemlog.Event,
	maximum int,
) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO system_events(
                occurred_at_ms, level, module, event_code, request_id,
                method, route_pattern, status_code, duration_ms
            ) VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''),
                      NULLIF(?, ''), NULLIF(?, 0), NULLIF(?, 0))`,
			s.nowMS(), event.Level, event.Module, event.EventCode,
			event.RequestID, event.Method, event.RoutePattern,
			event.StatusCode, event.DurationMS,
		)
		if err != nil {
			return fmt.Errorf("append system event: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            DELETE FROM system_events
            WHERE id < COALESCE((
                SELECT MIN(id) FROM (
                    SELECT id FROM system_events ORDER BY id DESC LIMIT ?
                ) retained
            ), 0)`, maximum); err != nil {
			return fmt.Errorf("bound system events: %w", err)
		}
		return nil
	})
}

func (s *Store) ListSystemEvents(
	ctx context.Context,
	query systemlog.Query,
) ([]systemlog.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, occurred_at_ms, level, module, event_code,
               COALESCE(request_id, ''), COALESCE(method, ''),
               COALESCE(route_pattern, ''), COALESCE(status_code, 0),
               COALESCE(duration_ms, 0)
        FROM system_events
        WHERE (? = '' OR level = ?)
          AND (? = '' OR module = ?)
          AND (? = 0 OR id < ?)
        ORDER BY id DESC
        LIMIT ?`,
		query.Level, query.Level, query.Module, query.Module,
		query.BeforeID, query.BeforeID, query.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list system events: %w", err)
	}
	defer rows.Close()
	events := make([]systemlog.Event, 0, query.Limit)
	for rows.Next() {
		var event systemlog.Event
		if err := rows.Scan(
			&event.ID, &event.OccurredAtMS, &event.Level, &event.Module,
			&event.EventCode, &event.RequestID, &event.Method,
			&event.RoutePattern, &event.StatusCode, &event.DurationMS,
		); err != nil {
			return nil, fmt.Errorf("read system event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate system events: %w", err)
	}
	return events, nil
}
