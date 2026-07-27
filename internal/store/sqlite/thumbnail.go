package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func (s *Store) GetAssetForDerivation(
	ctx context.Context,
	assetID int64,
) (thumbnail.Asset, error) {
	if assetID <= 0 {
		return thumbnail.Asset{}, thumbnail.ErrAssetNotFound
	}
	var asset thumbnail.Asset
	var kind, format, fingerprint string
	if err := s.db.QueryRowContext(ctx, `
        SELECT a.id, a.library_id, l.root_rel_path, a.relative_path,
               a.kind, a.media_format, a.size_bytes, a.mtime_ns,
               a.source_fingerprint
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        WHERE a.id = ?`,
		assetID,
	).Scan(
		&asset.ID, &asset.LibraryID, &asset.LibraryRoot, &asset.RelativePath,
		&kind, &format, &asset.SizeBytes, &asset.ModifiedAtNS, &fingerprint,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return thumbnail.Asset{}, thumbnail.ErrAssetNotFound
		}
		return thumbnail.Asset{}, fmt.Errorf("get thumbnail asset: %w", err)
	}
	asset.Kind = media.Kind(kind)
	asset.Format = media.Format(format)
	asset.SourceFingerprint = media.SourceFingerprint(fingerprint)
	return asset, nil
}

func (s *Store) CommitReady(ctx context.Context, ready thumbnail.Ready) error {
	if ready.AssetID <= 0 || ready.SourceFingerprint == "" ||
		ready.CacheRelativePath == "" || ready.ByteSize <= 0 ||
		ready.CreatedAtMS <= 0 {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var kind string
		result := ready.Result
		if err := tx.QueryRowContext(ctx, `
            SELECT kind FROM assets
            WHERE id = ? AND source_fingerprint = ?`,
			ready.AssetID, ready.SourceFingerprint.String(),
		).Scan(&kind); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return thumbnail.ErrSourceChanged
			}
			return fmt.Errorf("verify ready thumbnail source: %w", err)
		}
		if err := media.ValidateProcessingResult(media.Kind(kind), result); err != nil {
			return thumbnail.ErrInvalidState
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE assets
            SET width = ?, height = ?, duration_ms = ?,
                probe_status = 'ready', probe_error_code = NULL,
                playback_status = ?
            WHERE id = ? AND source_fingerprint = ?`,
			result.Metadata.Width, result.Metadata.Height, result.Metadata.DurationMS,
			string(result.Metadata.PlaybackStatus),
			ready.AssetID, ready.SourceFingerprint.String(),
		); err != nil {
			return fmt.Errorf("commit ready asset metadata: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO thumbnails(
                library_id, asset_id, variant, source_fingerprint,
                transform_version, cache_rel_path, status, error_code,
                width, height, byte_size, created_at_ms, last_accessed_at_ms
            )
            SELECT library_id, id, 'grid', source_fingerprint,
                   ?, ?, 'ready', NULL, ?, ?, ?, ?, ?
            FROM assets
            WHERE id = ? AND source_fingerprint = ?
            ON CONFLICT(asset_id, variant) DO UPDATE SET
                library_id = excluded.library_id,
                source_fingerprint = excluded.source_fingerprint,
                transform_version = excluded.transform_version,
                cache_rel_path = excluded.cache_rel_path,
                status = excluded.status,
                error_code = NULL,
                width = excluded.width,
                height = excluded.height,
                byte_size = excluded.byte_size,
                created_at_ms = excluded.created_at_ms,
                last_accessed_at_ms = excluded.last_accessed_at_ms`,
			thumbnail.GridTransformVersion, ready.CacheRelativePath,
			result.Thumbnail.Width, result.Thumbnail.Height, ready.ByteSize,
			ready.CreatedAtMS, ready.CreatedAtMS,
			ready.AssetID, ready.SourceFingerprint.String(),
		); err != nil {
			return fmt.Errorf("commit ready thumbnail: %w", err)
		}
		return nil
	})
}

func (s *Store) CommitFailure(ctx context.Context, failure thumbnail.Failure) error {
	if failure.AssetID <= 0 || failure.SourceFingerprint == "" ||
		(failure.Code != media.ErrorUnsupportedMedia &&
			failure.Code != media.ErrorInvalidMedia &&
			failure.Code != media.ErrorProcessingFailed &&
			failure.Code != media.ErrorProcessingTimed) {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		status := string(media.ProbeFailed)
		if failure.Code == media.ErrorUnsupportedMedia {
			status = string(media.ProbeUnsupported)
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE assets
            SET width = NULL, height = NULL, duration_ms = NULL,
                probe_status = ?, probe_error_code = ?,
                playback_status = 'unknown'
            WHERE id = ? AND source_fingerprint = ?`,
			status, string(failure.Code),
			failure.AssetID, failure.SourceFingerprint.String(),
		)
		if err != nil {
			return fmt.Errorf("commit failed asset metadata: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect failed asset update: %w", err)
		}
		if changed != 1 {
			return thumbnail.ErrSourceChanged
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
                library_id = excluded.library_id,
                source_fingerprint = excluded.source_fingerprint,
                transform_version = excluded.transform_version,
                cache_rel_path = NULL,
                status = 'failed',
                error_code = excluded.error_code,
                width = NULL,
                height = NULL,
                byte_size = NULL,
                created_at_ms = NULL,
                last_accessed_at_ms = NULL`,
			thumbnail.GridTransformVersion, string(failure.Code),
			failure.AssetID, failure.SourceFingerprint.String(),
		); err != nil {
			return fmt.Errorf("commit failed thumbnail: %w", err)
		}
		return nil
	})
}
