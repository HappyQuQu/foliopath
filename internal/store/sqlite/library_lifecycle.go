package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

const libraryDetailsSQL = `
SELECT
    l.id, l.name, l.root_rel_path, l.status, l.current_generation,
    l.revision, l.created_at_ms, l.updated_at_ms,
    (
        SELECT MAX(finished_at_ms)
        FROM scan_runs
        WHERE library_id = l.id AND status = 'succeeded'
    ),
    (
        SELECT id
        FROM scan_runs
        WHERE library_id = l.id
        ORDER BY created_at_ms DESC, id DESC
        LIMIT 1
    ),
    (SELECT COUNT(*) FROM assets WHERE library_id = l.id),
    (SELECT COUNT(*) FROM directories WHERE library_id = l.id)
FROM libraries l`

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) FindCreateReplay(
	ctx context.Context,
	keyHash [32]byte,
	requestHash [32]byte,
) (library.CreateResult, bool, error) {
	return findCreateReplay(ctx, s.db, keyHash, requestHash)
}

func findCreateReplay(
	ctx context.Context,
	query queryRower,
	keyHash [32]byte,
	requestHash [32]byte,
) (library.CreateResult, bool, error) {
	var (
		storedRequest []byte
		resultID      int64
	)
	err := query.QueryRowContext(ctx, `
        SELECT request_hash, result_id
        FROM idempotency_records
        WHERE operation = 'create_library' AND key_hash = ?`,
		keyHash[:],
	).Scan(&storedRequest, &resultID)
	if errors.Is(err, sql.ErrNoRows) {
		return library.CreateResult{}, false, nil
	}
	if err != nil {
		return library.CreateResult{}, false, fmt.Errorf("find library create replay: %w", err)
	}
	if !bytes.Equal(storedRequest, requestHash[:]) {
		return library.CreateResult{}, false, library.ErrIdempotencyConflict
	}
	details, err := getLibraryDetails(ctx, query, resultID)
	if errors.Is(err, library.ErrNotFound) {
		return library.CreateResult{}, false, library.ErrIdempotencyConflict
	}
	if err != nil {
		return library.CreateResult{}, false, err
	}
	scan, err := getCreationScan(ctx, query, resultID)
	if errors.Is(err, sql.ErrNoRows) {
		return library.CreateResult{}, false, library.ErrIdempotencyConflict
	}
	if err != nil {
		return library.CreateResult{}, false, err
	}
	return library.CreateResult{Library: details, Scan: scan, Replayed: true}, true, nil
}

