package contract_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
	"github.com/dlclark/regexp2"
	"github.com/getkin/kin-openapi/openapi3"
)

type operation struct {
	path   string
	method string
	id     string
	block  string
}

var (
	pathLine      = regexp.MustCompile(`^  (/[^:]+):$`)
	methodLine    = regexp.MustCompile(`^    (get|post|patch|delete|head):$`)
	operationID   = regexp.MustCompile(`(?m)^      operationId: ([A-Za-z][A-Za-z0-9]+)$`)
	componentRef  = regexp.MustCompile(`\$ref: '#/components/([A-Za-z]+)/([A-Za-z0-9]+)'`)
	componentKind = regexp.MustCompile(`^  (securitySchemes|parameters|headers|responses|schemas):$`)
	componentName = regexp.MustCompile(`^    ([A-Za-z][A-Za-z0-9]+):$`)
)

func TestOpenAPIYAMLParsesAndValidatesOffline(t *testing.T) {
	t.Parallel()

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	document, err := loader.LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatalf("parse and resolve OpenAPI YAML: %v", err)
	}
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("openapi = %q, want 3.0.3", document.OpenAPI)
	}
	if document.Info == nil {
		t.Error("info is missing")
	}
	if document.Paths == nil {
		t.Error("paths is missing")
	}
	if document.Components == nil {
		t.Error("components is missing")
	}
	if err := document.Validate(
		context.Background(),
		openapi3.SetRegexCompiler(compileECMAScriptPattern),
		openapi3.EnableMultiError(),
	); err != nil {
		t.Fatalf("validate OpenAPI structure and references: %v", err)
	}
}

func TestOpenAPIDeclaresRepositoryLicenseAndFrozenBaseline(t *testing.T) {
	t.Parallel()

	text := readOpenAPI(t)
	for _, required := range []string{
		"name: AGPL-3.0-or-later",
		"status: authoritative",
		"targetVersion: MVP-2026-07-23",
		"scopeRevision: 1",
		"baselineEvent: BASELINE-2026-07-23",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("OpenAPI metadata is missing %q", required)
		}
	}
}

func TestFrozenOpenAPIRevisionDigest(t *testing.T) {
	t.Parallel()

	fields := strings.Fields(readRepositoryFile(t, "api", "openapi.sha256"))
	if len(fields) != 2 || fields[1] != "openapi.yaml" {
		t.Fatalf("api/openapi.sha256 must contain '<sha256>  openapi.yaml'")
	}
	actual := fmt.Sprintf("%x", sha256.Sum256([]byte(readOpenAPI(t))))
	if actual != fields[0] {
		t.Fatalf(
			"OpenAPI revision lock differs: got %s, want %s; review compatibility before updating api/openapi.sha256",
			actual,
			fields[0],
		)
	}
}

func TestOpenAPIHasExactMVPResourceOperations(t *testing.T) {
	t.Parallel()

	operations := readOperations(t)
	want := []string{
		"GET /health/live",
		"GET /health/ready",
		"GET /api/v1/auth/status",
		"POST /api/v1/auth/setup",
		"POST /api/v1/auth/login",
		"GET /api/v1/auth/session",
		"POST /api/v1/auth/logout",
		"GET /api/v1/status",
		"GET /api/v1/settings",
		"PATCH /api/v1/settings",
		"GET /api/v1/library-paths",
		"GET /api/v1/libraries",
		"POST /api/v1/libraries",
		"GET /api/v1/libraries/{libraryId}",
		"PATCH /api/v1/libraries/{libraryId}",
		"DELETE /api/v1/libraries/{libraryId}",
		"GET /api/v1/library-removals/{removalId}",
		"GET /api/v1/libraries/{libraryId}/scans",
		"POST /api/v1/libraries/{libraryId}/scans",
		"GET /api/v1/scans/{scanId}",
		"POST /api/v1/scans/{scanId}/cancel",
		"GET /api/v1/libraries/{libraryId}/directories",
		"GET /api/v1/directories/{directoryId}",
		"GET /api/v1/libraries/{libraryId}/assets",
		"GET /api/v1/assets",
		"GET /api/v1/assets/{assetId}",
		"GET /api/v1/assets/{assetId}/thumbnail",
		"GET /api/v1/assets/{assetId}/content",
		"HEAD /api/v1/assets/{assetId}/content",
	}

	got := make([]string, 0, len(operations))
	for _, operation := range operations {
		got = append(got, operation.method+" "+operation.path)
	}
	sort.Strings(got)
	sort.Strings(want)
	if diff := sliceDiff(want, got); diff != "" {
		t.Fatalf("MVP operation set differs (-want +got):\n%s", diff)
	}
}

