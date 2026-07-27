package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestGetContentAssetReturnsIndexedIdentityAndOfflineState(t *testing.T) {
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

	asset, err := store.GetContentAsset(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	if asset.ID != assetID || asset.LibraryRoot != "photos" ||
		asset.RelativePath != "photo-2.jpg" || asset.Format != media.FormatJPEG ||
		asset.MIMEType != "image/jpeg" || asset.LibraryOffline ||
		!asset.SourceFingerprint.Matches(asset.SizeBytes, asset.ModifiedAtNS) {
		t.Fatalf("content asset = %#v", asset)
	}

	if _, err := store.db.ExecContext(context.Background(),
		`UPDATE libraries SET status = 'offline' WHERE id = ?`, libraryID,
	); err != nil {
		t.Fatal(err)
	}
	asset, err = store.GetContentAsset(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	if !asset.LibraryOffline {
		t.Fatal("offline library was reported available")
	}
	if _, err := store.GetContentAsset(context.Background(), -1); !errors.Is(
		err,
		media.ErrContentAssetNotFound,
	) {
		t.Fatalf("invalid asset error = %v", err)
	}
}
