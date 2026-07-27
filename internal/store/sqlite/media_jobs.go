package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func (s *Store) ReconcileMediaJobTransform(
	ctx context.Context,
	transformVersion int,
	limit int,
) (int64, error) {
	if transformVersion <= 0 || limit < 1 || limit > 256 {
		return 0, thumbnail.ErrInvalidJob
	}
	var changed int64
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		if _, err := tx.ExecContext(ctx, `
            INSERT OR IGNORE INTO cache_deletions(
                library_id, cache_rel_path, byte_size, created_at_ms
            )
            SELECT thumbnail.library_id, thumbnail.cache_rel_path,
                   thumbnail.byte_size, ?
            FROM thumbnails AS thumbnail
            JOIN media_jobs AS job ON job.asset_id = thumbnail.asset_id
            WHERE job.transform_version <> ?
              AND thumbnail.status = 'ready'
              AND job.id IN (
                  SELECT id FROM media_jobs
                  WHERE transform_version <> ?
                  ORDER BY id LIMIT ?
              )`,
			now, transformVersion, transformVersion, limit,
		); err != nil {
			return fmt.Errorf("schedule old transform cache cleanup: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            DELETE FROM thumbnails
            WHERE asset_id IN (
                SELECT asset_id FROM media_jobs
                WHERE transform_version <> ?
                ORDER BY id LIMIT ?
            )`,
			transformVersion, limit,
		); err != nil {
			return fmt.Errorf("invalidate old thumbnail transform: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE assets
            SET width = NULL, height = NULL, duration_ms = NULL,
                probe_status = 'pending', probe_error_code = NULL,
                playback_status = 'unknown'
            WHERE id IN (
                SELECT asset_id FROM media_jobs
                WHERE transform_version <> ?
                ORDER BY id LIMIT ?
            )`,
			transformVersion, limit,
		); err != nil {
			return fmt.Errorf("reset old media transform metadata: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE media_jobs
            SET transform_version = ?, status = 'queued',
                last_error_code = NULL, available_at_ms = ?,
                started_at_ms = NULL, heartbeat_at_ms = NULL,
                lease_expires_at_ms = NULL, attempt_count = 0,
                created_at_ms = ?, finished_at_ms = NULL
            WHERE id IN (
                SELECT id FROM media_jobs
                WHERE transform_version <> ?
                ORDER BY id LIMIT ?
            )`,
			transformVersion, now, now, transformVersion, limit,
		)
		if err != nil {
			return fmt.Errorf("requeue old media transform: %w", err)
		}
		changed, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect media transform reconciliation: %w", err)
		}
		return nil
	})
	return changed, err
}

func (s *Store) ClaimNextMediaJob(
	ctx context.Context,
	leaseDuration time.Duration,
) (thumbnail.Job, bool, error) {
	leaseMS, err := mediaLeaseMilliseconds(leaseDuration)
	if err != nil {
		return thumbnail.Job{}, false, err
	}
	var claimed thumbnail.Job
	var found bool
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		var fingerprint string
		err := tx.QueryRowContext(ctx, `
            WITH per_library AS (
                SELECT library_id, MIN(id) AS job_id
                FROM media_jobs
                WHERE status = 'queued' AND available_at_ms <= ?
                GROUP BY library_id
            ),
            candidate AS (
                SELECT per_library.job_id
                FROM per_library
                LEFT JOIN media_job_library_state AS fairness
                    ON fairness.library_id = per_library.library_id
                ORDER BY COALESCE(fairness.last_claim_sequence, 0),
                         per_library.job_id
                LIMIT 1
            )
            UPDATE media_jobs
            SET status = 'running',
                started_at_ms = COALESCE(started_at_ms, ?),
                heartbeat_at_ms = ?,
                lease_expires_at_ms = ?,
                attempt_count = attempt_count + 1,
                last_error_code = NULL,
                finished_at_ms = NULL
            WHERE id = (SELECT job_id FROM candidate)
              AND status = 'queued'
            RETURNING id, library_id, asset_id, transform_version,
                      source_fingerprint, attempt_count`,
			now, now, now, now+leaseMS,
		).Scan(
			&claimed.ID, &claimed.LibraryID, &claimed.AssetID,
			&claimed.TransformVersion,
			&fingerprint, &claimed.Attempt,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("claim media job: %w", err)
		}
		claimed.SourceFingerprint = media.SourceFingerprint(fingerprint)
		if !claimed.SourceFingerprint.Valid() ||
			claimed.ID <= 0 || claimed.LibraryID <= 0 || claimed.AssetID <= 0 ||
			claimed.TransformVersion <= 0 ||
			claimed.Attempt < 1 || claimed.Attempt > thumbnail.MaxJobAttempts {
			return errors.New("claimed media job is invalid")
		}
		var claimSequence int64
		if err := tx.QueryRowContext(ctx, `
            UPDATE media_job_queue_state
            SET next_claim_sequence = next_claim_sequence + 1
            WHERE singleton_key = 1
            RETURNING next_claim_sequence - 1`,
		).Scan(&claimSequence); err != nil {
			return fmt.Errorf("allocate media fairness sequence: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO media_job_library_state(library_id, last_claim_sequence)
            VALUES (?, ?)
            ON CONFLICT(library_id) DO UPDATE SET
                last_claim_sequence = excluded.last_claim_sequence`,
			claimed.LibraryID, claimSequence,
		); err != nil {
			return fmt.Errorf("advance media job fairness: %w", err)
		}
		found = true
		return nil
	})
	return claimed, found, err
}

