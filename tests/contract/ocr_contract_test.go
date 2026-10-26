package contract_test

import (
	"github.com/getkin/kin-openapi/openapi3"
	"testing"
)

func TestOCRManagementContractHasExplicitScopeAndNoPlaintext(t *testing.T) {
	d, err := openapi3.NewLoader().LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []struct{ path, method string }{
		{"/api/v1/libraries/{libraryId}/ai/ocr", "PUT"}, {"/api/v1/libraries/{libraryId}/ai/ocr/jobs", "POST"}, {"/api/v1/libraries/{libraryId}/ai/ocr/jobs/{jobId}/cancel", "POST"}, {"/api/v1/libraries/{libraryId}/ai/ocr/clear", "POST"},
	} {
		path := d.Paths.Find(v.path)
		if path == nil {
			t.Fatal(v.path)
		}
		op := path.GetOperation(v.method)
		if op == nil || op.Security == nil || op.Responses.Status(412) == nil || op.Responses.Status(428) == nil {
			t.Fatal("unsafe OCR contract", v)
		}
	}
	for _, name := range []string{"OCRStatus", "OCRJob", "OCRCoverage"} {
		s := requiredComponentSchema(t, d, name)
		for _, field := range []string{"text", "lines", "path", "url", "modelURL"} {
			if s.Properties[field] != nil {
				t.Fatal("plaintext/path authority leaked", field)
			}
		}
	}
	status := requiredComponentSchema(t, d, "OCRStatus")
	empty := map[string]any{"libraryId": "lib_1", "enabled": false, "clearing": false, "revision": 1, "epoch": 1, "fingerprint": nil, "expectedFingerprint": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "state": "disabled", "latestJob": nil, "coverage": map[string]any{"eligible": 0, "ready": 0, "empty": 0, "failed": 0, "pending": 0, "stale": 0}}
	if err := status.VisitJSON(empty); err != nil {
		t.Fatal("disabled wire response", err)
	}
}
