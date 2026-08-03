package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func (s *Store) ListMediaFailures(
	ctx context.Context,
	query thumbnail.FailureQuery,
) ([]thumbnail.MediaFailure, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT job.id, job.library_id, job.asset_id, library.name,
               asset.relative_path, job.variant, job.last_error_code,
               job.attempt_count, job.finished_at_ms,
               attempt.attempt_number, attempt.outcome, attempt.stage,
               attempt.reason_code, attempt.tool, attempt.exit_code,
               attempt.duration_ms, attempt.finished_at_ms
        FROM media_jobs AS job
        JOIN libraries AS library ON library.id = job.library_id
        JOIN assets AS asset
          ON asset.library_id = job.library_id AND asset.id = job.asset_id
        LEFT JOIN media_job_attempts AS attempt
          ON attempt.id = (
            SELECT recent.id FROM media_job_attempts AS recent
            WHERE recent.job_id = job.id ORDER BY recent.id DESC LIMIT 1
          )
        WHERE job.status = 'failed'
          AND (? = 0 OR job.library_id = ?)
          AND (? = '' OR job.variant = ?)
          AND (? = '' OR job.last_error_code = ?)
          AND (? = 0 OR job.id < ?)
        ORDER BY job.id DESC
        LIMIT ?`,
		query.LibraryID, query.LibraryID,
		string(query.Variant), string(query.Variant),
		string(query.ErrorCode), string(query.ErrorCode),
		query.BeforeID, query.BeforeID,
		query.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list media failures: %w", err)
	}
	defer rows.Close()
	failures := make([]thumbnail.MediaFailure, 0, query.Limit)
	for rows.Next() {
		var failure thumbnail.MediaFailure
		var variant, code string
		var finished, attemptNumber, exitCode, duration, attemptFinished sql.NullInt64
		var outcome, stage, reason, tool sql.NullString
		if err := rows.Scan(
			&failure.JobID, &failure.LibraryID, &failure.AssetID,
			&failure.LibraryName, &failure.RelativePath, &variant, &code,
			&failure.AttemptCount, &finished, &attemptNumber, &outcome,
			&stage, &reason, &tool, &exitCode, &duration, &attemptFinished,
		); err != nil {
			return nil, fmt.Errorf("read media failure: %w", err)
		}
		failure.Variant = thumbnail.Variant(variant)
		failure.ErrorCode = thumbnail.JobErrorCode(code)
		failure.FinishedAtMS = finished.Int64
		if attemptNumber.Valid {
			failure.LatestAttempt = mediaFailureAttempt(
				attemptNumber, outcome, stage, reason, tool, exitCode, duration, attemptFinished,
			)
		}
		failures = append(failures, failure)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate media failures: %w", err)
	}
	return failures, nil
}

func (s *Store) LatestMediaFailureRevision(
	ctx context.Context,
	query thumbnail.FailureQuery,
) (thumbnail.FailureRevision, bool, error) {
	var revision thumbnail.FailureRevision
	err := s.db.QueryRowContext(ctx, `
        SELECT job.finished_at_ms, job.id
        FROM media_jobs AS job
        WHERE job.status = 'failed'
          AND (? = 0 OR job.library_id = ?)
          AND (? = '' OR job.variant = ?)
          AND (? = '' OR job.last_error_code = ?)
        ORDER BY job.finished_at_ms DESC, job.id DESC
        LIMIT 1`,
		query.LibraryID, query.LibraryID,
		string(query.Variant), string(query.Variant),
		string(query.ErrorCode), string(query.ErrorCode),
	).Scan(&revision.FinishedAtMS, &revision.JobID)
	if errors.Is(err, sql.ErrNoRows) {
		return thumbnail.FailureRevision{}, false, nil
	}
	if err != nil {
		return thumbnail.FailureRevision{}, false,
			fmt.Errorf("get latest media failure revision: %w", err)
	}
	return revision, true, nil
}

func (s *Store) GetMediaFailure(
	ctx context.Context,
	jobID int64,
) (thumbnail.MediaFailure, error) {
	var failure thumbnail.MediaFailure
	var variant, code string
	var finished sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
        SELECT job.id, job.library_id, job.asset_id, library.name,
               asset.relative_path, job.variant, job.last_error_code,
               job.attempt_count, job.finished_at_ms
        FROM media_jobs AS job
        JOIN libraries AS library ON library.id = job.library_id
        JOIN assets AS asset
          ON asset.library_id = job.library_id AND asset.id = job.asset_id
        WHERE job.id = ? AND job.status = 'failed'`, jobID,
	).Scan(
		&failure.JobID, &failure.LibraryID, &failure.AssetID,
		&failure.LibraryName, &failure.RelativePath, &variant, &code,
		&failure.AttemptCount, &finished,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return thumbnail.MediaFailure{}, thumbnail.ErrDiagnosticsFailureNotFound
	}
	if err != nil {
		return thumbnail.MediaFailure{}, fmt.Errorf("get media failure: %w", err)
	}
	failure.Variant = thumbnail.Variant(variant)
	failure.ErrorCode = thumbnail.JobErrorCode(code)
	failure.FinishedAtMS = finished.Int64
	rows, err := s.db.QueryContext(ctx, `
        SELECT attempt_number, outcome, stage, reason_code, tool,
               exit_code, duration_ms, finished_at_ms
        FROM media_job_attempts
        WHERE job_id = ? ORDER BY id DESC LIMIT 10`, jobID)
	if err != nil {
		return thumbnail.MediaFailure{}, fmt.Errorf("list media failure attempts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var attemptNumber, exitCode, duration, attemptFinished sql.NullInt64
		var outcome, stage, reason, tool sql.NullString
		if err := rows.Scan(
			&attemptNumber, &outcome, &stage, &reason, &tool,
			&exitCode, &duration, &attemptFinished,
		); err != nil {
			return thumbnail.MediaFailure{}, fmt.Errorf("read media failure attempt: %w", err)
		}
		attempt := mediaFailureAttempt(
			attemptNumber, outcome, stage, reason, tool, exitCode, duration, attemptFinished,
		)
		failure.AttemptHistory = append(failure.AttemptHistory, *attempt)
	}
	if err := rows.Err(); err != nil {
		return thumbnail.MediaFailure{}, fmt.Errorf("iterate media failure attempts: %w", err)
	}
	if len(failure.AttemptHistory) > 0 {
		latest := failure.AttemptHistory[0]
		failure.LatestAttempt = &latest
	}
	return failure, nil
}