func TestEveryOperationHasUniqueIDAndRequirementTrace(t *testing.T) {
	t.Parallel()

	seen := make(map[string]string)
	for _, operation := range readOperations(t) {
		if operation.id == "" {
			t.Errorf("%s %s has no operationId", operation.method, operation.path)
			continue
		}
		key := operation.method + " " + operation.path
		if previous, ok := seen[operation.id]; ok {
			t.Errorf("operationId %q is shared by %s and %s", operation.id, previous, key)
		}
		seen[operation.id] = key
		if !strings.Contains(operation.block, "\n      x-requirements: [") {
			t.Errorf("%s has no frozen requirement trace", key)
		}
	}
}

func TestComponentReferencesResolve(t *testing.T) {
	t.Parallel()

	text := readOpenAPI(t)
	definitions := make(map[string]struct{})
	currentKind := ""
	for _, line := range strings.Split(text, "\n") {
		if match := componentKind.FindStringSubmatch(line); match != nil {
			currentKind = match[1]
			continue
		}
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			currentKind = ""
		}
		if currentKind == "" {
			continue
		}
		if match := componentName.FindStringSubmatch(line); match != nil {
			key := currentKind + "/" + match[1]
			if _, exists := definitions[key]; exists {
				t.Errorf("duplicate component definition %s", key)
			}
			definitions[key] = struct{}{}
		}
	}

	for _, match := range componentRef.FindAllStringSubmatch(text, -1) {
		key := match[1] + "/" + match[2]
		if _, ok := definitions[key]; !ok {
			t.Errorf("unresolved component reference %s", key)
		}
	}
}

func TestAuthenticationAndCSRFBoundaries(t *testing.T) {
	t.Parallel()

	text := readOpenAPI(t)
	if !strings.Contains(text, "security:\n  - cookieAuth: []") {
		t.Fatal("root cookie authentication requirement is missing")
	}

	public := map[string]bool{
		"GET /health/live":        true,
		"GET /health/ready":       true,
		"GET /api/v1/auth/status": true,
		"POST /api/v1/auth/setup": true,
		"POST /api/v1/auth/login": true,
	}
	protectedWrites := map[string]bool{
		"POST /api/v1/auth/logout":                 true,
		"PATCH /api/v1/settings":                   true,
		"POST /api/v1/libraries":                   true,
		"PATCH /api/v1/libraries/{libraryId}":      true,
		"DELETE /api/v1/libraries/{libraryId}":     true,
		"POST /api/v1/libraries/{libraryId}/scans": true,
		"POST /api/v1/scans/{scanId}/cancel":       true,
	}

	for _, operation := range readOperations(t) {
		key := operation.method + " " + operation.path
		if public[key] {
			if !strings.Contains(operation.block, "\n      security: []") {
				t.Errorf("%s must explicitly override cookie authentication", key)
			}
		} else if strings.Contains(operation.block, "\n      security: []") {
			t.Errorf("%s unexpectedly permits an anonymous request", key)
		}

		if protectedWrites[key] {
			for _, required := range []string{"cookieAuth: []", "csrfToken: []"} {
				if !strings.Contains(operation.block, required) {
					t.Errorf("%s does not require %s", key, required)
				}
			}
			if !strings.Contains(operation.block, "'403':") {
				t.Errorf("%s does not contract a CSRF/same-origin failure", key)
			}
		}
	}

	for _, key := range []string{
		"POST /api/v1/auth/setup",
		"POST /api/v1/auth/login",
	} {
		block := operationByKey(t, key).block
		for _, required := range []string{
			"#/components/parameters/OriginHeader",
			"'403':",
			"#/components/responses/Forbidden",
		} {
			if !strings.Contains(block, required) {
				t.Errorf("%s is missing same-origin contract %s", key, required)
			}
		}
	}
}

