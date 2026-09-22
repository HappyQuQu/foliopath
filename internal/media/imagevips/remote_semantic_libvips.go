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
	maxSourceBytes := semantic.RemoteImageMaxSourceBytes
	if format == media.FormatJPEG {
		maxSourceBytes = semantic.RemoteJPEGMaxSourceBytes
	}
	// The already-opened stream is seekable. Validate size before allocating. On
	// production Linux, large JPEGs stay behind the openat2-anchored descriptor
	// and libvips reads /proc/self/fd directly; non-file test/development streams
	// retain the exact-buffer fallback.
	size, err := source.Seek(0, io.SeekEnd)
	if err != nil || size < 1 || size > int64(maxSourceBytes) {
		return nil, semantic.ErrInvalidImageInput
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, semantic.ErrInvalidImageInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var encoded []byte
	importParams := imageImportParams()
	width, height := 0, 0
	jpegShrink := 1
	sourceFile, fileStream := source.(*os.File)
	fileStream = fileStream && runtime.GOOS == "linux" && format == media.FormatJPEG
	if !fileStream {
		encoded, err = readRemoteSemanticSource(ctx, source, int(size))
		if err != nil {
			return nil, err
		}
	}
	if format == media.FormatJPEG {
		headerSource := io.Reader(source)
		if !fileStream {
			headerSource = bytes.NewReader(encoded)
		}
		header, err := jpeg.DecodeConfig(headerSource)
		if err != nil {
			return nil, semantic.ErrInvalidImageInput
		}
		width, height = header.Width, header.Height
		shrink, ok := remoteSemanticJPEGShrink(header.Width, header.Height)
		if !ok {
			return nil, semantic.ErrInvalidImageInput
		}
		jpegShrink = shrink
		importParams.JpegShrinkFactor.Set(shrink)
	}
	load := func(params *vips.ImportParams) (*vips.ImageRef, error) {
		if fileStream {
			if _, seekErr := source.Seek(0, io.SeekStart); seekErr != nil {
				return nil, seekErr
			}
			return vips.LoadImageFromFile(
				"/proc/self/fd/"+strconv.FormatUint(uint64(sourceFile.Fd()), 10),
				params,
			)
		}
		return vips.LoadImageFromBuffer(encoded, params)
	}
	image, err := load(importParams)
	if err != nil {
		return nil, semantic.ErrInvalidImageInput
	}
	if format != media.FormatJPEG {
		width, height = image.Width(), image.Height()
	}
	if !validRemoteSemanticImage(image, expected, format, width, height) {
		image.Close()
		return nil, semantic.ErrInvalidImageInput
	}
	output, processErr := encodeRemoteSemanticImage(ctx, image)
	image.Close()
	if processErr == nil {
		return output, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if format != media.FormatJPEG || !jpegRecoveryAllowed(format, width, height, processErr) {
		return nil, semantic.ErrImagePreprocessFailed
	}
	recoveryParams := imageImportParams()
	recoveryParams.FailOnError.Set(false)
	recoveryParams.JpegShrinkFactor.Set(jpegShrink)
	recovered, err := load(recoveryParams)
	if err != nil {
		return nil, semantic.ErrImagePreprocessFailed
	}
	defer recovered.Close()
	if !validRemoteSemanticImage(recovered, expected, format, width, height) {
		return nil, semantic.ErrInvalidImageInput
	}
	output, err = encodeRemoteSemanticImage(ctx, recovered)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, semantic.ErrImagePreprocessFailed
	}
	return output, nil
}

func validRemoteSemanticImage(image *vips.ImageRef, expected vips.ImageType, format media.Format, width, height int) bool {
	return image != nil && image.OriginalFormat() == expected && media.ValidateImageDimensions(format, width, height) == nil &&
		int64(image.Width())*int64(image.Height()) <= semantic.RemoteImageMaxPixels
}

func encodeRemoteSemanticImage(ctx context.Context, image *vips.ImageRef) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := image.ToColorSpace(vips.InterpretationSRGB); err != nil {
		return nil, err
	}
	if image.Bands() > 3 {
		if err := image.ExtractBand(0, 3); err != nil {
			return nil, err
		}
	}
	if image.Bands() != 3 {
		return nil, semantic.ErrImagePreprocessFailed
	}
	if largest := max(image.Width(), image.Height()); largest > semantic.RemoteImageMaxDimension {
		if err := image.Resize(float64(semantic.RemoteImageMaxDimension)/float64(largest), vips.KernelLanczos3); err != nil {
			return nil, err
		}
	}
	if image.BandFormat() != vips.BandFormatUchar {
		if err := image.Cast(vips.BandFormatUchar); err != nil {
			return nil, err
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
		if err != nil {
			return nil, err
		}
		return nil, semantic.ErrImagePreprocessFailed
	}
	return output, nil
}

func readRemoteSemanticSource(ctx context.Context, source io.Reader, size int) ([]byte, error) {
	encoded := make([]byte, size)
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
			return nil, semantic.ErrInvalidImageInput
		}
		offset += n
		if readErr != nil && !(readErr == io.EOF && offset == len(encoded)) {
			return nil, semantic.ErrInvalidImageInput
		}
		if n == 0 {
			emptyReads++
			if emptyReads >= 100 {
				return nil, semantic.ErrInvalidImageInput
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
		return nil, semantic.ErrInvalidImageInput
	}
	return encoded, nil
}

var _ semantic.EncodedImagePreprocessor = (*Processor)(nil)

// Validate source dimensions before native allocation. Round up each decoded
// dimension, so odd-sized JPEGs cannot sneak past the working-pixel budget.
func remoteSemanticJPEGShrink(width, height int) (int, bool) {
	if media.ValidateImageDimensions(media.FormatJPEG, width, height) != nil {
		return 0, false
	}
	for _, factor := range []int{1, 2, 4, 8} {
		if int64((width+factor-1)/factor)*int64((height+factor-1)/factor) <= semantic.RemoteImageMaxPixels {
			return factor, true
		}
	}
	return 0, false
}
