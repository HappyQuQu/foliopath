//go:build libvips

package imagevips

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/davidbyttow/govips/v2/vips"
)

// PrepareSemanticImage creates the fixed SigLIP tensor from an already-opened
// media stream. It never accepts or resolves a filesystem path.
func (*Processor) PrepareSemanticImage(
	ctx context.Context,
	source io.ReadSeeker,
	format media.Format,
) ([]float32, error) {
	if format != media.FormatJPEG && format != media.FormatPNG &&
		format != media.FormatWebP && format != media.FormatGIF || source == nil {
		return nil, semantic.ErrInvalidImageInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, semantic.ErrInvalidImageInput
	}
	encoded, err := io.ReadAll(io.LimitReader(source, media.MaxImageSourceBytes+1))
	if err != nil || media.ValidateSourceSize(format, int64(len(encoded))) != nil {
		return nil, semantic.ErrInvalidImageInput
	}

	image, err := vips.LoadImageFromBuffer(encoded, imageImportParams())
	if err != nil {
		return nil, semantic.ErrInvalidImageInput
	}
	defer func() {
		if image != nil {
			image.Close()
		}
	}()
	if expected, ok := expectedImageType(format); !ok || image.OriginalFormat() != expected ||
		media.ValidateImageDimensions(format, image.Width(), image.Height()) != nil {
		return nil, semantic.ErrInvalidImageInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Large JPEGs must use decoder shrink-on-load before any pixel evaluation.
	// The following direct bicubic resize still targets both axes independently,
	// matching SigLIP's square resize rather than the gallery thumbnail geometry.
	if format == media.FormatJPEG && int64(image.Width())*int64(image.Height()) > media.MaxDecodedPixels {
		width, height := image.Width(), image.Height()
		image.Close()
		params := imageImportParams()
		params.JpegShrinkFactor.Set(jpegShrinkFactor(width, height))
		image, err = vips.LoadImageFromBuffer(encoded, params)
		if err != nil {
			return nil, semantic.ErrInvalidImageInput
		}
	}
	if err := image.ToColorSpace(vips.InterpretationSRGB); err != nil {
		return nil, semantic.ErrImagePreprocessFailed
	}
	// PIL Image.convert("RGB") discards alpha before resizing. Do the same;
	// compositing on an invented background would change model semantics.
	if image.Bands() > semantic.SigLIPImageChannels {
		if err := image.ExtractBand(0, semantic.SigLIPImageChannels); err != nil {
			return nil, semantic.ErrImagePreprocessFailed
		}
	}
	if image.Bands() != semantic.SigLIPImageChannels {
		return nil, semantic.ErrImagePreprocessFailed
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	hScale := float64(semantic.SigLIPImageWidth) / float64(image.Width())
	vScale := float64(semantic.SigLIPImageHeight) / float64(image.Height())
	if err := image.ResizeWithVScale(hScale, vScale, vips.KernelCubic); err != nil {
		return nil, semantic.ErrImagePreprocessFailed
	}
	if image.BandFormat() != vips.BandFormatUchar {
		if err := image.Cast(vips.BandFormatUchar); err != nil {
			return nil, semantic.ErrImagePreprocessFailed
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if image.Width() != semantic.SigLIPImageWidth || image.Height() != semantic.SigLIPImageHeight ||
		image.Bands() != semantic.SigLIPImageChannels {
		return nil, semantic.ErrImagePreprocessFailed
	}
	rgb, err := image.ToBytes()
	if err != nil {
		return nil, semantic.ErrImagePreprocessFailed
	}
	tensor, err := semantic.PrepareSigLIPImageTensor(rgb)
	if err != nil {
		return nil, semantic.ErrImagePreprocessFailed
	}
	return tensor, nil
}

// SplitSemanticStoryboard decodes one already-generated WebP sprite and
// exports its bounded cells. It never reads the original video or invokes
// FFmpeg, keeping storyboard generation under the thumbnail owner.
func (*Processor) SplitSemanticStoryboard(
	ctx context.Context,
	source io.ReadSeeker,
	frameCount, columns, cellWidth, cellHeight int,
) ([][]byte, error) {
	if source == nil || (frameCount != 4 && frameCount != 10) || columns != min(frameCount, 5) ||
		cellWidth < 1 || cellWidth > 320 || cellHeight < 1 || cellHeight > 320 {
		return nil, semantic.ErrInvalidVideoSemantic
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, semantic.ErrInvalidVideoSemantic
	}
	encoded, err := io.ReadAll(io.LimitReader(source, media.MaxImageSourceBytes+1))
	if err != nil || media.ValidateSourceSize(media.FormatWebP, int64(len(encoded))) != nil {
		return nil, semantic.ErrInvalidVideoSemantic
	}
	image, err := vips.LoadImageFromBuffer(encoded, imageImportParams())
	if err != nil {
		return nil, semantic.ErrInvalidVideoSemantic
	}
	defer image.Close()
	rows := (frameCount + columns - 1) / columns
	if image.OriginalFormat() != vips.ImageTypeWEBP || image.Width() != columns*cellWidth || image.Height() != rows*cellHeight {
		return nil, semantic.ErrInvalidVideoSemantic
	}
	result := make([][]byte, 0, frameCount)
	for ordinal := 0; ordinal < frameCount; ordinal++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cell, err := image.Copy()
		if err != nil {
			return nil, semantic.ErrImagePreprocessFailed
		}
		if err := cell.ExtractArea((ordinal%columns)*cellWidth, (ordinal/columns)*cellHeight, cellWidth, cellHeight); err != nil {
			cell.Close()
			return nil, semantic.ErrImagePreprocessFailed
		}
		params := vips.NewWebpExportParams()
		params.Lossless = true
		value, _, err := cell.ExportWebp(params)
		cell.Close()
		if err != nil || len(value) == 0 {
			return nil, semantic.ErrImagePreprocessFailed
		}
		result = append(result, value)
	}
	return result, nil
}

func jpegShrinkFactor(width, height int) int {
	minimum := min(width, height)
	switch {
	case minimum >= semantic.SigLIPImageWidth*8:
		return 8
	case minimum >= semantic.SigLIPImageWidth*4:
		return 4
	case minimum >= semantic.SigLIPImageWidth*2:
		return 2
	default:
		return 1
	}
}