func (s *Store) RefreshMediaJobLease(
	ctx context.Context,
	job thumbnail.Job,
	leaseDuration time.Duration,
) error {
	leaseMS, err := mediaLeaseMilliseconds(leaseDuration)
	if err != nil {
		return err
	}
	now := s.nowMS()
	result, err := s.db.ExecContext(ctx, `
        UPDATE media_jobs
        SET heartbeat_at_ms = ?, lease_expires_at_ms = ?
        WHERE id = ? AND transform_version = ?
          AND source_fingerprint = ? AND status = 'running'`,
		now, now+leaseMS, job.ID, job.TransformVersion,
		job.SourceFingerprint.String(),
	)
	if err != nil {
		return fmt.Errorf("refresh media job lease: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect media job lease refresh: %w", err)
	}
	if changed != 1 {
		return thumbnail.ErrJobNotActive
	}
	return nil
}

func (s *Store) RecoverExpiredMediaJobs(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		for recovered := 0; recovered < thumbnail.MediaWorkerCount; recovered++ {
			var job thumbnail.Job
			var fingerprint string
			err := tx.QueryRowContext(ctx, `
                SELECT id, library_id, asset_id, transform_version,
                       source_fingerprint, attempt_count
                FROM media_jobs
                WHERE status = 'running' AND lease_expires_at_ms <= ?
                ORDER BY lease_expires_at_ms, id
                LIMIT 1`,
				now,
			).Scan(
				&job.ID, &job.LibraryID, &job.AssetID,
				&job.TransformVersion,
				&fingerprint, &job.Attempt,
			)
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("find expired media job: %w", err)
			}
			job.SourceFingerprint = media.SourceFingerprint(fingerprint)
			if job.Attempt < thumbnail.MaxJobAttempts {
				if _, err := tx.ExecContext(ctx, `
                    UPDATE media_jobs
                    SET status = 'queued', available_at_ms = ?,
                        heartbeat_at_ms = NULL, lease_expires_at_ms = NULL
                    WHERE id = ? AND transform_version = ?
                      AND source_fingerprint = ? AND status = 'running'`,
					now, job.ID, job.TransformVersion, fingerprint,
				); err != nil {
					return fmt.Errorf("requeue expired media job: %w", err)
				}
				summary.Requeued++
				continue
			}
			if err := failMediaJobTx(
				ctx, tx, job, thumbnail.JobErrorProcessing, now,
			); err != nil {
				return err
			}
			summary.Interrupted++
		}
		var remaining int
		if err := tx.QueryRowContext(ctx, `
            SELECT EXISTS (
                SELECT 1 FROM media_jobs
                WHERE status = 'running' AND lease_expires_at_ms <= ?
            )`,
			now,
		).Scan(&remaining); err != nil {
			return fmt.Errorf("check bounded media recovery: %w", err)
		}
		if remaining != 0 {
			return errors.New("expired media job recovery exceeded worker bound")
		}
		return nil
	})
	return summary, err
}

