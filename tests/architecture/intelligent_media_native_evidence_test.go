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

	requireFragments(t, "Makefile", makefile, []string{
		"verify-intelligent-media-native-evidence:",
		`$(GO) run ./tests/release/intelligent_media_native_evidence`,
		`-run-id "$(WORKFLOW_RUN_ID)" -run-attempt "$(WORKFLOW_RUN_ATTEMPT)"`,
		"verify-intelligent-media-native-model-evidence:",
		`-require-model -output "$(SUMMARY_FILE)"`,
		"verify-intelligent-media-quality:",
		`cd spikes/int001-ai && $(GO) run . quality-score`,
		`-dataset-manifest "$(abspath $(DATASET_MANIFEST))"`,
	})
	requireFragments(t, ".github/workflows/intelligent-media-native.yml", workflow, []string{
		"Verify paired native evidence",
		"make verify-intelligent-media-native-evidence",
		"intelligent-media-native-paired-",
	})
}
