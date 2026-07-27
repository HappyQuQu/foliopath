package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func TestThumbnailStateTransitionsAreFingerprintGuarded(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var assetID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM assets
        WHERE library_id = ? AND relative_path = 'photo-2.jpg'`,
		libraryID,
	).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	asset, err := store.GetAssetForDerivation(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	if asset.LibraryRoot != "photos" || asset.RelativePath != "photo-2.jpg" ||
		asset.Format != media.FormatJPEG {
		t.Fatalf("derivation asset = %#v", asset)
	}
	if err := store.CommitFailure(context.Background(), thumbnail.Failure{
		AssetID: asset.ID, SourceFingerprint: asset.SourceFingerprint,
		Code: media.ErrorInvalidMedia,
	}); err != nil {
		t.Fatal(err)
	}
	var probeStatus, probeCode, thumbnailStatus, thumbnailCode string
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT a.probe_status, a.probe_error_code, t.status, t.error_code
        FROM assets a JOIN thumbnails t ON t.asset_id = a.id
        WHERE a.id = ?`,
		asset.ID,
	).Scan(&probeStatus, &probeCode, &thumbnailStatus, &thumbnailCode); err != nil {
		t.Fatal(err)
	}
	if probeStatus != "failed" || probeCode != "invalid_media" ||
		thumbnailStatus != "failed" || thumbnailCode != "invalid_media" {
		t.Fatalf("failure state = %q, %q, %q, %q",
			probeStatus, probeCode, thumbnailStatus, thumbnailCode)
	}

	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 96, Height: 64, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{
			Bytes: []byte("not stored in SQLite"), Width: 48, Height: 32,
		},
	}
	if err := store.CommitReady(context.Background(), thumbnail.Ready{
		AssetID: asset.ID, SourceFingerprint: asset.SourceFingerprint,
		Result: result, CacheRelativePath: "libraries/lib_1/aa/key.webp",
		ByteSize: 123, CreatedAtMS: 1000,
	}); err != nil {
		t.Fatal(err)
	}
	var width, height, byteSize int64
	var cachePath string
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT a.probe_status, t.status, t.cache_rel_path,
               t.width, t.height, t.byte_size
        FROM assets a JOIN thumbnails t ON t.asset_id = a.id
        WHERE a.id = ?`,
		asset.ID,
	).Scan(
		&probeStatus, &thumbnailStatus, &cachePath, &width, &height, &byteSize,
	); err != nil {
		t.Fatal(err)
	}
	if probeStatus != "ready" || thumbnailStatus != "ready" ||
		cachePath != "libraries/lib_1/aa/key.webp" ||
		width != 48 || height != 32 || byteSize != 123 {
		t.Fatalf("ready state = %q, %q, %q, %d, %d, %d",
			probeStatus, thumbnailStatus, cachePath, width, height, byteSize)
	}

	if err := store.CommitFailure(context.Background(), thumbnail.Failure{
		AssetID:           asset.ID,
		SourceFingerprint: media.SourceFingerprint("v1:999:999"),
		Code:              media.ErrorProcessingFailed,
	}); !errors.Is(err, thumbnail.ErrSourceChanged) {
		t.Fatalf("stale failure error = %v", err)
	}
}

func TestThumbnailDeliveryStateTouchOfflineAndMissingCacheRepair(t *testing.T) {
	store, _ := openTestStore(t)
	store.now = func() time.Time { return time.UnixMilli(2000) }
	libraryID := seedBrowseCatalog(t, store)
	var assetID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM assets
        WHERE library_id = ? AND relative_path = 'photo-2.jpg'`,
		libraryID,
	).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	asset, err := store.GetAssetForDerivation(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 96, Height: 64, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{Bytes: []byte("webp"), Width: 48, Height: 32},
	}
	const cachePath = "libraries/lib_1/aa/delivery.webp"
	if err := store.CommitReady(context.Background(), thumbnail.Ready{
		AssetID: asset.ID, SourceFingerprint: asset.SourceFingerprint,
		Result: result, CacheRelativePath: cachePath,
		ByteSize: 4, CreatedAtMS: 1000,
	}); err != nil {
		t.Fatal(err)
	}
	state, err := store.GetThumbnailDelivery(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != thumbnail.DeliveryReady ||
		state.CacheRelativePath != cachePath || state.ByteSize != 4 {
		t.Fatalf("ready delivery state = %#v", state)
	}
	if err := store.TouchThumbnail(
		context.Background(), assetID, asset.SourceFingerprint, cachePath,
	); err != nil {
		t.Fatal(err)
	}
	var accessed int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT last_accessed_at_ms FROM thumbnails WHERE asset_id = ?`,
		assetID,
	).Scan(&accessed); err != nil {
		t.Fatal(err)
	}
	if accessed != 2000 {
		t.Fatalf("last accessed = %d", accessed)
	}

	if err := store.RequeueMissingThumbnail(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	var probeStatus, jobStatus string
	var attempts int
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT a.probe_status, j.status, j.attempt_count
        FROM assets a JOIN media_jobs j ON j.asset_id = a.id
        WHERE a.id = ?`,
		assetID,
	).Scan(&probeStatus, &jobStatus, &attempts); err != nil {
		t.Fatal(err)
	}
	if probeStatus != "pending" || jobStatus != "queued" || attempts != 0 {
		t.Fatalf("repair state = %q, %q, %d", probeStatus, jobStatus, attempts)
	}
	state, err = store.GetThumbnailDelivery(context.Background(), assetID)
	if err != nil || state.Status != thumbnail.DeliveryQueued {
		t.Fatalf("repaired delivery state = %#v, %v", state, err)
	}

	if _, err := store.db.ExecContext(context.Background(), `
        UPDATE libraries SET status = 'offline' WHERE id = ?`,
		libraryID,
	); err != nil {
		t.Fatal(err)
	}
	state, err = store.GetThumbnailDelivery(context.Background(), assetID)
	if err != nil || state.Status != thumbnail.DeliveryOffline ||
		state.ErrorCode != media.ProcessingErrorCode("source_offline") {
		t.Fatalf("offline delivery state = %#v, %v", state, err)
	}
}
