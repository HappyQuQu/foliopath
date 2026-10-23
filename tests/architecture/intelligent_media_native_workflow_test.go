package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntelligentMediaNativeEvidenceWorkflowFailsClosed(t *testing.T) {
	root := repositoryRoot(t)
	path := filepath.Join(root, ".github", "workflows", "intelligent-media-native.yml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read intelligent media native workflow: %v", err)
	}
	workflow := string(content)
	requireFragments(t, ".github/workflows/intelligent-media-native.yml", workflow, []string{
		"workflow_dispatch:",
		"workflow_call:",
		"permissions:\n  contents: read",
		"sudo apt-get install --yes --no-install-recommends ripgrep",
		"runner: ubuntu-24.04\n            goarch: amd64\n            machine: x86_64",
		"runner: ubuntu-24.04-arm\n            goarch: arm64\n            machine: aarch64",
		`test "$(uname -m)" = "${EXPECTED_MACHINE}"`,
		`test "$(go env GOARCH)" = "${EXPECTED_GOARCH}"`,
		`test -z "${DOCKER_DEFAULT_PLATFORM:-}"`,
		"qemuAllowed: false",
		"make fmt-check arch-check generate-check lint test test-integration test-e2e",
		"make test-libvips",
		"FOLIOPATH_SEARCH_MATRIX_MULTILIB=1 GOMAXPROCS=4",
		"make spike-capacity",
		"if: always()",
		"actions/upload-artifact@v4",
		"if-no-files-found: error",
		"retention-days: 14",
	})

	for _, forbidden := range []string{
		"pull_request:",
		"push:",
		"setup-qemu",
		"DOCKER_DEFAULT_PLATFORM=",
		"continue-on-error:",
		"contents: write",
		"/library",
		"/app/data",
	} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("native evidence workflow contains forbidden fragment %q", forbidden)
		}
	}
}
