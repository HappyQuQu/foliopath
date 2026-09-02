package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func (s *Store) EnqueueReconcile(
	ctx context.Context,
	libraryID int64,
	relativeDirectory string,
	debounce time.Duration,
	maximumDebounce time.Duration,
) (scanner.ReconcileJob, error) {
	if libraryID <= 0 || debounce <= 0 || maximumDebounce < debounce {
		return scanner.ReconcileJob{}, scanner.ErrInvalidReconcileTarget
	}
	normalized, err := pathpolicy.Normalize(relativeDirectory)
	if err != nil || normalized != relativeDirectory {
		return scanner.ReconcileJob{}, scanner.ErrInvalidReconcileTarget
	}
	var job scanner.ReconcileJob
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM libraries WHERE id = ?
			)
		`, libraryID).Scan(&exists); err != nil {
			return fmt.Errorf("check reconciliation library: %w", err)
		}
		if exists == 0 {
			return library.ErrNotFound
		}
		var existing int
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM catalog_reconcile_jobs
				WHERE library_id = ? AND relative_dir_path = ?
			)
		`, libraryID, relativeDirectory).Scan(&existing); err != nil {
			return fmt.Errorf("check existing reconciliation: %w", err)
		}
		if existing == 0 {
			var pending int
			if err := tx.QueryRowContext(ctx, `
				SELECT count(*) FROM catalog_reconcile_jobs
			`).Scan(&pending); err != nil {
				return fmt.Errorf("count reconciliation admission: %w", err)
			}
			if pending >= scanner.MaxDirtyDirectories {
				return scanner.ErrReconcileCapacity
			}
		}
		now := s.nowMS()
		available := now + debounce.Milliseconds()
		maximum := now + maximumDebounce.Milliseconds()
		row := tx.QueryRowContext(ctx, `
			INSERT INTO catalog_reconcile_jobs(
				library_id, relative_dir_path, status, requested_revision,
				available_at_ms, attempt_count, created_at_ms, updated_at_ms
			) VALUES (?, ?, 'queued', 1, ?, 0, ?, ?)
			ON CONFLICT(library_id, relative_dir_path) DO UPDATE SET
				requested_revision =
					catalog_reconcile_jobs.requested_revision + 1,
				status = CASE
					WHEN catalog_reconcile_jobs.status = 'failed' THEN 'queued'
					ELSE catalog_reconcile_jobs.status
				END,
				claimed_revision = CASE
					WHEN catalog_reconcile_jobs.status = 'failed' THEN NULL
					ELSE catalog_reconcile_jobs.claimed_revision
				END,
				lease_expires_at_ms = CASE
					WHEN catalog_reconcile_jobs.status = 'failed' THEN NULL
					ELSE catalog_reconcile_jobs.lease_expires_at_ms
				END,
				attempt_count = CASE
					WHEN catalog_reconcile_jobs.status = 'failed' THEN 0
					ELSE catalog_reconcile_jobs.attempt_count
				END,
				last_error_code = CASE
					WHEN catalog_reconcile_jobs.status = 'failed' THEN NULL
					ELSE catalog_reconcile_jobs.last_error_code
				END,
				available_at_ms = CASE
					WHEN catalog_reconcile_jobs.status = 'running'
						THEN catalog_reconcile_jobs.available_at_ms
					WHEN ? < catalog_reconcile_jobs.created_at_ms + ?
						THEN ?
					ELSE catalog_reconcile_jobs.created_at_ms + ?
				END,
				updated_at_ms = ?
			RETURNING
				id, library_id, relative_dir_path, status,
				requested_revision, claimed_revision, available_at_ms,
				lease_expires_at_ms, attempt_count, last_error_code,
				created_at_ms, updated_at_ms
		`,
			libraryID,
			relativeDirectory,
			available,
			now,
			now,
			available,
			maximumDebounce.Milliseconds(),
			available,
			maximumDebounce.Milliseconds(),
			now,
		)
		mapped, err := scanReconcileJob(row)
		if err != nil {
			return fmt.Errorf("enqueue reconciliation: %w", err)
		}
		if mapped.AvailableAtMS > maximum {
			return errors.New("reconciliation debounce exceeded maximum")
		}
		job = mapped
		return nil
	})
	return job, err
}

