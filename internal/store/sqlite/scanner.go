package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type rowScanner interface {
	Scan(...any) error
}

const admissionScanColumns = `
    id, library_id, generation, trigger_kind, status,
    discovered_directories, discovered_assets, skipped_count, error_code,
    created_at_ms, started_at_ms, finished_at_ms, revision, phase,
    processed_assets, skipped_directories, skipped_files, error_count,
    issues_truncated, cancel_requested_at_ms`

func (s *Store) AdmitFullScan(
	ctx context.Context,
	libraryID int64,
	trigger scanner.Trigger,
) (scanner.AdmissionResult, error) {
	if libraryID <= 0 {
		return scanner.AdmissionResult{}, scanner.ErrLibraryNotFound
	}
	if !trigger.Valid() {
		return scanner.AdmissionResult{}, fmt.Errorf("invalid scan trigger %q", trigger)
	}
	var admission scanner.AdmissionResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx,
			`SELECT 1 FROM libraries WHERE id = ?`,
			libraryID,
		).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return scanner.ErrLibraryNotFound
			}
			return fmt.Errorf("check admission library: %w", err)
		}
		err := tx.QueryRowContext(ctx, `
            SELECT 1 FROM library_removals
            WHERE library_id = ? AND status IN ('queued', 'running')`,
			libraryID,
		).Scan(&exists)
		if err == nil {
			return scanner.ErrAdmissionConflict
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check removal before scan admission: %w", err)
		}

		run, err := scanAdmissionRun(tx.QueryRowContext(ctx, `
            SELECT `+admissionScanColumns+`
            FROM scan_runs
            WHERE library_id = ? AND status IN ('queued', 'running')
            ORDER BY id
            LIMIT 1`,
			libraryID,
		))
		if err == nil {
			admission = scanner.AdmissionResult{Run: run, Coalesced: true}
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("find active scan admission: %w", err)
		}

		var active int64
		if err := tx.QueryRowContext(ctx, `
            SELECT COUNT(*) FROM scan_runs
            WHERE status IN ('queued', 'running')`,
		).Scan(&active); err != nil {
			return fmt.Errorf("count scan admission capacity: %w", err)
		}
		if active >= scanner.MaxActiveFullScans {
			return scanner.ErrAdmissionCapacity
		}
		var generation int64
		if err := tx.QueryRowContext(ctx, `
            SELECT COALESCE(MAX(generation), 0) + 1
            FROM scan_runs WHERE library_id = ?`,
			libraryID,
		).Scan(&generation); err != nil {
			return fmt.Errorf("allocate scan admission generation: %w", err)
		}
		now := s.nowMS()
		result, err := tx.ExecContext(ctx, `
            INSERT INTO scan_runs(
                library_id, generation, trigger_kind, status, phase,
                created_at_ms, available_at_ms
            ) VALUES (?, ?, ?, 'queued', 'queued', ?, ?)`,
			libraryID, generation, string(trigger), now, now,
		)
		if err != nil {
			if strings.Contains(err.Error(), "scan_runs.library_id") {
				return scanner.ErrScanActive
			}
			return fmt.Errorf("insert scan admission: %w", err)
		}
		runID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read scan admission ID: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE libraries
            SET revision = revision + 1, updated_at_ms = ?
            WHERE id = ?`,
			now, libraryID,
		); err != nil {
			return fmt.Errorf("advance library scan validator: %w", err)
		}
		run, err = scanAdmissionRun(tx.QueryRowContext(ctx, `
            SELECT `+admissionScanColumns+` FROM scan_runs WHERE id = ?`,
			runID,
		))
		if err != nil {
			return fmt.Errorf("read inserted scan admission: %w", err)
		}
		admission = scanner.AdmissionResult{Run: run}
		return nil
	})
	return admission, err
}

func scanAdmissionRun(row rowScanner) (scanner.ScanRun, error) {
	var (
		run                                scanner.ScanRun
		rawTrigger, rawStatus              string
		errorCode                          sql.NullString
		started, finished, cancelRequested sql.NullInt64
		issuesTruncated                    int64
	)
	err := row.Scan(
		&run.ID, &run.LibraryID, &run.Generation, &rawTrigger, &rawStatus,
		&run.DiscoveredDirectories, &run.DiscoveredAssets, &run.SkippedCount,
		&errorCode, &run.CreatedAtMS, &started, &finished,
		&run.Revision, &run.Phase, &run.ProcessedAssets,
		&run.SkippedDirectories, &run.SkippedFiles, &run.ErrorCount,
		&issuesTruncated, &cancelRequested,
	)
	if err != nil {
		return scanner.ScanRun{}, err
	}
	run.Trigger = scanner.Trigger(rawTrigger)
	run.Status = scanner.RunStatus(rawStatus)
	run.IssuesTruncated = issuesTruncated != 0
	if errorCode.Valid {
		run.ErrorCode = errorCode.String
	}
	if started.Valid {
		run.StartedAtMS = &started.Int64
	}
	if finished.Valid {
		run.FinishedAtMS = &finished.Int64
	}
	if cancelRequested.Valid {
		run.CancelRequestedAtMS = &cancelRequested.Int64
	}
	return run, nil
}

func (s *Store) GetLibraryRoot(ctx context.Context, libraryID int64) (string, error) {
	if libraryID <= 0 {
		return "", scanner.ErrLibraryNotFound
	}
	var root string
	if err := s.db.QueryRowContext(ctx, `SELECT root_rel_path FROM libraries WHERE id = ?`, libraryID).Scan(&root); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", scanner.ErrLibraryNotFound
		}
		return "", fmt.Errorf("get scan library root: %w", err)
	}
	return root, nil
}

func (s *Store) BeginFullScan(ctx context.Context, libraryID int64, trigger scanner.Trigger) (scanner.ScanRun, error) {
	if libraryID <= 0 {
		return scanner.ScanRun{}, scanner.ErrLibraryNotFound
	}
	if !trigger.Valid() {
		return scanner.ScanRun{}, fmt.Errorf("invalid scan trigger %q", trigger)
	}

	var run scanner.ScanRun
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM libraries WHERE id = ?`, libraryID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return scanner.ErrLibraryNotFound
			}
			return fmt.Errorf("check scan library: %w", err)
		}
		removalErr := tx.QueryRowContext(ctx, `
            SELECT 1 FROM library_removals
            WHERE library_id = ? AND status IN ('queued', 'running')`,
			libraryID,
		).Scan(&exists)
		if removalErr == nil {
			return scanner.ErrLibraryNotFound
		}
		if !errors.Is(removalErr, sql.ErrNoRows) {
			return fmt.Errorf("check library removal before scan: %w", removalErr)
		}

		var activeID int64
		err := tx.QueryRowContext(ctx, `
            SELECT id
            FROM scan_runs
            WHERE library_id = ? AND status IN ('queued', 'running')`, libraryID).Scan(&activeID)
		switch {
		case err == nil:
			return scanner.ErrScanActive
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("check active scan: %w", err)
		}

		var generation int64
		if err := tx.QueryRowContext(ctx, `
            SELECT COALESCE(MAX(generation), 0) + 1 FROM scan_runs WHERE library_id = ?`, libraryID).Scan(&generation); err != nil {
			return fmt.Errorf("allocate scan generation: %w", err)
		}

		now := s.nowMS()
		result, err := tx.ExecContext(ctx, `
            INSERT INTO scan_runs(
                library_id, generation, trigger_kind, status, phase,
                created_at_ms, started_at_ms, heartbeat_at_ms,
                lease_expires_at_ms, attempt_count
            )
            VALUES (?, ?, ?, 'running', 'checking_root', ?, ?, ?, ?, 1)`,
			libraryID, generation, string(trigger), now, now, now, now+120000)
		if err != nil {
			if strings.Contains(err.Error(), "scan_runs.library_id") {
				return scanner.ErrScanActive
			}
			return fmt.Errorf("insert scan run: %w", err)
		}
		runID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read scan run id: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE libraries
            SET status = 'scanning', revision = revision + 1, updated_at_ms = ?
            WHERE id = ?`, now, libraryID); err != nil {
			return fmt.Errorf("mark library scanning: %w", err)
		}
		startedAt := now
		run = scanner.ScanRun{
			ID: runID, LibraryID: libraryID, Generation: generation,
			Trigger: trigger, Status: scanner.RunStatusRunning,
			CreatedAtMS: now, StartedAtMS: &startedAt,
		}
		return nil
	})
	return run, err
}

func (s *Store) UpsertCatalogBatch(ctx context.Context, runID int64, entries []scanner.CatalogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	if len(entries) > s.maxBatchSize {
		return scanner.ErrBatchTooLarge
	}

	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		run, err := activeScanRunTx(ctx, tx, runID)
		if err != nil {
			return err
		}

		var newDirectories, newAssets int64
		for _, entry := range entries {
			switch entry.Kind {
			case scanner.CatalogEntryDirectory:
				newlySeen, err := upsertDirectory(ctx, tx, run, entry)
				if err != nil {
					return err
				}
				if newlySeen {
					newDirectories++
				}
			case scanner.CatalogEntryAsset:
				newlySeen, err := upsertAsset(ctx, tx, run, entry)
				if err != nil {
					return err
				}
				if newlySeen {
					newAssets++
				}
			default:
				return fmt.Errorf("%w: unknown catalog entry kind %q", scanner.ErrInvalidEntry, entry.Kind)
			}
		}

		if _, err := tx.ExecContext(ctx, `
            UPDATE scan_runs
            SET discovered_directories = discovered_directories + ?,
                discovered_assets = discovered_assets + ?,
                processed_assets = processed_assets + ?,
                revision = revision + 1
            WHERE id = ? AND status = 'running'`,
			newDirectories, newAssets, newAssets, runID); err != nil {
			return fmt.Errorf("update scan progress: %w", err)
		}
		return nil
	})
	if err == nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, scanner.ErrInvalidEntry) ||
		errors.Is(err, scanner.ErrScanRunNotFound) ||
		errors.Is(err, scanner.ErrScanRunNotActive) {
		return err
	}
	return errors.Join(scanner.ErrDatabaseUnavailable, err)
}

func upsertDirectory(ctx context.Context, tx *sql.Tx, run scanner.ScanRun, entry scanner.CatalogEntry) (bool, error) {
	if entry.RelativePath == "" && entry.ParentPath != "" {
		return false, fmt.Errorf("%w: root directory has a parent", scanner.ErrInvalidEntry)
	}

	newlySeen, err := notSeenInGeneration(ctx, tx, "directories", run.LibraryID, entry.RelativePath, run.Generation)
	if err != nil {
		return false, err
	}

	var parentID any
	if entry.RelativePath != "" {
		var id int64
		if err := tx.QueryRowContext(ctx, `
            SELECT id
            FROM directories
            WHERE library_id = ? AND relative_path = ? AND last_seen_generation = ?`,
			run.LibraryID, entry.ParentPath, run.Generation).Scan(&id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, fmt.Errorf("%w: parent directory %q has not been indexed", scanner.ErrInvalidEntry, entry.ParentPath)
			}
			return false, fmt.Errorf("find parent directory: %w", err)
		}
		parentID = id
	}

	if _, err := tx.ExecContext(ctx, `
        INSERT INTO directories(
            library_id, parent_id, relative_path, name, natural_name_key,
            mtime_ns, last_seen_generation
        ) VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(library_id, relative_path) DO UPDATE SET
            parent_id = excluded.parent_id,
            name = excluded.name,
            natural_name_key = excluded.natural_name_key,
            mtime_ns = excluded.mtime_ns,
            last_seen_generation = excluded.last_seen_generation`,
		run.LibraryID, parentID, entry.RelativePath, entry.Name,
		catalog.NaturalNameKey(entry.Name), entry.MTimeNS, run.Generation); err != nil {
		return false, fmt.Errorf("upsert directory %q: %w", entry.RelativePath, err)
	}
	return newlySeen, nil
}

func upsertAsset(ctx context.Context, tx *sql.Tx, run scanner.ScanRun, entry scanner.CatalogEntry) (bool, error) {
	if entry.RelativePath == "" || entry.Name == "" || entry.SizeBytes < 0 {
		return false, scanner.ErrInvalidEntry
	}
	sourceFingerprint, err := media.NewSourceFingerprint(entry.SizeBytes, entry.MTimeNS)
	if err != nil {
		return false, fmt.Errorf("%w: %w", scanner.ErrInvalidEntry, err)
	}
	newlySeen, err := notSeenInGeneration(ctx, tx, "assets", run.LibraryID, entry.RelativePath, run.Generation)
	if err != nil {
		return false, err
	}

	var directoryID int64
	if err := tx.QueryRowContext(ctx, `
        SELECT id
        FROM directories
        WHERE library_id = ? AND relative_path = ? AND last_seen_generation = ?`,
		run.LibraryID, entry.ParentPath, run.Generation).Scan(&directoryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%w: asset parent %q has not been indexed", scanner.ErrInvalidEntry, entry.ParentPath)
		}
		return false, fmt.Errorf("find asset parent directory: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
        INSERT INTO assets(
            library_id, directory_id, relative_path, name, kind, media_format,
            mime_type, size_bytes, mtime_ns, source_fingerprint,
            natural_name_key, last_seen_generation
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(library_id, relative_path) DO UPDATE SET
            directory_id = excluded.directory_id,
            name = excluded.name,
            kind = excluded.kind,
            media_format = excluded.media_format,
            mime_type = excluded.mime_type,
            size_bytes = excluded.size_bytes,
            mtime_ns = excluded.mtime_ns,
            width = CASE
                WHEN assets.source_fingerprint <> excluded.source_fingerprint THEN NULL
                ELSE assets.width END,
            height = CASE
                WHEN assets.source_fingerprint <> excluded.source_fingerprint THEN NULL
                ELSE assets.height END,
            duration_ms = CASE
                WHEN assets.source_fingerprint <> excluded.source_fingerprint THEN NULL
                ELSE assets.duration_ms END,
            probe_status = CASE
                WHEN assets.source_fingerprint <> excluded.source_fingerprint THEN 'pending'
                ELSE assets.probe_status END,
            probe_error_code = CASE
                WHEN assets.source_fingerprint <> excluded.source_fingerprint THEN NULL
                ELSE assets.probe_error_code END,
            playback_status = CASE
                WHEN assets.source_fingerprint <> excluded.source_fingerprint THEN 'unknown'
                ELSE assets.playback_status END,
            source_fingerprint = excluded.source_fingerprint,
            natural_name_key = excluded.natural_name_key,
            last_seen_generation = excluded.last_seen_generation`,
		run.LibraryID, directoryID, entry.RelativePath, entry.Name,
		string(entry.AssetKind), string(entry.MediaFormat), entry.MIMEType,
		entry.SizeBytes, entry.MTimeNS, sourceFingerprint.String(),
		catalog.NaturalNameKey(entry.Name), run.Generation); err != nil {
		return false, fmt.Errorf("upsert asset %q: %w", entry.RelativePath, err)
	}
	var assetID int64
	if err := tx.QueryRowContext(ctx, `
        SELECT id FROM assets WHERE library_id = ? AND relative_path = ?`,
		run.LibraryID, entry.RelativePath,
	).Scan(&assetID); err != nil {
		return false, fmt.Errorf("read upserted asset identity: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
        INSERT OR IGNORE INTO cache_deletions(
            library_id, cache_rel_path, byte_size, created_at_ms
        )
        SELECT library_id, cache_rel_path, byte_size, ?
        FROM thumbnails
        WHERE asset_id = ? AND source_fingerprint <> ? AND status = 'ready'`,
		run.CreatedAtMS, assetID, sourceFingerprint.String(),
	); err != nil {
		return false, fmt.Errorf("schedule stale thumbnail cleanup: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
        DELETE FROM thumbnails
        WHERE asset_id = ? AND source_fingerprint <> ?`,
		assetID, sourceFingerprint.String(),
	); err != nil {
		return false, fmt.Errorf("invalidate stale thumbnail: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO media_jobs(
            library_id, asset_id, variant, transform_version, source_fingerprint,
            status, available_at_ms, attempt_count, created_at_ms
        ) VALUES (?, ?, 'grid', ?, ?, 'queued', ?, 0, ?)
        ON CONFLICT(asset_id, variant) DO UPDATE SET
            library_id = excluded.library_id,
            transform_version = excluded.transform_version,
            source_fingerprint = excluded.source_fingerprint,
            status = 'queued',
            last_error_code = NULL,
            available_at_ms = excluded.available_at_ms,
            started_at_ms = NULL,
            heartbeat_at_ms = NULL,
            lease_expires_at_ms = NULL,
            attempt_count = 0,
            created_at_ms = excluded.created_at_ms,
            finished_at_ms = NULL
        WHERE media_jobs.source_fingerprint <> excluded.source_fingerprint
           OR media_jobs.transform_version <> excluded.transform_version`,
		run.LibraryID, assetID, thumbnail.GridTransformVersion,
		sourceFingerprint.String(),
		run.CreatedAtMS, run.CreatedAtMS,
	); err != nil {
		return false, fmt.Errorf("admit media job: %w", err)
	}
	return newlySeen, nil
}

func notSeenInGeneration(
	ctx context.Context,
	tx *sql.Tx,
	table string,
	libraryID int64,
	relativePath string,
	generation int64,
) (bool, error) {
	if table != "directories" && table != "assets" {
		return false, errors.New("invalid generation table")
	}
	var previous int64
	err := tx.QueryRowContext(ctx,
		`SELECT last_seen_generation FROM `+table+` WHERE library_id = ? AND relative_path = ?`,
		libraryID, relativePath,
	).Scan(&previous)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read prior %s generation: %w", table, err)
	}
	return previous != generation, nil
}

func (s *Store) CompleteFullScan(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
) (scanner.ScanRun, error) {
	if !skipped.Valid() {
		return scanner.ScanRun{}, errors.New("skipped counts cannot be negative")
	}
	var completed scanner.ScanRun
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		run, err := activeScanRunTx(ctx, tx, runID)
		if err != nil {
			return err
		}
		var rootID int64
		if err := tx.QueryRowContext(ctx, `
			SELECT id
			FROM directories
			WHERE library_id = ? AND relative_path = '' AND last_seen_generation = ?`,
			run.LibraryID, run.Generation).Scan(&rootID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: scan generation did not index its root directory", scanner.ErrInvalidEntry)
			}
			return fmt.Errorf("verify scanned root directory: %w", err)
		}
		if err := validateCatalogRelationshipsTx(ctx, tx, run.LibraryID, run.Generation); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
            DELETE FROM assets WHERE library_id = ? AND last_seen_generation < ?`,
			run.LibraryID, run.Generation); err != nil {
			return fmt.Errorf("remove stale assets: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            DELETE FROM directories WHERE library_id = ? AND last_seen_generation < ?`,
			run.LibraryID, run.Generation); err != nil {
			return fmt.Errorf("remove stale directories: %w", err)
		}
		if err := recalculateDirectoryCountsTx(ctx, tx, run.LibraryID, rootID); err != nil {
			return err
		}

		now := s.nowMS()
		if _, err := tx.ExecContext(ctx, `
            UPDATE libraries
            SET current_generation = ?, status = 'ready',
                revision = revision + 1, updated_at_ms = ?
            WHERE id = ?`, run.Generation, now, run.LibraryID); err != nil {
			return fmt.Errorf("commit library generation: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE scan_runs
            SET status = 'succeeded', phase = 'completed',
                skipped_count = ?, skipped_directories = ?, skipped_files = ?,
                error_code = NULL, finished_at_ms = ?,
                heartbeat_at_ms = NULL, lease_expires_at_ms = NULL,
                revision = revision + 1
            WHERE id = ? AND status = 'running'`,
			skipped.Total(), skipped.Directories, skipped.Files, now, run.ID)
		if err != nil {
			return fmt.Errorf("complete scan run: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read completed scan row count: %w", err)
		}
		if rows != 1 {
			return scanner.ErrScanRunNotActive
		}
		completed, err = getScanRunTx(ctx, tx, run.ID)
		return err
	})
	return completed, err
}

const directoryRollupBatchSize = 500

// validateCatalogRelationshipsTx rejects relationship and generation
// corruption before stale cleanup can cascade into another library or delete
// a current-generation row through a stale parent. The schema's foreign keys
// cover normal writers; the finalizer also fails closed if the database was
// externally damaged.
func validateCatalogRelationshipsTx(
	ctx context.Context,
	tx *sql.Tx,
	libraryID int64,
	generation int64,
) error {
	var invalid int
	if err := tx.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM directories AS child
            LEFT JOIN directories AS parent ON parent.id = child.parent_id
            WHERE child.parent_id IS NOT NULL
              AND (
                  (child.library_id = ? AND (
                      parent.id IS NULL OR parent.library_id <> child.library_id
                  ))
                  OR
                  (parent.library_id = ? AND child.library_id <> parent.library_id)
              )
            UNION ALL
            SELECT 1
            FROM assets AS asset
            LEFT JOIN directories AS directory ON directory.id = asset.directory_id
            WHERE (
                asset.library_id = ? AND (
                    directory.id IS NULL OR directory.library_id <> asset.library_id
                )
            )
               OR (
                    directory.library_id = ? AND asset.library_id <> directory.library_id
               )
            LIMIT 1
        )`,
		libraryID, libraryID, libraryID, libraryID).Scan(&invalid); err != nil {
		return fmt.Errorf("validate catalog relationships: %w", err)
	}
	if invalid != 0 {
		return fmt.Errorf("%w: catalog contains a cross-library or missing relationship", scanner.ErrInvalidEntry)
	}

	if err := tx.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM directories AS child
            JOIN directories AS parent
              ON parent.library_id = child.library_id
             AND parent.id = child.parent_id
            WHERE child.library_id = ?
              AND child.last_seen_generation = ?
              AND parent.last_seen_generation <> ?
            UNION ALL
            SELECT 1
            FROM assets AS asset
            JOIN directories AS directory
              ON directory.library_id = asset.library_id
             AND directory.id = asset.directory_id
            WHERE asset.library_id = ?
              AND asset.last_seen_generation = ?
              AND directory.last_seen_generation <> ?
            UNION ALL
            SELECT 1
            FROM directories
            WHERE library_id = ? AND last_seen_generation > ?
            UNION ALL
            SELECT 1
            FROM assets
            WHERE library_id = ? AND last_seen_generation > ?
            LIMIT 1
        )`,
		libraryID, generation, generation,
		libraryID, generation, generation,
		libraryID, generation,
		libraryID, generation,
	).Scan(&invalid); err != nil {
		return fmt.Errorf("validate catalog generations: %w", err)
	}
	if invalid != 0 {
		return fmt.Errorf(
			"%w: current catalog row references a stale directory or contains a future generation",
			scanner.ErrInvalidEntry,
		)
	}
	return nil
}

// recalculateDirectoryCountsTx computes recursive counts without expanding one
// row per asset/ancestor pair. SQLite owns an O(D) temporary topology table;
// Go retains only counters. Leaf batches are folded into their parents exactly
// once, giving O(A + D log D) work with the current indexes and constant Go
// memory. A cycle, orphan root, or otherwise disconnected topology leaves no
// processable leaf and fails the surrounding finalization transaction closed.
func recalculateDirectoryCountsTx(
	ctx context.Context,
	tx *sql.Tx,
	libraryID int64,
	rootID int64,
) error {
	if _, err := tx.ExecContext(ctx, `
        CREATE TEMP TABLE IF NOT EXISTS foliopath_directory_rollup (
            id                    INTEGER PRIMARY KEY,
            parent_id             INTEGER,
            remaining_children    INTEGER NOT NULL CHECK (remaining_children >= 0),
            direct_asset_count    INTEGER NOT NULL CHECK (direct_asset_count >= 0),
            recursive_asset_count INTEGER NOT NULL CHECK (recursive_asset_count >= 0),
            processing            INTEGER NOT NULL DEFAULT 0 CHECK (processing IN (0, 1)),
            retired               INTEGER NOT NULL DEFAULT 0 CHECK (retired IN (0, 1))
        )`); err != nil {
		return fmt.Errorf("create directory rollup workspace: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
        CREATE INDEX IF NOT EXISTS temp.foliopath_directory_rollup_ready
        ON foliopath_directory_rollup(retired, processing, remaining_children, id)`); err != nil {
		return fmt.Errorf("index directory rollup workspace: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM temp.foliopath_directory_rollup`); err != nil {
		return fmt.Errorf("clear directory rollup workspace: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
        WITH
        direct_counts(directory_id, asset_count) AS MATERIALIZED (
            SELECT directory_id, count(*)
            FROM assets
            WHERE library_id = ?
            GROUP BY directory_id
        ),
        child_counts(parent_id, child_count) AS MATERIALIZED (
            SELECT parent_id, count(*)
            FROM directories
            WHERE library_id = ? AND parent_id IS NOT NULL
            GROUP BY parent_id
        )
        INSERT INTO temp.foliopath_directory_rollup(
            id, parent_id, remaining_children,
            direct_asset_count, recursive_asset_count
        )
        SELECT
            directory.id,
            directory.parent_id,
            COALESCE(children.child_count, 0),
            COALESCE(direct.asset_count, 0),
            COALESCE(direct.asset_count, 0)
        FROM directories AS directory
        LEFT JOIN child_counts AS children ON children.parent_id = directory.id
        LEFT JOIN direct_counts AS direct ON direct.directory_id = directory.id
        WHERE directory.library_id = ?`,
		libraryID, libraryID, libraryID); err != nil {
		return fmt.Errorf("populate directory rollup workspace: %w", err)
	}

	var remainingDirectories int64
	var malformedRoots int64
	if err := tx.QueryRowContext(ctx, `
        SELECT
            count(*),
            COALESCE(sum(CASE
                WHEN (id = ? AND parent_id IS NOT NULL)
                  OR (id <> ? AND parent_id IS NULL)
                THEN 1 ELSE 0 END), 0)
        FROM temp.foliopath_directory_rollup`,
		rootID, rootID).Scan(&remainingDirectories, &malformedRoots); err != nil {
		return fmt.Errorf("inspect directory rollup topology: %w", err)
	}
	if remainingDirectories == 0 || malformedRoots != 0 {
		return fmt.Errorf("%w: catalog directory topology has an invalid root", scanner.ErrInvalidEntry)
	}
	totalDirectories := remainingDirectories

	for remainingDirectories > 1 {
		result, err := tx.ExecContext(ctx, `
            UPDATE temp.foliopath_directory_rollup
            SET processing = 1
            WHERE id IN (
                SELECT id
                FROM temp.foliopath_directory_rollup
                WHERE retired = 0
                  AND processing = 0
                  AND remaining_children = 0
                  AND id <> ?
                ORDER BY id
                LIMIT ?
            )`, rootID, directoryRollupBatchSize)
		if err != nil {
			return fmt.Errorf("select directory rollup leaf batch: %w", err)
		}
		processed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read directory rollup leaf batch size: %w", err)
		}
		if processed == 0 {
			return fmt.Errorf("%w: catalog directory topology contains a cycle", scanner.ErrInvalidEntry)
		}

		if _, err := tx.ExecContext(ctx, `
            WITH propagated(parent_id, asset_count, child_count) AS MATERIALIZED (
                SELECT parent_id, sum(recursive_asset_count), count(*)
                FROM temp.foliopath_directory_rollup
                WHERE retired = 0 AND processing = 1
                GROUP BY parent_id
            )
            UPDATE temp.foliopath_directory_rollup AS parent
            SET recursive_asset_count =
                    parent.recursive_asset_count + propagated.asset_count,
                remaining_children =
                    parent.remaining_children - propagated.child_count
            FROM propagated
            WHERE parent.id = propagated.parent_id`); err != nil {
			return fmt.Errorf("propagate directory rollup leaf batch: %w", err)
		}
		result, err = tx.ExecContext(ctx, `
            UPDATE temp.foliopath_directory_rollup
            SET processing = 0, retired = 1
            WHERE retired = 0 AND processing = 1`)
		if err != nil {
			return fmt.Errorf("retire directory rollup leaf batch: %w", err)
		}
		retired, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read retired directory rollup batch size: %w", err)
		}
		if retired != processed {
			return fmt.Errorf("%w: directory rollup workspace changed unexpectedly", scanner.ErrInvalidEntry)
		}
		remainingDirectories -= retired
	}

	var rootChildren int64
	if err := tx.QueryRowContext(ctx, `
        SELECT remaining_children
        FROM temp.foliopath_directory_rollup
        WHERE id = ?`, rootID).Scan(&rootChildren); err != nil {
		return fmt.Errorf("%w: directory rollup did not converge to the library root", scanner.ErrInvalidEntry)
	}
	if rootChildren != 0 {
		return fmt.Errorf("%w: directory rollup root still has unprocessed children", scanner.ErrInvalidEntry)
	}

	result, err := tx.ExecContext(ctx, `
        UPDATE directories AS directory
        SET direct_asset_count = rollup.direct_asset_count,
            recursive_asset_count = rollup.recursive_asset_count
        FROM temp.foliopath_directory_rollup AS rollup
        WHERE directory.id = rollup.id
          AND directory.library_id = ?`, libraryID)
	if err != nil {
		return fmt.Errorf("apply directory rollup counts: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read applied directory rollup count: %w", err)
	}
	if updated != totalDirectories {
		return fmt.Errorf("%w: directory rollup did not update every directory", scanner.ErrInvalidEntry)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM temp.foliopath_directory_rollup`); err != nil {
		return fmt.Errorf("release directory rollup workspace: %w", err)
	}
	return nil
}

func (s *Store) FailFullScan(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
	errorCode string,
) (scanner.ScanRun, error) {
	return s.finishWithoutCleanup(
		ctx, runID, skipped, scanner.RunStatusFailed,
		normalizeErrorCode(errorCode), "error",
	)
}

func (s *Store) CancelFullScan(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
) (scanner.ScanRun, error) {
	return s.finishWithoutCleanup(
		ctx, runID, skipped, scanner.RunStatusCancelled, "", "previous",
	)
}

func (s *Store) OfflineFullScan(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
	errorCode string,
) (scanner.ScanRun, error) {
	return s.finishWithoutCleanup(
		ctx, runID, skipped, scanner.RunStatusOffline,
		normalizeErrorCode(errorCode), "offline",
	)
}

func (s *Store) finishWithoutCleanup(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
	status scanner.RunStatus,
	errorCode string,
	libraryStatus string,
) (scanner.ScanRun, error) {
	if !skipped.Valid() {
		return scanner.ScanRun{}, errors.New("skipped counts cannot be negative")
	}
	var finished scanner.ScanRun
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		run, err := activeScanRunTx(ctx, tx, runID)
		if err != nil {
			return err
		}
		now := s.nowMS()

		switch libraryStatus {
		case "error", "offline":
			if _, err := tx.ExecContext(ctx, `
                    UPDATE libraries
                    SET status = ?, revision = revision + 1, updated_at_ms = ?
                    WHERE id = ?`,
				libraryStatus, now, run.LibraryID); err != nil {
				return fmt.Errorf("update aborted library status: %w", err)
			}
		case "previous":
			if _, err := tx.ExecContext(ctx, `
                    UPDATE libraries
                    SET status = CASE WHEN current_generation > 0 THEN 'ready' ELSE 'pending' END,
                        revision = revision + 1,
                        updated_at_ms = ?
                    WHERE id = ?`, now, run.LibraryID); err != nil {
				return fmt.Errorf("restore library status after cancellation: %w", err)
			}
		default:
			return errors.New("invalid aborted library status")
		}

		var nullableCode any
		if errorCode != "" {
			nullableCode = errorCode
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE scan_runs
            SET status = ?, phase = 'completed',
                skipped_count = ?, skipped_directories = ?, skipped_files = ?,
                error_code = ?, finished_at_ms = ?,
                heartbeat_at_ms = NULL, lease_expires_at_ms = NULL,
                revision = revision + 1
            WHERE id = ? AND status = 'running'`,
			string(status), skipped.Total(), skipped.Directories, skipped.Files,
			nullableCode, now, run.ID)
		if err != nil {
			return fmt.Errorf("finish scan without cleanup: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read aborted scan row count: %w", err)
		}
		if rows != 1 {
			return scanner.ErrScanRunNotActive
		}
		finished, err = getScanRunTx(ctx, tx, run.ID)
		return err
	})
	return finished, err
}

func (s *Store) InterruptActiveScans(ctx context.Context) (int64, error) {
	var interrupted int64
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		now := s.nowMS()
		if _, err := tx.ExecContext(ctx, `
            UPDATE libraries
            SET status = CASE WHEN current_generation > 0 THEN 'ready' ELSE 'error' END,
                revision = revision + 1,
                updated_at_ms = ?
            WHERE id IN (SELECT library_id FROM scan_runs WHERE status = 'running')`, now); err != nil {
			return fmt.Errorf("restore libraries for interrupted scans: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE scan_runs
            SET status = 'interrupted', phase = 'completed',
                error_code = 'scan_interrupted', finished_at_ms = ?,
                heartbeat_at_ms = NULL, lease_expires_at_ms = NULL,
                revision = revision + 1
            WHERE status = 'running'`, now)
		if err != nil {
			return fmt.Errorf("interrupt active scans: %w", err)
		}
		interrupted, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read interrupted scan count: %w", err)
		}
		return nil
	})
	return interrupted, err
}