func TestCursorPaginationIsBoundedAndQueryBound(t *testing.T) {
	t.Parallel()

	for _, key := range []string{
		"GET /api/v1/library-paths",
		"GET /api/v1/libraries",
		"GET /api/v1/libraries/{libraryId}/scans",
		"GET /api/v1/libraries/{libraryId}/directories",
		"GET /api/v1/libraries/{libraryId}/assets",
		"GET /api/v1/assets",
	} {
		block := operationByKey(t, key).block
		for _, required := range []string{
			"#/components/parameters/CursorParameter",
			"#/components/parameters/LimitParameter",
			"'400':",
			"#/components/responses/BadRequest",
		} {
			if !strings.Contains(block, required) {
				t.Errorf("%s is missing %s", key, required)
			}
		}
	}

	text := readOpenAPI(t)
	normalized := strings.Join(strings.Fields(text), " ")
	for _, required := range []string{
		"maximum: 200",
		"default: 50",
		"query fingerprint",
		"fails with `invalid_cursor`",
		"never falls back to the first page",
	} {
		if !strings.Contains(normalized, required) {
			t.Errorf("cursor contract is missing %q", required)
		}
	}
}

func TestMediaContentLocksSingleRangeAndSafeHeaders(t *testing.T) {
	t.Parallel()

	get := operationByKey(t, "GET /api/v1/assets/{assetId}/content").block
	for _, required := range []string{
		"#/components/parameters/AssetIDParameter",
		"#/components/parameters/RangeHeader",
		"#/components/parameters/IfNoneMatchHeader",
		"#/components/parameters/IfModifiedSinceHeader",
		"#/components/parameters/IfRangeHeader",
		"'200':",
		"'206':",
		"'304':",
		"'416':",
		"#/components/responses/FullMediaContent",
		"#/components/responses/PartialMediaContent",
		"#/components/responses/RangeNotSatisfiable",
	} {
		if !strings.Contains(get, required) {
			t.Errorf("content GET is missing %s", required)
		}
	}

	head := operationByKey(t, "HEAD /api/v1/assets/{assetId}/content").block
	if strings.Contains(head, "#/components/parameters/RangeHeader") {
		t.Error("content HEAD must not contract partial-body behavior")
	}
	for _, required := range []string{
		"Accept-Ranges:",
		"Content-Length:",
		"Content-Type:",
		"Content-Disposition:",
		"ETag:",
		"Last-Modified:",
		"X-Content-Type-Options:",
	} {
		if !strings.Contains(head, required) {
			t.Errorf("content HEAD is missing %s", required)
		}
	}

	text := readOpenAPI(t)
	for _, required := range []string{
		"Exactly one byte range",
		"Multiple, malformed, and unsatisfiable ranges",
		"Content-Range:",
		"UnsatisfiedContentRange",
		"enum: [bytes]",
		"enum: [nosniff]",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("Range/header contract is missing %q", required)
		}
	}
}

func TestFilesystemPrivacyAndRootImmutabilityAreVisible(t *testing.T) {
	t.Parallel()

	text := readOpenAPI(t)
	normalized := strings.ToLower(strings.Join(strings.Fields(text), " "))
	for _, forbidden := range []string{
		"\n        hostPath:",
		"\n        absolutePath:",
		"\n        filesystemPath:",
		"\n        containerPath:",
		"/api/v1/assets/{assetId}/content?path=",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("contract exposes forbidden path field or parameter %q", strings.TrimSpace(forbidden))
		}
	}

	rename := schemaBlock(t, "RenameLibraryRequest")
	if strings.Contains(rename, "\n        rootPath:") {
		t.Error("RenameLibraryRequest allows changing immutable rootPath")
	}
	create := schemaBlock(t, "CreateLibraryRequest")
	if !strings.Contains(create, "\n        rootPath:") {
		t.Error("CreateLibraryRequest does not accept the allowed-root-relative rootPath")
	}
	if strings.Contains(create, "\n        displayPath:") {
		t.Error("CreateLibraryRequest accepts server-built displayPath")
	}

	for _, required := range []string{
		"The empty string means `/library` itself",
		"never a host path",
		"never accepts a path",
		"never a host",
	} {
		if !strings.Contains(normalized, strings.ToLower(required)) {
			t.Errorf("path privacy contract is missing %q", required)
		}
	}
}

