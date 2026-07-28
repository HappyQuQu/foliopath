package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestVerifyEvidenceRequiresBothArchitecturesFromSameRun(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	writeEvidence(t, directory, validEvidence("arm64"))

	if err := verifyEvidence(directory, testCommit); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyEvidenceRejectsDifferentRun(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	arm64 := validEvidence("arm64")
	arm64.WorkflowRunID = "9002"
	writeEvidence(t, directory, arm64)

	err := verifyEvidence(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "different workflow runs") {
		t.Fatalf("verifyEvidence() error = %v, want different workflow runs", err)
	}
}

func TestVerifyEvidenceRejectsCommitMismatch(t *testing.T) {
	directory := t.TempDir()
	amd64 := validEvidence("amd64")
	amd64.SourceCommit = "abcdef0123456789abcdef0123456789abcdef01"
	writeEvidence(t, directory, amd64)
	writeEvidence(t, directory, validEvidence("arm64"))

	err := verifyEvidence(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "sourceCommit") {
		t.Fatalf("verifyEvidence() error = %v, want sourceCommit mismatch", err)
	}
}

func TestVerifyEvidenceRejectsFailedSmoke(t *testing.T) {
	directory := t.TempDir()
	amd64 := validEvidence("amd64")
	amd64.Result = "failed"
	writeEvidence(t, directory, amd64)
	writeEvidence(t, directory, validEvidence("arm64"))

	err := verifyEvidence(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "want passed") {
		t.Fatalf("verifyEvidence() error = %v, want failed result", err)
	}
}

func validEvidence(architecture string) imageEvidence {
	return imageEvidence{
		SchemaVersion:      1,
		Release:            releaseName,
		SourceCommit:       testCommit,
		Architecture:       architecture,
		OS:                 "linux",
		ImageDigest:        "sha256:" + strings.Repeat("a", 64),
		ImageSizeBytes:     1234,
		SmokeSuite:         "tests/release/image_smoke.sh",
		Result:             "passed",
		WorkflowRunID:      "9001",
		WorkflowRunAttempt: 1,
		CreatedAt:          "2026-07-28T00:00:00Z",
	}
}

func writeEvidence(t *testing.T, directory string, evidence imageEvidence) {
	t.Helper()
	content, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(
		directory,
		"release-image-"+evidence.Architecture+".json",
	)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}
