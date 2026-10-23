//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"image/jpeg"
	"io"
	"os"
	"runtime"
	"strconv"

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
	size, err := source.Seek(0, io.SeekEnd)
	if err != nil || size < 1 || size > maxBytes || media.ValidateSourceSize(format, size) != nil {
		return face.DecodedImage{}, face.ErrInvalidInput
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return face.DecodedImage{}, face.ErrInvalidInput
	}
	sourceFile, fileStream := source.(*os.File)
	if largeImageSourceNeedsReclaim(source, format) {
		finish, err := beginLargeImageDecode(ctx)
		if err != nil {
			return face.DecodedImage{}, err
		}
		defer finish()
		if err := ctx.Err(); err != nil {
			return face.DecodedImage{}, err
		}
	}
	fileStream = fileStream && runtime.GOOS == "linux" && format == media.FormatJPEG
	var encoded []byte
	if !fileStream {
		encoded, err = io.ReadAll(io.LimitReader(source, maxBytes+1))
		if err != nil || int64(len(encoded)) != size {
			return face.DecodedImage{}, face.ErrInvalidInput
		}
	}
	params := imageImportParams()
	if format == media.FormatJPEG {
		headerSource := io.Reader(source)
		if !fileStream {
			headerSource = bytes.NewReader(encoded)
		}
		header, headerErr := jpeg.DecodeConfig(headerSource)
		if headerErr != nil || media.ValidateImageDimensions(format, header.Width, header.Height) != nil {
			return face.DecodedImage{}, face.ErrInvalidInput
		}
		// Select the existing shrink policy before the first native load. Some
		// JPEG loaders allocate working memory even while opening the image.
		if int64(header.Width)*int64(header.Height) > media.MaxDecodedPixels {
			params.JpegShrinkFactor.Set(jpegShrinkFactor(header.Width, header.Height))
		}
	}
	var image *vips.ImageRef
	if fileStream {
		if _, err := source.Seek(0, io.SeekStart); err != nil {
			return face.DecodedImage{}, face.ErrInvalidInput
		}
		image, err = vips.LoadImageFromFile(
			"/proc/self/fd/"+strconv.FormatUint(uint64(sourceFile.Fd()), 10),
			params,
		)
	} else {
		image, err = vips.LoadImageFromBuffer(encoded, params)
	}
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