func (s *Store) CreateLibraryWithScan(
	ctx context.Context,
	command library.CreateCommand,
) (library.CreateResult, error) {
	if len(command.NameSortKey) == 0 {
		command.NameSortKey = library.NaturalNameSortKey(command.Name)
	}
	var result library.CreateResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		replayed, found, err := findCreateReplay(ctx, tx, command.KeyHash, command.RequestHash)
		if err != nil {
			return err
		}
		if found {
			result = replayed
			return nil
		}

		if err := rejectLibraryConflicts(ctx, tx, command.NameKey, command.RootRelativePath); err != nil {
			return err
		}
		var activeCount int64
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM scan_runs WHERE status IN ('queued', 'running')`,
		).Scan(&activeCount); err != nil {
			return fmt.Errorf("count active scans: %w", err)
		}
		if activeCount >= scanner.MaxActiveFullScans {
			return library.ErrScanCapacity
		}

		now := s.nowMS()
		inserted, err := tx.ExecContext(ctx, `
            INSERT INTO libraries(
                name, name_key, name_sort_key, root_rel_path, status, current_generation,
                revision, created_at_ms, updated_at_ms
            ) VALUES (?, ?, ?, ?, 'pending', 0, 1, ?, ?)`,
			command.Name, command.NameKey, command.NameSortKey,
			command.RootRelativePath, now, now,
		)
		if err != nil {
			return mapLibraryConstraintError(err)
		}
		libraryID, err := inserted.LastInsertId()
		if err != nil {
			return fmt.Errorf("read created library ID: %w", err)
		}
		insertedScan, err := tx.ExecContext(ctx, `
            INSERT INTO scan_runs(
                library_id, generation, trigger_kind, status, phase,
                created_at_ms, available_at_ms
            ) VALUES (?, 1, 'library_created', 'queued', 'queued', ?, ?)`,
			libraryID, now, now,
		)
		if err != nil {
			return fmt.Errorf("queue creation scan: %w", err)
		}
		scanID, err := insertedScan.LastInsertId()
		if err != nil {
			return fmt.Errorf("read creation scan ID: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO idempotency_records(
                operation, key_hash, request_hash, result_kind, result_id,
                created_at_ms, expires_at_ms
            ) VALUES ('create_library', ?, ?, 'library', ?, ?, ?)`,
			command.KeyHash[:], command.RequestHash[:], libraryID,
			now, now+command.RetentionMS,
		); err != nil {
			if strings.Contains(err.Error(), "idempotency_records.operation") {
				return library.ErrIdempotencyConflict
			}
			return fmt.Errorf("record library creation idempotency: %w", err)
		}
		details, err := getLibraryDetails(ctx, tx, libraryID)
		if err != nil {
			return err
		}
		scan, err := getScan(ctx, tx, scanID)
		if err != nil {
			return err
		}
		result = library.CreateResult{Library: details, Scan: scan}
		return nil
	})
	return result, err
}

