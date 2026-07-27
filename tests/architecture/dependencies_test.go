package architecture_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

func TestLibraryPathFilesystemBoundaryIsCentralized(t *testing.T) {
	root := repositoryRoot(t)
	apiPath := filepath.Join(root, "internal", "api", "library_paths_http.go")
	capabilityPath := filepath.Join(root, "internal", "library", "paths.go")
	adapterPath := filepath.Join(root, "internal", "files", "enumerate.go")
	appPath := filepath.Join(root, "internal", "app", "run.go")
	mediaRootPath := filepath.Join(root, "internal", "app", "media_root.go")

	for _, required := range []struct {
		path    string
		content []string
	}{
		{
			path: apiPath,
			content: []string{
				`GET /api/v1/library-paths`,
				"LibraryPathService",
			},
		},
		{
			path: capabilityPath,
			content: []string{
				"type DirectorySource interface",
				"func (service *PathService) ListPaths",
			},
		},
		{
			path: adapterPath,
			content: []string{
				"func (source *DirectorySource) EnumerateDirectories",
				"source.root.OpenDir(parent)",
			},
		},
		{
			path: mediaRootPath,
			content: []string{
				"files.NewDirectorySource(root)",
				"func (service *mediaRootService) EnumerateDirectories",
			},
		},
		{
			path: appPath,
			content: []string{
				"newMediaRootService(configuration.mediaRoot)",
				"LibraryPaths:   libraryPaths",
			},
		},
	} {
		source, err := os.ReadFile(required.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.content {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing canonical boundary %q", required.path, content)
			}
		}
	}

	for _, forbidden := range []string{
		`"github.com/HappyQuQu/foliopath/internal/files"`,
		`"os"`,
		`"path/filepath"`,
	} {
		for _, sourcePath := range []string{apiPath, capabilityPath} {
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(source), forbidden) {
				t.Errorf("%s bypasses the directory-source port with %s", sourcePath, forbidden)
			}
		}
	}
}

