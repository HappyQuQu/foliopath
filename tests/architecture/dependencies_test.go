package architecture_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

const modulePath = "github.com/HappyQuQu/foliopath"

type listedPackage struct {
	ImportPath string
	Imports    []string
}

func TestGoDependencyDirection(t *testing.T) {
	root := repositoryRoot(t)
	command := exec.Command("go", "list", "-json", "./...")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("list Go packages: %v", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for decoder.More() {
		var current listedPackage
		if err := decoder.Decode(&current); err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		if !strings.HasPrefix(current.ImportPath, modulePath+"/") {
			continue
		}
		for _, imported := range current.Imports {
			checkDependency(t, current.ImportPath, imported)
		}
	}
}

func TestNoGenericCatchAllGoPackages(t *testing.T) {
	root := repositoryRoot(t)
	command := exec.Command("go", "list", "-f", "{{.ImportPath}}", "./...")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("list Go packages: %v", err)
	}

	for _, importPath := range strings.Fields(string(output)) {
		if !strings.HasPrefix(importPath, modulePath+"/") {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(importPath, modulePath+"/"), "/")
		for _, part := range parts {
			if slices.Contains([]string{"utils", "common", "helpers", "base"}, part) {
				t.Errorf("generic catch-all package is forbidden: %s", importPath)
			}
		}
	}

	if info, err := os.Stat(filepath.Join(root, "pkg")); err == nil && info.IsDir() {
		t.Error("top-level pkg/ requires an accepted external-consumer decision")
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatalf("inspect top-level pkg/: %v", err)
	}
}

