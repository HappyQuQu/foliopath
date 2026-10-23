package api

import (
	"encoding/json"
	"github.com/HappyQuQu/foliopath/internal/face"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFaceDiagnosticsHTTPUsesSafeBoundedProjection(t *testing.T) {
	now := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	current := false
	value := face.LibrarySettings{LibraryID: 4, State: "degraded", Revision: 1, Enabled: true, GenerationCurrent: &current, Diagnostics: face.FailureDiagnostics{
		LatestTask:       &face.DiagnosticTask{OperationID: "aio_diagnostic", State: "failed", ErrorCode: "library_not_ready", UpdatedAt: now},
		RecordedFailures: 1, UnrecordedFailures: 7, Reasons: []face.FailureReason{{Code: "invalid_media", Count: 1}},
		Samples: []face.FailureSample{{AssetID: 5, Name: "broken.jpg", Code: "invalid_media", UpdatedAt: now}},
	}}
	mux := http.NewServeMux()
	registerFaceSettingsReadRoute(mux, faceHTTPSettingsStub{value})
	r := httptest.NewRecorder()
	mux.ServeHTTP(r, httptest.NewRequest("GET", "/api/v1/libraries/lib_4/ai/faces", nil))
	if r.Code != 200 {
		t.Fatalf("%d %s", r.Code, r.Body.String())
	}
	var body struct {
		Diagnostics faceDiagnosticsJSON `json:"diagnostics"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Diagnostics.LatestTask.OperationID != "aio_diagnostic" || body.Diagnostics.UnrecordedFailures != 7 || body.Diagnostics.Samples[0].AssetID != "ast_5" {
		t.Fatalf("body=%+v", body)
	}
	if !strings.Contains(r.Body.String(), `"generationCurrent":false`) || !strings.Contains(r.Body.String(), `"sizeBytes":0`) {
		t.Fatal("false freshness or empty source size was lost")
	}
	unknown, _ := json.Marshal(faceSettingsResponse(face.LibrarySettings{}))
	if strings.Contains(string(unknown), "generationCurrent") {
		t.Fatal("unknown freshness must be omitted")
	}
	for _, forbidden := range []string{"sourceFingerprint", "embedding", "/library/", "/Volumes/", "stack", "sql"} {
		if strings.Contains(r.Body.String(), forbidden) {
			t.Fatalf("leaked %s", forbidden)
		}
	}
	zero := faceDiagnosticsResponse(face.FailureDiagnostics{})
	if zero.Reasons == nil || zero.Samples == nil || zero.LatestTask != nil {
		t.Fatalf("empty projection=%+v", zero)
	}
}