func rejectLibraryConflicts(
	ctx context.Context,
	tx *sql.Tx,
	nameKey, root string,
) error {
	var existing int64
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM libraries WHERE name_key = ?`, nameKey,
	).Scan(&existing)
	switch {
	case err == nil:
		return library.ErrNameExists
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("check library name: %w", err)
	}
	err = tx.QueryRowContext(ctx, `
        SELECT id FROM libraries
        WHERE root_rel_path = ?
           OR root_rel_path = ''
           OR ? = ''
           OR instr(?, root_rel_path || '/') = 1
           OR instr(root_rel_path, ? || '/') = 1
        LIMIT 1`, root, root, root, root).Scan(&existing)
	switch {
	case err == nil:
		return library.ErrRootOverlap
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("check library root overlap: %w", err)
	default:
		return nil
	}
}

func mapLibraryConstraintError(err error) error {
	message := err.Error()
	switch {
	case strings.Contains(message, "libraries.name_key"):
		return library.ErrNameExists
	case strings.Contains(message, "libraries.root_rel_path"):
		return library.ErrRootOverlap
	default:
		return fmt.Errorf("insert library: %w", err)
	}
}

func (s *Store) ListLibraryPage(
	ctx context.Context,
	params library.ListParams,
) ([]library.Details, error) {
	query := libraryDetailsSQL
	arguments := []any{}
	if params.After != nil {
		query += ` WHERE (
            l.name_sort_key > ?
            OR (l.name_sort_key = ? AND l.name > ?)
            OR (l.name_sort_key = ? AND l.name = ? AND l.id > ?)
        )`
		arguments = append(
			arguments,
			params.After.NameSortKey,
			params.After.NameSortKey, params.After.Name,
			params.After.NameSortKey, params.After.Name, params.After.ID,
		)
	}
	query += ` ORDER BY l.name_sort_key, l.name, l.id LIMIT ?`
	arguments = append(arguments, params.Limit)
	rows, err := s.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list library page: %w", err)
	}
	defer rows.Close()
	items := make([]library.Details, 0, params.Limit)
	for rows.Next() {
		item, err := scanLibraryDetails(rows)
		if err != nil {
			return nil, fmt.Errorf("map library page: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate library page: %w", err)
	}
	return items, nil
}

func (s *Store) GetLibraryDetails(ctx context.Context, id int64) (library.Details, error) {
	return getLibraryDetails(ctx, s.db, id)
}

func getLibraryDetails(
	ctx context.Context,
	query queryRower,
	id int64,
) (library.Details, error) {
	item, err := scanLibraryDetails(query.QueryRowContext(
		ctx,
		libraryDetailsSQL+` WHERE l.id = ?`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return library.Details{}, library.ErrNotFound
	}
	if err != nil {
		return library.Details{}, fmt.Errorf("get library details: %w", err)
	}
	return item, nil
}

type rowValues interface {
	Scan(...any) error
}

func scanLibraryDetails(row rowValues) (library.Details, error) {
	var (
		id, generation, revision, createdAt, updatedAt int64
		name, root, rawStatus                          string
		lastSuccess, latestScan                        sql.NullInt64
		assetCount, directoryCount                     int64
	)
	if err := row.Scan(
		&id, &name, &root, &rawStatus, &generation, &revision,
		&createdAt, &updatedAt, &lastSuccess, &latestScan,
		&assetCount, &directoryCount,
	); err != nil {
		return library.Details{}, err
	}
	base, err := libraryFromDatabase(
		id, name, root, rawStatus, generation, revision, createdAt, updatedAt,
	)
	if err != nil {
		return library.Details{}, err
	}
	details := library.Details{
		Library:        base,
		AssetCount:     assetCount,
		DirectoryCount: directoryCount,
	}
	if lastSuccess.Valid {
		details.LastSuccessfulScanAtMS = &lastSuccess.Int64
	}
	if latestScan.Valid {
		details.LatestScanID = &latestScan.Int64
	}
	return details, nil
}

func (s *Store) RenameLibraryIfRevision(
	ctx context.Context,
	command library.RenameCommand,
) (library.Details, error) {
	if len(command.NameSortKey) == 0 {
		command.NameSortKey = library.NaturalNameSortKey(command.Name)
	}
	var renamed library.Details
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := getLibraryDetails(ctx, tx, command.ID)
		if err != nil {
			return err
		}
		if current.Revision != command.ExpectedRevision {
			return library.ErrPreconditionFailed
		}
		var activeRemoval int64
		err = tx.QueryRowContext(ctx, `
            SELECT id FROM library_removals
            WHERE library_id = ? AND status IN ('queued', 'running')`,
			command.ID,
		).Scan(&activeRemoval)
		if err == nil {
			return library.ErrRemovalActive
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check active library removal: %w", err)
		}
		if current.Name == command.Name {
			renamed = current
			return nil
		}
		var duplicate int64
		err = tx.QueryRowContext(ctx,
			`SELECT id FROM libraries WHERE name_key = ? AND id <> ?`,
			command.NameKey, command.ID,
		).Scan(&duplicate)
		if err == nil {
			return library.ErrNameExists
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check renamed library name: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE libraries
            SET name = ?, name_key = ?, name_sort_key = ?,
                revision = revision + 1, updated_at_ms = ?
            WHERE id = ? AND revision = ?`,
			command.Name, command.NameKey, command.NameSortKey, s.nowMS(),
			command.ID, command.ExpectedRevision,
		)
		if err != nil {
			return mapLibraryConstraintError(err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read rename result: %w", err)
		}
		if affected != 1 {
			return library.ErrPreconditionFailed
		}
		renamed, err = getLibraryDetails(ctx, tx, command.ID)
		return err
	})
	return renamed, err
}

