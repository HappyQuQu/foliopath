//go:build !libvips

package imagevips

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

func (*Processor) PrepareOCRImage(ctx context.Context, _ io.ReadSeeker, _ media.Format) (ocr.PreparedImage, error) {
	if err := ctx.Err(); err != nil {
		return ocr.PreparedImage{}, err
	}
	return ocr.PreparedImage{}, ocr.ErrPreprocessUnavailable
}

var _ ocr.ImagePreprocessor = (*Processor)(nil)
