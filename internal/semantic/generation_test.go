package semantic

import (
	"errors"
	"strings"
	"testing"
)

func TestLegacySigLIPRejectsBoundSpaceEvenWithSameDimension(t *testing.T) {
	for _, state := range []string{"active", "building", "ready", "retired", "failed"} {
		for _, dimension := range []int{512, 768} {
			for _, fingerprint := range []string{"", strings.Repeat("a", 64), "invalid"} {
				err := RequireLegacySigLIPGeneration(GenerationRuntime{State: state, EmbeddingDimension: dimension, SpaceFingerprint: fingerprint})
				allowed := state == "active" && dimension == 768 && fingerprint == ""
				if allowed && err != nil || !allowed && !errors.Is(err, ErrSemanticGenerationUnavailable) {
					t.Fatalf("state=%s dimension=%d bound=%t err=%v", state, dimension, fingerprint != "", err)
				}
			}
		}
	}
}