func (s *Store) ClaimNextReconcile(
	ctx context.Context,
	leaseDuration time.Duration,
) (scanner.ReconcileJob, bool, error) {
	leaseMS, err := scanLeaseMilliseconds(leaseDuration)
	if err != nil {
		return scanner.ReconcileJob{}, false, err
	}
	var (
		job   scanner.ReconcileJob
		found bool
	)
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		row := tx.QueryRowContext(ctx, `
			UPDATE catalog_reconcile_jobs
			SET status = 'running',
			    claimed_revision = requested_revision,
			    lease_expires_at_ms = ?,
			    attempt_count = attempt_count + 1,
			    last_error_code = NULL,
			    updated_at_ms = ?
			WHERE id = (
				SELECT candidate.id
				FROM catalog_reconcile_jobs AS candidate
				WHERE candidate.status = 'queued'
				  AND candidate.available_at_ms <= ?
				  AND candidate.attempt_count < ?
				  AND NOT EXISTS (
				      SELECT 1
				      FROM scan_runs
				      WHERE scan_runs.library_id = candidate.library_id
				        AND scan_runs.status IN ('queued', 'running')
				  )
				  AND NOT EXISTS (
				      SELECT 1
				      FROM catalog_reconcile_jobs AS active
				      WHERE active.library_id = candidate.library_id
				        AND active.status = 'running'
				  )
				ORDER BY candidate.available_at_ms, candidate.library_id, candidate.id
				LIMIT 1
			)
			  AND status = 'queued'
			RETURNING
				id, library_id, relative_dir_path, status,
				requested_revision, claimed_revision, available_at_ms,
				lease_expires_at_ms, attempt_count, last_error_code,
				created_at_ms, updated_at_ms
		`,
			now+leaseMS,
			now,
			now,
			scanner.MaxReconcileAttempts,
		)
		mapped, err := scanReconcileJob(row)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("claim reconciliation: %w", err)
		}
		job = mapped
		found = true
		return nil
	})
	return job, found, err
}

func (s *Store) RefreshReconcileLease(
	ctx context.Context,
	job scanner.ReconcileJob,
	leaseDuration time.Duration,
) (scanner.ReconcileJob, error) {
	if job.ID <= 0 || job.ClaimedRevision == nil {
		return scanner.ReconcileJob{}, scanner.ErrReconcileNotFound
	}
	leaseMS, err := scanLeaseMilliseconds(leaseDuration)
	if err != nil {
		return scanner.ReconcileJob{}, err
	}
	var refreshed scanner.ReconcileJob
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		row := tx.QueryRowContext(ctx, `
			UPDATE catalog_reconcile_jobs
			SET lease_expires_at_ms = ?, updated_at_ms = ?
			WHERE id = ?
			  AND status = 'running'
			  AND claimed_revision = ?
			RETURNING
				id, library_id, relative_dir_path, status,
				requested_revision, claimed_revision, available_at_ms,
				lease_expires_at_ms, attempt_count, last_error_code,
				created_at_ms, updated_at_ms
		`, now+leaseMS, now, job.ID, *job.ClaimedRevision)
		mapped, err := scanReconcileJob(row)
		if errors.Is(err, sql.ErrNoRows) {
			return scanner.ErrReconcileNotActive
		}
		if err != nil {
			return fmt.Errorf("refresh reconciliation lease: %w", err)
		}
		refreshed = mapped
		return nil
	})
	return refreshed, err
}