func TestLibraryRemovalHasNoOriginalMediaCapability(t *testing.T) {
	root := repositoryRoot(t)
	workerPath := filepath.Join(root, "internal", "library", "removal_worker.go")
	cleanerPath := filepath.Join(root, "internal", "app", "library_removal.go")
	compositionPath := filepath.Join(root, "internal", "app", "run.go")

	workerSource, err := os.ReadFile(workerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		`"os"`,
		`"path/filepath"`,
		`"github.com/HappyQuQu/foliopath/internal/files"`,
	} {
		if strings.Contains(string(workerSource), forbidden) {
			t.Errorf("removal worker gained original-media filesystem capability %s", forbidden)
		}
	}
	for _, required := range []string{
		"type RemovalRepository interface",
		"type DerivedCacheCleaner interface",
		"CleanupLibraryRemovalBatch",
		"RemoveLibraryCache",
	} {
		if !strings.Contains(string(workerSource), required) {
			t.Errorf("removal worker is missing narrow derived-state port %q", required)
		}
	}

	cleanerSource, err := os.ReadFile(cleanerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"mediaRoot", `"/library"`, "internal/files"} {
		if strings.Contains(string(cleanerSource), forbidden) {
			t.Errorf("cache cleaner references original-media boundary %q", forbidden)
		}
	}
	for _, required := range []string{`"cache"`, `"libraries"`, `"lib_"`} {
		if !strings.Contains(string(cleanerSource), required) {
			t.Errorf("cache cleaner is missing derived-cache segment %s", required)
		}
	}

	compositionSource, err := os.ReadFile(compositionPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(
		string(compositionSource),
		"libraryCacheCleaner{dataRoot: configuration.dataRoot}",
	) {
		t.Error("composition root does not constrain removal cache cleanup to application data")
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
		"-- name: RenameLibrary :one",
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

func TestLibraryDomainRulesHaveCanonicalOwners(t *testing.T) {
	root := repositoryRoot(t)
	capabilityPath := filepath.Join(root, "internal", "library", "library.go")
	adapterPath := filepath.Join(root, "internal", "store", "sqlite", "library.go")
	queryPath := filepath.Join(root, "internal", "store", "sqlite", "queries", "libraries.sql")
	migrationPath := filepath.Join(root, "migrations", "00001_initial.sql")

	for _, required := range []struct {
		path     string
		contents []string
	}{
		{
			path: capabilityPath,
			contents: []string{
				"func NormalizeName(",
				"func NormalizeRoot(",
				"func RootsOverlap(",
				"cases.Fold().String(norm.NFKC.String(display))",
			},
		},
		{
			path: adapterPath,
			contents: []string{
				"func (s *Store) CreateLibrary(",
				"s.withWriteTx(ctx",
				"queries.FindOverlappingLibraryID(ctx, root)",
				"func libraryFromDatabase(",
			},
		},
		{
			path: queryPath,
			contents: []string{
				"-- name: FindOverlappingLibraryID :one",
				"-- name: RenameLibrary :one",
				"RETURNING",
			},
		},
		{
			path: migrationPath,
			contents: []string{
				"name_key           TEXT NOT NULL UNIQUE",
				"root_rel_path      TEXT NOT NULL UNIQUE",
				"CREATE TRIGGER libraries_root_is_immutable",
			},
		},
	} {
		source, err := os.ReadFile(required.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing canonical library rule %q", required.path, content)
			}
		}
	}
}

func TestScanWorkerUsesOneDurableBoundedQueue(t *testing.T) {
	root := repositoryRoot(t)
	workerPath := filepath.Join(root, "internal", "jobs", "worker.go")
	processorPath := filepath.Join(root, "internal", "scanner", "claimed_processor.go")
	signalPath := filepath.Join(root, "internal", "jobs", "signal.go")
	admissionPath := filepath.Join(root, "internal", "scanner", "admission.go")
	storePath := filepath.Join(root, "internal", "store", "sqlite", "scan_worker.go")
	queryPath := filepath.Join(root, "internal", "store", "sqlite", "queries", "scans.sql")
	compositionPath := filepath.Join(root, "internal", "app", "run.go")

	for _, required := range []struct {
		path     string
		contents []string
	}{
		{
			path: workerPath,
			contents: []string{
				"DefaultWorkerCount       = 2",
				"DefaultHeartbeatInterval = 15 * time.Second",
				"DefaultLeaseDuration     = 120 * time.Second",
				"pool.queue.RecoverExpired(ctx)",
				"func (pool *WorkerPool[T]) runRecovery(",
				"pool.queue.Claim(ctx, pool.leaseDuration)",
			},
		},
		{
			path: admissionPath,
			contents: []string{
				"func (service *AdmissionService) RequestStartup(",
				"ListStartupLibraryIDs(",
				"TriggerStartup",
				"ErrAdmissionCapacity",
			},
		},
		{
			path: storePath,
			contents: []string{
				"func (s *Store) ListStartupLibraryIDs(",
				"ORDER BY libraries.id",
				"library_removals.status IN ('queued', 'running')",
			},
		},
		{
			path: signalPath,
			contents: []string{
				"make(chan struct{}, 1)",
				"Signals never carry work",
			},
		},
		{
			path: queryPath,
			contents: []string{
				"-- name: ClaimNextQueuedScan :one",
				"ORDER BY available_at_ms, created_at_ms, id",
				"attempt_count < 3",
				"-- name: RecoverNextExpiredScan :one",
			},
		},
		{
			path: compositionPath,
			contents: []string{
				"jobs.NewWorkerPool(",
				"scanner.NewClaimedProcessor(",
				"newScanWorkerComponent(scanWorker, scanAdmission, scanScheduler)",
				"scanComponent,",
			},
		},
		{
			path: processorPath,
			contents: []string{
				"RunClaimedFullScan(ctx, run, processor.walker)",
				"never allocates or claims queue work itself",
			},
		},
	} {
		source, err := os.ReadFile(required.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing scan queue boundary %q", required.path, content)
			}
		}
	}

	processorSource, err := os.ReadFile(processorPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		`"os"`,
		`"path/filepath"`,
		`"/internal/files"`,
		`"/internal/store/sqlite"`,
		"ClaimNext",
		"RecoverExpired",
	} {
		if strings.Contains(string(processorSource), forbidden) {
			t.Errorf("scan capability bypasses a port or creates an in-memory work queue: %q", forbidden)
		}
	}
}

func TestScanCapacityGateUsesCanonicalProductionBounds(t *testing.T) {
	root := repositoryRoot(t)
	for _, required := range []struct {
		path     string
		contents []string
	}{
		{
			path: filepath.Join(root, "internal", "scanner", "admission.go"),
			contents: []string{
				"MaxActiveFullScans     = 256",
			},
		},
		{
			path: filepath.Join(root, "internal", "scanner", "scanner.go"),
			contents: []string{
				"const DefaultBatchSize = 256",
				"config.BatchSize = DefaultBatchSize",
			},
		},
		{
			path: filepath.Join(root, "internal", "store", "sqlite", "scanner.go"),
			contents: []string{
				"active >= scanner.MaxActiveFullScans",
			},
		},
		{
			path: filepath.Join(root, "internal", "store", "sqlite", "library_lifecycle.go"),
			contents: []string{
				"activeCount >= scanner.MaxActiveFullScans",
			},
		},
		{
			path: filepath.Join(root, ".github", "workflows", "ci.yml"),
			contents: []string{
				"scan-capacity:",
				"--cpus=2",
				"--memory=4g",
				"GOMAXPROCS=2",
				"FOLIOPATH_CAPACITY_ENFORCE_BUDGET=1",
				"Test(CapacityBaseline|DirectoryRollupDeepChainBaseline)",
			},
		},
	} {
		source, err := os.ReadFile(required.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing scan capacity rule %q", required.path, content)
			}
		}
	}
}

