package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

const maxCacheEvictionBatch = 256

func (s *Store) CacheQuota(ctx context.Context) (int64, error) {
	var quota int64
	if err := s.db.QueryRowContext(ctx, `
        SELECT thumbnail_cache_quota_bytes
        FROM settings WHERE singleton_key = 1`,
	).Scan(&quota); err != nil {
		return 0, fmt.Errorf("read thumbnail cache quota: %w", err)
	}
	return quota, nil
}

func (s *Store) ReadyCacheUsage(ctx context.Context) (int64, error) {
	var usage int64
	if err := s.db.QueryRowContext(ctx, `
        SELECT COALESCE(SUM(byte_size), 0)
        FROM thumbnails WHERE status = 'ready'`,
	).Scan(&usage); err != nil {
		return 0, fmt.Errorf("read thumbnail cache usage: %w", err)
	}
	return usage, nil
}

func (s *Store) ListLRUCacheEntries(
	ctx context.Context,
	limit int,
) ([]thumbnail.CacheEntry, error) {
	if limit < 1 || limit > 256 {
		return nil, errors.New("cache eviction limit must be between 1 and 256")
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT asset_id, cache_rel_path, byte_size
        FROM thumbnails
        WHERE status = 'ready'
        ORDER BY last_accessed_at_ms, asset_id
        LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list LRU thumbnail cache entries: %w", err)
	}
	defer rows.Close()
	entries := make([]thumbnail.CacheEntry, 0, limit)
	for rows.Next() {
		var entry thumbnail.CacheEntry
		if err := rows.Scan(
			&entry.AssetID, &entry.CacheRelativePath, &entry.ByteSize,
		); err != nil {
			return nil, fmt.Errorf("read LRU thumbnail cache entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate LRU thumbnail cache entries: %w", err)
	}
	return entries, nil
}

func (s *Store) DeleteReadyCacheEntries(
	ctx context.Context,
	entries []thumbnail.CacheEntry,
) error {
	if len(entries) < 1 || len(entries) > maxCacheEvictionBatch {
		return thumbnail.ErrInvalidState
	}
	for _, entry := range entries {
		if entry.AssetID <= 0 || entry.CacheRelativePath == "" || entry.ByteSize <= 0 {
			return thumbnail.ErrInvalidState
		}
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		for _, entry := range entries {
			result, err := tx.ExecContext(ctx, `
            DELETE FROM thumbnails
            WHERE asset_id = ? AND cache_rel_path = ?
              AND byte_size = ? AND status = 'ready'`,
				entry.AssetID, entry.CacheRelativePath, entry.ByteSize,
			)
			if err != nil {
				return fmt.Errorf("delete evicted thumbnail state: %w", err)
			}
			changed, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("inspect evicted thumbnail state: %w", err)
			}
			if changed > 1 {
				return errors.New("cache eviction deleted multiple states")
			}
		}
		return nil
	})
}