func (s *Store) RecoverExpiredReconciles(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	var summary jobs.RecoverySummary
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		for recovered := 0; recovered < scanner.MaxConcurrentReconciles; recovered++ {
			var (
				id       int64
				attempts int64
			)
			err := tx.QueryRowContext(ctx, `
				SELECT id, attempt_count
				FROM catalog_reconcile_jobs
				WHERE status = 'running' AND lease_expires_at_ms <= ?
				ORDER BY lease_expires_at_ms, id
				LIMIT 1
			`, now).Scan(&id, &attempts)
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("find expired reconciliation: %w", err)
			}
			if attempts >= scanner.MaxReconcileAttempts {
				if _, err := tx.ExecContext(ctx, `
					UPDATE catalog_reconcile_jobs
					SET status = 'failed',
					    claimed_revision = NULL,
					    lease_expires_at_ms = NULL,
					    last_error_code = 'internal_error',
					    updated_at_ms = ?
					WHERE id = ? AND status = 'running'
				`, now, id); err != nil {
					return fmt.Errorf("fail expired reconciliation: %w", err)
				}
				summary.Interrupted++
				continue
			}
			backoff := reconcileBackoffMS(attempts)
			if _, err := tx.ExecContext(ctx, `
				UPDATE catalog_reconcile_jobs
				SET status = 'queued',
				    claimed_revision = NULL,
				    lease_expires_at_ms = NULL,
				    available_at_ms = ?,
				    last_error_code = 'internal_error',
				    updated_at_ms = ?
				WHERE id = ? AND status = 'running'
			`, now+backoff, now, id); err != nil {
				return fmt.Errorf("requeue expired reconciliation: %w", err)
			}
			summary.Requeued++
		}
		var remaining int
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM catalog_reconcile_jobs
				WHERE status = 'running' AND lease_expires_at_ms <= ?
			)
		`, now).Scan(&remaining); err != nil {
			return fmt.Errorf("check bounded reconciliation recovery: %w", err)
		}
		if remaining != 0 {
			return errors.New("expired reconciliation recovery exceeded concurrency bound")
		}
		return nil
	})
	return summary, err
}

func reconcileBackoffMS(attempt int64) int64 {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > scanner.MaxReconcileAttempts {
		attempt = scanner.MaxReconcileAttempts
	}
	return int64(time.Second/time.Millisecond) << (attempt - 1)
}

func (s *Store) CommitDirectoryReconcile(
	ctx context.Context,
	job scanner.ReconcileJob,
	entries []scanner.CatalogEntry,
) (scanner.ReconcileCommitResult, error) {
	if job.ID <= 0 || job.LibraryID <= 0 || job.ClaimedRevision == nil ||
		job.Status != scanner.ReconcileRunning ||
		len(entries) > scanner.MaxReconcileEntries {
		return scanner.ReconcileCommitResult{}, scanner.ErrReconcileNotActive
	}
	var committed scanner.ReconcileCommitResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := getRunningReconcileTx(ctx, tx, job)
		if err != nil {
			return err
		}
		var generation int64
		if err := tx.QueryRowContext(ctx, `
			SELECT current_generation
			FROM libraries
			WHERE id = ?
		`, job.LibraryID).Scan(&generation); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return library.ErrNotFound
			}
			return fmt.Errorf("read reconciliation generation: %w", err)
		}
		if generation < 1 {
			return scanner.ErrInvalidEntry
		}

		targetID, targetParent, oldRecursive, err :=
			readReconcileCountTargetTx(
				ctx,
				tx,
				job.LibraryID,
				job.RelativeDirectory,
			)
		if err != nil {
			return err
		}

		existingDirectories, err := listDirectDirectoryState(
			ctx,
			tx,
			job.LibraryID,
			targetID,
		)
		if err != nil {
			return err
		}
		existingAssets, err := listDirectAssetState(
			ctx,
			tx,
			job.LibraryID,
			targetID,
		)
		if err != nil {
			return err
		}

		directories := make([]scanner.CatalogEntry, 0, len(entries))
		assets := make([]scanner.CatalogEntry, 0, len(entries))
		seen := make(map[string]struct{}, len(entries))
		for _, entry := range entries {
			normalized, err := pathpolicy.Normalize(entry.RelativePath)
			if err != nil || normalized != entry.RelativePath ||
				entry.ParentPath != job.RelativeDirectory ||
				entry.Name == "" {
				return scanner.ErrInvalidEntry
			}
			if _, duplicate := seen[entry.RelativePath]; duplicate {
				return scanner.ErrInvalidEntry
			}
			seen[entry.RelativePath] = struct{}{}
			switch entry.Kind {
			case scanner.CatalogEntryDirectory:
				directories = append(directories, entry)
			case scanner.CatalogEntryAsset:
				assets = append(assets, entry)
			default:
				return scanner.ErrInvalidEntry
			}
		}

		now := s.nowMS()
		run := scanner.ScanRun{
			LibraryID:   job.LibraryID,
			Generation:  generation,
			CreatedAtMS: now,
		}
		for _, entry := range directories {
			previous, exists := existingDirectories[entry.RelativePath]
			if !exists {
				committed.Changed = true
				committed.NewDirectories = append(
					committed.NewDirectories,
					entry.RelativePath,
				)
			} else if previous.mtimeNS != entry.MTimeNS ||
				previous.name != entry.Name {
				committed.Changed = true
			}
			if _, err := upsertDirectory(ctx, tx, run, entry); err != nil {
				return err
			}
			delete(existingDirectories, entry.RelativePath)
		}
		mediaAdmissions := make([]mediaJobAdmission, 0, len(assets))
		for _, entry := range assets {
			previous, exists := existingAssets[entry.RelativePath]
			if !exists || !previous.matches(entry) {
				committed.Changed = true
			}
			_, admission, err := upsertAsset(ctx, tx, run, entry)
			if err != nil {
				return err
			}
			if admission != nil {
				mediaAdmissions = append(mediaAdmissions, *admission)
			}
			delete(existingAssets, entry.RelativePath)
		}
		if err := admitMediaJobsBatch(ctx, tx, run, mediaAdmissions); err != nil {
			return err
		}

		for _, stale := range existingAssets {
			if err := scheduleAssetCacheDeletionTx(ctx, tx, stale.id, now); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM assets
				WHERE id = ? AND library_id = ?
			`, stale.id, job.LibraryID); err != nil {
				return fmt.Errorf("delete stale direct asset: %w", err)
			}
			committed.Changed = true
		}
		for _, stale := range existingDirectories {
			if err := scheduleDirectoryCacheDeletionTx(
				ctx,
				tx,
				job.LibraryID,
				stale.relativePath,
				now,
			); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM directories
				WHERE id = ? AND library_id = ?
			`, stale.id, job.LibraryID); err != nil {
				return fmt.Errorf("delete stale direct directory: %w", err)
			}
			committed.Changed = true
		}

		if err := updateReconcileCountsTx(
			ctx,
			tx,
			job.LibraryID,
			targetID,
			targetParent,
			oldRecursive,
		); err != nil {
			return err
		}

		if current.RequestedRevision == *current.ClaimedRevision {
			result, err := tx.ExecContext(ctx, `
				DELETE FROM catalog_reconcile_jobs
				WHERE id = ? AND status = 'running' AND claimed_revision = ?
			`, current.ID, *current.ClaimedRevision)
			if err != nil {
				return fmt.Errorf("complete reconciliation job: %w", err)
			}
			rows, err := result.RowsAffected()
			if err != nil || rows != 1 {
				return scanner.ErrReconcileNotActive
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
				UPDATE catalog_reconcile_jobs
				SET status = 'queued',
				    claimed_revision = NULL,
				    lease_expires_at_ms = NULL,
				    attempt_count = 0,
				    last_error_code = NULL,
				    available_at_ms = ?,
				    updated_at_ms = ?
				WHERE id = ? AND status = 'running' AND claimed_revision = ?
			`,
				now+scanner.ReconcileDebounce.Milliseconds(),
				now,
				current.ID,
				*current.ClaimedRevision,
			); err != nil {
				return fmt.Errorf("requeue changed reconciliation job: %w", err)
			}
			committed.Requeued = true
		}

		contentIncrement := 0
		if committed.Changed {
			contentIncrement = 1
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE libraries
			SET automatic_discovery_status = 'active',
			    automatic_discovery_error_code = NULL,
			    last_automatic_discovery_at_ms = ?,
			    content_revision = content_revision + ?,
			    revision = revision + 1,
			    updated_at_ms = ?
			WHERE id = ?
		`, now, contentIncrement, now, job.LibraryID); err != nil {
			return fmt.Errorf("publish library reconciliation state: %w", err)
		}
		if committed.Changed {
			if _, err := tx.ExecContext(ctx, `
				UPDATE catalog_search_state
				SET content_revision = content_revision + 1
				WHERE singleton_key = 1
			`); err != nil {
				return fmt.Errorf("publish catalog content revision: %w", err)
			}
		}
		return nil
	})
	return committed, err
}

func (s *Store) FailDirectoryReconcile(
	ctx context.Context,
	job scanner.ReconcileJob,
	errorCode string,
) error {
	if job.ID <= 0 || job.ClaimedRevision == nil {
		return scanner.ErrReconcileNotActive
	}
	switch errorCode {
	case "source_unavailable", "source_unreadable", "source_changed",
		"watch_overflow", "watch_resource_limit", "internal_error":
	default:
		return scanner.ErrInvalidEntry
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := getRunningReconcileTx(ctx, tx, job)
		if err != nil {
			return err
		}
		now := s.nowMS()
		status := "queued"
		available := now + reconcileBackoffMS(current.AttemptCount)
		if current.AttemptCount >= scanner.MaxReconcileAttempts {
			status = "failed"
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE catalog_reconcile_jobs
			SET status = ?,
			    claimed_revision = NULL,
			    lease_expires_at_ms = NULL,
			    available_at_ms = ?,
			    last_error_code = ?,
			    updated_at_ms = ?
			WHERE id = ? AND status = 'running' AND claimed_revision = ?
		`,
			status,
			available,
			errorCode,
			now,
			current.ID,
			*current.ClaimedRevision,
		); err != nil {
			return fmt.Errorf("record reconciliation failure: %w", err)
		}
		libraryError := "internal_error"
		switch errorCode {
		case "source_unavailable", "source_unreadable", "source_changed":
			libraryError = "source_unavailable"
		case "watch_overflow":
			libraryError = "watch_overflow"
		case "watch_resource_limit":
			libraryError = "watch_resource_limit"
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE libraries
			SET automatic_discovery_status = 'degraded',
			    automatic_discovery_error_code = ?,
			    revision = revision + 1,
			    updated_at_ms = ?
			WHERE id = ?
		`, libraryError, now, job.LibraryID); err != nil {
			return fmt.Errorf("degrade automatic discovery: %w", err)
		}
		return nil
	})
}

func (s *Store) SetAutomaticDiscoveryState(
	ctx context.Context,
	libraryID int64,
	status string,
	errorCode string,
) error {
	if libraryID <= 0 {
		return library.ErrNotFound
	}
	if _, err := library.ValidateAutomaticDiscoveryState(status, errorCode); err != nil {
		return err
	}
	var nullableError any
	if errorCode != "" {
		nullableError = errorCode
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE libraries
			SET automatic_discovery_status = ?,
			    automatic_discovery_error_code = ?,
			    revision = CASE
			        WHEN automatic_discovery_status <> ?
			          OR COALESCE(automatic_discovery_error_code, '') <> ?
			        THEN revision + 1
			        ELSE revision
			    END,
			    updated_at_ms = CASE
			        WHEN automatic_discovery_status <> ?
			          OR COALESCE(automatic_discovery_error_code, '') <> ?
			        THEN ?
			        ELSE updated_at_ms
			    END
			WHERE id = ?
		`,
			status,
			nullableError,
			status,
			errorCode,
			status,
			errorCode,
			s.nowMS(),
			libraryID,
		)
		if err != nil {
			return fmt.Errorf("set automatic discovery state: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return library.ErrNotFound
		}
		return nil
	})
}

