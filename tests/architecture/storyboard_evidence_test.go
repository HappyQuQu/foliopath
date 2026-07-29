package architecture_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoryboardEvidenceRemainsPairedAcrossNativeArchitectures(t *testing.T) {
	root := repositoryRoot(t)
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		return string(content)
	}

	workflow := read(".github/workflows/ci.yml")
	smoke := read("tests/release/storyboard_vertical_smoke.sh")
	makefile := read("Makefile")

	requireFragments(t, ".github/workflows/ci.yml", workflow, []string{
		"name: Storyboard candidate (${{ matrix.arch }})",
		"runner: ubuntu-24.04",
		"runner: ubuntu-24.04-arm",
		"FOLIOPATH_STORYBOARD_EXPECTED_ARCH: ${{ matrix.arch }}",
		"FOLIOPATH_STORYBOARD_SOURCE_COMMIT: ${{ github.sha }}",
		"name: storyboard-evidence-${{ github.sha }}-${{ matrix.arch }}",
		"name: Verify paired storyboard evidence",
		"actions/download-artifact@018cc2cf5baa6db3ef3c5f8a56943fffe632ef53",
		"merge-multiple: true",
		"make verify-storyboard-evidence",
		`SUMMARY_FILE="${RUNNER_TEMP}/storyboard-paired-evidence.json"`,
		"name: Upload verified paired storyboard summary",
		"name: storyboard-paired-evidence-${{ github.sha }}",
	})
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
