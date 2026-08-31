package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

type aiModelManagementStub struct {
	snapshot           aimodel.Snapshot
	scan               aimodel.CandidateScan
	operation          aimodel.Operation
	installResult      aimodel.InstallResult
	activationResult   aimodel.ActivationResult
	err                error
	candidateID        string
	storageMode        aimodel.StorageMode
	key                string
	cancelID           string
	cancelRevision     int64
	activationModelID  string
	activationRevision int64
}

func (stub *aiModelManagementStub) StartActivation(
	_ context.Context,
	modelID string,
	revision int64,
	key string,
) (aimodel.ActivationResult, error) {
	stub.activationModelID, stub.activationRevision, stub.key = modelID, revision, key
	return stub.activationResult, stub.err
}

func (stub *aiModelManagementStub) List(context.Context) (aimodel.Snapshot, error) {
	return stub.snapshot, stub.err
}

func (stub *aiModelManagementStub) ScanCandidates(context.Context) (aimodel.CandidateScan, error) {
	return stub.scan, stub.err
}

func (stub *aiModelManagementStub) StartInstall(
	_ context.Context,
	candidateID string,
	storageMode aimodel.StorageMode,
	key string,
) (aimodel.InstallResult, error) {
	stub.candidateID, stub.storageMode, stub.key = candidateID, storageMode, key
	return stub.installResult, stub.err
}

func (stub *aiModelManagementStub) GetOperation(context.Context, string) (aimodel.Operation, error) {
	return stub.operation, stub.err
}

func (stub *aiModelManagementStub) CancelOperation(
	_ context.Context,
	id string,
	revision int64,
) (aimodel.Operation, error) {
	stub.cancelID, stub.cancelRevision = id, revision
	return stub.operation, stub.err
}

func TestAIModelRoutesExposeOnlyReviewedMetadata(t *testing.T) {
	now := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	model := aimodel.Model{
		ID: "aim_reviewed123", Package: aimodel.VerifiedPackage{
			PackageID: "reviewed-package", Purpose: aimodel.PurposeSemanticImageText,
			Version: "1.0.0", Architecture: "arm64",
			ContentHash: strings.Repeat("a", 64), LicenseID: "Apache-2.0", PackageSizeByte: 1024,
		},
		StorageMode: aimodel.StorageDirect, State: aimodel.StateAvailable,
		SourceIdentity: "/models/private-name.foliomodel", AvailabilityRevision: 3,
		CreatedAt: now, UpdatedAt: now,
	}
	stub := &aiModelManagementStub{snapshot: aimodel.Snapshot{
		Items: []aimodel.Model{model}, Revision: 7,
	}}
	mux := http.NewServeMux()
	registerAIModelRoutes(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/ai/models", nil))
	if response.Code != http.StatusOK || response.Header().Get("ETag") != `"ai-models-r7"` {
		t.Fatalf("list status/etag = %d/%q; body=%s", response.Code, response.Header().Get("ETag"), response.Body.String())
	}
	body := response.Body.String()
	for _, leaked := range []string{"/models", "private-name", "contentHash", "packageId"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("list response leaked %q: %s", leaked, body)
		}
	}
	for _, expected := range []string{`"id":"aim_reviewed123"`, `"licenseId":"Apache-2.0"`, `"availabilityRevision":3`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("list response missing %q: %s", expected, body)
		}
	}
}