func TestAuthenticationHTTPBoundaryIsCentralizedAndFailClosed(t *testing.T) {
	root := repositoryRoot(t)
	apiRoot := filepath.Join(root, "internal", "api")
	routePath := filepath.Join(apiRoot, "auth_http.go")
	middlewarePath := filepath.Join(apiRoot, "auth_middleware.go")
	appRunPath := filepath.Join(root, "internal", "app", "run.go")

	routeSource, err := os.ReadFile(routePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{
		`GET /api/v1/auth/status`,
		`POST /api/v1/auth/setup`,
		`POST /api/v1/auth/login`,
		`GET /api/v1/auth/session`,
		`POST /api/v1/auth/logout`,
	} {
		if !strings.Contains(string(routeSource), operation) {
			t.Errorf("canonical authentication routes are missing %q", operation)
		}
	}

	middlewareSource, err := os.ReadFile(middlewarePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"requireAPIAuthentication",
		"anonymousAuthenticationOperation",
		"stateChangingMethod",
		"constantTimeTokenEqual",
		"requestHasSameOrigin",
	} {
		if !strings.Contains(string(middlewareSource), required) {
			t.Errorf("authentication middleware is missing %q", required)
		}
	}

	appRunSource, err := os.ReadFile(appRunPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(appRunSource), "Authentication: authentication") {
		t.Error("composition root does not wire the canonical authentication service into HTTP")
	}
	if strings.Contains(string(appRunSource), "denySystemStatus") {
		t.Error("composition root still uses the pre-authentication deny stub")
	}

	if err := filepath.WalkDir(apiRoot, func(
		path string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() ||
			filepath.Ext(path) != ".go" ||
			strings.HasSuffix(path, "_test.go") ||
			path == routePath {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(source), `HandleFunc("GET /api/v1/auth/`) ||
			strings.Contains(string(source), `HandleFunc("POST /api/v1/auth/`) {
			t.Errorf("authentication route registration is duplicated outside %s: %s", routePath, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("inspect authentication route ownership: %v", err)
	}
}

func TestWebUsesSingleGeneratedAPIClientBoundary(t *testing.T) {
	root := repositoryRoot(t)
	webSource := filepath.Join(root, "web", "src")
	clientPath := filepath.Join(webSource, "lib", "api", "client.ts")
	generatedPath := filepath.Join(webSource, "lib", "api", "generated", "schema.ts")

	for _, required := range []string{clientPath, generatedPath} {
		if info, err := os.Stat(required); err != nil {
			t.Fatalf("required web API boundary %s: %v", required, err)
		} else if !info.Mode().IsRegular() {
			t.Fatalf("required web API boundary is not a regular file: %s", required)
		}
	}

	if err := filepath.WalkDir(webSource, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (filepath.Ext(path) != ".ts" && filepath.Ext(path) != ".tsx") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(data)
		relative, err := filepath.Rel(webSource, path)
		if err != nil {
			return err
		}
		inAPIBoundary := strings.HasPrefix(relative, filepath.Join("lib", "api")+string(filepath.Separator))
		if strings.Contains(source, `"openapi-fetch"`) && path != clientPath {
			t.Errorf("%s imports openapi-fetch outside the canonical client", relative)
		}
		if strings.Contains(source, "generated/schema") && !inAPIBoundary {
			t.Errorf("%s imports generated OpenAPI types outside lib/api", relative)
		}
		if strings.Contains(source, "fetch(") && path != clientPath {
			t.Errorf("%s calls fetch outside the canonical client", relative)
		}
		return nil
	}); err != nil {
		t.Fatalf("inspect web API imports: %v", err)
	}

	generated, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "Do not edit this file directly.") {
		t.Error("generated OpenAPI types are missing the generated-file marker")
	}
}

func TestSQLiteQuerySourceAndGeneratedBoundary(t *testing.T) {
	root := repositoryRoot(t)
	sqliteRoot := filepath.Join(root, "internal", "store", "sqlite")
	configPath := filepath.Join(sqliteRoot, "sqlc.yaml")
	queryPath := filepath.Join(sqliteRoot, "queries", "libraries.sql")
	authQueryPath := filepath.Join(sqliteRoot, "queries", "auth.sql")
	generatedRoot := filepath.Join(sqliteRoot, "dbgen")

	for _, required := range []string{
		configPath,
		queryPath,
		authQueryPath,
		filepath.Join(generatedRoot, "db.go"),
		filepath.Join(generatedRoot, "auth.sql.go"),
		filepath.Join(generatedRoot, "libraries.sql.go"),
		filepath.Join(generatedRoot, "models.go"),
	} {
		if _, err := os.Stat(required); err != nil {
			t.Fatalf("required sqlc boundary %s: %v", required, err)
		}
	}

	authQueries, err := os.ReadFile(authQueryPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"-- name: IsAdministratorInitialized :one",
		"-- name: InsertAdministrator :one",
		"-- name: FindAdministratorCredential :one",
		"-- name: InsertSession :one",
		"-- name: FindSession :one",
		"-- name: TouchSession :execrows",
		"-- name: RevokeSession :execrows",
		"-- name: DeleteObsoleteSessions :execrows",
	} {
		if !strings.Contains(string(authQueries), required) {
			t.Errorf("canonical authentication queries are missing %q", required)
		}
	}

	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`version: "2"`,
		"schema: ../../../migrations",
		"queries: queries",
		"out: dbgen",
	} {
		if !strings.Contains(string(config), required) {
			t.Errorf("sqlc config is missing %q", required)
		}
	}

	queries, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"-- name: InsertLibrary :one",
		"-- name: RenameLibrary :execrows",
		"-- name: GetLibrary :one",
		"-- name: ListLibraries :many",
	} {
		if !strings.Contains(string(queries), required) {
			t.Errorf("canonical library queries are missing %q", required)
		}
	}

	if err := filepath.WalkDir(generatedRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			t.Errorf("generated SQL directory contains a non-Go file: %s", path)
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(source), "Code generated by sqlc. DO NOT EDIT.") {
			t.Errorf("generated SQL file is missing the generated marker: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatalf("inspect generated SQL package: %v", err)
	}

	adapterSource, err := os.ReadFile(filepath.Join(sqliteRoot, "library.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(adapterSource), `/store/sqlite/dbgen"`) {
		t.Error("library adapter does not consume the generated SQL package")
	}
	normalizedAdapterSource := strings.ToUpper(string(adapterSource))
	for _, forbidden := range []string{"SELECT ", "INSERT INTO libraries", "UPDATE libraries"} {
		if strings.Contains(normalizedAdapterSource, strings.ToUpper(forbidden)) {
			t.Errorf("library adapter duplicates canonical SQL containing %q", forbidden)
		}
	}

	authAdapterSource, err := os.ReadFile(filepath.Join(sqliteRoot, "auth.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(authAdapterSource), `/store/sqlite/dbgen"`) {
		t.Error("authentication adapter does not consume the generated SQL package")
	}
	normalizedAuthAdapterSource := strings.ToUpper(string(authAdapterSource))
	for _, forbidden := range []string{"SELECT ", "INSERT INTO users", "UPDATE users"} {
		if strings.Contains(normalizedAuthAdapterSource, strings.ToUpper(forbidden)) {
			t.Errorf("authentication adapter duplicates canonical SQL containing %q", forbidden)
		}
	}
}

func checkDependency(t *testing.T, source, imported string) {
	t.Helper()
	if !strings.HasPrefix(imported, modulePath+"/") {
		return
	}

	sourceArea := internalArea(source)
	importedArea := internalArea(imported)

	switch sourceArea {
	case "api":
		if slices.Contains([]string{"app", "files", "pathpolicy", "store", "webassets"}, importedArea) {
			t.Errorf("HTTP adapter %s must not import concrete/runtime area %s", source, imported)
		}
	case "pathpolicy":
		t.Errorf("pure path policy %s must not import repository package %s", source, imported)
	case "auth", "library", "catalog", "scanner", "thumbnail", "media", "jobs":
		if slices.Contains([]string{"api", "app", "files", "store", "webassets"}, importedArea) {
			t.Errorf("capability/policy package %s must not import outer area %s", source, imported)
		}
	case "files", "store":
		if slices.Contains([]string{"api", "app", "webassets"}, importedArea) {
			t.Errorf("adapter %s must not import delivery/composition area %s", source, imported)
		}
		if (sourceArea == "files" && importedArea == "store") ||
			(sourceArea == "store" && importedArea == "files") {
			t.Errorf("adapter %s must not import sibling adapter %s", source, imported)
		}
	case "webassets":
		t.Errorf("embedded web assets %s must not import repository package %s", source, imported)
	}

	if strings.HasPrefix(source, modulePath+"/cmd/") && imported != modulePath+"/internal/app" {
		t.Errorf("process entry point %s may import only internal/app, imported %s", source, imported)
	}
}

func internalArea(importPath string) string {
	prefix := modulePath + "/internal/"
	if !strings.HasPrefix(importPath, prefix) {
		return ""
	}
	remainder := strings.TrimPrefix(importPath, prefix)
	return strings.SplitN(remainder, "/", 2)[0]
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve architecture test location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
