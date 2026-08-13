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
	var kind, format, fingerprint, probeStatus string
	var width, height, duration sql.NullInt64
	var gridReady int
	if err := s.db.QueryRowContext(ctx, `
        SELECT a.id, a.library_id, l.root_rel_path, a.relative_path,
               a.kind, a.media_format, a.size_bytes, a.mtime_ns,
               a.source_fingerprint, a.width, a.height, a.duration_ms,
               a.probe_status,
               EXISTS (
                   SELECT 1 FROM thumbnails AS grid
                   WHERE grid.asset_id = a.id
                     AND grid.variant = 'grid'
                     AND grid.status = 'ready'
                     AND grid.source_fingerprint = a.source_fingerprint
               )
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        WHERE a.id = ?`,
		assetID,
	).Scan(
		&asset.ID, &asset.LibraryID, &asset.LibraryRoot, &asset.RelativePath,
		&kind, &format, &asset.SizeBytes, &asset.ModifiedAtNS, &fingerprint,
		&width, &height, &duration, &probeStatus, &gridReady,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return thumbnail.Asset{}, thumbnail.ErrAssetNotFound
		}
		return thumbnail.Asset{}, fmt.Errorf("get thumbnail asset: %w", err)
	}
	asset.Kind = media.Kind(kind)
	asset.Format = media.Format(format)
	asset.SourceFingerprint = media.SourceFingerprint(fingerprint)
	asset.Width = int(width.Int64)
	asset.Height = int(height.Int64)
	if duration.Valid {
		value := duration.Int64
		asset.DurationMS = &value
	}
	asset.ProbeStatus = media.ProbeStatus(probeStatus)
	asset.GridReady = gridReady != 0
	return asset, nil
}

func (s *Store) CommitStoryboardReady(
	ctx context.Context,
	ready thumbnail.StoryboardReady,
) error {
	if ready.AssetID <= 0 || !ready.SourceFingerprint.Valid() ||
		ready.CacheRelativePath == "" || ready.ByteSize <= 0 ||
		ready.CreatedAtMS <= 0 {
		return thumbnail.ErrInvalidState
	}
	layout := thumbnail.StoryboardLayout{
		FrameCount: ready.Result.FrameCount,
		Columns:    ready.Result.Columns,
		Rows:       ready.Result.Rows,
		CellWidth:  ready.Result.CellWidth,
		CellHeight: ready.Result.CellHeight,
	}
	if layout.Validate() != nil || len(ready.Result.Bytes) != int(ready.ByteSize) {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var duration int64
		if err := tx.QueryRowContext(ctx, `
            SELECT duration_ms
            FROM assets
            WHERE id = ? AND source_fingerprint = ?
              AND kind = 'video' AND probe_status = 'ready'
              AND duration_ms >= ?`,
			ready.AssetID,
			ready.SourceFingerprint.String(),
			thumbnail.StoryboardMinimumDurationMS,
		).Scan(&duration); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return thumbnail.ErrSourceChanged
			}
			return fmt.Errorf("verify ready storyboard source: %w", err)
		}
		request := media.StoryboardRequest{
			TimestampsMS: make([]int64, ready.Result.FrameCount),
			Columns:      ready.Result.Columns,
			Rows:         ready.Result.Rows,
			CellWidth:    ready.Result.CellWidth,
			CellHeight:   ready.Result.CellHeight,
		}
		plan, err := thumbnail.NewStoryboardPlan(duration)
		if err != nil || len(plan.TimestampsMS) != ready.Result.FrameCount {
			return thumbnail.ErrInvalidState
		}
		request.TimestampsMS = plan.TimestampsMS
		if media.ValidateStoryboardResult(request, ready.Result) != nil {
			return thumbnail.ErrInvalidState
		}
		width := ready.Result.Columns * ready.Result.CellWidth
		height := ready.Result.Rows * ready.Result.CellHeight
		result, err := tx.ExecContext(ctx, `
            INSERT INTO thumbnails(
                library_id, asset_id, variant, source_fingerprint,
                transform_version, cache_rel_path, status, error_code,
                width, height, byte_size, created_at_ms, last_accessed_at_ms,
                frame_count, sprite_columns, sprite_rows, cell_width, cell_height
            )
            SELECT library_id, id, 'storyboard', source_fingerprint,
                   ?, ?, 'ready', NULL, ?, ?, ?, ?, ?,
                   ?, ?, ?, ?, ?
            FROM assets
            WHERE id = ? AND source_fingerprint = ?
            ON CONFLICT(asset_id, variant) DO UPDATE SET
                library_id = excluded.library_id,
                source_fingerprint = excluded.source_fingerprint,
                transform_version = excluded.transform_version,
                cache_rel_path = excluded.cache_rel_path,
                status = 'ready', error_code = NULL,
                width = excluded.width, height = excluded.height,
                byte_size = excluded.byte_size,
                created_at_ms = excluded.created_at_ms,
                last_accessed_at_ms = excluded.last_accessed_at_ms,
                frame_count = excluded.frame_count,
                sprite_columns = excluded.sprite_columns,
                sprite_rows = excluded.sprite_rows,
                cell_width = excluded.cell_width,
                cell_height = excluded.cell_height`,
			thumbnail.StoryboardTransformVersion,
			ready.CacheRelativePath,
			width,
			height,
			ready.ByteSize,
			ready.CreatedAtMS,
			ready.CreatedAtMS,
			ready.Result.FrameCount,
			ready.Result.Columns,
			ready.Result.Rows,
			ready.Result.CellWidth,
			ready.Result.CellHeight,
			ready.AssetID,
			ready.SourceFingerprint.String(),
		)
		if err != nil {
			return fmt.Errorf("commit ready storyboard: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect ready storyboard commit: %w", err)
		}
		if changed != 1 {
			return thumbnail.ErrSourceChanged
		}
		return nil
	})
}

