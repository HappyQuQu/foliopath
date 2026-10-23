package semantic

import (
	"context"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
)

// EncodedTransport supplies vectors, not model identity attestation.
type EncodedTransport interface {
	EncodeText(context.Context, string) ([]float32, error)
	EncodeImage(context.Context, []byte) ([]float32, error)
}

// BoundEncoder shares generation admission across text and encoded image paths.
// Construction requires deployment-verified identity; it does not verify deployment.
type BoundEncoder struct {
	repository GenerationRuntimeRepository
	images     EncodedImagePreprocessor
	transport  EncodedTransport
	identity   SpaceIdentity
}

func NewBoundEncoder(repository GenerationRuntimeRepository, images EncodedImagePreprocessor, transport EncodedTransport, identity SpaceIdentity) (*BoundEncoder, error) {
	if repository == nil || images == nil || transport == nil {
		return nil, ErrImageEncoderUnavailable
	}
	if _, err := identity.Fingerprint(); err != nil {
		return nil, err
	}
	return &BoundEncoder{repository: repository, images: images, transport: transport, identity: identity}, nil
}

func (e *BoundEncoder) admit(ctx context.Context, id string, image bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	generation, err := e.repository.GetSemanticGenerationRuntime(ctx, id)
	if err != nil {
		return err
	}
	if generation.ID != id || generation.EmbeddingDimension != e.identity.Dimension ||
		(generation.State != "active" && !(image && (generation.State == "building" || generation.State == "ready"))) {
		return ErrSemanticGenerationUnavailable
	}
	return RequireSpace(generation.SpaceFingerprint, e.identity)
}

func (e *BoundEncoder) EncodeSemanticText(ctx context.Context, generationID, text string) ([]float32, error) {
	if err := e.admit(ctx, generationID, false); err != nil {
		return nil, err
	}
	vector, err := e.transport.EncodeText(ctx, text)
	if err != nil {
		return nil, err
	}
	if err := e.admit(ctx, generationID, false); err != nil {
		return nil, err
	}
	return NormalizeEmbedding(vector, e.identity.Dimension)
}

// EncodeSemanticStream never closes the caller-owned, already-opened stream.
func (e *BoundEncoder) EncodeSemanticStream(ctx context.Context, generationID string, source io.ReadSeeker, format media.Format) ([]float32, error) {
	if err := e.admit(ctx, generationID, true); err != nil {
		return nil, err
	}
	encoded, err := e.images.PrepareRemoteSemanticImage(ctx, source, format)
	if err != nil {
		return nil, err
	}
	if len(encoded) == 0 || len(encoded) > RemoteImageMaxBytes {
		return nil, ErrInvalidImageInput
	}
	if err := e.admit(ctx, generationID, true); err != nil {
		return nil, err
	}
	vector, err := e.transport.EncodeImage(ctx, encoded)
	if err != nil {
		return nil, err
	}
	if err := e.admit(ctx, generationID, true); err != nil {
		return nil, err
	}
	return NormalizeEmbedding(vector, e.identity.Dimension)
}

var _ TextEncoder = (*BoundEncoder)(nil)
var _ StreamImageEncoder = (*BoundEncoder)(nil)
