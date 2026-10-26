package ocr

import (
	"encoding/json"
	"testing"
)

func TestImageInputContractIsIndependentAndExplicit(t *testing.T) {
	var fields map[string]any
	if err := json.Unmarshal([]byte(InputContract), &fields); err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{"profile": "ocr-rgb-png-v1", "max_source_bytes": float64(MaxSourceBytes), "max_source_pixels": float64(MaxSourcePixels), "max_output_bytes": float64(MaxImageBytes), "max_output_pixels": float64(MaxPreparedPixels), "max_dimension": float64(MaxPreparedDimension), "min_dimension": float64(MinImageDimension), "max_header_bytes": float64(MaxHeaderBytes), "alpha": "white", "metadata": "strip"}
	for key, value := range expected {
		if fields[key] != value {
			t.Fatal(key, fields[key])
		}
	}
	if InputContractSHA256() != "753de4384b85c73f55020829b8dc96d41700881f01e9064afbbb8f963df46a4e" {
		t.Fatal("missing deployment contract fingerprint")
	}
}
