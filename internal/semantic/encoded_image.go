package semantic

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
)

// Remote image preparation is separate from the legacy SigLIP tensor contract.
const (
	RemoteImageMaxSourceBytes = 32 << 20
	RemoteJPEGMaxSourceBytes  = media.MaxImageSourceBytes
	RemoteImageMaxPixels      = 16_000_000
	RemoteImageMaxDimension   = 1024
	RemoteImageMaxBytes       = 8 << 20
	// Versioned main-application preprocessing, included in deployment identity.
	RemoteImagePreprocessingContract = "foliopath-remote-image-v3:jpeg,png,webp,gif:first-frame:autorotate:srgb:no-alpha:uchar:lanczos3:aspect-longedge1024:png-strip:source32MiB:jpeg-source256MiB-fd-stream-180MP:jpeg-shrink-min-1-2-4-8-ceil:working-pixels16000000:output8MiB"
)

// EncodedImagePreprocessor accepts only streams already opened by internal/files.
// The caller owns stream closure and bounded worker concurrency.
type EncodedImagePreprocessor interface {
	PrepareRemoteSemanticImage(context.Context, io.ReadSeeker, media.Format) ([]byte, error)
}
