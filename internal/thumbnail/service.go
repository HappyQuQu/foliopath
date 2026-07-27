package thumbnail

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

var (
	ErrSourceUnavailable = errors.New("thumbnail source unavailable")
	ErrPublishFailed     = errors.New("thumbnail publish failed")
)

type SourceFile interface {
	io.ReadSeekCloser
	Stat() (fs.FileInfo, error)
}

type Source interface {
	OpenAsset(context.Context, string, string) (SourceFile, error)
}

type Published struct {
	CacheRelativePath string
	ByteSize          int64
}

type Publisher interface {
	Publish(context.Context, Derivation, []byte) (Published, error)
}

type Service struct {
	repository Repository
	source     Source
	publisher  Publisher
	image      media.Processor
	video      media.Processor
	now        func() time.Time
}

type ServiceOptions struct {
	Now func() time.Time
}

func NewService(
	repository Repository,
	source Source,
	publisher Publisher,
	image media.Processor,
	video media.Processor,
	options ServiceOptions,
) (*Service, error) {
	if repository == nil || source == nil || publisher == nil ||
		image == nil || video == nil {
		return nil, errors.New("thumbnail service dependencies are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Service{
		repository: repository,
		source:     source,
		publisher:  publisher,
		image:      image,
		video:      video,
		now:        options.Now,
	}, nil
}

func (service *Service) Process(ctx context.Context, assetID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	asset, err := service.repository.GetAssetForDerivation(ctx, assetID)
	if err != nil {
		return err
	}
	derivation, err := GridDerivation(
		asset.LibraryID, asset.ID, asset.SourceFingerprint,
	)
	if err != nil {
		return ErrInvalidState
	}
	source, err := service.source.OpenAsset(ctx, asset.LibraryRoot, asset.RelativePath)
	if err != nil {
		return ErrSourceUnavailable
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return ErrSourceUnavailable
	}
	if !asset.SourceFingerprint.Matches(info.Size(), info.ModTime().UnixNano()) {
		return ErrSourceChanged
	}
	processor := service.image
	if asset.Kind == media.KindVideo {
		processor = service.video
	}
	result, err := processor.Process(ctx, source, asset.Format)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		code := media.ProcessingCode(err)
		if commitErr := service.repository.CommitFailure(ctx, Failure{
			AssetID: asset.ID, SourceFingerprint: asset.SourceFingerprint, Code: code,
		}); commitErr != nil {
			return errors.Join(err, commitErr)
		}
		return err
	}
	if err := media.ValidateProcessingResult(asset.Kind, result); err != nil {
		return ErrInvalidState
	}
	published, err := service.publisher.Publish(ctx, derivation, result.Thumbnail.Bytes)
	if err != nil {
		return errors.Join(ErrPublishFailed, err)
	}
	expectedPath, err := derivation.CacheRelativePath()
	if err != nil || published.CacheRelativePath != expectedPath ||
		published.ByteSize != int64(len(result.Thumbnail.Bytes)) {
		return ErrPublishFailed
	}
	if err := service.repository.CommitReady(ctx, Ready{
		AssetID: asset.ID, SourceFingerprint: asset.SourceFingerprint,
		Result: result, CacheRelativePath: published.CacheRelativePath,
		ByteSize: published.ByteSize, CreatedAtMS: service.now().UnixMilli(),
	}); err != nil {
		return err
	}
	return nil
}
