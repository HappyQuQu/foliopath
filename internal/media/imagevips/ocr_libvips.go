//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"github.com/davidbyttow/govips/v2/vips"
	"golang.org/x/image/webp"
)

// PrepareOCRImage consumes an already-opened stream only. Production callers
// must obtain it through internal/files and own source-fingerprint consistency.
// Its independent contract must never change legacy semantic/tag/face pixels.
func (*Processor) PrepareOCRImage(ctx context.Context, source io.ReadSeeker, format media.Format) (ocr.PreparedImage, error) {
	if err := ctx.Err(); err != nil {
		return ocr.PreparedImage{}, err
	}
	expected, ok := expectedImageType(format)
	if source == nil || !ok {
		return ocr.PreparedImage{}, ocr.ErrInvalidImageInput
	}
	size, err := source.Seek(0, io.SeekEnd)
	if err != nil {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrPreprocessFailed)
	}
	if size < 1 {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrInvalidMedia)
	}
	if size > ocr.MaxSourceBytes {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrInputLimit)
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrPreprocessFailed)
	}
	if err := ctx.Err(); err != nil {
		return ocr.PreparedImage{}, err
	}
	if largeImageSourceNeedsReclaim(source, format) {
		finish, err := beginLargeImageDecode(ctx)
		if err != nil {
			return ocr.PreparedImage{}, err
		}
		defer finish()
	}
	// Reuse the canonical bounded/cancellable read owner. A short/changing or IO
	// failure is processing failure, not evidence of a corrupt original image.
	encoded, err := readRemoteSemanticSource(ctx, source, int(size))
	if err != nil {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrPreprocessFailed)
	}
	header, err := ocrImageHeader(encoded, format)
	if err != nil {
		if errors.Is(err, ocr.ErrInputLimit) {
			return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrInputLimit)
		}
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrInvalidMedia)
	}
	if media.ValidateImageDimensions(format, header.Width, header.Height) != nil || min(header.Width, header.Height) < ocr.MinImageDimension ||
		int64(header.Width)*int64(header.Height) > ocr.MaxSourcePixels {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrInputLimit)
	}
	if err := ctx.Err(); err != nil {
		return ocr.PreparedImage{}, err
	}
	params := imageImportParams()
	params.FailOnError.Set(true)
	native, err := vips.LoadImageFromBuffer(encoded, params)
	if err != nil {
		return ocr.PreparedImage{}, ocrNativeImageError(ctx, encoded, format)
	}
	defer native.Close() // Registered after finish: ImageRef closes before admission.
	if native.OriginalFormat() != expected || !((native.Width() == header.Width && native.Height() == header.Height) || (native.Width() == header.Height && native.Height() == header.Width)) {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrInvalidMedia)
	}
	if err := ctx.Err(); err != nil {
		return ocr.PreparedImage{}, err
	}
	if err := native.ToColorSpace(vips.InterpretationSRGB); err != nil {
		return ocr.PreparedImage{}, ocrNativeImageError(ctx, encoded, format)
	}
	if native.HasAlpha() {
		if err := native.Flatten(&vips.Color{R: 255, G: 255, B: 255}); err != nil {
			return ocr.PreparedImage{}, ocrNativeImageError(ctx, encoded, format)
		}
	}
	if native.Bands() != 3 {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrPreprocessFailed)
	}
	if largest := max(native.Width(), native.Height()); largest > ocr.MaxPreparedDimension {
		if err := native.Resize(float64(ocr.MaxPreparedDimension)/float64(largest), vips.KernelLanczos3); err != nil {
			return ocr.PreparedImage{}, ocrNativeImageError(ctx, encoded, format)
		}
	}
	if max(native.Width(), native.Height()) > ocr.MaxPreparedDimension || min(native.Width(), native.Height()) < ocr.MinImageDimension || int64(native.Width())*int64(native.Height()) > ocr.MaxPreparedPixels {
		return ocr.PreparedImage{}, ocrImageError(ctx, ocr.ErrInputLimit)
	}
	if native.BandFormat() != vips.BandFormatUchar {
		if err := native.Cast(vips.BandFormatUchar); err != nil {
			return ocr.PreparedImage{}, ocrNativeImageError(ctx, encoded, format)
		}
	}
	if err := ctx.Err(); err != nil {
		return ocr.PreparedImage{}, err
	}
	export := vips.NewPngExportParams()
	export.StripMetadata = true
	// PNG filtering changes compression only; the OCR pixel contract is unchanged.
	export.Filter = vips.PngFilterAll
	output, _, err := native.ExportPng(export)
	if err != nil {
		return ocr.PreparedImage{}, ocrNativeImageError(ctx, encoded, format)
	}
	if err := ctx.Err(); err != nil {
		return ocr.PreparedImage{}, err
	}
	if len(output) == 0 {
		return ocr.PreparedImage{}, ocr.ErrPreprocessFailed
	}
	if len(output) > ocr.MaxImageBytes {
		return ocr.PreparedImage{}, ocr.ErrInputLimit
	}
	return ocr.PreparedImage{ContractSHA256: ocr.InputContractSHA256(), PNG: output, Width: native.Width(), Height: native.Height()}, nil
}

func ocrImageHeader(encoded []byte, format media.Format) (image.Config, error) {
	source := &io.LimitedReader{R: bytes.NewReader(encoded), N: ocr.MaxHeaderBytes}
	var config image.Config
	var err error
	switch format {
	case media.FormatJPEG:
		config, err = jpeg.DecodeConfig(source)
	case media.FormatPNG:
		config, err = png.DecodeConfig(source)
	case media.FormatGIF:
		config, err = gif.DecodeConfig(source)
	case media.FormatWebP:
		config, err = webp.DecodeConfig(source)
	default:
		return image.Config{}, ocr.ErrInvalidImageInput
	}
	if err != nil && source.N == 0 {
		return image.Config{}, ocr.ErrInputLimit
	}
	return config, err
}

func ocrImageError(ctx context.Context, fallback error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fallback
}

func ocrNativeImageError(ctx context.Context, encoded []byte, format media.Format) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Native decoding can fail lazily during export. Only a definitive decoder
	// error proves a bad JPEG; resource/native failures remain processing faults.
	if format == media.FormatJPEG {
		_, err := jpeg.Decode(bytes.NewReader(encoded))
		var malformed jpeg.FormatError
		if errors.As(err, &malformed) || errors.Is(err, io.ErrUnexpectedEOF) {
			return ocrImageError(ctx, ocr.ErrInvalidMedia)
		}
	}
	return ocrImageError(ctx, ocr.ErrPreprocessFailed)
}

var _ ocr.ImagePreprocessor = (*Processor)(nil)