func TestScanHTTPQueriesUseCapabilityAndGeneratedClientBoundary(t *testing.T) {
	root := repositoryRoot(t)
	for _, required := range []struct {
		path     string
		contents []string
	}{
		{
			path: filepath.Join(root, "internal", "scanner", "queries.go"),
			contents: []string{
				"type QueryService struct",
				"MaxScanPageSize     = 200",
				"ErrInvalidScanCursor",
				"ErrScanAlreadyFinished",
			},
		},
		{
			path: filepath.Join(root, "internal", "api", "scans_http.go"),
			contents: []string{
				`"GET /api/v1/libraries/{libraryId}/scans"`,
				`"GET /api/v1/scans/{scanId}"`,
				`"POST /api/v1/scans/{scanId}/cancel"`,
			},
		},
		{
			path: filepath.Join(root, "internal", "app", "run.go"),
			contents: []string{
				"scanner.NewQueryService(database, nil)",
				"Scans:          scanQueries",
				"scanner.NewScheduler(",
				"Settings:       settingsService",
			},
		},
		{
			path: filepath.Join(root, "internal", "store", "sqlite", "scan_queries.go"),
			contents: []string{
				"ListLibraryScanContractRuns",
				"ListScanIssues",
				"RequestRunningScanCancellation",
				"CancelQueuedScan",
			},
		},
		{
			path: filepath.Join(root, "internal", "api", "settings_http.go"),
			contents: []string{
				`"GET /api/v1/settings"`,
				`"PATCH /api/v1/settings"`,
				`"settings-r`,
			},
		},
		{
			path: filepath.Join(root, "internal", "scanner", "scheduler.go"),
			contents: []string{
				"TriggerScheduled",
				"ListDueLibraryIDs",
				"scheduledLibraryPageSize = 64",
			},
		},
	} {
		source, err := os.ReadFile(required.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing scan HTTP boundary %q", required.path, content)
			}
		}
	}
}

