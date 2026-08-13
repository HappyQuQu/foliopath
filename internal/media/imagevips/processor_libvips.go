//go:build libvips

// Package imagevips implements image probing and thumbnail generation through
// govips/libvips. Release builds that enable this adapter must use the libvips
// build tag and provide the native library.
package imagevips

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/davidbyttow/govips/v2/vips"
)

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
	encoded, err := io.ReadAll(io.LimitReader(source, media.MaxImageSourceBytes+1))
	if err != nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	if err := media.ValidateSourceSize(format, int64(len(encoded))); err != nil {
		return media.ProcessingResult{}, err
	}
	params := imageImportParams()
	probe, err := vips.LoadImageFromBuffer(encoded, params)
	if err != nil {
		return media.ProcessingResult{}, imageDecodeError(
			err,
			media.ReasonInvalidData,
		)
	}
	expectedFormat, ok := expectedImageType(format)
	if !ok || probe.OriginalFormat() != expectedFormat {
		probe.Close()
		return media.ProcessingResult{}, media.WithFailureDiagnostic(
			media.ErrInvalidMedia,
			media.FailureDiagnostic{
				Stage:  media.StageProbe,
				Reason: media.ReasonInvalidData,
				Tool:   "libvips",
			},
		)
	}
	width, height := probe.Width(), probe.Height()
	if err := media.ValidateImageDimensions(format, width, height); err != nil {
		probe.Close()
		return media.ProcessingResult{}, err
	}
	if err := ctx.Err(); err != nil {
		probe.Close()
		return media.ProcessingResult{}, err
	}
	image := probe
	if format == media.FormatJPEG &&
		int64(width)*int64(height) > media.MaxDecodedPixels {
		probe.Close()
		image, err = vips.LoadThumbnailFromBuffer(
			encoded,
			media.GridThumbnailWidth,
			media.GridThumbnailHeight,
			vips.InterestingNone,
			vips.SizeDown,
			imageThumbnailParams(),
		)
	} else {
		err = image.ThumbnailWithSize(
			media.GridThumbnailWidth,
			media.GridThumbnailHeight,
			vips.InterestingNone,
			vips.SizeDown,
		)
	}
	if err != nil {
		if image != nil {
			image.Close()
		}
		if ctx.Err() != nil {
			return media.ProcessingResult{}, ctx.Err()
		}
		if jpegRecoveryAllowed(format, width, height, err) {
			return recoverJPEG(ctx, encoded, width, height)
		}
		return media.ProcessingResult{}, imageDecodeError(err, media.ReasonDecodeFailed)
	}
	if err := ctx.Err(); err != nil {
		image.Close()
		return media.ProcessingResult{}, err
	}
	thumbnail, metadata, err := exportWebP(image)
	image.Close()
	if err != nil {
		if ctx.Err() != nil {
			return media.ProcessingResult{}, ctx.Err()
		}
		if jpegRecoveryAllowed(format, width, height, err) {
			return recoverJPEG(ctx, encoded, width, height)
		}
		return media.ProcessingResult{}, imageOutputError(err)
	}
	return imageProcessingResult(format, width, height, thumbnail, metadata, nil)
}

func recoverJPEG(
	ctx context.Context,
	encoded []byte,
	width int,
	height int,
) (media.ProcessingResult, error) {
	if err := ctx.Err(); err != nil {
		return media.ProcessingResult{}, err
	}
	image, err := vips.LoadThumbnailFromBuffer(
		encoded,
		media.GridThumbnailWidth,
		media.GridThumbnailHeight,
		vips.InterestingNone,
		vips.SizeDown,
		imageRecoveryParams(),
	)
	if err != nil {
		return media.ProcessingResult{}, imageDecodeError(err, media.ReasonDecodeFailed)
	}
	defer image.Close()
	if err := ctx.Err(); err != nil {
		return media.ProcessingResult{}, err
	}
	thumbnail, metadata, err := exportWebP(image)
	if err != nil {
		return media.ProcessingResult{}, imageOutputError(err)
	}
	warning := &media.FailureDiagnostic{
		Stage: media.StageValidation, Reason: media.ReasonDecodeRecovered,
		Tool: "libvips",
	}
	return imageProcessingResult(
		media.FormatJPEG, width, height, thumbnail, metadata, warning,
	)
}

