//go:build libvips

package imagevips

import (
	"context"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/davidbyttow/govips/v2/vips"
)

func (*Processor) ReadExif(ctx context.Context, source io.ReadSeeker, format media.Format) ([]media.ExifField, error) {
	finish, err := beginLargeImageDecode(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, media.ErrProcessingFailed
	}
	encoded, err := readImageSource(ctx, source, format)
	if err != nil {
		return nil, err
	}
	params := imageImportParams()
	params.AutoRotate.Set(false)
	image, err := vips.LoadImageFromBuffer(encoded, params)
	if err != nil {
		return nil, media.ErrInvalidMedia
	}
	defer image.Close()
	expected, ok := expectedImageType(format)
	if !ok || image.OriginalFormat() != expected {
		return nil, media.ErrInvalidMedia
	}
	fields := make([]media.ExifField, 0, 8)
	for _, tag := range []struct{ key, name string }{
		{"capturedAt", "exif-ifd2-DateTimeOriginal"},
		{"cameraMake", "exif-ifd0-Make"},
		{"cameraModel", "exif-ifd0-Model"},
		{"lensModel", "exif-ifd2-LensModel"},
		{"aperture", "exif-ifd2-FNumber"},
		{"exposureTime", "exif-ifd2-ExposureTime"},
		{"iso", "exif-ifd2-ISOSpeedRatings"},
		{"focalLength", "exif-ifd2-FocalLength"},
	} {
		value := exifValue(tag.key, image.GetString(tag.name))
		if value != "" {
			runes := []rune(value)
			fields = append(fields, media.ExifField{Key: tag.key, Value: string(runes[:min(len(runes), 160)])})
		}
	}
	return fields, ctx.Err()
}

func exifText(value string) string {
	// libvips appends a native type/component description to EXIF values.
	if suffix := strings.LastIndex(value, " ("); suffix >= 0 {
		value = value[:suffix]
	}
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	return string(runes[:min(len(runes), 160)])
}

func exifValue(key, raw string) string {
	value := exifText(raw)
	if value == "" {
		return ""
	}
	if key == "capturedAt" {
		date, err := time.Parse("2006:01:02 15:04:05", value)
		if err != nil {
			return ""
		}
		return date.Format("2006-01-02 15:04:05")
	}
	if key != "aperture" && key != "focalLength" && key != "exposureTime" {
		return value
	}
	parts := strings.Split(value, "/")
	if len(parts) > 2 {
		return ""
	}
	number, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return ""
	}
	if len(parts) == 2 {
		denominator, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || denominator <= 0 {
			return ""
		}
		number /= denominator
	}
	if number <= 0 || math.IsNaN(number) || math.IsInf(number, 0) {
		return ""
	}
	switch key {
	case "aperture":
		return "f/" + strconv.FormatFloat(number, 'g', -1, 64)
	case "focalLength":
		return strconv.FormatFloat(number, 'g', -1, 64) + " mm"
	default:
		return value + " s"
	}
}