func TestDirectoryIndexAndCountPolicyHasCanonicalOwners(t *testing.T) {
	root := repositoryRoot(t)
	scannerPath := filepath.Join(root, "internal", "scanner", "scanner.go")
	storePath := filepath.Join(root, "internal", "store", "sqlite", "scanner.go")
	catalogReadPath := filepath.Join(root, "internal", "store", "sqlite", "catalog.go")

	scannerSource, err := os.ReadFile(scannerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"appendEntry(CatalogEntry{Kind: CatalogEntryDirectory})",
		"if entry.IsDirectory",
		"IsSystemDirectory(path.Base(relativePath))",
		"skipped.Directories++",
		"skipped.Files++",
	} {
		if !strings.Contains(string(scannerSource), required) {
			t.Errorf("scanner directory policy is missing %q", required)
		}
	}

	storeSource, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"func recalculateDirectoryCountsTx(",
		"COALESCE(direct.asset_count, 0)",
		"sum(recursive_asset_count)",
		"updated != totalDirectories",
	} {
		if !strings.Contains(string(storeSource), required) {
			t.Errorf("SQLite directory count owner is missing %q", required)
		}
	}

	internalRoot := filepath.Join(root, "internal")
	if err := filepath.WalkDir(internalRoot, func(
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
			path == storePath {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if path == catalogReadPath {
			if strings.Contains(string(source), "UPDATE directories") ||
				strings.Contains(string(source), "SET direct_asset_count") ||
				strings.Contains(string(source), "SET recursive_asset_count") {
				t.Errorf("catalog read adapter mutates canonical directory counts: %s", path)
			}
			return nil
		}
		if strings.Contains(string(source), "direct_asset_count") ||
			strings.Contains(string(source), "recursive_asset_count") {
			t.Errorf("directory count SQL is duplicated outside its owner: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatalf("inspect directory count ownership: %v", err)
	}
}

func TestMediaCandidateFingerprintAndConvergenceHaveCanonicalOwners(t *testing.T) {
	root := repositoryRoot(t)
	formatPath := filepath.Join(root, "internal", "media", "formats.go")
	fingerprintPath := filepath.Join(root, "internal", "media", "fingerprint.go")
	scannerPath := filepath.Join(root, "internal", "scanner", "formats.go")
	storePath := filepath.Join(root, "internal", "store", "sqlite", "scanner.go")
	migrationPath := filepath.Join(
		root,
		"migrations",
		"00006_asset_source_fingerprint.sql",
	)

	for _, required := range []struct {
		path     string
		contents []string
	}{
		{
			path: formatPath,
			contents: []string{
				"var supportedExtensions",
				`".jpg"`,
				`".mkv"`,
			},
		},
		{
			path: fingerprintPath,
			contents: []string{
				`const sourceFingerprintPrefix = "v1:"`,
				"func NewSourceFingerprint(sizeBytes, mtimeNS int64)",
				"strconv.FormatInt(sizeBytes, 10)",
				"strconv.FormatInt(mtimeNS, 10)",
			},
		},
		{
			path: scannerPath,
			contents: []string{
				"media.ClassifyPath(relativePath)",
			},
		},
		{
			path: storePath,
			contents: []string{
				"media.NewSourceFingerprint(entry.SizeBytes, entry.MTimeNS)",
				"source_fingerprint = excluded.source_fingerprint",
				"processed_assets = processed_assets + ?",
				"DELETE FROM assets WHERE library_id = ? AND last_seen_generation < ?",
			},
		},
		{
			path: migrationPath,
			contents: []string{
				"ADD COLUMN source_fingerprint TEXT NOT NULL",
				"'v1:' || size_bytes || ':' || mtime_ns",
			},
		},
	} {
		source, err := os.ReadFile(required.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing S2-104 policy %q", required.path, content)
			}
		}
	}

	internalRoot := filepath.Join(root, "internal")
	if err := filepath.WalkDir(internalRoot, func(
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
			path == fingerprintPath {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(source), `"v1:"`) {
			t.Errorf("source fingerprint encoding is duplicated outside its owner: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatalf("inspect source fingerprint ownership: %v", err)
	}
}

func TestMediaProcessingAndThumbnailDerivationHaveCanonicalOwners(t *testing.T) {
	root := repositoryRoot(t)
	processingPath := filepath.Join(root, "internal", "media", "processing.go")
	imagePath := filepath.Join(
		root, "internal", "media", "imagevips", "processor_libvips.go",
	)
	videoPath := filepath.Join(
		root, "internal", "media", "videoffmpeg", "processor.go",
	)
	derivationPath := filepath.Join(root, "internal", "thumbnail", "derivation.go")
	cachePath := filepath.Join(
		root, "internal", "thumbnail", "cachefs", "cache.go",
	)
	migrationPath := filepath.Join(root, "migrations", "00008_media_processing.sql")
	for _, required := range []struct {
		path     string
		contents []string
	}{
		{
			path: processingPath,
			contents: []string{
				"type ProcessingResult struct",
				"func ProcessingCode(err error)",
				"MaxImageSourceBytes",
				"MaxVideoSourceBytes",
				"MaxDecodedPixels",
				"MaxMediaDimension",
				"func ValidateSourceSize(",
				"func ValidateDimensions(",
			},
		},
		{
			path: imagePath,
			contents: []string{
				`"github.com/davidbyttow/govips/v2/vips"`,
				"vips.LoadImageFromBuffer",
				"media.ValidateDimensions",
				"image.ExportWebp",
			},
		},
		{
			path: videoPath,
			contents: []string{
				"exec.CommandContext",
				"configureCommandCancellation(command)",
				"command.ExtraFiles",
				`"-threads", "1"`,
				`"-filter_threads", "1"`,
				`"-show_streams"`,
				`"-frames:v", "1"`,
			},
		},
		{
			path: derivationPath,
			contents: []string{
				"GridTransformVersion",
				"func (value Derivation) CacheRelativePath()",
			},
		},
		{
			path: cachePath,
			contents: []string{
				"os.CreateTemp",
				"temp.Sync()",
				"os.Rename",
			},
		},
		{
			path: migrationPath,
			contents: []string{
				"CREATE TABLE thumbnails",
				"probe_status",
				"source_fingerprint",
				"transform_version",
			},
		},
	} {
		source, err := os.ReadFile(required.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing media boundary %q", required.path, content)
			}
		}
	}
	videoSource, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"sh -c", "bash -c", "CombinedOutput"} {
		if strings.Contains(string(videoSource), forbidden) {
			t.Errorf("FFmpeg adapter contains forbidden execution pattern %q", forbidden)
		}
	}
}

func TestMediaJobsAndCachePolicyHaveCanonicalOwners(t *testing.T) {
	root := repositoryRoot(t)
	for _, required := range []struct {
		relative string
		contents []string
	}{
		{
			relative: "migrations/00009_media_jobs.sql",
			contents: []string{
				"CREATE TABLE media_jobs",
				"CREATE TABLE cache_deletions",
				"lease_expires_at_ms",
				"attempt_count",
				"transform_version",
			},
		},
		{
			relative: "internal/thumbnail/jobs.go",
			contents: []string{
				"MediaWorkerCount = 2",
				"MaxJobAttempts   = 3",
				"type ClaimedProcessor struct",
			},
		},
		{
			relative: "internal/thumbnail/capacity.go",
			contents: []string{
				"CacheHighWaterPercent",
				"CacheLowWaterPercent",
				"CacheSafeFreeBytes",
				"ListLRUCacheEntries",
			},
		},
		{
			relative: "internal/store/sqlite/scanner.go",
			contents: []string{
				"INSERT INTO media_jobs",
				"INSERT OR IGNORE INTO cache_deletions",
				"invalidate stale thumbnail",
			},
		},
		{
			relative: "internal/app/run.go",
			contents: []string{
				"jobs.NewWorkerPool(",
				"newImageRuntimeComponent(imagevips.NewRuntime())",
				"newMediaWorkerComponent(",
				"thumbnail.NewCacheManager(",
			},
		},
		{
			relative: "internal/media/imagevips/runtime_libvips.go",
			contents: []string{
				"ConcurrencyLevel: NativeConcurrency",
				"MaxCacheFiles:    NativeCacheFiles",
				"MaxCacheMem:      NativeCacheMemory",
				"MaxCacheSize:     NativeCacheEntries",
			},
		},
		{
			relative: ".github/workflows/ci.yml",
			contents: []string{
				"Run the real full-filesystem cache fixture",
				"FOLIOPATH_FULL_CACHE_ROOT",
				"-tags mediafull",
			},
		},
	} {
		path := filepath.Join(root, filepath.FromSlash(required.relative))
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing media job/cache boundary %q",
					required.relative, content)
			}
		}
	}
}

