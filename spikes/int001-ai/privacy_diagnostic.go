package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

var diagnosticCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

// AIDiagnosticSnapshot is a closed spike contract. It deliberately has no
// fields for media paths, queries, person names, vectors, crops, or raw errors.
type AIDiagnosticSnapshot struct {
	SchemaVersion int                        `json:"schema_version"`
	Component     string                     `json:"component"`
	Model         AIDiagnosticModel          `json:"model"`
	State         string                     `json:"state"`
	Counts        AIDiagnosticCounts         `json:"counts"`
	ErrorCode     string                     `json:"error_code,omitempty"`
	Resources     AIDiagnosticResourceTotals `json:"resources"`
}

type AIDiagnosticModel struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type AIDiagnosticCounts struct {
	AssetsProcessed   int64 `json:"assets_processed"`
	FacesDetected     int64 `json:"faces_detected"`
	AnonymousClusters int64 `json:"anonymous_clusters"`
	PendingReviews    int64 `json:"pending_reviews"`
}

type AIDiagnosticResourceTotals struct {
	ResidentBytes int64 `json:"resident_bytes"`
	PeakBytes     int64 `json:"peak_bytes"`
	QueueDepth    int64 `json:"queue_depth"`
}

func MarshalAIDiagnostic(snapshot AIDiagnosticSnapshot) ([]byte, error) {
	if err := snapshot.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(snapshot)
}

func (snapshot AIDiagnosticSnapshot) Validate() error {
	if snapshot.SchemaVersion != 1 {
		return errors.New("AI diagnostic requires schema_version 1")
	}
	if snapshot.Component != "semantic" && snapshot.Component != "face" && snapshot.Component != "model-runtime" {
		return fmt.Errorf("unsupported AI diagnostic component %q", snapshot.Component)
	}
	if snapshot.State != "disabled" && snapshot.State != "ready" && snapshot.State != "degraded" && snapshot.State != "unavailable" {
		return fmt.Errorf("unsupported AI diagnostic state %q", snapshot.State)
	}
	if !governanceReferencePattern.MatchString(snapshot.Model.ID) || !governanceReferencePattern.MatchString(snapshot.Model.Version) || !sha256Pattern.MatchString(snapshot.Model.SHA256) {
		return errors.New("AI diagnostic requires safe model id/version and sha256")
	}
	if snapshot.ErrorCode != "" && !diagnosticCodePattern.MatchString(snapshot.ErrorCode) {
		return errors.New("AI diagnostic error_code must be a stable code")
	}
	values := []int64{
		snapshot.Counts.AssetsProcessed,
		snapshot.Counts.FacesDetected,
		snapshot.Counts.AnonymousClusters,
		snapshot.Counts.PendingReviews,
		snapshot.Resources.ResidentBytes,
		snapshot.Resources.PeakBytes,
		snapshot.Resources.QueueDepth,
	}
	for _, value := range values {
		if value < 0 {
			return errors.New("AI diagnostic counters and resource totals must be non-negative")
		}
	}
	return nil
}