func (s *Store) CommitStoryboardFailure(
	ctx context.Context,
	failure thumbnail.StoryboardFailure,
) error {
	if failure.AssetID <= 0 || !failure.SourceFingerprint.Valid() ||
		(failure.Code != media.ErrorUnsupportedMedia &&
			failure.Code != media.ErrorInvalidMedia &&
			failure.Code != media.ErrorProcessingFailed &&
			failure.Code != media.ErrorProcessingTimed) {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            INSERT INTO thumbnails(
                library_id, asset_id, variant, source_fingerprint,
                transform_version, status, error_code
            )
            SELECT library_id, id, 'storyboard', source_fingerprint,
                   ?, 'failed', ?
            FROM assets
            WHERE id = ? AND source_fingerprint = ?
            ON CONFLICT(asset_id, variant) DO UPDATE SET
                library_id = excluded.library_id,
                source_fingerprint = excluded.source_fingerprint,
                transform_version = excluded.transform_version,
                cache_rel_path = NULL, status = 'failed',
                error_code = excluded.error_code,
                width = NULL, height = NULL, byte_size = NULL,
                created_at_ms = NULL, last_accessed_at_ms = NULL,
                frame_count = NULL, sprite_columns = NULL, sprite_rows = NULL,
                cell_width = NULL, cell_height = NULL`,
			thumbnail.StoryboardTransformVersion,
			string(failure.Code),
			failure.AssetID,
			failure.SourceFingerprint.String(),
		)
		if err != nil {
			return fmt.Errorf("commit failed storyboard: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect failed storyboard commit: %w", err)
		}
		if changed != 1 {
			return thumbnail.ErrSourceChanged
		}
		return nil
	})
}

func (s *Store) CommitReady(ctx context.Context, ready thumbnail.Ready) error {
	if ready.AssetID <= 0 || ready.SourceFingerprint == "" ||
		ready.CacheRelativePath == "" || ready.ByteSize <= 0 ||
		ready.CreatedAtMS <= 0 {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var kind, format string
		result := ready.Result
		if err := tx.QueryRowContext(ctx, `
            SELECT kind, media_format FROM assets
            WHERE id = ? AND source_fingerprint = ?`,
			ready.AssetID, ready.SourceFingerprint.String(),
		).Scan(&kind, &format); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return thumbnail.ErrSourceChanged
			}
			return fmt.Errorf("verify ready thumbnail source: %w", err)
		}
		if err := media.ValidateProcessingResult(
			media.Kind(kind), media.Format(format), result,
		); err != nil {
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

func (s *Store) CommitMetadataReadyFailure(
	ctx context.Context,
	failure thumbnail.MetadataReadyFailure,
) error {
	if failure.AssetID <= 0 || !failure.SourceFingerprint.Valid() ||
		media.ValidateMetadata(media.KindVideo, failure.Metadata) != nil ||
		(failure.Code != media.ErrorUnsupportedMedia &&
			failure.Code != media.ErrorInvalidMedia &&
			failure.Code != media.ErrorProcessingFailed &&
			failure.Code != media.ErrorProcessingTimed) {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            UPDATE assets
            SET width = ?, height = ?, duration_ms = ?,
                probe_status = 'ready', probe_error_code = NULL,
                playback_status = ?
            WHERE id = ? AND source_fingerprint = ? AND kind = 'video'`,
			failure.Metadata.Width,
			failure.Metadata.Height,
			failure.Metadata.DurationMS,
			string(failure.Metadata.PlaybackStatus),
			failure.AssetID,
			failure.SourceFingerprint.String(),
		)
		if err != nil {
			return fmt.Errorf("commit ready video metadata: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect ready video metadata update: %w", err)
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
			thumbnail.GridTransformVersion,
			string(failure.Code),
			failure.AssetID,
			failure.SourceFingerprint.String(),
		); err != nil {
			return fmt.Errorf("commit failed video thumbnail: %w", err)
		}
		return nil
	})
}