type directDirectoryState struct {
	id           int64
	relativePath string
	name         string
	mtimeNS      int64
}

func listDirectDirectoryState(
	ctx context.Context,
	tx *sql.Tx,
	libraryID int64,
	parentID int64,
) (map[string]directDirectoryState, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, relative_path, name, mtime_ns
		FROM directories
		WHERE library_id = ? AND parent_id = ?
	`, libraryID, parentID)
	if err != nil {
		return nil, fmt.Errorf("list direct directory state: %w", err)
	}
	defer rows.Close()
	result := make(map[string]directDirectoryState)
	for rows.Next() {
		var item directDirectoryState
		if err := rows.Scan(
			&item.id,
			&item.relativePath,
			&item.name,
			&item.mtimeNS,
		); err != nil {
			return nil, err
		}
		if len(result) >= scanner.MaxReconcileEntries {
			return nil, scanner.ErrBatchTooLarge
		}
		result[item.relativePath] = item
	}
	return result, rows.Err()
}

type directAssetState struct {
	id           int64
	relativePath string
	name         string
	kind         string
	format       string
	mimeType     string
	sizeBytes    int64
	mtimeNS      int64
}

func (item directAssetState) matches(entry scanner.CatalogEntry) bool {
	return item.name == entry.Name &&
		item.kind == string(entry.AssetKind) &&
		item.format == string(entry.MediaFormat) &&
		item.mimeType == entry.MIMEType &&
		item.sizeBytes == entry.SizeBytes &&
		item.mtimeNS == entry.MTimeNS
}

func listDirectAssetState(
	ctx context.Context,
	tx *sql.Tx,
	libraryID int64,
	directoryID int64,
) (map[string]directAssetState, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, relative_path, name, kind, media_format,
		       mime_type, size_bytes, mtime_ns
		FROM assets
		WHERE library_id = ? AND directory_id = ?
	`, libraryID, directoryID)
	if err != nil {
		return nil, fmt.Errorf("list direct asset state: %w", err)
	}
	defer rows.Close()
	result := make(map[string]directAssetState)
	for rows.Next() {
		var item directAssetState
		if err := rows.Scan(
			&item.id,
			&item.relativePath,
			&item.name,
			&item.kind,
			&item.format,
			&item.mimeType,
			&item.sizeBytes,
			&item.mtimeNS,
		); err != nil {
			return nil, err
		}
		if len(result) >= scanner.MaxReconcileEntries {
			return nil, scanner.ErrBatchTooLarge
		}
		result[item.relativePath] = item
	}
	return result, rows.Err()
}

