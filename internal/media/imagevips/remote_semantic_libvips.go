//go:build libvips

package imagevips

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/davidbyttow/govips/v2/vips"
)

// PrepareRemoteSemanticImage emits upright, metadata-free RGB PNG. It does not
// perform CLIP normalization/cropping, which belongs to the pinned ML runtime.
func (*Processor) PrepareRemoteSemanticImage(ctx context.Context, source io.ReadSeeker, format media.Format) ([]byte, error) {
	expected, ok := expectedImageType(format)
	if !ok || source == nil {
		return nil, semantic.ErrInvalidImageInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, semantic.ErrInvalidImageInput
	}
	encoded, err := io.ReadAll(io.LimitReader(source, semantic.RemoteImageMaxSourceBytes+1))
	if err != nil || len(encoded) == 0 || len(encoded) > semantic.RemoteImageMaxSourceBytes {
		return nil, semantic.ErrInvalidImageInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	image, err := vips.LoadImageFromBuffer(encoded, imageImportParams())
	if err != nil {
		return nil, semantic.ErrInvalidImageInput
	}
	defer image.Close()
	if image.OriginalFormat() != expected || media.ValidateImageDimensions(format, image.Width(), image.Height()) != nil ||
		int64(image.Width())*int64(image.Height()) > semantic.RemoteImageMaxPixels {
		return nil, semantic.ErrInvalidImageInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := image.ToColorSpace(vips.InterpretationSRGB); err != nil {
		return nil, semantic.ErrImagePreprocessFailed
	}
	if image.Bands() > 3 {
		if err := image.ExtractBand(0, 3); err != nil {
			return nil, semantic.ErrImagePreprocessFailed
		}
	}
	if image.Bands() != 3 {
		return nil, semantic.ErrImagePreprocessFailed
	}
	if largest := max(image.Width(), image.Height()); largest > semantic.RemoteImageMaxDimension {
		if err := image.Resize(float64(semantic.RemoteImageMaxDimension)/float64(largest), vips.KernelLanczos3); err != nil {
			return nil, semantic.ErrImagePreprocessFailed
		}
	}
	if image.BandFormat() != vips.BandFormatUchar {
		if err := image.Cast(vips.BandFormatUchar); err != nil {
			return nil, semantic.ErrImagePreprocessFailed
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	params := vips.NewPngExportParams()
	params.StripMetadata = true
	output, _, err := image.ExportPng(params)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err != nil || len(output) == 0 || len(output) > semantic.RemoteImageMaxBytes {
		return nil, semantic.ErrImagePreprocessFailed
	}
	return output, nil
}

var _ semantic.EncodedImagePreprocessor = (*Processor)(nil)
