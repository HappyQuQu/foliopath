//go:build !libvips

package imagevips

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
)

// Processor keeps development and contract builds independent of the native
// library. Release builds must enable the libvips tag; this stub fails closed.
type Processor struct{}

func New() *Processor {
	return &Processor{}
}

func (*Processor) Process(
	context.Context,
	io.ReadSeeker,
	media.Format,
) (media.ProcessingResult, error) {
	return media.ProcessingResult{}, media.ErrProcessingFailed
}