func mediaFailureAttempt(
	attemptNumber sql.NullInt64,
	outcome, stage, reason, tool sql.NullString,
	exitCode, duration, finished sql.NullInt64,
) *thumbnail.MediaFailureAttempt {
	attempt := &thumbnail.MediaFailureAttempt{
		AttemptNumber: int(attemptNumber.Int64),
		Outcome:       thumbnail.JobOutcome(outcome.String),
		Stage:         media.FailureStage(stage.String),
		Reason:        media.FailureReason(reason.String),
		Tool:          tool.String,
		DurationMS:    duration.Int64,
		FinishedAtMS:  finished.Int64,
	}
	if exitCode.Valid {
		value := int(exitCode.Int64)
		attempt.ExitCode = &value
	}
	return attempt
}

func (s *Store) RequeueMediaProcessing(
	ctx context.Context,
	libraryID int64,
	mode thumbnail.RequeueMode,
	limit int,
) (thumbnail.RetrySummary, error) {
	var summary thumbnail.RetrySummary
	if mode != thumbnail.RequeueMissing && mode != thumbnail.RequeueAll {
		return summary, thumbnail.ErrInvalidDiagnosticsRequest
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx,
			`SELECT 1 FROM libraries WHERE id = ?`, libraryID,
		).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return thumbnail.ErrDiagnosticsLibraryNotFound
		} else if err != nil {
			return fmt.Errorf("find media diagnostics library: %w", err)
		}
		return nil
	})
	if err != nil {
		return summary, err
	}

	candidate := `job.status IN ('failed', 'succeeded')`
	if mode == thumbnail.RequeueMissing {
		candidate = `
            job.status = 'failed'
            OR (
              job.status = 'succeeded'
              AND NOT EXISTS (
                SELECT 1 FROM thumbnails AS derived
                WHERE derived.asset_id = job.asset_id
                  AND derived.variant = job.variant
                  AND derived.status = 'ready'
                  AND derived.source_fingerprint = job.source_fingerprint
                  AND derived.transform_version = job.transform_version
              )
            )`
	}
	now := s.nowMS()
	var afterID int64
	for {
		var requeued int64
		err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
			rows, queryErr := tx.QueryContext(ctx, fmt.Sprintf(`
            UPDATE media_jobs
            SET status = 'queued', last_error_code = NULL,
                available_at_ms = ?, started_at_ms = NULL,
                heartbeat_at_ms = NULL, lease_expires_at_ms = NULL,
                attempt_count = 0, created_at_ms = ?, finished_at_ms = NULL
            WHERE id IN (
                SELECT job.id
                FROM media_jobs AS job
                WHERE job.library_id = ?
                  AND job.id > ?
                  AND (%s)
                ORDER BY id
                LIMIT ?
            )
            RETURNING id`, candidate), now, now, libraryID, afterID, limit)
			if queryErr != nil {
				return fmt.Errorf("requeue media processing: %w", queryErr)
			}
			defer rows.Close()
			for rows.Next() {
				var jobID int64
				if scanErr := rows.Scan(&jobID); scanErr != nil {
					return fmt.Errorf("inspect requeued media processing: %w", scanErr)
				}
				requeued++
				if jobID > afterID {
					afterID = jobID
				}
			}
			return rows.Err()
		})
		if err != nil {
			return summary, err
		}
		summary.Requeued += requeued
		if requeued == 0 {
			break
		}
	}
	// Manual missing/all actions intentionally include terminal failures. Those
	// codes only stop automatic retry loops; they do not block an administrator.
	return summary, nil
}
