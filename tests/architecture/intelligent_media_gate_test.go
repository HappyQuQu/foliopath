package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntelligentMediaNoGoFailsClosedInProductionComposition(t *testing.T) {
	root := repositoryRoot(t)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "adr", "0014-siglip-sentencepiece-tokenizer-runtime.md"),
		"**Accepted for fail-closed backend implementation（",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "gates", "POST-MVP-5", "int-s2a-backend-evidence-ready.md"),
		"当前判断：**Backend Ready / Release No-Go**",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "gates", "POST-MVP-5", "int-s2b-backend-evidence-ready.md"),
		"当前判断：**Backend Ready / Release No-Go**",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "gates", "POST-MVP-5", "int-s2c-privacy-ready.md"),
		"当前判断：**Backend Ready / Release No-Go**",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "changes", "CR-2026-022-s2-backend-release-gate-separation.md"),
		"S4 任一最终证据缺失或失败时",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "gates", "POST-MVP-5", "adr-0014-acceptance-audit-2026-08-29.md"),
		"当前裁决：**保持提议 / Blocked**",
	)

	runPath := filepath.Join(root, "internal", "app", "run.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), runPath, nil, 0)
	if err != nil {
		t.Fatalf("parse production composition: %v", err)
	}
	for _, imported := range parsed.Imports {
		if imported.Path != nil && strings.Contains(imported.Path.Value, "/internal/inference/faceonnx") {
			t.Error("face runtime adapter must remain absent from production composition while INT-S2C release gate is No-Go")
		}
	}

	emptyCatalogCalls := 0
	routeCompositions := 0
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.CallExpr:
			selector, ok := current.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "NewCatalog" || selectorPackage(selector) != "aimodel" {
				return true
			}
			if len(current.Args) != 1 || !isNilIdentifier(current.Args[0]) {
				t.Errorf("production reviewed AI catalog must remain empty while intelligent-media release gate is No-Go")
				return true
			}
			emptyCatalogCalls++
		case *ast.CompositeLit:
			selector, ok := current.Type.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "RouteDependencies" || selectorPackage(selector) != "api" {
				return true
			}
			routeCompositions++
			hasSemantic, hasSemanticSearch, hasFace := false, false, false
			for _, element := range current.Elts {
				field, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := field.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch key.Name {
				case "Semantic":
					hasSemantic = true
				case "SemanticSearch":
					hasSemanticSearch = true
				default:
					hasFace = hasFace || strings.HasPrefix(key.Name, "Face")
				}
			}
			if !hasSemantic {
				t.Errorf("approved semantic settings/backfill/clear routes must remain composed")
			}
			if !hasSemanticSearch {
				t.Errorf("accepted fail-closed semantic search route must remain composed")
			}
			if hasFace {
				t.Errorf("face routes must remain absent while INT-S2C release gate is No-Go")
			}
		}
		return true
	})

	if emptyCatalogCalls != 1 {
		t.Errorf("production composition has %d empty reviewed AI catalog constructors, want 1", emptyCatalogCalls)
	}
	if routeCompositions != 1 {
		t.Errorf("production composition has %d API route dependency literals, want 1", routeCompositions)
	}
	assertDocumentContains(t, runPath, "routeDependencies.VideoSemanticSearch = videoSemanticSearch")
}

func TestIntelligentMediaRevision2ContractAndFacePrivacyGates(t *testing.T) {
	root := repositoryRoot(t)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "releases", "POST-MVP-5-scope-r2.md"),
		"状态：**Frozen scope / S1R2 contract accepted**",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "gates", "POST-MVP-5", "int-s1r2-contract-ready.md"),
		"当前判断：**Go / Contract Ready**",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "gates", "POST-MVP-5", "int-s2c-privacy-ready.md"),
		"当前判断：**Backend Ready / Release No-Go**",
	)
	if _, err := os.Stat(filepath.Join(root, "internal", "face")); err != nil {
		t.Fatal(err)
	}
	openAPI := filepath.Join(root, "api", "openapi.yaml")
	for _, required := range []string{
		"/api/v1/libraries/{libraryId}/ai/tag-suggestions:",
		"/api/v1/semantic/videos:",
		"/api/v1/libraries/{libraryId}/ai/faces:",
		"/api/v1/people:",
	} {
		assertDocumentContains(t, openAPI, required)
	}
}

func TestAcceptedFacePackageArchitectureKeepsReleaseClosedAndSpikeIsolated(t *testing.T) {
	root := repositoryRoot(t)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "adr", "0015-face-model-package-and-generation-activation.md"),
		"**Accepted for fail-closed backend implementation（",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "gates", "POST-MVP-5", "int-s2c-privacy-ready.md"),
		"当前判断：**Backend Ready / Release No-Go**",
	)

	for _, sourceRoot := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, sourceRoot), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imported := range parsed.Imports {
				if imported.Path != nil && strings.Contains(imported.Path.Value, "/spikes/int001-model-package-v2") {
					t.Errorf("production source %s imports the proposed face package parser", path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("inspect production %s imports: %v", sourceRoot, err)
		}
	}
}

func assertDocumentContains(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), expected) {
		t.Fatalf("%s is missing %q; update the No-Go fitness check with the accepted Gate transition", path, expected)
	}
}

func selectorPackage(selector *ast.SelectorExpr) string {
	identifier, _ := selector.X.(*ast.Ident)
	if identifier == nil {
		return ""
	}
	return identifier.Name
}

func isNilIdentifier(expression ast.Expr) bool {
	identifier, _ := expression.(*ast.Ident)
	return identifier != nil && identifier.Name == "nil"
}
