package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/inference/sentencepiece"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type legacyTextRuntimeMustNotOpen struct{}

func (legacyTextRuntimeMustNotOpen) OpenTextSession(context.Context, aimodel.Manifest, aimodel.RuntimeFileOpener) (onnx.TextSession, error) {
	panic("bound space must not open legacy text runtime")
}

func (legacyTextRuntimeMustNotOpen) Open(context.Context, aimodel.RuntimeModelFile) (sentencepiece.Session, error) {
	panic("bound space must not open legacy tokenizer")
}

func TestSemanticLegacyFactoriesRejectBoundSpace(t *testing.T) {
	models, err := aimodel.NewService(&semanticModelRepositoryStub{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, fingerprint := range []string{"", strings.Repeat("a", 64)} {
		t.Run("fingerprint="+fingerprint, func(t *testing.T) {
			generations := semanticGenerationRuntimeStub{generation: semantic.GenerationRuntime{
				ID: "generation", State: "active", EmbeddingDimension: semantic.SigLIPEmbeddingDimension,
				SpaceFingerprint: fingerprint,
			}}
			imageFactory := semanticProductionSessionFactory{
				generations: generations, models: models, catalog: &aimodel.Catalog{},
				source: semanticActivationSourceStub{}, availability: &semanticAvailabilityMarkerStub{},
				runtime: semanticProductionRuntimeStub{},
			}
			textFactory := semanticProductionTextSessionFactory{
				generations: generations, models: models, catalog: &aimodel.Catalog{},
				source: semanticActivationSourceStub{}, availability: &semanticAvailabilityMarkerStub{},
				onnxRuntime: legacyTextRuntimeMustNotOpen{}, tokenizerRuntime: legacyTextRuntimeMustNotOpen{},
			}
			want := semantic.ErrSemanticGenerationUnavailable
			if fingerprint == "" {
				// The legacy generation passes admission and reaches model lookup.
				want = aimodel.ErrModelUnavailable
			}
			for name, validate := range map[string]func(context.Context, string) error{
				"image": imageFactory.ValidateSemanticImageSession,
				"text":  textFactory.ValidateSemanticTextSession,
			} {
				if err := validate(context.Background(), "generation"); !errors.Is(err, want) {
					t.Fatalf("%s error = %v, want %v", name, err, want)
				}
			}
		})
	}
}
