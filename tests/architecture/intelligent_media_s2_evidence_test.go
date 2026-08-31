package architecture_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntelligentMediaS2EvidenceBindingRemainsExecutable(t *testing.T) {
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
	testingStrategy := read("docs/testing-strategy.md")
	requireFragments(t, "Makefile", makefile, []string{
		"verify-intelligent-media-s2-evidence:",
		`$(GO) run ./tests/release/intelligent_media_s2_evidence`,
		`-quality "$(QUALITY_SUMMARY)" -native "$(NATIVE_SUMMARY)"`,
		`-supply-chain "$(SUPPLY_CHAIN_SUMMARY)" -commit "$(RELEASE_SHA)"`,
	})
	requireFragments(t, "testing strategy", testingStrategy, []string{
		"make verify-intelligent-media-s2-evidence",
		"同一 source commit、同一 model package",
		"finalModelEvidence=false",
	})
}