func (s *Store) RequestLibraryRemoval(
	ctx context.Context,
	command library.RemoveCommand,
) (library.RemoveResult, error) {
	var result library.RemoveResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		replayed, found, err := findRemovalReplay(
			ctx, tx, command.KeyHash, command.RequestHash,
		)
		if err != nil {
			return err
		}
		if found {
			result = library.RemoveResult{Removal: replayed, Replayed: true}
			return nil
		}
		details, err := getLibraryDetails(ctx, tx, command.LibraryID)
		if err != nil {
			return err
		}
		if details.Revision != command.ExpectedRevision {
			return library.ErrPreconditionFailed
		}
		var activeID int64
		err = tx.QueryRowContext(ctx, `
            SELECT id FROM library_removals
            WHERE library_id = ? AND status IN ('queued', 'running')`,
			command.LibraryID,
		).Scan(&activeID)
		if err == nil {
			return library.ErrIdempotencyConflict
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check active removal: %w", err)
		}
		now := s.nowMS()
		inserted, err := tx.ExecContext(ctx, `
            INSERT INTO library_removals(
                library_id, library_name, status, revision, created_at_ms
            ) VALUES (?, ?, 'queued', 1, ?)`,
			command.LibraryID, details.Name, now,
		)
		if err != nil {
			return fmt.Errorf("queue library removal: %w", err)
		}
		removalID, err := inserted.LastInsertId()
		if err != nil {
			return fmt.Errorf("read removal ID: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE scan_runs
            SET status = 'cancelled', phase = 'completed',
                cancel_requested_at_ms = COALESCE(cancel_requested_at_ms, ?),
                finished_at_ms = ?, revision = revision + 1
            WHERE library_id = ? AND status = 'queued'`,
			now, now, command.LibraryID,
		); err != nil {
			return fmt.Errorf("cancel queued library scan: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE scan_runs
            SET cancel_requested_at_ms = COALESCE(cancel_requested_at_ms, ?),
                revision = CASE
                    WHEN cancel_requested_at_ms IS NULL THEN revision + 1
                    ELSE revision
                END
            WHERE library_id = ? AND status = 'running'`,
			now, command.LibraryID,
		); err != nil {
			return fmt.Errorf("request running library scan cancellation: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO idempotency_records(
                operation, key_hash, request_hash, result_kind, result_id,
                created_at_ms, expires_at_ms
            ) VALUES ('remove_library', ?, ?, 'library_removal', ?, ?, ?)`,
			command.KeyHash[:], command.RequestHash[:], removalID,
			now, now+command.RetentionMS,
		); err != nil {
			if strings.Contains(err.Error(), "idempotency_records.operation") {
				return library.ErrIdempotencyConflict
			}
			return fmt.Errorf("record library removal idempotency: %w", err)
		}
		removal, err := getRemoval(ctx, tx, removalID)
		if err != nil {
			return err
		}
		result = library.RemoveResult{Removal: removal}
		return nil
	})
	return result, err
}

func findRemovalReplay(
	ctx context.Context,
	query queryRower,
	keyHash [32]byte,
	requestHash [32]byte,
) (library.Removal, bool, error) {
	var (
		storedRequest []byte
		resultID      int64
	)
	err := query.QueryRowContext(ctx, `
        SELECT request_hash, result_id
        FROM idempotency_records
        WHERE operation = 'remove_library' AND key_hash = ?`,
		keyHash[:],
	).Scan(&storedRequest, &resultID)
	if errors.Is(err, sql.ErrNoRows) {
		return library.Removal{}, false, nil
	}
	if err != nil {
		return library.Removal{}, false, fmt.Errorf("find removal replay: %w", err)
	}
	if !bytes.Equal(storedRequest, requestHash[:]) {
		return library.Removal{}, false, library.ErrIdempotencyConflict
	}
	removal, err := getRemoval(ctx, query, resultID)
	if errors.Is(err, sql.ErrNoRows) {
		return library.Removal{}, false, library.ErrIdempotencyConflict
	}
	return removal, err == nil, err
}

func (s *Store) GetLibraryRemoval(ctx context.Context, id int64) (library.Removal, error) {
	removal, err := getRemoval(ctx, s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return library.Removal{}, library.ErrRemovalNotFound
	}
	return removal, err
}

func getRemoval(
	ctx context.Context,
	query queryRower,
	id int64,
) (library.Removal, error) {
	var (
		removal               library.Removal
		rawStatus             string
		errorCode             sql.NullString
		startedAt, finishedAt sql.NullInt64
	)
	err := query.QueryRowContext(ctx, `
        SELECT id, library_id, library_name, status, revision, error_code,
               created_at_ms, started_at_ms, finished_at_ms
        FROM library_removals WHERE id = ?`,
		id,
	).Scan(
		&removal.ID, &removal.LibraryID, &removal.LibraryName,
		&rawStatus, &removal.Revision, &errorCode,
		&removal.CreatedAtMS, &startedAt, &finishedAt,
	)
	if err != nil {
		return library.Removal{}, err
	}
	removal.Status = library.RemovalStatus(rawStatus)
	if errorCode.Valid {
		removal.ErrorCode = errorCode.String
	}
	if startedAt.Valid {
		removal.StartedAtMS = &startedAt.Int64
	}
	if finishedAt.Valid {
		removal.FinishedAtMS = &finishedAt.Int64
	}
	return removal, nil
}

func getCreationScan(
	ctx context.Context,
	query queryRower,
	libraryID int64,
) (library.Scan, error) {
	return scanScan(query.QueryRowContext(ctx, `
        SELECT id, library_id, generation, trigger_kind, status, phase, revision,
               discovered_directories, discovered_assets, processed_assets,
               skipped_directories, skipped_files, error_count, issues_truncated,
               error_code, created_at_ms, started_at_ms, finished_at_ms
        FROM scan_runs
        WHERE library_id = ? AND trigger_kind = 'library_created'`,
		libraryID,
	))
}

func getScan(ctx context.Context, query queryRower, id int64) (library.Scan, error) {
	return scanScan(query.QueryRowContext(ctx, `
        SELECT id, library_id, generation, trigger_kind, status, phase, revision,
               discovered_directories, discovered_assets, processed_assets,
               skipped_directories, skipped_files, error_count, issues_truncated,
               error_code, created_at_ms, started_at_ms, finished_at_ms
        FROM scan_runs WHERE id = ?`,
		id,
	))
}

func scanScan(row rowValues) (library.Scan, error) {
	var (
		scan                  library.Scan
		issuesTruncated       int64
		errorCode             sql.NullString
		startedAt, finishedAt sql.NullInt64
	)
	err := row.Scan(
		&scan.ID, &scan.LibraryID, &scan.Generation,
		&scan.Trigger, &scan.Status, &scan.Phase, &scan.Revision,
		&scan.DiscoveredDirectories, &scan.DiscoveredAssets,
		&scan.ProcessedAssets, &scan.SkippedDirectories, &scan.SkippedFiles,
		&scan.ErrorCount, &issuesTruncated, &errorCode,
		&scan.CreatedAtMS, &startedAt, &finishedAt,
	)
	if err != nil {
		return library.Scan{}, err
	}
	scan.IssuesTruncated = issuesTruncated != 0
	if errorCode.Valid {
		scan.ErrorCode = errorCode.String
	}
	if startedAt.Valid {
		scan.StartedAtMS = &startedAt.Int64
	}
	if finishedAt.Valid {
		scan.FinishedAtMS = &finishedAt.Int64
	}
	return scan, nil
}

func (s *Store) ClaimNextLibraryRemoval(
	ctx context.Context,
) (library.Removal, bool, error) {
	var claimed library.Removal
	found := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var id int64
		err := tx.QueryRowContext(ctx, `
            SELECT id
            FROM library_removals r
            WHERE r.status IN ('queued', 'running')
              AND NOT EXISTS (
                  SELECT 1 FROM scan_runs s
                  WHERE s.library_id = r.library_id
                    AND s.status IN ('queued', 'running')
              )
            ORDER BY r.created_at_ms, r.id
            LIMIT 1`,
		).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("find queued library removal: %w", err)
		}
		now := s.nowMS()
		if _, err := tx.ExecContext(ctx, `
            UPDATE library_removals
            SET status = 'running',
                started_at_ms = COALESCE(started_at_ms, ?),
                revision = CASE WHEN status = 'queued' THEN revision + 1 ELSE revision END
            WHERE id = ?`,
			now, id,
		); err != nil {
			return fmt.Errorf("claim library removal: %w", err)
		}
		claimed, err = getRemoval(ctx, tx, id)
		found = err == nil
		return err
	})
	return claimed, found, err
}

func (s *Store) LibraryRemovalReady(ctx context.Context, removalID int64) (bool, error) {
	var active int64
	err := s.db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM scan_runs s
        JOIN library_removals r ON r.library_id = s.library_id
        WHERE r.id = ? AND r.status = 'running'
          AND s.status IN ('queued', 'running')`,
		removalID,
	).Scan(&active)
	if err != nil {
		return false, fmt.Errorf("check removal scan state: %w", err)
	}
	return active == 0, nil
}

func (s *Store) CleanupLibraryRemovalBatch(
	ctx context.Context,
	removalID int64,
	limit int,
) (bool, error) {
	done := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		removal, err := getRemoval(ctx, tx, removalID)
		if errors.Is(err, sql.ErrNoRows) {
			return library.ErrRemovalNotFound
		}
		if err != nil {
			return err
		}
		if removal.Status == library.RemovalSucceeded {
			done = true
			return nil
		}
		if removal.Status != library.RemovalRunning {
			return errors.New("library removal is not running")
		}
		var active int64
		if err := tx.QueryRowContext(ctx, `
            SELECT COUNT(*) FROM scan_runs
            WHERE library_id = ? AND status IN ('queued', 'running')`,
			removal.LibraryID,
		).Scan(&active); err != nil {
			return fmt.Errorf("check cleanup scan state: %w", err)
		}
		if active != 0 {
			return nil
		}
		for _, statement := range []string{
			`DELETE FROM assets WHERE id IN (
                SELECT id FROM assets WHERE library_id = ? LIMIT ?
            )`,
			`DELETE FROM directories WHERE id IN (
                SELECT d.id FROM directories d
                WHERE d.library_id = ?
                  AND NOT EXISTS (
                      SELECT 1 FROM directories child WHERE child.parent_id = d.id
                  )
                LIMIT ?
            )`,
			`DELETE FROM scan_issues WHERE id IN (
                SELECT i.id FROM scan_issues i
                JOIN scan_runs s ON s.id = i.scan_run_id
                WHERE s.library_id = ? LIMIT ?
            )`,
			`DELETE FROM scan_runs WHERE id IN (
                SELECT id FROM scan_runs WHERE library_id = ? LIMIT ?
            )`,
		} {
			result, err := tx.ExecContext(ctx, statement, removal.LibraryID, limit)
			if err != nil {
				return fmt.Errorf("delete library-derived batch: %w", err)
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("read cleanup batch result: %w", err)
			}
			if affected > 0 {
				return nil
			}
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM libraries WHERE id = ?`,
			removal.LibraryID,
		); err != nil {
			return fmt.Errorf("delete library configuration: %w", err)
		}
		now := s.nowMS()
		if _, err := tx.ExecContext(ctx, `
            UPDATE library_removals
            SET status = 'succeeded', finished_at_ms = ?,
                error_code = NULL, revision = revision + 1
            WHERE id = ? AND status = 'running'`,
			now, removalID,
		); err != nil {
			return fmt.Errorf("complete library removal: %w", err)
		}
		done = true
		return nil
	})
	return done, err
}

func (s *Store) FailLibraryRemoval(
	ctx context.Context,
	removalID int64,
	errorCode string,
) error {
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            UPDATE library_removals
            SET status = 'failed', error_code = ?, finished_at_ms = ?,
                revision = revision + 1
            WHERE id = ? AND status = 'running'`,
			errorCode, s.nowMS(), removalID,
		)
		if err != nil {
			return fmt.Errorf("fail library removal: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read failed removal result: %w", err)
		}
		if affected == 0 {
			return library.ErrRemovalNotFound
		}
		return nil
	})
}
