package thumbnail

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type StoryboardService struct {
	repository StoryboardRepository
	source     Source
	publisher  Publisher
	capacity   Capacity
	processor  media.StoryboardProcessor
	now        func() time.Time
}

func NewStoryboardService(
	repository StoryboardRepository,
	source Source,
	publisher Publisher,
	capacity Capacity,
	processor media.StoryboardProcessor,
	options ServiceOptions,
) (*StoryboardService, error) {
	if repository == nil || source == nil || publisher == nil ||
		capacity == nil || processor == nil {
		return nil, errors.New("storyboard service dependencies are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &StoryboardService{
		repository: repository,
		source:     source,
		publisher:  publisher,
		capacity:   capacity,
		processor:  processor,
		now:        options.Now,
	}, nil
}

func (service *StoryboardService) Process(
	ctx context.Context,
	assetID int64,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	asset, err := service.repository.GetAssetForDerivation(ctx, assetID)
	if err != nil {
		return err
	}
	if !StoryboardEligible(
		asset.Kind,
		asset.ProbeStatus,
		asset.DurationMS,
		asset.GridReady,
	) {
		return ErrStoryboardNotEligible
	}
	plan, err := NewStoryboardPlan(*asset.DurationMS)
	if err != nil {
		return err
	}
	cellWidth, cellHeight, err := StoryboardCellDimensions(
		asset.Width,
		asset.Height,
	)
	if err != nil {
		return err
	}
	request := media.StoryboardRequest{
		TimestampsMS: plan.TimestampsMS,
		Columns:      plan.Columns,
		Rows:         plan.Rows,
		CellWidth:    cellWidth,
		CellHeight:   cellHeight,
	}
	if media.ValidateStoryboardRequest(request) != nil {
		return ErrInvalidState
	}
	derivation, err := StoryboardDerivation(
		asset.LibraryID,
		asset.ID,
		asset.SourceFingerprint,
	)
	if err != nil {
		return ErrInvalidState
	}
	source, err := service.source.OpenAsset(
		ctx,
		asset.LibraryRoot,
		asset.RelativePath,
	)
	if err != nil {
		return ErrSourceUnavailable
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return ErrSourceUnavailable
	}
	if !asset.SourceFingerprint.Matches(
		info.Size(),
		info.ModTime().UnixNano(),
	) {
		return ErrSourceChanged
	}
	if err := media.ValidateSourceSize(asset.Format, info.Size()); err != nil {
		return service.commitFailure(ctx, asset, err)
	}
	result, err := service.processor.Storyboard(
		ctx,
		source,
		asset.Format,
		request,
	)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return service.commitFailure(ctx, asset, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if media.ValidateStoryboardResult(request, result) != nil {
		return ErrInvalidState
	}
	layout := StoryboardLayout{
		FrameCount: result.FrameCount,
		Columns:    result.Columns,
		Rows:       result.Rows,
		CellWidth:  result.CellWidth,
		CellHeight: result.CellHeight,
	}
	if layout.Validate() != nil {
		return ErrInvalidState
	}
	reservation, err := service.capacity.Reserve(
		ctx,
		int64(len(result.Bytes)),
	)
	if err != nil {
		return err
	}
	defer reservation.Release()
	published, err := service.publisher.Publish(ctx, derivation, result.Bytes)
	if err != nil {
		return errors.Join(ErrPublishFailed, err)
	}
	expectedPath, err := derivation.CacheRelativePath()
	if err != nil ||
		published.CacheRelativePath != expectedPath ||
		published.ByteSize != int64(len(result.Bytes)) {
		return ErrPublishFailed
	}
	return service.repository.CommitStoryboardReady(
		ctx,
		StoryboardReady{
			AssetID:           asset.ID,
			SourceFingerprint: asset.SourceFingerprint,
			Result:            result,
			CacheRelativePath: published.CacheRelativePath,
			ByteSize:          published.ByteSize,
			CreatedAtMS:       service.now().UnixMilli(),
		},
	)
}

func (service *StoryboardService) commitFailure(
	ctx context.Context,
	asset Asset,
	processingErr error,
) error {
	commitErr := service.repository.CommitStoryboardFailure(
		ctx,
		StoryboardFailure{
			AssetID:           asset.ID,
			SourceFingerprint: asset.SourceFingerprint,
			Code:              media.ProcessingCode(processingErr),
		},
	)
	if commitErr != nil {
		return errors.Join(processingErr, commitErr)
	}
	return processingErr
}