func TestAllowedRootPathContractMatchesCanonicalPolicy(t *testing.T) {
	t.Parallel()

	document := loadOpenAPIDocument(t)
	rootPath := document.Components.Schemas["AllowedRootRelativePath"]
	if rootPath == nil || rootPath.Value == nil {
		t.Fatal("AllowedRootRelativePath schema is missing or unresolved")
	}

	for _, valid := range []string{
		"",
		"family",
		"family/2026",
		"相册/旅行 100%.jpg",
		"literal%20name",
		"100%25-real",
	} {
		if err := rootPath.Value.VisitJSON(
			valid,
			openapi3.SetSchemaRegexCompiler(compileECMAScriptPattern),
		); err != nil {
			t.Errorf("valid allowed-root-relative path %q was rejected: %v", valid, err)
		}
		if normalized, err := pathpolicy.Normalize(valid); err != nil || normalized != valid {
			t.Errorf("pathpolicy.Normalize(%q) = %q, %v; want raw path preserved", valid, normalized, err)
		}
	}
	for _, invalid := range []string{
		"/absolute",
		".",
		"..",
		"family/../private",
		"family//2026",
		"family/",
		`family\2026`,
		"family\x00private",
	} {
		if err := rootPath.Value.VisitJSON(
			invalid,
			openapi3.SetSchemaRegexCompiler(compileECMAScriptPattern),
		); err == nil {
			t.Errorf("non-canonical or encoded allowed-root path %q was accepted", invalid)
		}
		if _, err := pathpolicy.Normalize(invalid); !errors.Is(err, pathpolicy.ErrInvalid) {
			t.Errorf("pathpolicy.Normalize(%q) error = %v, want ErrInvalid", invalid, err)
		}
	}
	for _, encodedTraversal := range []string{
		"family%2f2026",
		"%2e%2e/private",
		"%252e%252e/private",
		"%25252E%25252e/private",
		"%5cwindows",
		"%255Cwindows",
		"%00",
		"%2500",
	} {
		if _, err := pathpolicy.Normalize(encodedTraversal); !errors.Is(err, pathpolicy.ErrInvalid) {
			t.Errorf("pathpolicy.Normalize(%q) error = %v, want ErrInvalid", encodedTraversal, err)
		}
	}

	block := schemaBlock(t, "AllowedRootRelativePath")
	normalizedBlock := strings.Join(strings.Fields(block), " ")
	for _, required := range []string{
		"Canonical UTF-8",
		"preserves safe literal percent",
		"iteratively checks percent-decoded views",
		"not a security boundary",
		"openat2",
		"RESOLVE_BENEATH",
		"RESOLVE_NO_SYMLINKS",
		"RESOLVE_NO_XDEV",
	} {
		if !strings.Contains(normalizedBlock, required) {
			t.Errorf("AllowedRootRelativePath is missing boundary statement %q", required)
		}
	}
	if strings.Contains(strings.ToLower(normalizedBlock), "realpath") {
		t.Error("AllowedRootRelativePath must not claim realpath is the Linux trust boundary")
	}
}

func TestAllRelativePathSchemasRejectTrailingEmptyComponent(t *testing.T) {
	t.Parallel()

	document := loadOpenAPIDocument(t)
	for _, name := range []string{
		"AllowedRootRelativePath",
		"LibraryRelativePath",
		"NullableLibraryRelativePath",
	} {
		schemaRef := document.Components.Schemas[name]
		if schemaRef == nil || schemaRef.Value == nil {
			t.Fatalf("%s schema is missing or unresolved", name)
		}
		if err := schemaRef.Value.VisitJSON(
			"family/",
			openapi3.SetSchemaRegexCompiler(compileECMAScriptPattern),
		); err == nil {
			t.Errorf("%s accepted a trailing empty path component", name)
		}
	}
}