func scheduleAssetCacheDeletionTx(
	ctx context.Context,
	tx *sql.Tx,
	assetID int64,
	now int64,
) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO cache_deletions(
			library_id, cache_rel_path, byte_size, created_at_ms
		)
		SELECT library_id, cache_rel_path, byte_size, ?
		FROM thumbnails
		WHERE asset_id = ? AND status = 'ready'
	`, now, assetID); err != nil {
		return fmt.Errorf("schedule removed asset cache cleanup: %w", err)
	}
	return nil
}

func scheduleDirectoryCacheDeletionTx(
	ctx context.Context,
	tx *sql.Tx,
	libraryID int64,
	relativeDirectory string,
	now int64,
) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO cache_deletions(
			library_id, cache_rel_path, byte_size, created_at_ms
		)
		SELECT thumbnail.library_id, thumbnail.cache_rel_path,
		       thumbnail.byte_size, ?
		FROM thumbnails AS thumbnail
		JOIN assets AS asset ON asset.id = thumbnail.asset_id
		WHERE asset.library_id = ?
		  AND thumbnail.status = 'ready'
		  AND (
		      asset.relative_path = ?
		      OR instr(asset.relative_path, ? || '/') = 1
		  )
	`, now, libraryID, relativeDirectory, relativeDirectory); err != nil {
		return fmt.Errorf("schedule removed directory cache cleanup: %w", err)
	}
	return nil
}