func exportWebP(image *vips.ImageRef) ([]byte, *vips.ImageMetadata, error) {
	export := vips.NewWebpExportParams()
	export.Quality = media.GridWebPQuality
	export.StripMetadata = true
	return image.ExportWebp(export)
}

func imageProcessingResult(
	format media.Format,
	width int,
	height int,
	thumbnail []byte,
	metadata *vips.ImageMetadata,
	warning *media.FailureDiagnostic,
) (media.ProcessingResult, error) {
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
		Warning: warning,
	}
	kind := media.KindImage
	if format == media.FormatGIF {
		kind = media.KindAnimated
	}
	if err := media.ValidateProcessingResult(kind, format, result); err != nil {
		return media.ProcessingResult{}, errors.Join(media.ErrProcessingFailed, err)
	}
	return result, nil
}

func expectedImageType(format media.Format) (vips.ImageType, bool) {
	switch format {
	case media.FormatJPEG:
		return vips.ImageTypeJPEG, true
	case media.FormatPNG:
		return vips.ImageTypePNG, true
	case media.FormatWebP:
		return vips.ImageTypeWEBP, true
	case media.FormatGIF:
		return vips.ImageTypeGIF, true
	default:
		return vips.ImageTypeUnknown, false
	}
}

func imageImportParams() *vips.ImportParams {
	params := vips.NewImportParams()
	params.AutoRotate.Set(true)
	params.NumPages.Set(1)
	params.Access.Set(vips.AccessSequential)
	return params
}

func imageThumbnailParams() *vips.ImportParams {
	// thumbnail_buffer applies these values as loader option strings. Keep the
	// strict decoder setting here; thumbnail_buffer itself rotates upright after
	// the JPEG decoder has performed shrink-on-load.
	return vips.NewImportParams()
}

func imageRecoveryParams() *vips.ImportParams {
	params := vips.NewImportParams()
	params.FailOnError.Set(false)
	return params
}

func jpegRecoveryAllowed(
	format media.Format,
	width int,
	height int,
	err error,
) bool {
	if format != media.FormatJPEG || err == nil ||
		media.ValidateDimensions(width, height) != nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "premature end") ||
		strings.Contains(message, "incomplete scan") ||
		strings.Contains(message, "corrupt jpeg data")
}

func imageDecodeError(err error, fallback media.FailureReason) error {
	message := strings.ToLower(err.Error())
	truncated := corruptImageMessage(message)
	invalid := truncated ||
		errors.Is(err, vips.ErrUnsupportedImageFormat) ||
		strings.Contains(message, "not a known file format") ||
		strings.Contains(message, "not in a known format") ||
		strings.Contains(message, "unsupported image format") ||
		strings.Contains(message, "not enough data") ||
		strings.Contains(message, "invalid")
	if !invalid {
		return media.WithFailureDiagnostic(
			media.ErrProcessingFailed,
			media.FailureDiagnostic{
				Stage: media.StageProbe, Reason: media.ReasonToolFailed, Tool: "libvips",
			},
		)
	}
	reason := fallback
	if truncated {
		reason = media.ReasonDecodeFailed
	}
	return media.WithFailureDiagnostic(media.ErrInvalidMedia, media.FailureDiagnostic{
		Stage: media.StageProbe, Reason: reason, Tool: "libvips",
	})
}

func imageOutputError(err error) error {
	message := strings.ToLower(err.Error())
	if corruptImageMessage(message) {
		return media.WithFailureDiagnostic(
			media.ErrInvalidMedia,
			media.FailureDiagnostic{
				Stage: media.StageValidation, Reason: media.ReasonDecodeFailed, Tool: "libvips",
			},
		)
	}
	return media.WithFailureDiagnostic(media.ErrProcessingFailed, media.FailureDiagnostic{
		Stage: media.StageValidation, Reason: media.ReasonToolFailed, Tool: "libvips",
	})
}

func corruptImageMessage(message string) bool {
	return strings.Contains(message, "premature end") ||
		strings.Contains(message, "unexpected end") ||
		strings.Contains(message, "incomplete scan") ||
		strings.Contains(message, "corrupt") ||
		strings.Contains(message, "truncated")
}
