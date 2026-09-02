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
	strictJSON := read("tests/release/evidencejson/decode.go")
	nativeVerifier := read("tests/release/intelligent_media_native_evidence/main.go")
	supplyVerifier := read("tests/release/intelligent_media_supplychain_evidence/main.go")
	aggregateVerifier := read("tests/release/intelligent_media_s2_evidence/main.go")
	requireFragments(t, "Makefile", makefile, []string{
		"verify-intelligent-media-s2-evidence:",
		`$(GO) run ./tests/release/intelligent_media_s2_evidence`,
		`-quality "$(QUALITY_SUMMARY)" -face-quality "$(FACE_QUALITY_SUMMARY)" -native "$(NATIVE_SUMMARY)"`,
		`-supply-chain "$(SUPPLY_CHAIN_SUMMARY)" -commit "$(RELEASE_SHA)"`,
	})
	requireFragments(t, "testing strategy", testingStrategy, []string{
		"make verify-intelligent-media-s2-evidence",
		"同一 source commit、同一 model package",
		"finalModelEvidence=false",
		"duplicate key、unknown field 与 trailing value",
	})
	requireFragments(t, "release evidence JSON owner", strictJSON, []string{
		"func Decode(data []byte, target any) error",
		"rejectDuplicateKeys(data)",
		"decoder.DisallowUnknownFields()",
		"trailing JSON value",
	})
	for name, source := range map[string]string{
		"native verifier": nativeVerifier, "supply verifier": supplyVerifier, "aggregate verifier": aggregateVerifier,
	} {
		requireFragments(t, name, source, []string{"evidencejson.Decode("})
	}
}
