package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

type downloadHTTPStub struct{ id, key string }

func (*downloadHTTPStub) ListDownloads(context.Context) (aimodel.DownloadPage, error) {
	return aimodel.DownloadPage{Presets: aimodel.BuiltinModelPresets(), Items: []aimodel.DownloadOffer{{ID: "vision_model", Name: "Vision", SourceLabel: "Official", PackageSizeBytes: 1024, Compatibility: "compatible"}}}, nil
}
func (stub *downloadHTTPStub) StartDownload(_ context.Context, id, key string) (aimodel.InstallResult, error) {
	stub.id, stub.key = id, key
	return aimodel.InstallResult{Operation: aimodel.Operation{ID: "aio_download", Revision: 1, Kind: aimodel.OperationModelInstall, State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued}}, nil
}
func TestDownloadRoutesUseIDAndExposeSafeMetadata(t *testing.T) {
	stub := &downloadHTTPStub{}
	mux := http.NewServeMux()
	registerAIModelDownloadRoutes(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/ai/model-downloads", nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"runtimeAvailable":false`) || !strings.Contains(response.Body.String(), `"operation":null`) {
		t.Fatal(response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"readiness":"package_pending"`) || !strings.Contains(response.Body.String(), `"id":"preset_buffalo_l"`) {
		t.Fatal("missing built-in preset metadata", response.Body.String())
	}
	request := httptest.NewRequest("POST", "/api/v1/ai/model-downloads/vision_model/install", nil)
	request.Header.Set("Idempotency-Key", "request_download_001")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != 202 || stub.id != "vision_model" || response.Header().Get("Location") != "/api/v1/ai/operations/aio_download" {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest("POST", "/api/v1/ai/model-downloads/vision_model/install", strings.NewReader(`{"url":"https://arbitrary.example"}`))
	request.Header.Set("Idempotency-Key", "request_download_002")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != 400 {
		t.Fatal("accepted request body")
	}
}