func TestLibraryPickerHasSafeMountBoundaryReason(t *testing.T) {
	t.Parallel()

	document := loadOpenAPIDocument(t)
	entry := document.Components.Schemas["LibraryPathEntry"]
	if entry == nil || entry.Value == nil {
		t.Fatal("LibraryPathEntry schema is missing or unresolved")
	}
	reason := entry.Value.Properties["selectionBlockedReason"]
	if reason == nil || reason.Value == nil {
		t.Fatal("LibraryPathEntry.selectionBlockedReason is missing or unresolved")
	}
	found := false
	for _, value := range reason.Value.Enum {
		if value == "mount_boundary" {
			found = true
			break
		}
	}
	if !found {
		t.Error("selectionBlockedReason enum is missing mount_boundary")
	}
	requireSchemaEnumValue(t, document, "ErrorCode", "library_root_mount_boundary")

	block := schemaBlock(t, "LibraryPathEntry")
	for _, required := range []string{
		"`mount_boundary`",
		"descendant mount",
		"never reveals",
		"host path",
	} {
		if !strings.Contains(block, required) {
			t.Errorf("LibraryPathEntry is missing safe mount-boundary statement %q", required)
		}
	}
	for _, forbidden := range []string{"hostPath:", "mountSource:", "deviceId:"} {
		if strings.Contains(block, forbidden) {
			t.Errorf("LibraryPathEntry exposes forbidden mount detail %q", forbidden)
		}
	}
}

func TestUnifiedErrorsAreStableAndSanitized(t *testing.T) {
	t.Parallel()

	errorSchema := schemaBlock(t, "Error")
	normalizedErrorSchema := strings.ToLower(strings.Join(strings.Fields(errorSchema), " "))
	for _, required := range []string{
		"required: [code, message, requestId]",
		"Clients must branch on code, never message.",
		"never a host",
		"SQL",
		"stack trace",
		"raw subprocess output",
		"cookie",
		"CSRF token",
		"password",
		"secret",
	} {
		if !strings.Contains(normalizedErrorSchema, strings.ToLower(required)) {
			t.Errorf("Error schema is missing sanitization rule %q", required)
		}
	}

	codeSchema := schemaBlock(t, "ErrorCode")
	for _, code := range []string{
		"invalid_cursor",
		"invalid_credentials",
		"csrf_invalid",
		"library_path_overlap",
		"library_root_outside_allowed",
		"library_root_mount_boundary",
		"idempotency_conflict",
		"scan_already_finished",
		"source_offline",
		"source_missing",
		"range_not_satisfiable",
		"internal_error",
	} {
		if !strings.Contains(codeSchema, "- "+code) {
			t.Errorf("stable ErrorCode enum is missing %s", code)
		}
	}

	for _, response := range []string{
		"BadRequest",
		"Unauthorized",
		"Forbidden",
		"NotFound",
		"Conflict",
		"PreconditionFailed",
		"PreconditionRequired",
		"UnprocessableEntity",
		"TooManyRequests",
		"InternalError",
		"RangeNotSatisfiable",
	} {
		block := componentBlock(t, "responses", response)
		if !strings.Contains(block, "#/components/schemas/ErrorResponse") {
			t.Errorf("response %s does not use the unified ErrorResponse", response)
		}
	}
}

