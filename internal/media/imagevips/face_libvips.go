//go:build libvips

package imagevips

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/davidbyttow/govips/v2/vips"
)

// DecodeFaceImage decodes an already-opened image to bounded interleaved RGB.
// It never accepts or resolves a filesystem path.
func (*Processor) DecodeFaceImage(
	ctx context.Context,
	source io.ReadSeeker,
	format media.Format,
	maxBytes int64,
	maxDimension int,
) (face.DecodedImage, error) {
	if source == nil || maxBytes < 1 || maxBytes > face.MaxInputBytes || maxDimension < 1 || maxDimension > face.MaxDecodeDimension ||
		(format != media.FormatJPEG && format != media.FormatPNG && format != media.FormatWebP && format != media.FormatGIF) {
		return face.DecodedImage{}, face.ErrInvalidInput
	}
	if err := ctx.Err(); err != nil {
		return face.DecodedImage{}, err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return face.DecodedImage{}, face.ErrInvalidInput
	}
	encoded, err := io.ReadAll(io.LimitReader(source, maxBytes+1))
	if err != nil || int64(len(encoded)) > maxBytes || media.ValidateSourceSize(format, int64(len(encoded))) != nil {
		return face.DecodedImage{}, face.ErrInvalidInput
	}
	image, err := vips.LoadImageFromBuffer(encoded, imageImportParams())
	if err != nil {
		return face.DecodedImage{}, face.ErrInvalidInput
	}
	defer func() {
		if image != nil {
			image.Close()
		}
	}()
	if expected, ok := expectedImageType(format); !ok || image.OriginalFormat() != expected ||
		media.ValidateImageDimensions(format, image.Width(), image.Height()) != nil {
		return face.DecodedImage{}, face.ErrInvalidInput
	}
	if format == media.FormatJPEG && int64(image.Width())*int64(image.Height()) > media.MaxDecodedPixels {
		width, height := image.Width(), image.Height()
		image.Close()
		params := imageImportParams()
		params.JpegShrinkFactor.Set(jpegShrinkFactor(width, height))
		image, err = vips.LoadImageFromBuffer(encoded, params)
		if err != nil {
			return face.DecodedImage{}, face.ErrInvalidInput
		}
	}
	if err := image.ToColorSpace(vips.InterpretationSRGB); err != nil {
		return face.DecodedImage{}, face.ErrInvalidOutput
	}
	if image.Bands() > 3 {
		if err := image.ExtractBand(0, 3); err != nil {
			return face.DecodedImage{}, face.ErrInvalidOutput
		}
	}
	if image.Bands() != 3 {
		return face.DecodedImage{}, face.ErrInvalidOutput
	}
	if largest := max(image.Width(), image.Height()); largest > maxDimension {
		if err := image.Resize(float64(maxDimension)/float64(largest), vips.KernelLanczos3); err != nil {
			return face.DecodedImage{}, face.ErrInvalidOutput
		}
	}
	if image.BandFormat() != vips.BandFormatUchar {
		if err := image.Cast(vips.BandFormatUchar); err != nil {
			return face.DecodedImage{}, face.ErrInvalidOutput
		}
	}
	if err := ctx.Err(); err != nil {
		return face.DecodedImage{}, err
	}
	rgb, err := image.ToBytes()
	if err != nil || len(rgb) != image.Width()*image.Height()*3 {
		return face.DecodedImage{}, face.ErrInvalidOutput
	}
	return face.DecodedImage{Width: image.Width(), Height: image.Height(), RGB: rgb}, nil
}

var _ face.ImageDecoder = (*Processor)(nil)