func TestThumbnailHTTPUsesCanonicalDeliveryBoundary(t *testing.T) {
	root := repositoryRoot(t)
	for _, required := range []struct {
		relative string
		contents []string
	}{
		{
			relative: "internal/thumbnail/delivery.go",
			contents: []string{
				"type DeliveryRepository interface",
				"type CacheReader interface",
				"RequeueMissingThumbnail",
				"TouchThumbnail",
			},
		},
		{
			relative: "internal/api/thumbnail_http.go",
			contents: []string{
				`GET /api/v1/assets/{assetId}/thumbnail`,
				"type ThumbnailService interface",
				"writeReadyThumbnail",
			},
		},
		{
			relative: "internal/app/run.go",
			contents: []string{
				"thumbnail.NewDeliveryService(",
				"Thumbnails:     thumbnailDelivery",
			},
		},
	} {
		path := filepath.Join(root, filepath.FromSlash(required.relative))
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing thumbnail delivery boundary %q",
					required.relative, content)
			}
		}
	}

	apiPath := filepath.Join(root, "internal", "api", "thumbnail_http.go")
	apiSource, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		`"os"`,
		`"path/filepath"`,
		`"github.com/HappyQuQu/foliopath/internal/store/sqlite"`,
		`"github.com/HappyQuQu/foliopath/internal/thumbnail/cachefs"`,
	} {
		if strings.Contains(string(apiSource), forbidden) {
			t.Errorf("thumbnail HTTP bypasses canonical delivery service with %s", forbidden)
		}
	}
}