func TestErrorSchemaCannotCarryArbitraryDetails(t *testing.T) {
	t.Parallel()

	document := loadOpenAPIDocument(t)
	errorRef := document.Components.Schemas["Error"]
	if errorRef == nil || errorRef.Value == nil {
		t.Fatal("Error schema is missing or unresolved")
	}

	value := map[string]any{
		"code":      "invalid_request",
		"message":   "The request is invalid.",
		"requestId": "req_contract_test",
	}
	if err := errorRef.Value.VisitJSON(
		value,
		openapi3.SetSchemaRegexCompiler(compileECMAScriptPattern),
	); err != nil {
		t.Fatalf("minimal safe Error response was rejected: %v", err)
	}
	value["details"] = map[string]any{
		"path": "/mnt/private/secret.jpg",
	}
	if err := errorRef.Value.VisitJSON(
		value,
		openapi3.SetSchemaRegexCompiler(compileECMAScriptPattern),
	); err == nil {
		t.Fatal("Error schema accepted an arbitrary details object")
	}

	block := schemaBlock(t, "Error")
	if strings.Contains(block, "\n        details:") {
		t.Error("Error schema exposes an arbitrary details property")
	}
	if strings.Contains(block, "additionalProperties: true") {
		t.Error("Error schema permits arbitrary public error fields")
	}
}

func TestHealthResponsesHaveStatusSpecificSchemas(t *testing.T) {
	t.Parallel()

	document := loadOpenAPIDocument(t)
	testCases := []struct {
		name    string
		path    string
		status  string
		wantRef string
		valid   map[string]any
		invalid []map[string]any
	}{
		{
			name:    "liveness",
			path:    "/health/live",
			status:  "200",
			wantRef: "#/components/schemas/LivenessResponse",
			valid:   map[string]any{"status": "live", "reasonCode": nil},
			invalid: []map[string]any{
				{"status": "ready", "reasonCode": nil},
				{"status": "not_ready", "reasonCode": "database_unavailable"},
			},
		},
		{
			name:    "readiness success",
			path:    "/health/ready",
			status:  "200",
			wantRef: "#/components/schemas/ReadinessResponse",
			valid:   map[string]any{"status": "ready", "reasonCode": nil},
			invalid: []map[string]any{
				{"status": "live", "reasonCode": nil},
				{"status": "not_ready", "reasonCode": "database_unavailable"},
			},
		},
		{
			name:    "readiness failure",
			path:    "/health/ready",
			status:  "503",
			wantRef: "#/components/schemas/NotReadyResponse",
			valid:   map[string]any{"status": "not_ready", "reasonCode": "database_unavailable"},
			invalid: []map[string]any{
				{"status": "live", "reasonCode": nil},
				{"status": "ready", "reasonCode": nil},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			schemaRef := responseSchema(t, document, testCase.path, testCase.status)
			if schemaRef.Ref != testCase.wantRef {
				t.Fatalf("response schema ref = %q, want %q", schemaRef.Ref, testCase.wantRef)
			}
			if schemaRef.Value == nil {
				t.Fatal("response schema reference is unresolved")
			}
			if err := schemaRef.Value.VisitJSON(testCase.valid); err != nil {
				t.Fatalf("status-specific valid response was rejected: %v", err)
			}
			for _, invalid := range testCase.invalid {
				if err := schemaRef.Value.VisitJSON(invalid); err == nil {
					t.Errorf("cross-endpoint health state %#v was accepted", invalid)
				}
			}
		})
	}
}