func (s *Store) GetThumbnailDelivery(
	ctx context.Context,
	assetID int64,
	variant thumbnail.Variant,
) (thumbnail.DeliveryState, error) {
	if assetID <= 0 ||
		(variant != thumbnail.VariantGrid &&
			variant != thumbnail.VariantStoryboard) {
		return thumbnail.DeliveryState{}, thumbnail.ErrAssetNotFound
	}
	state := thumbnail.DeliveryState{Variant: variant}
	var fingerprint, libraryStatus string
	var kind string
	var duration sql.NullInt64
	var thumbnailStatus, jobStatus, cachePath, errorCode sql.NullString
	var byteSize sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
        SELECT a.id, a.source_fingerprint, l.status, a.kind, a.duration_ms,
               t.status, t.cache_rel_path, t.byte_size, t.error_code,
               j.status
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        LEFT JOIN thumbnails t ON t.asset_id = a.id AND t.variant = ?
        LEFT JOIN media_jobs j ON j.asset_id = a.id AND j.variant = ?
        WHERE a.id = ?`,
		string(variant), string(variant), assetID,
	).Scan(
		&state.AssetID, &fingerprint, &libraryStatus, &kind, &duration,
		&thumbnailStatus, &cachePath, &byteSize, &errorCode, &jobStatus,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return thumbnail.DeliveryState{}, thumbnail.ErrAssetNotFound
	}
	if err != nil {
		return thumbnail.DeliveryState{}, fmt.Errorf("get thumbnail delivery: %w", err)
	}
	state.SourceFingerprint = media.SourceFingerprint(fingerprint)
	if !state.SourceFingerprint.Valid() {
		return thumbnail.DeliveryState{}, thumbnail.ErrInvalidState
	}
	if variant == thumbnail.VariantStoryboard &&
		(kind != string(media.KindVideo) ||
			!duration.Valid ||
			duration.Int64 < thumbnail.StoryboardMinimumDurationMS) {
		return thumbnail.DeliveryState{}, thumbnail.ErrStoryboardNotEligible
	}
	if libraryStatus == "offline" {
		state.Status = thumbnail.DeliveryOffline
		state.ErrorCode = media.ProcessingErrorCode("source_offline")
		return state, nil
	}
	if thumbnailStatus.String == "ready" {
		state.Status = thumbnail.DeliveryReady
		state.CacheRelativePath = cachePath.String
		state.ByteSize = byteSize.Int64
		return state, nil
	}
	if thumbnailStatus.String == "failed" {
		state.Status = thumbnail.DeliveryFailed
		state.ErrorCode = media.ProcessingErrorCode(errorCode.String)
		return state, nil
	}
	if jobStatus.String == "running" {
		state.Status = thumbnail.DeliveryRunning
	} else {
		state.Status = thumbnail.DeliveryQueued
	}
	return state, nil
}

func (s *Store) TouchThumbnail(
	ctx context.Context,
	assetID int64,
	variant thumbnail.Variant,
	fingerprint media.SourceFingerprint,
	cachePath string,
) error {
	if assetID <= 0 || !fingerprint.Valid() || cachePath == "" ||
		(variant != thumbnail.VariantGrid &&
			variant != thumbnail.VariantStoryboard) {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            UPDATE thumbnails
            SET last_accessed_at_ms = ?
            WHERE asset_id = ? AND variant = ? AND status = 'ready'
              AND source_fingerprint = ? AND cache_rel_path = ?`,
			s.nowMS(), assetID, string(variant), fingerprint.String(), cachePath,
		)
		if err != nil {
			return fmt.Errorf("touch thumbnail delivery: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect thumbnail delivery touch: %w", err)
		}
		if changed != 1 {
			return thumbnail.ErrSourceChanged
		}
		return nil
	})
}

