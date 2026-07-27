package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite/dbgen"
)

func (s *Store) ListStartupLibraryIDs(
	ctx context.Context,
	afterID int64,
	limit int,
) ([]int64, error) {
	if afterID < 0 {
		return nil, errors.New("startup library cursor must not be negative")
	}
	if limit < 1 || limit > 256 {
		return nil, errors.New("startup library page limit must be between 1 and 256")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT libraries.id
		FROM libraries
		WHERE libraries.id > ?
			AND NOT EXISTS (
				SELECT 1
				FROM library_removals
				WHERE library_removals.library_id = libraries.id
					AND library_removals.status IN ('queued', 'running')
			)
		ORDER BY libraries.id
		LIMIT ?`,
		afterID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list startup scan libraries: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read startup scan library: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate startup scan libraries: %w", err)
	}
	return ids, nil
}

func (s *Store) ClaimNextFullScan(
	ctx context.Context,
	leaseDuration time.Duration,
) (scanner.ScanRun, bool, error) {
	leaseMS, err := scanLeaseMilliseconds(leaseDuration)
	if err != nil {
		return scanner.ScanRun{}, false, err
	}
	var (
		claimed scanner.ScanRun
		found   bool
	)
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		row, err := dbgen.New(tx).ClaimNextQueuedScan(ctx, dbgen.ClaimNextQueuedScanParams{
			NowMs:            sql.NullInt64{Int64: now, Valid: true},
			LeaseExpiresAtMs: sql.NullInt64{Int64: now + leaseMS, Valid: true},
		})
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("claim queued scan: %w", err)
		}
		claimed, err = contractScanRun(row)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE libraries
			SET status = 'scanning', revision = revision + 1, updated_at_ms = ?
			WHERE id = ?`,
			now, claimed.LibraryID,
		); err != nil {
			return fmt.Errorf("mark claimed scan library active: %w", err)
		}
		found = true
		return nil
	})
	return claimed, found, err
}

func (s *Store) RefreshFullScanLease(
	ctx context.Context,
	runID int64,
	leaseDuration time.Duration,
) (scanner.ScanRun, error) {
	if runID <= 0 {
		return scanner.ScanRun{}, scanner.ErrScanRunNotFound
	}
	leaseMS, err := scanLeaseMilliseconds(leaseDuration)
	if err != nil {
		return scanner.ScanRun{}, err
	}
	var refreshed scanner.ScanRun
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		row, err := dbgen.New(tx).TouchScanLease(ctx, dbgen.TouchScanLeaseParams{
			NowMs:            sql.NullInt64{Int64: now, Valid: true},
			LeaseExpiresAtMs: sql.NullInt64{Int64: now + leaseMS, Valid: true},
			ID:               runID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return scanner.ErrScanRunNotActive
		}
		if err != nil {
			return fmt.Errorf("touch scan lease: %w", err)
		}
		refreshed, err = contractScanRun(row)
		return err
	})
	return refreshed, err
}

func (s *Store) SetFullScanPhase(ctx context.Context, runID int64, phase string) error {
	if runID <= 0 {
		return scanner.ErrScanRunNotFound
	}
	switch phase {
	case scanner.PhaseCheckingRoot, scanner.PhaseWalking, scanner.PhaseFinalizing:
	default:
		return errors.New("invalid running scan phase")
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		queries := dbgen.New(tx)
		_, err := queries.UpdateRunningScanPhase(ctx, dbgen.UpdateRunningScanPhaseParams{
			Phase: phase,
			ID:    runID,
		})
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("update running scan phase: %w", err)
		}
		current, getErr := queries.GetScanContractRun(ctx, runID)
		switch {
		case errors.Is(getErr, sql.ErrNoRows):
			return scanner.ErrScanRunNotFound
		case getErr != nil:
			return fmt.Errorf("read scan after unchanged phase: %w", getErr)
		case current.Status != string(scanner.RunStatusRunning):
			return scanner.ErrScanRunNotActive
		case current.Phase == phase:
			return nil
		default:
			return scanner.ErrScanRunNotActive
		}
	})
}