func TestScanAndAssetContractMatchesDomainAndMigration(t *testing.T) {
	t.Parallel()

	document := loadOpenAPIDocument(t)
	requireSchemaEnumValue(t, document, "ScanStatus", "queued")
	requireSchemaEnumValue(t, document, "AssetKind", "animated")

	scanRun := document.Components.Schemas["ScanRun"]
	if scanRun == nil || scanRun.Value == nil {
		t.Fatal("ScanRun schema is missing or unresolved")
	}
	startedAt := scanRun.Value.Properties["startedAt"]
	if startedAt == nil || startedAt.Value == nil || !startedAt.Value.Nullable {
		t.Fatal("ScanRun.startedAt must resolve to a nullable schema for queued scans")
	}

	scannerSource := readRepositoryFile(t, "internal", "scanner", "scanner.go")
	for _, required := range []struct {
		name    string
		pattern string
	}{
		{name: "queued status", pattern: `RunStatusQueued\s+RunStatus\s*=\s*"queued"`},
		{name: "animated asset kind", pattern: `AssetKindAnimated\s+AssetKind\s*=\s*"animated"`},
		{name: "nullable scan start", pattern: `StartedAtMS\s+\*int64`},
	} {
		if !regexp.MustCompile(required.pattern).MatchString(scannerSource) {
			t.Errorf("scanner domain is missing %s", required.name)
		}
	}

	formatSource := readRepositoryFile(t, "internal", "scanner", "formats.go")
	if !regexp.MustCompile(
		`"\.gif"\s*:\s*\{\s*AssetKindAnimated\s*,\s*MediaFormatGIF\s*,\s*"image/gif"\s*\}`,
	).MatchString(formatSource) {
		t.Error("scanner format classification does not map GIF to the animated asset kind")
	}

	migration := readRepositoryFile(t, "migrations", "00001_initial.sql")
	for _, required := range []struct {
		name    string
		pattern string
	}{
		{name: "queued scan status", pattern: `(?s)CHECK\s*\(status\s+IN\s*\([^)]*'queued'[^)]*'running'`},
		{name: "animated asset kind", pattern: `(?s)CHECK\s*\(kind\s+IN\s*\([^)]*'animated'`},
	} {
		if !regexp.MustCompile(required.pattern).MatchString(migration) {
			t.Errorf("initial migration is missing %s", required.name)
		}
	}
	startedAtColumn := regexp.MustCompile(`(?m)^\s*started_at_ms\s+INTEGER\s*,?\s*$`)
	match := startedAtColumn.FindString(migration)
	if match == "" {
		t.Fatal("initial migration does not declare started_at_ms as a nullable INTEGER")
	}
	if strings.Contains(strings.ToUpper(match), "NOT NULL") {
		t.Fatal("initial migration makes started_at_ms non-null despite queued scan state")
	}
}

func TestFirstContractDecisionsCannotSilentlyDrift(t *testing.T) {
	t.Parallel()

	checks := map[string][]string{
		"POST /api/v1/libraries": {
			"#/components/parameters/IdempotencyKeyHeader",
			"'201':",
			"#/components/schemas/CreateLibraryResult",
		},
		"DELETE /api/v1/libraries/{libraryId}": {
			"'202':",
			"#/components/schemas/LibraryRemoval",
			"never moves, edits, or deletes original",
		},
		"POST /api/v1/libraries/{libraryId}/scans": {
			"coalesces the request",
			"'200':",
			"'202':",
		},
		"GET /api/v1/assets/{assetId}/thumbnail": {
			"'202':",
			"#/components/schemas/ThumbnailPending",
			"Retry-After:",
		},
	}
	for key, required := range checks {
		block := operationByKey(t, key).block
		for _, value := range required {
			if !strings.Contains(block, value) {
				t.Errorf("%s is missing fixed decision %q", key, value)
			}
		}
	}

	scanStatus := schemaBlock(t, "ScanStatus")
	if !strings.Contains(scanStatus, "offline") {
		t.Error("ScanStatus omits the offline terminal outcome required by the reliable-index contract")
	}
}

type ecmaRegexMatcher struct {
	expression *regexp2.Regexp
}

func (matcher *ecmaRegexMatcher) MatchString(value string) bool {
	matched, err := matcher.expression.MatchString(value)
	return err == nil && matched
}

func compileECMAScriptPattern(expression string) (openapi3.RegexMatcher, error) {
	compiled, err := regexp2.Compile(expression, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return &ecmaRegexMatcher{expression: compiled}, nil
}

func openAPIPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate contract test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "api", "openapi.yaml"))
}

func loadOpenAPIDocument(t *testing.T) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	document, err := loader.LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatalf("parse and resolve OpenAPI YAML: %v", err)
	}
	return document
}

func responseSchema(t *testing.T, document *openapi3.T, path, status string) *openapi3.SchemaRef {
	t.Helper()
	pathItem := document.Paths.Find(path)
	if pathItem == nil || pathItem.Get == nil {
		t.Fatalf("GET %s is missing", path)
	}
	response := pathItem.Get.Responses.Value(status)
	if response == nil || response.Value == nil {
		t.Fatalf("GET %s response %s is missing or unresolved", path, status)
	}
	mediaType := response.Value.Content.Get("application/json")
	if mediaType == nil || mediaType.Schema == nil {
		t.Fatalf("GET %s response %s has no application/json schema", path, status)
	}
	return mediaType.Schema
}