func (s *Store) FinishMediaJob(
	ctx context.Context,
	job thumbnail.Job,
	result thumbnail.JobResult,
) error {
	if job.ID <= 0 || job.AssetID <= 0 || job.TransformVersion <= 0 ||
		!job.SourceFingerprint.Valid() ||
		job.Attempt < 1 || job.Attempt > thumbnail.MaxJobAttempts {
		return thumbnail.ErrInvalidJob
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		switch result.Outcome {
		case thumbnail.JobSucceeded:
			return finishMediaJobTx(ctx, tx, job, "succeeded", "", now)
		case thumbnail.JobPermanent:
			if !validJobErrorCode(result.Code) {
				return thumbnail.ErrInvalidJob
			}
			return finishMediaJobTx(ctx, tx, job, "failed", result.Code, now)
		case thumbnail.JobStale:
			_, err := tx.ExecContext(ctx, `
                DELETE FROM media_jobs
                WHERE id = ? AND transform_version = ?
                  AND source_fingerprint = ? AND status = 'running'`,
				job.ID, job.TransformVersion, job.SourceFingerprint.String(),
			)
			if err != nil {
				return fmt.Errorf("discard stale media job: %w", err)
			}
			return nil
		case thumbnail.JobRetry:
			if !validJobErrorCode(result.Code) || result.RetryDelay <= 0 ||
				result.RetryDelay > time.Hour {
				return thumbnail.ErrInvalidJob
			}
			if job.Attempt >= thumbnail.MaxJobAttempts {
				return failMediaJobTx(ctx, tx, job, result.Code, now)
			}
			delay := result.RetryDelay * time.Duration(1<<(job.Attempt-1))
			update, err := tx.ExecContext(ctx, `
                UPDATE media_jobs
                SET status = 'queued', last_error_code = ?,
                    available_at_ms = ?, heartbeat_at_ms = NULL,
                    lease_expires_at_ms = NULL, finished_at_ms = NULL
                WHERE id = ? AND transform_version = ?
                  AND source_fingerprint = ? AND status = 'running'
                  AND attempt_count = ?`,
				string(result.Code), now+delay.Milliseconds(),
				job.ID, job.TransformVersion,
				job.SourceFingerprint.String(), job.Attempt,
			)
			if err != nil {
				return fmt.Errorf("retry media job: %w", err)
			}
			return requireOneMediaJobRow(update)
		default:
			return thumbnail.ErrInvalidJob
		}
	})
}

func finishMediaJobTx(
	ctx context.Context,
	tx *sql.Tx,
	job thumbnail.Job,
	status string,
	code thumbnail.JobErrorCode,
	now int64,
) error {
	var storedCode any
	if code != "" {
		storedCode = string(code)
	}
	result, err := tx.ExecContext(ctx, `
        UPDATE media_jobs
        SET status = ?, last_error_code = ?, heartbeat_at_ms = NULL,
            lease_expires_at_ms = NULL, finished_at_ms = ?
        WHERE id = ? AND transform_version = ?
          AND source_fingerprint = ? AND status = 'running'
          AND attempt_count = ?`,
		status, storedCode, now,
		job.ID, job.TransformVersion,
		job.SourceFingerprint.String(), job.Attempt,
	)
	if err != nil {
		return fmt.Errorf("finish media job: %w", err)
	}
	return requireOneMediaJobRow(result)
}

func failMediaJobTx(
	ctx context.Context,
	tx *sql.Tx,
	job thumbnail.Job,
	code thumbnail.JobErrorCode,
	now int64,
) error {
	if err := finishMediaJobTx(ctx, tx, job, "failed", code, now); err != nil {
		return err
	}
	processingCode := media.ErrorProcessingFailed
	if code == thumbnail.JobErrorTimeout {
		processingCode = media.ErrorProcessingTimed
	}
	if _, err := tx.ExecContext(ctx, `
        UPDATE assets
        SET width = NULL, height = NULL, duration_ms = NULL,
            probe_status = 'failed', probe_error_code = ?,
            playback_status = 'unknown'
        WHERE id = ? AND source_fingerprint = ?`,
		string(processingCode), job.AssetID, job.SourceFingerprint.String(),
	); err != nil {
		return fmt.Errorf("mark exhausted media asset failed: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO thumbnails(
            library_id, asset_id, variant, source_fingerprint,
            transform_version, status, error_code
        )
        SELECT library_id, id, 'grid', source_fingerprint,
               ?, 'failed', ?
        FROM assets
        WHERE id = ? AND source_fingerprint = ?
        ON CONFLICT(asset_id, variant) DO UPDATE SET
            source_fingerprint = excluded.source_fingerprint,
            transform_version = excluded.transform_version,
            cache_rel_path = NULL, status = 'failed',
            error_code = excluded.error_code,
            width = NULL, height = NULL, byte_size = NULL,
            created_at_ms = NULL, last_accessed_at_ms = NULL`,
		thumbnail.GridTransformVersion, string(processingCode),
		job.AssetID, job.SourceFingerprint.String(),
	); err != nil {
		return fmt.Errorf("mark exhausted thumbnail failed: %w", err)
	}
	return nil
}

func requireOneMediaJobRow(result sql.Result) error {
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect media job transition: %w", err)
	}
	if changed != 1 {
		return thumbnail.ErrJobNotActive
	}
	return nil
}

func validJobErrorCode(code thumbnail.JobErrorCode) bool {
	switch code {
	case thumbnail.JobErrorInvalidMedia,
		thumbnail.JobErrorUnsupportedMedia,
		thumbnail.JobErrorProcessing,
		thumbnail.JobErrorTimeout,
		thumbnail.JobErrorSource,
		thumbnail.JobErrorCache:
		return true
	default:
		return false
	}
}

func mediaLeaseMilliseconds(duration time.Duration) (int64, error) {
	if duration < time.Millisecond || duration > 10*time.Minute {
		return 0, errors.New(
			"media lease duration must be between one millisecond and ten minutes",
		)
	}
	return duration.Milliseconds(), nil
}
