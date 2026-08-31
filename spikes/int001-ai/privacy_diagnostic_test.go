package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAIDiagnosticSnapshotHasClosedNonSensitiveShape(t *testing.T) {
	snapshot := validAIDiagnosticSnapshot()
	payload, err := MarshalAIDiagnostic(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	assertExactJSONKeys(t, decoded, "schema_version", "component", "model", "state", "counts", "error_code", "resources")
	assertExactJSONKeys(t, decoded["model"].(map[string]any), "id", "version", "sha256")
	assertExactJSONKeys(t, decoded["counts"].(map[string]any), "assets_processed", "faces_detected", "anonymous_clusters", "pending_reviews")
	assertExactJSONKeys(t, decoded["resources"].(map[string]any), "resident_bytes", "peak_bytes", "queue_depth")

	serialized := string(payload)
	for _, forbidden := range []string{
		"query", "person_name", "media_path", "face_crop", "embedding", "vector", "raw_error", "runtime_error",
	} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("diagnostic payload contains forbidden field marker %q: %s", forbidden, serialized)
		}
	}
}

func TestAIDiagnosticSnapshotRejectsRawOrUnboundedValues(t *testing.T) {
	tests := []struct {
		name string
		edit func(*AIDiagnosticSnapshot)
	}{
		{name: "raw error", edit: func(snapshot *AIDiagnosticSnapshot) {
			snapshot.ErrorCode = "open /library/private/alice.jpg: permission denied"
		}},
		{name: "unsafe model id", edit: func(snapshot *AIDiagnosticSnapshot) { snapshot.Model.ID = "../private/model" }},
		{name: "unsafe version", edit: func(snapshot *AIDiagnosticSnapshot) { snapshot.Model.Version = "v1 person name" }},
		{name: "invalid digest", edit: func(snapshot *AIDiagnosticSnapshot) { snapshot.Model.SHA256 = "latest" }},
		{name: "unknown state", edit: func(snapshot *AIDiagnosticSnapshot) { snapshot.State = "stack trace follows" }},
		{name: "negative count", edit: func(snapshot *AIDiagnosticSnapshot) { snapshot.Counts.FacesDetected = -1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validAIDiagnosticSnapshot()
			test.edit(&snapshot)
			if _, err := MarshalAIDiagnostic(snapshot); err == nil {
				t.Fatal("expected unsafe diagnostic value to be rejected")
			}
		})
	}
}

func validAIDiagnosticSnapshot() AIDiagnosticSnapshot {
	return AIDiagnosticSnapshot{
		SchemaVersion: 1,
		Component:     "face",
		Model: AIDiagnosticModel{
			ID:      "yunet-candidate",
			Version: "2023mar",
			SHA256:  "0000000000000000000000000000000000000000000000000000000000000000",
		},
		State:     "degraded",
		ErrorCode: "MODEL_UNAVAILABLE",
		Counts: AIDiagnosticCounts{
			AssetsProcessed:   100,
			FacesDetected:     80,
			AnonymousClusters: 12,
			PendingReviews:    5,
		},
		Resources: AIDiagnosticResourceTotals{
			ResidentBytes: 128 * 1024 * 1024,
			PeakBytes:     192 * 1024 * 1024,
			QueueDepth:    2,
		},
	}
}

func assertExactJSONKeys(t *testing.T, object map[string]any, expected ...string) {
	t.Helper()
	if len(object) != len(expected) {
		t.Fatalf("unexpected JSON object size: got %v, want keys %v", object, expected)
	}
	for _, key := range expected {
		if _, ok := object[key]; !ok {
			t.Fatalf("missing JSON key %q in %v", key, object)
		}
	}
}
