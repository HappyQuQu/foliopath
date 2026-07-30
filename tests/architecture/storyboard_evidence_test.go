package architecture_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoryboardEvidenceCanBeVerifiedLocally(t *testing.T) {
	root := repositoryRoot(t)
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		return string(content)
	}

	smoke := read("tests/release/storyboard_vertical_smoke.sh")
	makefile := read("Makefile")

	requireFragments(t, "tests/release/storyboard_vertical_smoke.sh", smoke, []string{
		`test "${image_arch}" = "${FOLIOPATH_STORYBOARD_EXPECTED_ARCH}"`,
		`decodedPixelSHA256: $storyboard_pixel_sha256`,
		`missingStatus: 202`,
		`decodedPixelsMatch: true`,
		`originalMediaUnchanged: true`,
		`workflowRunId: $run_id`,
	})
	requireFragments(t, "Makefile", makefile, []string{
		"verify-storyboard-evidence:",
		`$(GO) run ./tests/release/storyboard_evidence`,
		`-dir "$(EVIDENCE_DIR)" -commit "$(RELEASE_SHA)"`,
		`-output "$(SUMMARY_FILE)"`,
	})
}
