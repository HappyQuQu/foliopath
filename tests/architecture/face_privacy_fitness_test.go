package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestFacePrivacyProjectionRemainsClosed(t *testing.T) {
	root := repositoryRoot(t)
	forbidden := map[string]struct{}{
		"embedding": {}, "embeddings": {}, "vector": {}, "vectors": {},
		"crop": {}, "cropPath": {}, "path": {}, "sourceFingerprint": {},
		"detection": {}, "quality": {}, "score": {}, "similarity": {},
		"modelOutput": {}, "runtimeError": {},
	}

	path := filepath.Join(root, "internal", "api", "face_http.go")
	parsed := parsePrivacyFile(t, path)
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.Field:
			if current.Tag == nil {
				return true
			}
			raw, err := strconv.Unquote(current.Tag.Value)
			if err != nil {
				t.Fatalf("decode struct tag in %s: %v", path, err)
			}
			name := strings.Split(reflect.StructTag(raw).Get("json"), ",")[0]
			if _, blocked := forbidden[name]; blocked {
				t.Errorf("face HTTP JSON exposes forbidden field %q", name)
			}
		case *ast.BasicLit:
			if current.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(current.Value)
			if err == nil {
				if _, blocked := forbidden[value]; blocked {
					t.Errorf("face HTTP constructs forbidden literal key %q", value)
				}
			}
		}
		return true
	})

	assertPackagesDoNotLogFaceData(t, root)
	assertDiagnosticsDoNotImportFace(t, root)
}

func assertPackagesDoNotLogFaceData(t *testing.T, root string) {
	t.Helper()
	patterns := []string{
		filepath.Join(root, "internal", "face", "*.go"),
		filepath.Join(root, "internal", "inference", "faceonnx", "*.go"),
		filepath.Join(root, "internal", "inference", "onnx", "*.go"),
		filepath.Join(root, "internal", "store", "sqlite", "face*.go"),
	}
	for _, pattern := range patterns {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			parsed := parsePrivacyFile(t, path)
			for _, imported := range parsed.Imports {
				name, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				if name == "log" || name == "log/slog" {
					t.Errorf("face persistence/capability package must not log sensitive state directly: %s imports %s", path, name)
				}
			}
		}
	}
}

func assertDiagnosticsDoNotImportFace(t *testing.T, root string) {
	t.Helper()
	paths := []string{
		filepath.Join(root, "internal", "api", "media_diagnostics_http.go"),
		filepath.Join(root, "internal", "systemlog", "systemlog.go"),
	}
	for _, path := range paths {
		parsed := parsePrivacyFile(t, path)
		for _, imported := range parsed.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if name == "github.com/HappyQuQu/foliopath/internal/face" {
				t.Errorf("existing diagnostics/log boundary must not consume face state: %s", path)
			}
		}
	}
}

func parsePrivacyFile(t *testing.T, path string) *ast.File {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), path, content, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return parsed
}
