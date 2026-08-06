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

const MaxStoryboardAdmissionBatch = 128

func (s *Store) GetMediaProcessingProgress(
	ctx context.Context,
	libraryID int64,
) (thumbnail.ProcessingProgress, bool, error) {
	var progress thumbnail.ProcessingProgress
	var found int
	err := s.db.QueryRowContext(ctx, `
        WITH library_jobs AS (
            SELECT job.variant, job.status, asset.kind
            FROM media_jobs AS job
            JOIN assets AS asset
              ON asset.library_id = job.library_id
             AND asset.id = job.asset_id
            WHERE job.library_id = ?
        )
        SELECT
            EXISTS(SELECT 1 FROM libraries WHERE id = ?),
            COALESCE(SUM(variant = 'grid' AND status = 'queued'), 0),
            COALESCE(SUM(variant = 'grid' AND status = 'running'), 0),
            COALESCE(SUM(variant = 'grid' AND status = 'succeeded'), 0),
            COALESCE(SUM(variant = 'grid' AND status = 'failed'), 0),
            COALESCE(SUM(variant = 'storyboard' AND status = 'queued'), 0),
            COALESCE(SUM(variant = 'storyboard' AND status = 'running'), 0),
            COALESCE(SUM(variant = 'storyboard' AND status = 'succeeded'), 0),
            COALESCE(SUM(variant = 'storyboard' AND status = 'failed'), 0),
			(
				SELECT COUNT(*)
				FROM assets AS video
				JOIN media_jobs AS grid
				  ON grid.asset_id = video.id AND grid.variant = 'grid'
				WHERE video.library_id = ? AND video.kind = 'video'
				  AND (
					grid.status IN ('queued', 'running')
					OR (
						grid.status = 'succeeded'
						AND video.probe_status = 'ready'
						AND video.duration_ms >= ?
						AND NOT EXISTS (
							SELECT 1 FROM media_jobs AS storyboard
							WHERE storyboard.asset_id = video.id
							  AND storyboard.variant = 'storyboard'
						)
					)
				)
			)
        FROM library_jobs`,
		libraryID,
		libraryID,
		libraryID,
		thumbnail.StoryboardMinimumDurationMS,
	).Scan(
		&found,
		&progress.Grid.Queued,
		&progress.Grid.Running,
		&progress.Grid.Succeeded,
		&progress.Grid.Failed,
		&progress.Storyboard.Queued,
		&progress.Storyboard.Running,
		&progress.Storyboard.Succeeded,
		&progress.Storyboard.Failed,
		&progress.StoryboardPendingEligibility,
	)
	if err != nil {
		return thumbnail.ProcessingProgress{}, false,
			fmt.Errorf("get media processing progress: %w", err)
	}
	return progress, found != 0, nil
}