func requireSchemaEnumValue(t *testing.T, document *openapi3.T, name string, want any) {
	t.Helper()
	schemaRef := document.Components.Schemas[name]
	if schemaRef == nil || schemaRef.Value == nil {
		t.Fatalf("%s schema is missing or unresolved", name)
	}
	for _, value := range schemaRef.Value.Enum {
		if value == want {
			return
		}
	}
	t.Errorf("%s enum is missing %#v", name, want)
}

func readRepositoryFile(t *testing.T, path ...string) string {
	t.Helper()
	repositoryRoot := filepath.Dir(filepath.Dir(openAPIPath(t)))
	data, err := os.ReadFile(filepath.Join(append([]string{repositoryRoot}, path...)...))
	if err != nil {
		t.Fatalf("read repository source %s: %v", filepath.Join(path...), err)
	}
	return string(data)
}

func readOpenAPI(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(openAPIPath(t))
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "\t") {
		t.Error("OpenAPI YAML contains a tab")
	}
	for lineNumber, line := range strings.Split(text, "\n") {
		if strings.TrimRight(line, " ") != line {
			t.Errorf("OpenAPI YAML line %d has trailing whitespace", lineNumber+1)
		}
	}
	return text
}

func readOperations(t *testing.T) []operation {
	t.Helper()
	lines := strings.Split(readOpenAPI(t), "\n")
	var operations []operation
	currentPath := ""
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if line == "components:" {
			break
		}
		if match := pathLine.FindStringSubmatch(line); match != nil {
			currentPath = match[1]
			continue
		}
		match := methodLine.FindStringSubmatch(line)
		if currentPath == "" || match == nil {
			continue
		}
		end := index + 1
		for end < len(lines) {
			if lines[end] == "components:" || pathLine.MatchString(lines[end]) || methodLine.MatchString(lines[end]) {
				break
			}
			end++
		}
		block := strings.Join(lines[index:end], "\n")
		id := ""
		if idMatch := operationID.FindStringSubmatch(block); idMatch != nil {
			id = idMatch[1]
		}
		operations = append(operations, operation{
			path:   currentPath,
			method: strings.ToUpper(match[1]),
			id:     id,
			block:  block,
		})
		index = end - 1
	}
	return operations
}

func operationByKey(t *testing.T, key string) operation {
	t.Helper()
	for _, operation := range readOperations(t) {
		if operation.method+" "+operation.path == key {
			return operation
		}
	}
	t.Fatalf("operation %s not found", key)
	return operation{}
}

func schemaBlock(t *testing.T, name string) string {
	t.Helper()
	return componentBlock(t, "schemas", name)
}

func componentBlock(t *testing.T, kind, name string) string {
	t.Helper()
	lines := strings.Split(readOpenAPI(t), "\n")
	kindLine := "  " + kind + ":"
	nameLine := "    " + name + ":"
	inKind := false
	for index, line := range lines {
		if line == kindLine {
			inKind = true
			continue
		}
		if inKind && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			break
		}
		if !inKind || line != nameLine {
			continue
		}
		end := index + 1
		for end < len(lines) {
			if componentName.MatchString(lines[end]) || (strings.HasPrefix(lines[end], "  ") && !strings.HasPrefix(lines[end], "    ")) {
				break
			}
			end++
		}
		return strings.Join(lines[index:end], "\n")
	}
	t.Fatalf("component %s/%s not found", kind, name)
	return ""
}

func sliceDiff(want, got []string) string {
	var lines []string
	wantSet := make(map[string]bool, len(want))
	gotSet := make(map[string]bool, len(got))
	for _, value := range want {
		wantSet[value] = true
	}
	for _, value := range got {
		gotSet[value] = true
	}
	for _, value := range want {
		if !gotSet[value] {
			lines = append(lines, fmt.Sprintf("- %s", value))
		}
	}
	for _, value := range got {
		if !wantSet[value] {
			lines = append(lines, fmt.Sprintf("+ %s", value))
		}
	}
	return strings.Join(lines, "\n")
}
