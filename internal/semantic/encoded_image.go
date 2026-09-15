package semantic

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
)

// Remote image preparation is separate from the legacy SigLIP tensor contract.
const (
	RemoteImageMaxSourceBytes = 32 << 20
	RemoteImageMaxPixels      = 16_000_000
	RemoteImageMaxDimension   = 1024
	RemoteImageMaxBytes       = 8 << 20
)

// EncodedImagePreprocessor accepts only streams already opened by internal/files.
// The caller owns stream closure and bounded worker concurrency.
type EncodedImagePreprocessor interface {
	PrepareRemoteSemanticImage(context.Context, io.ReadSeeker, media.Format) ([]byte, error)
}