func (s *Store) AdmitStoryboardJobs(
	ctx context.Context,
	limit int,
) (int64, error) {
	if limit < 1 || limit > MaxStoryboardAdmissionBatch {
		return 0, thumbnail.ErrInvalidJob
	}
	var admitted int64
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            INSERT OR IGNORE INTO media_jobs(
                library_id, asset_id, variant, priority, transform_version,
                source_fingerprint, status, available_at_ms, attempt_count,
                created_at_ms
            )
            SELECT
                asset.library_id, asset.id, 'storyboard', 100, ?,
                asset.source_fingerprint, 'queued', ?, 0, ?
            FROM assets AS asset
            JOIN thumbnails AS grid
              ON grid.asset_id = asset.id
             AND grid.variant = 'grid'
             AND grid.status = 'ready'
             AND grid.source_fingerprint = asset.source_fingerprint
            WHERE asset.kind = 'video'
              AND asset.probe_status = 'ready'
              AND asset.duration_ms >= ?
              AND NOT EXISTS (
                  SELECT 1 FROM media_jobs AS existing
                  WHERE existing.asset_id = asset.id
                    AND existing.variant = 'storyboard'
              )
            ORDER BY asset.id
            LIMIT ?`,
			thumbnail.StoryboardTransformVersion,
			s.nowMS(),
			s.nowMS(),
			thumbnail.StoryboardMinimumDurationMS,
			limit,
		)
		if err != nil {
			return fmt.Errorf("admit storyboard jobs: %w", err)
		}
		admitted, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect storyboard admission: %w", err)
		}
		return nil
	})
	return admitted, err
}

func (s *Store) ReconcileMediaJobTransform(
	ctx context.Context,
	transformVersion int,
	limit int,
) (int64, error) {
	return s.reconcileMediaJobTransform(
		ctx,
		thumbnail.VariantGrid,
		transformVersion,
		limit,
	)
}

func (s *Store) ReconcileStoryboardJobTransform(
	ctx context.Context,
	transformVersion int,
	limit int,
) (int64, error) {
	return s.reconcileMediaJobTransform(
		ctx,
		thumbnail.VariantStoryboard,
		transformVersion,
		limit,
	)
}

func (s *Store) reconcileMediaJobTransform(
	ctx context.Context,
	variant thumbnail.Variant,
	transformVersion int,
	limit int,
) (int64, error) {
	if transformVersion <= 0 || limit < 1 || limit > 256 {
		return 0, thumbnail.ErrInvalidJob
	}
	if variant != thumbnail.VariantGrid &&
		variant != thumbnail.VariantStoryboard {
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
            JOIN media_jobs AS job
              ON job.asset_id = thumbnail.asset_id
             AND job.variant = thumbnail.variant
            WHERE job.variant = ?
              AND job.transform_version <> ?
              AND thumbnail.status = 'ready'
              AND job.id IN (
                  SELECT id FROM media_jobs
                  WHERE variant = ? AND transform_version <> ?
                  ORDER BY id LIMIT ?
              )`,
			now,
			string(variant),
			transformVersion,
			string(variant),
			transformVersion,
			limit,
		); err != nil {
			return fmt.Errorf("schedule old transform cache cleanup: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            DELETE FROM thumbnails
            WHERE variant = ?
              AND asset_id IN (
                SELECT asset_id FROM media_jobs
                WHERE variant = ? AND transform_version <> ?
                ORDER BY id LIMIT ?
            )`,
			string(variant), string(variant), transformVersion, limit,
		); err != nil {
			return fmt.Errorf("invalidate old thumbnail transform: %w", err)
		}
		if variant == thumbnail.VariantGrid {
			if _, err := tx.ExecContext(ctx, `
                UPDATE assets
                SET width = NULL, height = NULL, duration_ms = NULL,
                    probe_status = 'pending', probe_error_code = NULL,
                    playback_status = 'unknown'
                WHERE id IN (
                    SELECT asset_id FROM media_jobs
                    WHERE variant = ? AND transform_version <> ?
                    ORDER BY id LIMIT ?
                )`,
				string(variant), transformVersion, limit,
			); err != nil {
				return fmt.Errorf("reset old media transform metadata: %w", err)
			}
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
                WHERE variant = ? AND transform_version <> ?
                ORDER BY id LIMIT ?
            )`,
			transformVersion,
			now,
			now,
			string(variant),
			transformVersion,
			limit,
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
		var fingerprint, variant string
		err := tx.QueryRowContext(ctx, `
            WITH eligible_priority AS (
                SELECT MIN(priority) AS priority
                FROM media_jobs
                WHERE status = 'queued' AND available_at_ms <= ?
                  AND (
                      priority = 0
                      OR NOT EXISTS (
                          SELECT 1 FROM media_jobs AS active_storyboard
                          WHERE active_storyboard.status = 'running'
                            AND active_storyboard.priority = 100
                      )
                  )
            ),
            per_library AS (
                SELECT jobs.library_id, jobs.priority, MIN(jobs.id) AS job_id
                FROM media_jobs AS jobs
                JOIN eligible_priority
                  ON eligible_priority.priority = jobs.priority
                WHERE jobs.status = 'queued' AND jobs.available_at_ms <= ?
                  AND (
                      jobs.priority = 0
                      OR NOT EXISTS (
                          SELECT 1 FROM media_jobs AS active_storyboard
                          WHERE active_storyboard.status = 'running'
                            AND active_storyboard.priority = 100
                      )
                  )
                GROUP BY jobs.library_id, jobs.priority
            ),
            candidate AS (
                SELECT per_library.job_id
                FROM media_jobs
                JOIN per_library ON per_library.job_id = media_jobs.id
                LEFT JOIN media_job_library_state AS fairness
                    ON fairness.library_id = per_library.library_id
                   AND fairness.priority = per_library.priority
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
            RETURNING id, library_id, asset_id, variant, transform_version,
                      source_fingerprint, attempt_count`,
			now, now, now, now, now+leaseMS,
		).Scan(
			&claimed.ID, &claimed.LibraryID, &claimed.AssetID,
			&variant, &claimed.TransformVersion,
			&fingerprint, &claimed.Attempt,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("claim media job: %w", err)
		}
		claimed.Variant = thumbnail.Variant(variant)
		claimed.SourceFingerprint = media.SourceFingerprint(fingerprint)
		if !claimed.SourceFingerprint.Valid() ||
			claimed.ID <= 0 || claimed.LibraryID <= 0 || claimed.AssetID <= 0 ||
			(claimed.Variant != thumbnail.VariantGrid &&
				claimed.Variant != thumbnail.VariantStoryboard) ||
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
            INSERT INTO media_job_library_state(
                library_id, priority, last_claim_sequence
            )
            SELECT library_id, priority, ?
            FROM media_jobs
            WHERE id = ?
            ON CONFLICT(library_id, priority) DO UPDATE SET
                last_claim_sequence = excluded.last_claim_sequence`,
			claimSequence, claimed.ID,
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
			var fingerprint, variant string
			err := tx.QueryRowContext(ctx, `
                SELECT id, library_id, asset_id, variant, transform_version,
                       source_fingerprint, attempt_count
                FROM media_jobs
                WHERE status = 'running' AND lease_expires_at_ms <= ?
                ORDER BY lease_expires_at_ms, id
                LIMIT 1`,
				now,
			).Scan(
				&job.ID, &job.LibraryID, &job.AssetID,
				&variant, &job.TransformVersion,
				&fingerprint, &job.Attempt,
			)
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("find expired media job: %w", err)
			}
			job.Variant = thumbnail.Variant(variant)
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
		(job.Variant != thumbnail.VariantGrid &&
			job.Variant != thumbnail.VariantStoryboard) ||
		!job.SourceFingerprint.Valid() ||
		job.Attempt < 1 || job.Attempt > thumbnail.MaxJobAttempts {
		return thumbnail.ErrInvalidJob
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		switch result.Outcome {
		case thumbnail.JobSucceeded:
			if err := finishMediaJobTx(ctx, tx, job, "succeeded", "", now); err != nil {
				return err
			}
			return recordMediaJobAttemptTx(ctx, tx, job, result, now)
		case thumbnail.JobPermanent:
			if !validJobErrorCode(result.Code) {
				return thumbnail.ErrInvalidJob
			}
			if err := finishMediaJobTx(ctx, tx, job, "failed", result.Code, now); err != nil {
				return err
			}
			return recordMediaJobAttemptTx(ctx, tx, job, result, now)
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
				if err := failMediaJobTx(ctx, tx, job, result.Code, now); err != nil {
					return err
				}
				return recordMediaJobAttemptTx(ctx, tx, job, result, now)
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
			if err := requireOneMediaJobRow(update); err != nil {
				return err
			}
			return recordMediaJobAttemptTx(ctx, tx, job, result, now)
		default:
			return thumbnail.ErrInvalidJob
		}
	})
}

func recordMediaJobAttemptTx(
	ctx context.Context,
	tx *sql.Tx,
	job thumbnail.Job,
	result thumbnail.JobResult,
	now int64,
) error {
	durationMS := result.Duration.Milliseconds()
	if durationMS < 0 {
		return thumbnail.ErrInvalidJob
	}
	var stage, reason, tool any
	if result.Diagnostic.Stage != "" {
		stage = string(result.Diagnostic.Stage)
	}
	if result.Diagnostic.Reason != "" {
		reason = string(result.Diagnostic.Reason)
	}
	if result.Diagnostic.Tool != "" {
		tool = result.Diagnostic.Tool
	}
	var exitCode any
	if result.Diagnostic.ExitCode != nil {
		exitCode = *result.Diagnostic.ExitCode
	}
	outcome := string(result.Outcome)
	if result.Outcome == thumbnail.JobRetry && job.Attempt >= thumbnail.MaxJobAttempts {
		outcome = string(thumbnail.JobPermanent)
	}
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO media_job_attempts(
            job_id, attempt_number, outcome, stage, reason_code, tool,
            exit_code, duration_ms, finished_at_ms
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Attempt, outcome, stage, reason, tool,
		exitCode, durationMS, now,
	); err != nil {
		return fmt.Errorf("record media job attempt: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
        DELETE FROM media_job_attempts
        WHERE job_id = ? AND id NOT IN (
            SELECT id FROM media_job_attempts
            WHERE job_id = ? ORDER BY id DESC LIMIT 10
        )`, job.ID, job.ID); err != nil {
		return fmt.Errorf("trim media job attempts: %w", err)
	}
	return nil
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
	if job.Variant == thumbnail.VariantStoryboard {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO thumbnails(
                library_id, asset_id, variant, source_fingerprint,
                transform_version, status, error_code
            )
            SELECT library_id, id, 'storyboard', source_fingerprint,
                   ?, 'failed', ?
            FROM assets
            WHERE id = ? AND source_fingerprint = ?
            ON CONFLICT(asset_id, variant) DO UPDATE SET
                source_fingerprint = excluded.source_fingerprint,
                transform_version = excluded.transform_version,
                cache_rel_path = NULL, status = 'failed',
                error_code = excluded.error_code,
                width = NULL, height = NULL, byte_size = NULL,
                created_at_ms = NULL, last_accessed_at_ms = NULL,
                frame_count = NULL, sprite_columns = NULL, sprite_rows = NULL,
                cell_width = NULL, cell_height = NULL`,
			job.TransformVersion,
			string(processingCode),
			job.AssetID,
			job.SourceFingerprint.String(),
		); err != nil {
			return fmt.Errorf("mark exhausted storyboard failed: %w", err)
		}
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
        UPDATE assets
        SET width = NULL, height = NULL, duration_ms = NULL,
            probe_status = 'failed', probe_error_code = ?,
            playback_status = 'unknown'
        WHERE id = ? AND source_fingerprint = ?
          AND probe_status <> 'ready'`,
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
