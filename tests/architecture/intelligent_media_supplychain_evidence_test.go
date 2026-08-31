package architecture_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntelligentMediaSupplyChainEvidenceHasOneExecutableOwner(t *testing.T) {
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
	gate := read("docs/gates/POST-MVP-5/int-s2a-backend-evidence-ready.md")
	testingStrategy := read("docs/testing-strategy.md")

	requireFragments(t, "Makefile", makefile, []string{
		"verify-intelligent-media-supply-chain:",
		`$(GO) run ./tests/release/intelligent_media_supplychain_evidence`,
		`-input "$(SUPPLY_CHAIN_INPUT)" -commit "$(RELEASE_SHA)"`,
	})
	requireFragments(t, "INT-S2A Gate", gate, []string{
		"verify-intelligent-media-supply-chain",
		"本 Gate 继续 No-Go",
	})
	requireFragments(t, "testing strategy", testingStrategy, []string{
		"make verify-intelligent-media-supply-chain",
		"security、compliance、release、inference owner 仍须审阅并签署",
	})
}
