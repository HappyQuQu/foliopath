//go:build !libvips

package imagevips

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
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

func (*Processor) PrepareSemanticImage(
	context.Context,
	io.ReadSeeker,
	media.Format,
) ([]float32, error) {
	return nil, semantic.ErrImagePreprocessUnavailable
}

func (*Processor) SplitSemanticStoryboard(
	context.Context,
	io.ReadSeeker,
	int, int, int, int,
) ([][]byte, error) {
	return nil, semantic.ErrImagePreprocessUnavailable
}

func (*Processor) DecodeFaceImage(
	context.Context,
	io.ReadSeeker,
	media.Format,
	int64,
	int,
) (face.DecodedImage, error) {
	return face.DecodedImage{}, face.ErrRuntimeUnavailable
}
