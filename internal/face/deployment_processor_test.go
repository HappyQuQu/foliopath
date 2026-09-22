package face

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type deploymentRuntimeStub struct {
	value DeploymentGenerationRuntime
	err   error
}

func (r *deploymentRuntimeStub) GetFaceDeploymentGenerationRuntime(context.Context, string) (DeploymentGenerationRuntime, error) {
	return r.value, r.err
}

type deploymentAnalyzerFunc func(context.Context, int64, int64, string) ([]Observation, error)

func (f deploymentAnalyzerFunc) Analyze(ctx context.Context, l, a int64, s string) ([]Observation, error) {
	return f(ctx, l, a, s)
}

type deploymentCatalogStub struct{ JobCatalog }

func (deploymentCatalogStub) FaceJobLibraryState(context.Context, int64, string) (JobLibraryState, error) {
	return JobLibraryReady, nil
}

func TestDeploymentJobBinding(t *testing.T) {
	identity := DeploymentIdentity{Version: 1, Protocol: "foliopath-face-v1", Model: "buffalo_l", Dimension: 512,
		RuntimeSHA256: strings.Repeat("a", 64), CacheManifestSHA256: strings.Repeat("b", 64), PreprocessingSHA256: strings.Repeat("c", 64),
		DetectorSHA256: strings.Repeat("d", 64), EmbedderSHA256: strings.Repeat("e", 64),
		Threshold: ThresholdProfile{SchemaVersion: 1, ProfileID: "test-only", CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2, QualitySummarySHA256: strings.Repeat("f", 64)}}
	fingerprint, err := identity.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"success", "wrong-id", "wrong-space", "retired", "offline", "wrong-dimension", "read-error", "switch-during-analysis", "offline-empty-result", "bad-vector", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			repo := &deploymentRuntimeStub{value: DeploymentGenerationRuntime{ID: "face_generation", Fingerprint: fingerprint, State: "active", Dimension: 512, Available: true}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			binding := &deploymentJobBinding{JobCatalog: deploymentCatalogStub{}, repository: repo, identity: identity, generationID: repo.value.ID,
				analyzer: deploymentAnalyzerFunc(func(context.Context, int64, int64, string) ([]Observation, error) {
					calls++
					if scenario == "switch-during-analysis" {
						repo.value.State = "retired"
					}
					if scenario == "offline-empty-result" {
						repo.value.Available = false
						return nil, nil
					}
					dimension := 512
					if scenario == "bad-vector" {
						dimension = 256
					}
					return []Observation{{Embedding: make([]float32, dimension)}}, nil
				})}
			switch scenario {
			case "wrong-id":
				repo.value.ID = "other_generation"
			case "wrong-space":
				repo.value.Fingerprint = strings.Repeat("0", 64)
			case "retired":
				repo.value.State = "retired"
			case "offline":
				repo.value.Available = false
			case "wrong-dimension":
				repo.value.Dimension = 256
			case "read-error":
				repo.err = errors.New("synthetic database failure")
			case "cancelled":
				cancel()
			}
			items, err := binding.Analyze(ctx, 1, 1, "source")
			if scenario == "success" {
				if err != nil || len(items) != 1 || calls != 1 {
					t.Fatal(items, err, calls)
				}
				return
			}
			want := ErrRuntimeUnavailable
			if scenario == "bad-vector" {
				want = ErrInvalidOutput
			}
			if scenario == "cancelled" {
				want = context.Canceled
			}
			if !errors.Is(err, want) || len(items) != 0 {
				t.Fatal(items, err)
			}
			if scenario != "switch-during-analysis" && scenario != "offline-empty-result" && scenario != "bad-vector" && calls != 0 {
				t.Fatal("unadmitted inference")
			}
			if scenario != "bad-vector" && scenario != "cancelled" {
				state, err := binding.FaceJobLibraryState(ctx, 1, binding.generationID)
				if err != nil || state != JobLibraryModelUnavailable {
					t.Fatal("empty-job control bypass", state, err)
				}
			}
		})
	}
}