func (s *Store) GetScanRun(ctx context.Context, runID int64) (scanner.ScanRun, error) {
	run, err := scanAdmissionRun(s.db.QueryRowContext(
		ctx,
		`SELECT `+admissionScanColumns+` FROM scan_runs WHERE id = ?`,
		runID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return scanner.ScanRun{}, scanner.ErrScanRunNotFound
	}
	if err != nil {
		return scanner.ScanRun{}, fmt.Errorf("get scan run: %w", err)
	}
	return run, nil
}

func activeScanRunTx(ctx context.Context, tx *sql.Tx, runID int64) (scanner.ScanRun, error) {
	run, err := scanAdmissionRun(tx.QueryRowContext(ctx, `
        SELECT `+admissionScanColumns+` FROM scan_runs WHERE id = ?`, runID))
	if errors.Is(err, sql.ErrNoRows) {
		return scanner.ScanRun{}, scanner.ErrScanRunNotFound
	}
	if err != nil {
		return scanner.ScanRun{}, fmt.Errorf("get active scan run: %w", err)
	}
	if run.Status != scanner.RunStatusRunning {
		return scanner.ScanRun{}, scanner.ErrScanRunNotActive
	}
	return run, nil
}

func getScanRunTx(ctx context.Context, tx *sql.Tx, runID int64) (scanner.ScanRun, error) {
	run, err := scanAdmissionRun(tx.QueryRowContext(
		ctx,
		`SELECT `+admissionScanColumns+` FROM scan_runs WHERE id = ?`,
		runID,
	))
	if err != nil {
		return scanner.ScanRun{}, fmt.Errorf("read scan run: %w", err)
	}
	return run, nil
}

func normalizeErrorCode(code string) string {
	switch code {
	case "library_root_unavailable",
		"library_root_outside_allowed",
		"library_root_symlink",
		"library_root_mount_boundary",
		"library_root_identity_changed",
		"partial_tree_unreadable",
		"scan_io_error",
		"database_unavailable",
		"scan_interrupted",
		"internal_error":
		return code
	default:
		return "internal_error"
	}
}
