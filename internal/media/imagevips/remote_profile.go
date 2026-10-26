package imagevips

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type remoteImagePreprocessor struct{ profile semantic.RemoteImageProfile }

// NewRemoteImagePreprocessor binds a reviewed deployment profile. New() retains
// the legacy contract for unconfigured automatic tags and existing callers.
func NewRemoteImagePreprocessor(profile semantic.RemoteImageProfile) (semantic.EncodedImagePreprocessor, error) {
	if _, ok := profile.Contract(); !ok {
		return nil, semantic.ErrInvalidImageInput
	}
	return remoteImagePreprocessor{profile: profile}, nil
}

func (p remoteImagePreprocessor) PrepareRemoteSemanticImage(ctx context.Context, source io.ReadSeeker, format media.Format) ([]byte, error) {
	return prepareRemoteSemanticImage(ctx, source, format, p.profile)
}

func (*Processor) PrepareRemoteSemanticImage(ctx context.Context, source io.ReadSeeker, format media.Format) ([]byte, error) {
	return prepareRemoteSemanticImage(ctx, source, format, semantic.RemoteImageLegacy)
}
