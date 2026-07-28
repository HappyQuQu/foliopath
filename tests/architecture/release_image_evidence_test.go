package architecture_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeReleaseImageEvidenceRemainsBoundToTheWorkflowRun(t *testing.T) {
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
	smoke := read("tests/release/image_smoke.sh")
	provenance := read("scripts/generate-provenance.sh")
	makefile := read("Makefile")

	requireFragments(t, ".github/workflows/ci.yml", workflow, []string{
		"FOLIOPATH_RELEASE_EVIDENCE: ${{ runner.temp }}/release-image-${{ matrix.arch }}.json",
		"FOLIOPATH_RELEASE_EXPECTED_ARCH: ${{ matrix.arch }}",
		"FOLIOPATH_RELEASE_COMMIT: ${{ github.sha }}",
		"FOLIOPATH_RELEASE_RUN_ID: ${{ github.run_id }}",
		"FOLIOPATH_RELEASE_RUN_ATTEMPT: ${{ github.run_attempt }}",
		"FOLIOPATH_RELEASE_PROVENANCE: ${{ runner.temp }}/release-provenance-${{ matrix.arch }}.json",
		"name: release-image-${{ github.sha }}-${{ matrix.arch }}",
		"if-no-files-found: error",
	})
	requireFragments(t, "tests/release/image_smoke.sh", smoke, []string{
		`test "${image_arch}" = "${FOLIOPATH_RELEASE_EXPECTED_ARCH}"`,
		`sourceCommit: $source_commit`,
		`imageDigest: $image_digest`,
		`result: "passed"`,
		`workflowRunId: $run_id`,
		`workflowRunAttempt: $run_attempt`,
		`generate-provenance.sh`,
	})
	requireFragments(t, "scripts/generate-provenance.sh", provenance, []string{
		`provenance requires a clean source tree`,
		`https://in-toto.io/Statement/v1`,
		`https://slsa.dev/provenance/v1`,
		`"gitCommit": $source_commit`,
		`"sha256": $dockerfile_digest`,
	})
	requireFragments(t, "Makefile", makefile, []string{
		"verify-release-image-evidence:",
		`$(GO) run ./tests/release/evidence`,
		`-dir "$(EVIDENCE_DIR)" -commit "$(RELEASE_SHA)"`,
	})
}