func (s *Store) RequeueMissingThumbnail(
	ctx context.Context,
	state thumbnail.DeliveryState,
) error {
	if state.AssetID <= 0 || !state.SourceFingerprint.Valid() ||
		state.Status != thumbnail.DeliveryReady ||
		state.CacheRelativePath == "" || state.ByteSize <= 0 ||
		(state.Variant != thumbnail.VariantGrid &&
			state.Variant != thumbnail.VariantStoryboard) {
		return thumbnail.ErrInvalidState
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            DELETE FROM thumbnails
            WHERE asset_id = ? AND variant = ? AND status = 'ready'
              AND source_fingerprint = ? AND cache_rel_path = ? AND byte_size = ?`,
			state.AssetID, string(state.Variant), state.SourceFingerprint.String(),
			state.CacheRelativePath, state.ByteSize,
		)
		if err != nil {
			return fmt.Errorf("invalidate missing thumbnail cache: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect missing thumbnail invalidation: %w", err)
		}
		if changed != 1 {
			return thumbnail.ErrSourceChanged
		}
		if state.Variant == thumbnail.VariantGrid {
			if _, err := tx.ExecContext(ctx, `
            UPDATE assets
            SET width = NULL, height = NULL, duration_ms = NULL,
                probe_status = 'pending', probe_error_code = NULL,
                playback_status = 'unknown'
            WHERE id = ? AND source_fingerprint = ?`,
				state.AssetID, state.SourceFingerprint.String(),
			); err != nil {
				return fmt.Errorf("reset missing thumbnail metadata: %w", err)
			}
		}
		now := s.nowMS()
		job, err := tx.ExecContext(ctx, `
            UPDATE media_jobs
            SET transform_version = ?, status = 'queued',
                last_error_code = NULL, available_at_ms = ?,
                started_at_ms = NULL, heartbeat_at_ms = NULL,
                lease_expires_at_ms = NULL, attempt_count = 0,
                created_at_ms = ?, finished_at_ms = NULL
            WHERE asset_id = ? AND variant = ?
              AND source_fingerprint = ?`,
			transformVersion(state.Variant), now, now,
			state.AssetID, string(state.Variant), state.SourceFingerprint.String(),
		)
		if err != nil {
			return fmt.Errorf("requeue missing thumbnail: %w", err)
		}
		changed, err = job.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect missing thumbnail requeue: %w", err)
		}
		if changed != 1 {
			return thumbnail.ErrSourceChanged
		}
		return nil
	})
}

func transformVersion(variant thumbnail.Variant) int {
	if variant == thumbnail.VariantStoryboard {
		return thumbnail.StoryboardTransformVersion
	}
	return thumbnail.GridTransformVersion
}
