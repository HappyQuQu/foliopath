//go:build libvips

// Package imagevips implements image probing and thumbnail generation through
// govips/libvips. Release builds that enable this adapter must use the libvips
// build tag and provide the native library.
package imagevips

import (
	"context"
	"errors"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/davidbyttow/govips/v2/vips"
)

const maxImageInputBytes = 512 << 20

type Processor struct{}

func New() *Processor {
	return &Processor{}
}

func (*Processor) Process(
	ctx context.Context,
	source io.ReadSeeker,
	format media.Format,
) (media.ProcessingResult, error) {
	if format != media.FormatJPEG && format != media.FormatPNG &&
		format != media.FormatWebP && format != media.FormatGIF {
		return media.ProcessingResult{}, media.ErrUnsupportedMedia
	}
	if err := ctx.Err(); err != nil {
		return media.ProcessingResult{}, err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	encoded, err := io.ReadAll(io.LimitReader(source, maxImageInputBytes+1))
	if err != nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	if len(encoded) == 0 || len(encoded) > maxImageInputBytes {
		return media.ProcessingResult{}, media.ErrInvalidMedia
	}
	params := vips.NewImportParams()
	params.AutoRotate.Set(true)
	params.NumPages.Set(1)
	image, err := vips.LoadImageFromBuffer(encoded, params)
	if err != nil {
		return media.ProcessingResult{}, media.ErrInvalidMedia
	}
	defer image.Close()
	if err := ctx.Err(); err != nil {
		return media.ProcessingResult{}, err
	}
	width, height := image.Width(), image.Height()
	if width < 1 || height < 1 {
		return media.ProcessingResult{}, media.ErrInvalidMedia
	}
	if err := image.ThumbnailWithSize(
		media.GridThumbnailWidth,
		media.GridThumbnailHeight,
		vips.InterestingNone,
		vips.SizeDown,
	); err != nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	if err := ctx.Err(); err != nil {
		return media.ProcessingResult{}, err
	}
	export := vips.NewWebpExportParams()
	export.Quality = media.GridWebPQuality
	export.StripMetadata = true
	thumbnail, metadata, err := image.ExportWebp(export)
	if err != nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	if metadata == nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: width, Height: height,
			PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{
			Bytes: thumbnail, Width: metadata.Width, Height: metadata.Height,
		},
	}
	kind := media.KindImage
	if format == media.FormatGIF {
		kind = media.KindAnimated
	}
	if err := media.ValidateProcessingResult(kind, result); err != nil {
		return media.ProcessingResult{}, errors.Join(media.ErrProcessingFailed, err)
	}
	return result, nil
}