func getRunningReconcileTx(
	ctx context.Context,
	tx *sql.Tx,
	job scanner.ReconcileJob,
) (scanner.ReconcileJob, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT
			id, library_id, relative_dir_path, status,
			requested_revision, claimed_revision, available_at_ms,
			lease_expires_at_ms, attempt_count, last_error_code,
			created_at_ms, updated_at_ms
		FROM catalog_reconcile_jobs
		WHERE id = ? AND library_id = ?
		  AND status = 'running' AND claimed_revision = ?
	`, job.ID, job.LibraryID, *job.ClaimedRevision)
	current, err := scanReconcileJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return scanner.ReconcileJob{}, scanner.ErrReconcileNotActive
	}
	if err != nil {
		return scanner.ReconcileJob{}, err
	}
	return current, nil
}

func scanReconcileJob(row interface{ Scan(...any) error }) (scanner.ReconcileJob, error) {
	var (
		job            scanner.ReconcileJob
		status         string
		claimed, lease sql.NullInt64
		lastError      sql.NullString
	)
	if err := row.Scan(
		&job.ID,
		&job.LibraryID,
		&job.RelativeDirectory,
		&status,
		&job.RequestedRevision,
		&claimed,
		&job.AvailableAtMS,
		&lease,
		&job.AttemptCount,
		&lastError,
		&job.CreatedAtMS,
		&job.UpdatedAtMS,
	); err != nil {
		return scanner.ReconcileJob{}, err
	}
	job.Status = scanner.ReconcileStatus(status)
	switch job.Status {
	case scanner.ReconcileQueued, scanner.ReconcileRunning, scanner.ReconcileFailed:
	default:
		return scanner.ReconcileJob{}, errors.New("invalid reconciliation status")
	}
	if claimed.Valid {
		value := claimed.Int64
		job.ClaimedRevision = &value
	}
	if lease.Valid {
		value := lease.Int64
		job.LeaseExpiresAtMS = &value
	}
	if lastError.Valid {
		job.LastErrorCode = lastError.String
	}
	return job, nil
}
