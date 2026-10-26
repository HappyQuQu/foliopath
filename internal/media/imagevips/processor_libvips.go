//go:build libvips

// Package imagevips implements image probing and thumbnail generation through
// govips/libvips. Release builds that enable this adapter must use the libvips
// build tag and provide the native library.
package imagevips

import (
	"context"
	"errors"
	"image/jpeg"
	"io"
	"os"
	"runtime"
	"strconv"
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
	if largeImageSourceNeedsReclaim(source, format) {
		finish, err := beginLargeImageDecode(ctx)
		if err != nil {
			return media.ProcessingResult{}, err
		}
		defer finish()
		if err := ctx.Err(); err != nil {
			return media.ProcessingResult{}, err
		}
	}
	if format == media.FormatJPEG {
		if _, err := imageSourceSize(source, format); err != nil {
			return media.ProcessingResult{}, err
		}
		header, headerErr := jpeg.DecodeConfig(source)
		if headerErr != nil {
			return media.ProcessingResult{}, imageDecodeError(
				headerErr,
				media.ReasonInvalidData,
			)
		}
		if err := media.ValidateImageDimensions(format, header.Width, header.Height); err != nil {
			return media.ProcessingResult{}, err
		}
		if sourceFile, ok := source.(*os.File); ok && runtime.GOOS == "linux" {
			return processJPEGFile(ctx, sourceFile, header.Width, header.Height)
		}
		encoded, err := readImageSource(ctx, source, format)
		if err != nil {
			return media.ProcessingResult{}, err
		}
		return processJPEGThumbnail(ctx, encoded, header.Width, header.Height)
	}
	encoded, err := readImageSource(ctx, source, format)
	if err != nil {
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
	err = image.ThumbnailWithSize(
		media.GridThumbnailWidth,
		media.GridThumbnailHeight,
		vips.InterestingNone,
		vips.SizeDown,
	)
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

func processJPEGThumbnail(
	ctx context.Context,
	encoded []byte,
	width int,
	height int,
) (media.ProcessingResult, error) {
	// vips_thumbnail_buffer selects decoder-level JPEG shrink itself. Loading a
	// full image first, even with an explicit shrink hint, can still materialize
	// enough native state for two large concurrent jobs to exceed the 1 GiB
	// application limit.
	strictParams := vips.NewImportParams()
	image, err := vips.LoadThumbnailFromBuffer(
		encoded,
		media.GridThumbnailWidth,
		media.GridThumbnailHeight,
		vips.InterestingNone,
		vips.SizeDown,
		strictParams,
	)
	if err != nil {
		if jpegRecoveryAllowed(media.FormatJPEG, width, height, err) {
			return recoverJPEG(ctx, encoded, width, height)
		}
		return media.ProcessingResult{}, imageDecodeError(err, media.ReasonDecodeFailed)
	}
	defer image.Close()
	return finishJPEGThumbnail(ctx, image, width, height, func() ([]byte, error) {
		return encoded, nil
	})
}

func processJPEGFile(
	ctx context.Context,
	source *os.File,
	width int,
	height int,
) (media.ProcessingResult, error) {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	image, err := vips.LoadThumbnailFromFile(
		"/proc/self/fd/"+strconv.FormatUint(uint64(source.Fd()), 10),
		media.GridThumbnailWidth,
		media.GridThumbnailHeight,
		vips.InterestingNone,
		vips.SizeDown,
		vips.NewImportParams(),
	)
	if err != nil {
		if !jpegRecoveryAllowed(media.FormatJPEG, width, height, err) {
			return media.ProcessingResult{}, imageDecodeError(err, media.ReasonDecodeFailed)
		}
		encoded, readErr := readImageSource(ctx, source, media.FormatJPEG)
		if readErr != nil {
			return media.ProcessingResult{}, readErr
		}
		return recoverJPEG(ctx, encoded, width, height)
	}
	defer image.Close()
	return finishJPEGThumbnail(ctx, image, width, height, func() ([]byte, error) {
		return readImageSource(ctx, source, media.FormatJPEG)
	})
}

func finishJPEGThumbnail(
	ctx context.Context,
	image *vips.ImageRef,
	width int,
	height int,
	recoverySource func() ([]byte, error),
) (media.ProcessingResult, error) {
	if err := image.AutoRotate(); err != nil {
		return media.ProcessingResult{}, imageDecodeError(err, media.ReasonDecodeFailed)
	}
	// The thumbnail honors EXIF autorotation. Preserve full-resolution metadata
	// while using its aspect to identify the two orientations that swap axes.
	if (width > height) != (image.Width() > image.Height()) {
		width, height = height, width
	}
	if err := ctx.Err(); err != nil {
		return media.ProcessingResult{}, err
	}
	thumbnail, metadata, err := exportWebP(image)
	if err != nil {
		if ctx.Err() != nil {
			return media.ProcessingResult{}, ctx.Err()
		}
		if jpegRecoveryAllowed(media.FormatJPEG, width, height, err) && recoverySource != nil {
			encoded, readErr := recoverySource()
			if readErr != nil {
				return media.ProcessingResult{}, readErr
			}
			return recoverJPEG(ctx, encoded, width, height)
		}
		return media.ProcessingResult{}, imageOutputError(err)
	}
	return imageProcessingResult(media.FormatJPEG, width, height, thumbnail, metadata, nil)
}

func readImageSource(
	ctx context.Context,
	source io.ReadSeeker,
	format media.Format,
) ([]byte, error) {
	size, err := imageSourceSize(source, format)
	if err != nil {
		return nil, err
	}
	encoded := make([]byte, int(size))
	for offset, emptyReads := 0, 0; offset < len(encoded); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := min(offset+64*1024, len(encoded))
		n, readErr := source.Read(encoded[offset:end])
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if n < 0 || n > end-offset {
			return nil, media.ErrProcessingFailed
		}
		offset += n
		if readErr != nil && !(readErr == io.EOF && offset == len(encoded)) {
			return nil, media.ErrProcessingFailed
		}
		if n == 0 {
			emptyReads++
			if emptyReads >= 100 {
				return nil, media.ErrProcessingFailed
			}
		} else {
			emptyReads = 0
		}
	}
	var extra [1]byte
	n, readErr := source.Read(extra[:])
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if n != 0 || readErr != io.EOF {
		return nil, media.ErrProcessingFailed
	}
	return encoded, nil
}

func imageSourceSize(source io.ReadSeeker, format media.Format) (int64, error) {
	size, err := source.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, media.ErrProcessingFailed
	}
	if err := media.ValidateSourceSize(format, size); err != nil {
		return 0, err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return 0, media.ErrProcessingFailed
	}
	return size, nil
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
