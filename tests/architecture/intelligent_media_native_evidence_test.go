package architecture_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntelligentMediaNativeEvidenceCanBeVerifiedLocally(t *testing.T) {
	root := repositoryRoot(t)
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		return string(content)
	}
	makefile := read("Makefile")
	workflow := read(".github/workflows/intelligent-media-native.yml")
	faceCandidate := read("tests/release/face_candidate_native_smoke.sh")
	faceCandidateDockerfile := read("tests/release/face_candidate_native.Dockerfile")

	requireFragments(t, "Makefile", makefile, []string{
		"verify-intelligent-media-native-evidence:",
		`$(GO) run ./tests/release/intelligent_media_native_evidence`,
		`-run-id "$(WORKFLOW_RUN_ID)" -run-attempt "$(WORKFLOW_RUN_ATTEMPT)"`,
		"verify-intelligent-media-native-model-evidence:",
		`-require-model -output "$(SUMMARY_FILE)"`,
		"verify-intelligent-media-quality:",
		`cd spikes/int001-ai && $(GO) run . quality-score`,
		"verify-intelligent-media-face-quality:",
		`cd spikes/int001-ai && $(GO) run . face-quality-score`,
		`-dataset-manifest "$(abspath $(DATASET_MANIFEST))"`,
	})
	requireFragments(t, ".github/workflows/intelligent-media-native.yml", workflow, []string{
		"Verify paired native evidence",
		"make verify-intelligent-media-native-evidence",
		"intelligent-media-native-paired-",
		"Run pinned face candidate and synthetic capacity preflights",
		"faceCandidate",
	})
	requireFragments(t, "tests/release/face_candidate_native_smoke.sh", faceCandidate, []string{
		`test "$(uname -s)" = Linux`,
		`amd64:x86_64:x86_64`,
		`arm64:aarch64:aarch64`,
		`--network none --read-only`,
		`candidate-native-functional-preflight-only`,
		`synthetic-native-capacity-only`,
		`--cpus 4 --memory 4g`,
		`embedding_dimension`,
		`paired_cluster_count`,
		`singleton_cluster_count`,
		`identityGroundTruth: false`,
		`productionApproved: false`,
		`qualityGate: false`,
		`complianceGate: false`,
	})
	requireFragments(t, "tests/release/face_candidate_native.Dockerfile", faceCandidateDockerfile, []string{
		`test "$(cat /opt/onnxruntime/VERSION_NUMBER)" = "1.28.0"`,
		`test "$(cat /opt/onnxruntime/GIT_COMMIT_ID)" = "${ORT_COMMIT}"`,
		`-tags "libvips onnxruntime"`,
		`-o /out/face-capacity.test ./internal/face`,
	})
}
