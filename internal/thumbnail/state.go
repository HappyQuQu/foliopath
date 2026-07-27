package thumbnail

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/media"
)

var (
	ErrAssetNotFound      = errors.New("thumbnail asset not found")
	ErrSourceChanged      = errors.New("thumbnail source changed")
	ErrInvalidState       = errors.New("invalid thumbnail state")
	ErrRepositoryNotReady = errors.New("thumbnail repository is not ready")
)

type Asset struct {
	ID                int64
	LibraryID         int64
	LibraryRoot       string
	RelativePath      string
	Kind              media.Kind
	Format            media.Format
	SizeBytes         int64
	ModifiedAtNS      int64
	SourceFingerprint media.SourceFingerprint
}

type Ready struct {
	AssetID           int64
	SourceFingerprint media.SourceFingerprint
	Result            media.ProcessingResult
	CacheRelativePath string
	ByteSize          int64
	CreatedAtMS       int64
}

type Failure struct {
	AssetID           int64
	SourceFingerprint media.SourceFingerprint
	Code              media.ProcessingErrorCode
}

type Repository interface {
	GetAssetForDerivation(context.Context, int64) (Asset, error)
	CommitReady(context.Context, Ready) error
	CommitFailure(context.Context, Failure) error
}
