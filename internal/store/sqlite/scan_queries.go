package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite/dbgen"
)

func (s *Store) ListScanRuns(
	ctx context.Context,
	libraryID int64,
	before scanner.QueryPosition,
	limit int,
) ([]scanner.ScanRun, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM libraries WHERE id = ?`, libraryID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, scanner.ErrLibraryNotFound
	} else if err != nil {
		return nil, fmt.Errorf("check scan library: %w", err)
	}
	rows, err := s.queries.ListLibraryScanContractRuns(ctx, dbgen.ListLibraryScanContractRunsParams{
		LibraryID:         libraryID,
		BeforeCreatedAtMs: before.CreatedAtMS,
		BeforeID:          before.ID,
		PageSize:          int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list scan runs: %w", err)
	}
	runs := make([]scanner.ScanRun, 0, len(rows))
	for _, row := range rows {
		run, err := contractScanRun(row)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (s *Store) GetScanDetails(ctx context.Context, scanID int64) (scanner.Details, error) {
	row, err := s.queries.GetScanContractRun(ctx, scanID)
	if errors.Is(err, sql.ErrNoRows) {
		return scanner.Details{}, scanner.ErrScanRunNotFound
	}
	if err != nil {
		return scanner.Details{}, fmt.Errorf("get scan details: %w", err)
	}
	run, err := contractScanRun(row)
	if err != nil {
		return scanner.Details{}, err
	}
	rows, err := s.queries.ListScanIssues(ctx, scanID)
	if err != nil {
		return scanner.Details{}, fmt.Errorf("list scan issues: %w", err)
	}
	issues := make([]scanner.Issue, 0, len(rows))
	for _, row := range rows {
		issue := scanner.Issue{Code: row.Code, Count: row.IssueCount}
		if row.SampleRelPath.Valid {
			value := row.SampleRelPath.String
			issue.SampleRelativePath = &value
		}
		issues = append(issues, issue)
	}
	return scanner.Details{Run: run, Issues: issues}, nil
}

func (s *Store) RequestScanCancellation(ctx context.Context, scanID int64) (scanner.ScanRun, error) {
	var result scanner.ScanRun
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		queries := dbgen.New(tx)
		row, err := queries.GetScanContractRun(ctx, scanID)
		if errors.Is(err, sql.ErrNoRows) {
			return scanner.ErrScanRunNotFound
		}
		if err != nil {
			return fmt.Errorf("get scan for cancellation: %w", err)
		}
		switch scanner.RunStatus(row.Status) {
		case scanner.RunStatusQueued:
			row, err = queries.CancelQueuedScan(ctx, dbgen.CancelQueuedScanParams{
				NowMs: sql.NullInt64{Int64: s.nowMS(), Valid: true},
				ID:    scanID,
			})
			if err == nil {
				_, err = tx.ExecContext(ctx, `
					UPDATE libraries
					SET status = CASE WHEN current_generation > 0 THEN 'ready' ELSE 'pending' END,
					    revision = revision + 1, updated_at_ms = ?
					WHERE id = ?`, s.nowMS(), row.LibraryID)
			}
		case scanner.RunStatusRunning:
			row, err = queries.RequestRunningScanCancellation(ctx, dbgen.RequestRunningScanCancellationParams{
				NowMs: sql.NullInt64{Int64: s.nowMS(), Valid: true},
				ID:    scanID,
			})
		default:
			return scanner.ErrScanAlreadyFinished
		}
		if err != nil {
			return fmt.Errorf("request scan cancellation: %w", err)
		}
		result, err = contractScanRun(row)
		return err
	})
	return result, err
}
