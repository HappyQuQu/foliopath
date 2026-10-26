package semantic

import "context"

type GenerationRuntime struct {
	ID                 string
	ModelID            string
	EmbeddingDimension int
	State              string
	SpaceFingerprint   string
}

// GenerationSpaceRepository binds a verified space before generation work begins.
// It does not perform deployment admission or activate the generation.
type GenerationSpaceRepository interface {
	BindSemanticGenerationSpace(context.Context, string, SpaceIdentity) error
}

type GenerationRuntimeRepository interface {
	GetSemanticGenerationRuntime(context.Context, string) (GenerationRuntime, error)
}

// RequireLegacySigLIPGeneration prevents deployment-bound spaces from falling
// through to the old package/tensor runtime, even if dimensions happen to match.
func RequireLegacySigLIPGeneration(generation GenerationRuntime) error {
	if generation.State != "active" || generation.EmbeddingDimension != SigLIPEmbeddingDimension || generation.SpaceFingerprint != "" {
		return ErrSemanticGenerationUnavailable
	}
	return nil
}