func TestOpaqueCursorCodecHasOneCanonicalOwner(t *testing.T) {
	root := repositoryRoot(t)
	codecPath := filepath.Join(root, "internal", "cursor", "codec.go")
	codecSource, err := os.ReadFile(codecPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"aes.NewCipher", "cipher.NewGCM", ".Seal(", ".Open("} {
		if !strings.Contains(string(codecSource), required) {
			t.Errorf("canonical cursor codec is missing %q", required)
		}
	}

	for _, relative := range []string{
		"internal/library/lifecycle.go",
		"internal/library/paths.go",
		"internal/scanner/queries.go",
	} {
		path := filepath.Join(root, relative)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(source), "/internal/cursor") {
			t.Errorf("%s does not use the canonical cursor codec", relative)
		}
		for _, forbidden := range []string{
			`"crypto/aes"`,
			"cipher.NewGCM",
			".Seal(",
			".Open(",
		} {
			if strings.Contains(string(source), forbidden) {
				t.Errorf("%s duplicates cursor mechanism %q", relative, forbidden)
			}
		}
	}
}

func TestSearchOwnershipAndFTSBoundaryAreCentralized(t *testing.T) {
	root := repositoryRoot(t)
	capabilityPath := filepath.Join(root, "internal", "catalog", "catalog.go")
	apiPath := filepath.Join(root, "internal", "api", "catalog_http.go")
	adapterPath := filepath.Join(root, "internal", "store", "sqlite", "catalog.go")
	scannerPath := filepath.Join(root, "internal", "store", "sqlite", "scanner.go")
	migrationPath := filepath.Join(root, "migrations", "00010_catalog_search.sql")

	for _, required := range []struct {
		path     string
		contents []string
	}{
		{
			path: capabilityPath,
			contents: []string{
				"func NormalizeSearchTerms",
				"func SearchTextKey",
				"func (service *Service) SearchAssets",
				"searchProfileV1",
			},
		},
		{
			path: apiPath,
			contents: []string{
				`GET /api/v1/assets`,
				"parseGlobalSearchQuery",
				"parseUTCInstant",
			},
		},
		{
			path: adapterPath,
			contents: []string{
				"asset_search MATCH ?",
				"instr(a.search_name_key, ?)",
				"ResolveGlobalCatalogRevision",
			},
		},
		{
			path: scannerPath,
			contents: []string{
				"catalog.SearchTextKey(entry.Name)",
				"catalog.SearchTextKey(entry.RelativePath)",
			},
		},
		{
			path: migrationPath,
			contents: []string{
				"CREATE VIRTUAL TABLE asset_search USING fts5",
				"tokenize='trigram case_sensitive 1'",
				"CREATE TABLE catalog_search_state",
				"catalog_revision_generation_publish",
			},
		},
	} {
		source, err := os.ReadFile(required.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range required.contents {
			if !strings.Contains(string(source), content) {
				t.Errorf("%s is missing canonical search boundary %q", required.path, content)
			}
		}
	}

	apiSource, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{" MATCH ", "cases.Fold", "norm.NFKC", "/internal/files"} {
		if strings.Contains(string(apiSource), forbidden) {
			t.Errorf("catalog HTTP adapter owns forbidden search behavior %q", forbidden)
		}
	}

	for _, symbol := range []string{"func NormalizeSearchTerms", "func SearchTextKey"} {
		owners := 0
		if err := filepath.WalkDir(filepath.Join(root, "internal"), func(
			path string,
			entry fs.DirEntry,
			walkErr error,
		) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(source), symbol) {
				owners++
			}
			return nil
		}); err != nil {
			t.Fatalf("inspect search ownership: %v", err)
		}
		if owners != 1 {
			t.Errorf("%s owner count = %d, want 1", symbol, owners)
		}
	}
}

func TestSQLiteProductionQueriesDoNotUseOffsetPagination(t *testing.T) {
	root := repositoryRoot(t)
	storeRoot := filepath.Join(root, "internal", "store", "sqlite")
	offsetPattern := regexp.MustCompile(`(?i)\bOFFSET\b`)
	if err := filepath.WalkDir(storeRoot, func(
		path string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() ||
			strings.HasSuffix(path, "_test.go") ||
			(filepath.Ext(path) != ".go" && filepath.Ext(path) != ".sql") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if offsetPattern.Match(source) {
			t.Errorf("production SQLite query uses forbidden OFFSET pagination: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatalf("inspect SQLite pagination: %v", err)
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
	case "pathpolicy", "cursor":
		t.Errorf("pure policy/mechanism package %s must not import repository package %s", source, imported)
	case "auth", "settings", "library", "catalog", "scanner", "thumbnail", "media", "jobs":
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
