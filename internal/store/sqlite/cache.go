package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

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

func (s *Store) DeleteReadyCacheEntry(
	ctx context.Context,
	entry thumbnail.CacheEntry,
) error {
	if entry.AssetID <= 0 || entry.CacheRelativePath == "" || entry.ByteSize <= 0 {
		return thumbnail.ErrInvalidState
	}
	result, err := s.db.ExecContext(ctx, `
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
	return nil
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
}
