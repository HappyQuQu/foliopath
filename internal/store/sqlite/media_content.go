package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func (s *Store) GetContentAsset(
	ctx context.Context,
	assetID int64,
) (media.ContentAsset, error) {
	if assetID <= 0 {
		return media.ContentAsset{}, media.ErrContentAssetNotFound
	}
	var asset media.ContentAsset
	var format, fingerprint, libraryStatus string
	err := s.db.QueryRowContext(ctx, `
        SELECT a.id, l.root_rel_path, a.relative_path, a.media_format,
               a.mime_type, a.size_bytes, a.mtime_ns, a.source_fingerprint,
               l.status
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        WHERE a.id = ?`,
		assetID,
	).Scan(
		&asset.ID, &asset.LibraryRoot, &asset.RelativePath, &format,
		&asset.MIMEType, &asset.SizeBytes, &asset.ModifiedAtNS, &fingerprint,
		&libraryStatus,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return media.ContentAsset{}, media.ErrContentAssetNotFound
	}
	if err != nil {
		return media.ContentAsset{}, fmt.Errorf("get content asset: %w", err)
	}
	asset.Format = media.Format(format)
	asset.SourceFingerprint = media.SourceFingerprint(fingerprint)
	asset.LibraryOffline = libraryStatus == "offline"
	return asset, nil
}