func (s *Store) RecoverExpiredFullScans(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		queries := dbgen.New(tx)
		now := s.nowMS()
		for recovered := 0; recovered < scanner.MaxActiveFullScans; recovered++ {
			row, err := queries.RecoverNextExpiredScan(ctx, now)
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("recover expired scan: %w", err)
			}
			switch row.Status {
			case string(scanner.RunStatusQueued):
				summary.Requeued++
			case string(scanner.RunStatusInterrupted):
				summary.Interrupted++
			default:
				return errors.New("expired scan recovery returned invalid status")
			}
			libraryStatus := "pending"
			if row.Status == string(scanner.RunStatusInterrupted) {
				libraryStatus = "error"
			}
			if _, err := tx.ExecContext(ctx, `
				UPDATE libraries
				SET status = CASE
						WHEN current_generation > 0 THEN 'ready'
						ELSE ?
					END,
					revision = revision + 1,
					updated_at_ms = ?
				WHERE id = ?`,
				libraryStatus, now, row.LibraryID,
			); err != nil {
				return fmt.Errorf("restore recovered scan library: %w", err)
			}
		}
		var expired int
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM scan_runs
				WHERE status = 'running' AND lease_expires_at_ms <= ?
			)`,
			now,
		).Scan(&expired); err != nil {
			return fmt.Errorf("check bounded scan recovery: %w", err)
		}
		if expired != 0 {
			return errors.New("expired scan recovery exceeded active queue bound")
		}
		return nil
	})
	return summary, err
}

func scanLeaseMilliseconds(duration time.Duration) (int64, error) {
	if duration < time.Millisecond || duration > 10*time.Minute {
		return 0, errors.New("scan lease duration must be between one millisecond and ten minutes")
	}
	return duration.Milliseconds(), nil
}

func contractScanRun(row dbgen.ScanRun) (scanner.ScanRun, error) {
	run := scanner.ScanRun{
		ID:                    row.ID,
		LibraryID:             row.LibraryID,
		Generation:            row.Generation,
		Trigger:               scanner.Trigger(row.TriggerKind),
		Status:                scanner.RunStatus(row.Status),
		DiscoveredDirectories: row.DiscoveredDirectories,
		DiscoveredAssets:      row.DiscoveredAssets,
		SkippedCount:          row.SkippedCount,
		CreatedAtMS:           row.CreatedAtMs,
		Revision:              row.Revision,
		Phase:                 row.Phase,
		ProcessedAssets:       row.ProcessedAssets,
		SkippedDirectories:    row.SkippedDirectories,
		SkippedFiles:          row.SkippedFiles,
		ErrorCount:            row.ErrorCount,
		IssuesTruncated:       row.IssuesTruncated != 0,
	}
	if !run.Trigger.Valid() {
		return scanner.ScanRun{}, errors.New("scan queue returned invalid trigger")
	}
	switch run.Status {
	case scanner.RunStatusQueued, scanner.RunStatusRunning,
		scanner.RunStatusSucceeded, scanner.RunStatusFailed,
		scanner.RunStatusCancelled, scanner.RunStatusOffline,
		scanner.RunStatusInterrupted:
	default:
		return scanner.ScanRun{}, errors.New("scan queue returned invalid status")
	}
	if row.ErrorCode.Valid {
		run.ErrorCode = row.ErrorCode.String
	}
	if row.StartedAtMs.Valid {
		value := row.StartedAtMs.Int64
		run.StartedAtMS = &value
	}
	if row.FinishedAtMs.Valid {
		value := row.FinishedAtMs.Int64
		run.FinishedAtMS = &value
	}
	if row.CancelRequestedAtMs.Valid {
		value := row.CancelRequestedAtMs.Int64
		run.CancelRequestedAtMS = &value
	}
	return run, nil
}