func (s *Store) ListPendingCacheDeletions(
	ctx context.Context,
	limit int,
) ([]thumbnail.PendingCacheDeletion, error) {
	if limit < 1 || limit > 256 {
		return nil, errors.New("cache deletion limit must be between 1 and 256")
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, cache_rel_path
        FROM cache_deletions
        ORDER BY created_at_ms, id
        LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending cache deletions: %w", err)
	}
	defer rows.Close()
	pending := make([]thumbnail.PendingCacheDeletion, 0, limit)
	for rows.Next() {
		var item thumbnail.PendingCacheDeletion
		if err := rows.Scan(&item.ID, &item.CacheRelativePath); err != nil {
			return nil, fmt.Errorf("read pending cache deletion: %w", err)
		}
		pending = append(pending, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending cache deletions: %w", err)
	}
	return pending, nil
}

func (s *Store) CompleteCacheDeletion(
	ctx context.Context,
	item thumbnail.PendingCacheDeletion,
) error {
	if item.ID <= 0 || item.CacheRelativePath == "" {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteGate(ctx, func() error {
		result, err := s.db.ExecContext(ctx, `
            DELETE FROM cache_deletions WHERE id = ? AND cache_rel_path = ?`,
			item.ID, item.CacheRelativePath,
		)
		if err != nil {
			return fmt.Errorf("complete cache deletion: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect completed cache deletion: %w", err)
		}
		if changed == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
}

func (s *Store) GetCacheCleanup(ctx context.Context) (thumbnail.Cleanup, error) {
	return readCacheCleanup(ctx, s.db)
}

func (s *Store) RequestCacheCleanup(
	ctx context.Context,
	keyHash [sha256.Size]byte,
	requestedAtMS int64,
) (thumbnail.CleanupRequestResult, error) {
	var result thumbnail.CleanupRequestResult
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := readCacheCleanup(ctx, tx)
		if err != nil {
			return err
		}
		if current.Status == thumbnail.CleanupQueued ||
			current.Status == thumbnail.CleanupRunning {
			result.Cleanup = current
			result.Replayed = current.IdempotencyKeyHash == keyHash
			return nil
		}
		var existing int
		err = tx.QueryRowContext(ctx, `
            SELECT EXISTS(
                SELECT 1 FROM idempotency_records
                WHERE operation = 'cache_cleanup' AND key_hash = ?
                  AND expires_at_ms > ?
            )`,
			keyHash[:],
			requestedAtMS,
		).Scan(&existing)
		if err != nil {
			return fmt.Errorf("read cache cleanup idempotency: %w", err)
		}
		if existing != 0 {
			result.Cleanup = current
			result.Replayed = true
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
            DELETE FROM idempotency_records
            WHERE expires_at_ms <= ?`,
			requestedAtMS,
		); err != nil {
			return fmt.Errorf("expire cache cleanup idempotency: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
            UPDATE cache_cleanup_state
            SET revision = revision + 1,
                status = 'queued',
                idempotency_key_hash = ?,
                requested_at_ms = ?,
                started_at_ms = NULL,
                finished_at_ms = NULL,
                initial_usage_bytes = 0,
                remaining_usage_bytes = 0,
                reclaimed_bytes = 0,
                deleted_entries = 0,
                error_code = NULL
            WHERE singleton_key = 1`,
			keyHash[:],
			requestedAtMS,
		)
		if err != nil {
			return fmt.Errorf("queue cache cleanup: %w", err)
		}
		result.Cleanup, err = readCacheCleanup(ctx, tx)
		if err != nil {
			return err
		}
		requestHash := sha256.Sum256(nil)
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO idempotency_records(
                operation, key_hash, request_hash, result_kind, result_id,
                created_at_ms, expires_at_ms
            ) VALUES ('cache_cleanup', ?, ?, 'cache_cleanup', 1, ?, ?)`,
			keyHash[:],
			requestHash[:],
			requestedAtMS,
			requestedAtMS+int64(24*time.Hour/time.Millisecond),
		); err != nil {
			return fmt.Errorf("record cache cleanup idempotency: %w", err)
		}
		result.Created = err == nil
		return nil
	})
	return result, err
}

func (s *Store) ClaimCacheCleanup(
	ctx context.Context,
	usageBytes int64,
	startedAtMS int64,
) (thumbnail.Cleanup, bool, error) {
	var (
		cleanup thumbnail.Cleanup
		claimed bool
	)
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := readCacheCleanup(ctx, tx)
		if err != nil {
			return err
		}
		if current.Status != thumbnail.CleanupQueued &&
			current.Status != thumbnail.CleanupRunning {
			cleanup = current
			return nil
		}
		if current.Status == thumbnail.CleanupQueued {
			_, err = tx.ExecContext(ctx, `
                UPDATE cache_cleanup_state
                SET revision = revision + 1,
                    status = 'running',
                    started_at_ms = ?,
                    initial_usage_bytes = ?,
                    remaining_usage_bytes = ?
                WHERE singleton_key = 1 AND status = 'queued'`,
				startedAtMS,
				usageBytes,
				usageBytes,
			)
			if err != nil {
				return fmt.Errorf("claim cache cleanup: %w", err)
			}
		}
		cleanup, err = readCacheCleanup(ctx, tx)
		claimed = err == nil
		return err
	})
	return cleanup, claimed, err
}

func (s *Store) UpdateCacheCleanupProgress(
	ctx context.Context,
	progress thumbnail.CleanupProgress,
) error {
	return s.withWriteGate(ctx, func() error {
		result, err := s.db.ExecContext(ctx, `
            UPDATE cache_cleanup_state
            SET revision = revision + 1,
                remaining_usage_bytes = ?,
                reclaimed_bytes = ?,
                deleted_entries = ?
            WHERE singleton_key = 1 AND status = 'running'`,
			progress.RemainingUsageBytes,
			progress.ReclaimedBytes,
			progress.DeletedEntries,
		)
		if err != nil {
			return fmt.Errorf("update cache cleanup progress: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return thumbnail.ErrInvalidState
		}
		return nil
	})
}

func (s *Store) FinishCacheCleanup(
	ctx context.Context,
	status thumbnail.CleanupStatus,
	errorCode *string,
	finishedAtMS int64,
) (thumbnail.Cleanup, error) {
	if status != thumbnail.CleanupSucceeded && status != thumbnail.CleanupFailed {
		return thumbnail.Cleanup{}, thumbnail.ErrInvalidState
	}
	var cleanup thumbnail.Cleanup
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            UPDATE cache_cleanup_state
            SET revision = revision + 1,
                status = ?,
                finished_at_ms = ?,
                error_code = ?
            WHERE singleton_key = 1 AND status = 'running'`,
			string(status),
			finishedAtMS,
			errorCode,
		)
		if err != nil {
			return fmt.Errorf("finish cache cleanup: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return thumbnail.ErrInvalidState
		}
		cleanup, err = readCacheCleanup(ctx, tx)
		return err
	})
	return cleanup, err
}

type cacheCleanupQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readCacheCleanup(
	ctx context.Context,
	querier cacheCleanupQuerier,
) (thumbnail.Cleanup, error) {
	var (
		cleanup     thumbnail.Cleanup
		requestedAt sql.NullInt64
		startedAt   sql.NullInt64
		finishedAt  sql.NullInt64
		errorCode   sql.NullString
		keyHash     []byte
	)
	err := querier.QueryRowContext(ctx, `
        SELECT revision, status, idempotency_key_hash,
               requested_at_ms, started_at_ms, finished_at_ms,
               initial_usage_bytes, remaining_usage_bytes,
               reclaimed_bytes, deleted_entries, error_code
        FROM cache_cleanup_state WHERE singleton_key = 1`,
	).Scan(
		&cleanup.Revision,
		&cleanup.Status,
		&keyHash,
		&requestedAt,
		&startedAt,
		&finishedAt,
		&cleanup.InitialUsageBytes,
		&cleanup.RemainingUsageBytes,
		&cleanup.ReclaimedBytes,
		&cleanup.DeletedEntries,
		&errorCode,
	)
	if err != nil {
		return thumbnail.Cleanup{}, fmt.Errorf("read cache cleanup: %w", err)
	}
	if requestedAt.Valid {
		cleanup.RequestedAtMS = &requestedAt.Int64
	}
	if startedAt.Valid {
		cleanup.StartedAtMS = &startedAt.Int64
	}
	if finishedAt.Valid {
		cleanup.FinishedAtMS = &finishedAt.Int64
	}
	if errorCode.Valid {
		cleanup.ErrorCode = &errorCode.String
	}
	copy(cleanup.IdempotencyKeyHash[:], keyHash)
	return cleanup, nil
}
