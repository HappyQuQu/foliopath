package media

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"strconv"
	"time"
)

var (
	ErrContentAssetNotFound = errors.New("content asset not found")
	ErrContentSourceOffline = errors.New("content source offline")
	ErrContentSourceChanged = errors.New("content source changed")
	ErrContentUnavailable   = errors.New("content unavailable")
	ErrInvalidContentState  = errors.New("invalid content state")
)

// ContentAsset is the indexed source identity needed to open an original. The
// repository remains the only adapter that knows the database schema.
type ContentAsset struct {
	ID                int64
	LibraryRoot       string
	RelativePath      string
	Format            Format
	MIMEType          string
	SizeBytes         int64
	ModifiedAtNS      int64
	SourceFingerprint SourceFingerprint
	LibraryOffline    bool
}

type ContentRepository interface {
	GetContentAsset(context.Context, int64) (ContentAsset, error)
}

type ContentFile interface {
	io.ReadSeeker
	io.Closer
	Stat() (fs.FileInfo, error)
}

type ContentSource interface {
	OpenContent(context.Context, string, string) (ContentFile, error)
}

type Content struct {
	File        ContentFile
	Name        string
	MIMEType    string
	SizeBytes   int64
	ModifiedAt  time.Time
	ETag        string
	Fingerprint SourceFingerprint
}

type ContentService struct {
	repository ContentRepository
	source     ContentSource
}

func NewContentService(
	repository ContentRepository,
	source ContentSource,
) (*ContentService, error) {
	if repository == nil || source == nil {
		return nil, ErrInvalidContentState
	}
	return &ContentService{repository: repository, source: source}, nil
}

func (service *ContentService) Open(ctx context.Context, assetID int64) (Content, error) {
	if assetID <= 0 {
		return Content{}, ErrContentAssetNotFound
	}
	asset, err := service.repository.GetContentAsset(ctx, assetID)
	if err != nil {
		return Content{}, err
	}
	if asset.LibraryOffline {
		return Content{}, ErrContentSourceOffline
	}
	if err := validateContentAsset(asset); err != nil {
		return Content{}, err
	}
	file, err := service.source.OpenContent(ctx, asset.LibraryRoot, asset.RelativePath)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Content{}, ctxErr
		}
		return Content{}, ErrContentUnavailable
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return Content{}, ErrContentUnavailable
	}
	if !info.Mode().IsRegular() ||
		!asset.SourceFingerprint.Matches(info.Size(), info.ModTime().UnixNano()) {
		_ = file.Close()
		return Content{}, ErrContentSourceChanged
	}
	return Content{
		File:        file,
		Name:        asset.RelativePath,
		MIMEType:    asset.MIMEType,
		SizeBytes:   asset.SizeBytes,
		ModifiedAt:  time.Unix(0, asset.ModifiedAtNS),
		ETag:        contentETag(asset.ID, asset.SourceFingerprint),
		Fingerprint: asset.SourceFingerprint,
	}, nil
}

func validateContentAsset(asset ContentAsset) error {
	if asset.ID <= 0 || asset.RelativePath == "" || asset.SizeBytes < 0 ||
		!asset.SourceFingerprint.Valid() ||
		!asset.SourceFingerprint.Matches(asset.SizeBytes, asset.ModifiedAtNS) {
		return ErrInvalidContentState
	}
	_, format, mimeType, ok := ClassifyPath(asset.RelativePath)
	if !ok || format != asset.Format || mimeType != asset.MIMEType {
		return ErrInvalidContentState
	}
	return nil
}

func contentETag(assetID int64, fingerprint SourceFingerprint) string {
	sum := sha256.Sum256([]byte(
		"foliopath-content-v1\x00" +
			strconv.FormatInt(assetID, 10) + "\x00" +
			fingerprint.String(),
	))
	return `"` + base64.RawURLEncoding.EncodeToString(sum[:]) + `"`
}
