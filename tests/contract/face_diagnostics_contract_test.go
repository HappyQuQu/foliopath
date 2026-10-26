package contract_test

import (
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestFaceDiagnosticsContractIncludesTaskFailureCodes(t *testing.T) {
	t.Parallel()
	document, err := openapi3.NewLoader().LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatal(err)
	}
	operation := requiredComponentSchema(t, document, "AIModelOperation")
	codes := operation.Properties["errorCode"].Value.Enum
	for _, code := range []string{"library_not_ready", "library_scanning", "library_offline", "face_disabled", "model_unavailable", "operation_interrupted", "cancelled", "internal_error"} {
		if !slices.Contains(codes, any(code)) {
			t.Errorf("face task failure code %q is absent from the operation contract", code)
		}
	}
	settings := requiredComponentSchema(t, document, "FaceLibrarySettings")
	if settings.Properties["diagnostics"] == nil || slices.Contains(settings.Required, "diagnostics") {
		t.Fatal("diagnostics must remain an additive optional field")
	}
	if settings.Properties["generationCurrent"] == nil || slices.Contains(settings.Required, "generationCurrent") {
		t.Fatal("generation freshness must remain additive and optional")
	}
	diagnostics := requiredComponentSchema(t, document, "FaceFailureDiagnostics")
	sample := diagnostics.Properties["samples"].Value.Items.Value
	if sample.Properties["sizeBytes"] == nil || slices.Contains(sample.Required, "sizeBytes") {
		t.Fatal("source size must remain additive and optional")
	}
}
