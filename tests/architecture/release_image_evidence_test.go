package architecture_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeReleaseImageEvidenceCanBeVerifiedLocally(t *testing.T) {
	root := repositoryRoot(t)
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		return string(content)
	}

	smoke := read("tests/release/image_smoke.sh")
	provenance := read("scripts/generate-provenance.sh")
	makefile := read("Makefile")

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

func TestNativeSupplyChainEvidenceCanBeVerifiedLocally(t *testing.T) {
	root := repositoryRoot(t)
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		return string(content)
	}

	generator := read("scripts/generate-supply-chain-evidence.sh")
	makefile := read("Makefile")

	requireFragments(t, "scripts/generate-supply-chain-evidence.sh", generator, []string{
		`.total == 0`,
		`.critical == 0`,
		`.high == 0`,
		`.name == "foliopath-glib"`,
		`.name == "libblkid1"`,
		`.name == "libglib2.0-0t64"`,
		`.name == "libmount1"`,
		`.name == "libselinux1"`,
		`policy: "all"`,
		`sourceCommit: $source_commit`,
		`workflowRunId: $run_id`,
		`result: "passed"`,
	})
	requireFragments(t, "Makefile", makefile, []string{
		"verify-supply-chain-evidence:",
		`$(GO) run ./tests/release/supplychain_evidence`,
		`-dir "$(EVIDENCE_DIR)" -commit "$(RELEASE_SHA)"`,
	})
}