func TestAIModelCandidateScanUsesSafeUnknownValues(t *testing.T) {
	stub := &aiModelManagementStub{scan: aimodel.CandidateScan{
		Revision: 4,
		Candidates: []aimodel.Candidate{
			{ID: "aic_compatible1", Compatibility: "compatible", Package: aimodel.VerifiedPackage{
				Purpose: aimodel.PurposeSemanticImageText, Version: "1.0", Architecture: "amd64",
				PackageSizeByte: 99, LicenseID: "Apache-2.0",
			}},
			{ID: "aic_rejected123", Compatibility: "incompatible", SourceIdentity: "/models/secret.foliomodel"},
		},
	}}
	mux := http.NewServeMux()
	registerAIModelRoutes(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/ai/model-candidate-scans", nil))
	if response.Code != http.StatusOK || response.Header().Get("ETag") != `"ai-candidates-r4"` {
		t.Fatalf("scan status/etag = %d/%q; body=%s", response.Code, response.Header().Get("ETag"), response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "/models") || strings.Contains(body, "secret.foliomodel") {
		t.Fatalf("scan response leaked source identity: %s", body)
	}
	for _, expected := range []string{`"compatibility":"compatible"`, `"compatibility":"invalid_manifest"`, `"purpose":"unknown"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("scan response missing %q: %s", expected, body)
		}
	}
}

func TestAIModelInstallAndOperationCancellationContract(t *testing.T) {
	now := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	operation := aimodel.Operation{
		ID: "aio_install123", Kind: aimodel.OperationModelInstall,
		State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued,
		Revision: 2, CreatedAt: now, UpdatedAt: now,
	}
	stub := &aiModelManagementStub{
		operation:     operation,
		installResult: aimodel.InstallResult{Operation: operation, Created: true},
	}
	mux := http.NewServeMux()
	registerAIModelRoutes(mux, stub)

	install := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/model-candidates/aic_candidate123/install", strings.NewReader(`{"storageMode":"managed"}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	request.Header.Set("Idempotency-Key", "install-request-123")
	mux.ServeHTTP(install, request)
	if install.Code != http.StatusAccepted ||
		install.Header().Get("Location") != "/api/v1/ai/operations/aio_install123" ||
		install.Header().Get("ETag") != `"aio_install123-r2"` ||
		stub.candidateID != "aic_candidate123" || stub.storageMode != aimodel.StorageManaged || stub.key != "install-request-123" {
		t.Fatalf("install contract failed: status=%d headers=%v stub=%#v body=%s", install.Code, install.Header(), stub, install.Body.String())
	}

	cancel := httptest.NewRecorder()
	cancelRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ai/operations/aio_install123/cancel", nil)
	cancelRequest.Header.Set("If-Match", `"aio_install123-r2"`)
	mux.ServeHTTP(cancel, cancelRequest)
	if cancel.Code != http.StatusAccepted || stub.cancelID != "aio_install123" || stub.cancelRevision != 2 {
		t.Fatalf("cancel contract failed: status=%d stub=%#v body=%s", cancel.Code, stub, cancel.Body.String())
	}
}

func TestAIModelActivationContract(t *testing.T) {
	now := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	operation := aimodel.Operation{ID: "aio_activate123", Kind: aimodel.OperationModelActivate,
		State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued, Revision: 1, CreatedAt: now, UpdatedAt: now}
	stub := &aiModelManagementStub{activationResult: aimodel.ActivationResult{Operation: operation, Created: true}}
	mux := http.NewServeMux()
	registerAIModelRoutes(mux, stub)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/models/aim_reviewed123/activate", nil)
	request.Header.Set("If-Match", `"aim_reviewed123-r3"`)
	request.Header.Set("Idempotency-Key", "activate-request-123")
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted ||
		response.Header().Get("Location") != "/api/v1/ai/operations/aio_activate123" ||
		response.Header().Get("ETag") != `"aio_activate123-r1"` ||
		stub.activationModelID != "aim_reviewed123" || stub.activationRevision != 3 || stub.key != "activate-request-123" {
		t.Fatalf("activation contract failed: status=%d headers=%v stub=%#v body=%s", response.Code, response.Header(), stub, response.Body.String())
	}
}

func TestParseAIRevisionETagRejectsNonCanonicalAliases(t *testing.T) {
	for _, value := range []string{
		`"aim_reviewed123-r01"`,
		`"aim_reviewed123-r+1"`,
		`"aim_reviewed123-r0000000000000000001"`,
	} {
		t.Run(value, func(t *testing.T) {
			if revision, err := parseAIModelIfMatch(value, "aim_reviewed123"); revision != 0 || !errors.Is(err, aimodel.ErrPreconditionFailed) {
				t.Fatalf("parseAIModelIfMatch(%q) = %d, %v", value, revision, err)
			}
		})
	}
	if revision, err := parseAIModelIfMatch(`"aim_reviewed123-r1"`, "aim_reviewed123"); err != nil || revision != 1 {
		t.Fatalf("canonical ETag = %d, %v", revision, err)
	}
}

func TestAIModelRoutesRejectInvalidMutationInputsAndMapStableErrors(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		body    string
		headers map[string]string
		stubErr error
		status  int
		code    string
	}{
		{name: "missing idempotency key", target: "/api/v1/ai/model-candidates/aic_candidate123/install", body: `{"storageMode":"managed"}`, headers: map[string]string{"Content-Type": "application/json"}, status: 400, code: "invalid_request"},
		{name: "unknown JSON field", target: "/api/v1/ai/model-candidates/aic_candidate123/install", body: `{"storageMode":"managed","path":"/models"}`, headers: map[string]string{"Content-Type": "application/json", "Idempotency-Key": "install-request-123"}, status: 400, code: "invalid_request"},
		{name: "stale candidate", target: "/api/v1/ai/model-candidates/aic_candidate123/install", body: `{"storageMode":"direct"}`, headers: map[string]string{"Content-Type": "application/json", "Idempotency-Key": "install-request-123"}, stubErr: aimodel.ErrCandidateStale, status: 409, code: "model_candidate_stale"},
		{name: "missing precondition", target: "/api/v1/ai/operations/aio_install123/cancel", status: 428, code: "precondition_required"},
		{name: "finished operation", target: "/api/v1/ai/operations/aio_install123/cancel", headers: map[string]string{"If-Match": `"aio_install123-r2"`}, stubErr: aimodel.ErrOperationAlreadyFinished, status: 409, code: "ai_operation_already_finished"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := &aiModelManagementStub{err: test.stubErr}
			mux := http.NewServeMux()
			registerAIModelRoutes(mux, stub)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, test.target, strings.NewReader(test.body))
			for name, value := range test.headers {
				request.Header.Set(name, value)
			}
			mux.ServeHTTP(response, request)
			if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response = %d %s, want %d/%s", response.Code, response.Body.String(), test.status, test.code)
			}
		})
	}
}

func TestAIModelInstallRejectsOversizedTrailingWhitespace(t *testing.T) {
	stub := &aiModelManagementStub{}
	mux := http.NewServeMux()
	registerAIModelRoutes(mux, stub)
	body := `{"storageMode":"managed"}` + strings.Repeat(" ", maxAIModelJSONBytes)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/model-candidates/aic_candidate123/install", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "install-request-123")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || stub.candidateID != "" {
		t.Fatalf("response=%d body=%s candidate=%q", response.Code, response.Body.String(), stub.candidateID)
	}
	assertSafeErrorResponse(t, response, codeInvalidRequest)
}

var _ AIModelManagementService = (*aiModelManagementStub)(nil)
