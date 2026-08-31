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
		"**提议（",
	)
	assertDocumentContains(t,
		filepath.Join(root, "docs", "gates", "POST-MVP-5", "int-s2a-backend-evidence-ready.md"),
		"当前判断：**No-Go**",
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
				t.Errorf("production reviewed AI catalog must remain empty while INT-S2A is No-Go")
				return true
			}
			emptyCatalogCalls++
		case *ast.CompositeLit:
			selector, ok := current.Type.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "RouteDependencies" || selectorPackage(selector) != "api" {
				return true
			}
			routeCompositions++
			hasSemantic, hasSemanticSearch := false, false
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
				}
			}
			if !hasSemantic {
				t.Errorf("approved semantic settings/backfill/clear routes must remain composed")
			}
			if hasSemanticSearch {
				t.Errorf("semantic search route must remain absent while INT-S2A is No-Go")
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
		"当前判断：**No-Go / External Admission Pending**",
	)
	if _, err := os.Stat(filepath.Join(root, "internal", "face")); err == nil {
		t.Fatal("internal/face must not exist before the S2C privacy admission gate passes")
	} else if !os.IsNotExist(err) {
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
