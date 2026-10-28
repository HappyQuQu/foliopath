package media

import (
	"context"
	"io"
	"time"
)

type ExifField struct {
	Key   string
	Value string
}

type ExifResult struct {
	Fields []ExifField
}

type ExifReader interface {
	ReadExif(context.Context, io.ReadSeeker, Format) ([]ExifField, error)
}

type ExifService struct {
	content *ContentService
	reader  ExifReader
}

func NewExifService(content *ContentService, reader ExifReader) *ExifService {
	return &ExifService{content: content, reader: reader}
}

func (service *ExifService) Read(ctx context.Context, assetID int64) (ExifResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	content, err := service.content.Open(ctx, assetID)
	if err != nil {
		return ExifResult{}, err
	}
	defer content.File.Close()
	result := ExifResult{Fields: []ExifField{}}
	kind, format, _, _ := ClassifyPath(content.Name)
	if kind == KindVideo {
		return result, nil
	}
	fields, err := service.reader.ReadExif(ctx, content.File, format)
	if err != nil {
		return ExifResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ExifResult{}, err
	}
	info, err := content.File.Stat()
	if err != nil {
		return ExifResult{}, ErrContentUnavailable
	}
	if !content.Fingerprint.Matches(info.Size(), info.ModTime().UnixNano()) {
		return ExifResult{}, ErrContentSourceChanged
	}
	result.Fields = fields
	return result, nil
}
