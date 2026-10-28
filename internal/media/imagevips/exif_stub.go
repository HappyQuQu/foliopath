//go:build !libvips

package imagevips

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func (*Processor) ReadExif(context.Context, io.ReadSeeker, media.Format) ([]media.ExifField, error) {
	return nil, media.ErrProcessingFailed
}
