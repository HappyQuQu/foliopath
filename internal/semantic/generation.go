package semantic

import "context"

type GenerationRuntime struct {
	ID                 string
	ModelID            string
	EmbeddingDimension int
	State              string
}

type GenerationRuntimeRepository interface {
	GetSemanticGenerationRuntime(context.Context, string) (GenerationRuntime, error)
}
