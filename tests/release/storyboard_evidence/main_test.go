package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestVerifyEvidenceAcceptsMatchingNativePair(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	writeEvidence(t, directory, validEvidence("arm64"))

	if err := verifyEvidence(directory, testCommit); err != nil {
		t.Fatalf("verifyEvidence() error = %v", err)
	}
}

func TestVerifyEvidenceRejectsCrossArchitecturePixelDrift(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	arm64 := validEvidence("arm64")
	arm64.Fixture.DecodedPixelSHA256 = strings.Repeat("b", 64)
	writeEvidence(t, directory, arm64)

	err := verifyEvidence(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "pixels differ") {
		t.Fatalf("verifyEvidence() error = %v, want pixel drift", err)
	}
}

func TestVerifyEvidenceRejectsCrossAttemptPair(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	arm64 := validEvidence("arm64")
	arm64.WorkflowRunAttempt = 2
	writeEvidence(t, directory, arm64)

	err := verifyEvidence(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "run attempts") {
		t.Fatalf("verifyEvidence() error = %v, want run attempt drift", err)
	}
}

func TestWriteSummaryPublishesVerifiedPair(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	writeEvidence(t, directory, validEvidence("arm64"))

	summary, err := verifyEvidencePair(directory, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "paired", "summary.json")
	if err := writeSummary(output, summary); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var decoded pairedEvidenceSummary
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Result != "passed" ||
		decoded.SourceCommit != testCommit ||
		decoded.WorkflowRunID != "123" ||
		decoded.WorkflowRunAttempt != 1 ||
		len(decoded.Candidates) != 2 ||
		decoded.Candidates[0].Architecture != "amd64" ||
		decoded.Candidates[1].Architecture != "arm64" ||
		!decoded.Checks.WorkflowAttemptMatches ||
		!decoded.Checks.DecodedPixelsMatch {
		t.Fatalf("unexpected paired summary: %+v", decoded)
	}
}

func TestValidateEvidenceRejectsIncompleteCacheRepair(t *testing.T) {
	evidence := validEvidence("amd64")
	evidence.CacheRepair.MissingStatus = 200

	err := validateEvidence(evidence, "amd64", testCommit)
	if err == nil || !strings.Contains(err.Error(), "cache repair") {
		t.Fatalf("validateEvidence() error = %v, want cache repair failure", err)
	}
}

func validEvidence(architecture string) storyboardEvidence {
	return storyboardEvidence{
		SchemaVersion:  1,
		Feature:        "FTR-VID-001",
		SourceCommit:   testCommit,
		Architecture:   architecture,
		OS:             "linux",
		ImageDigest:    "sha256:" + strings.Repeat("a", 64),
		ImageSizeBytes: 1,
		FFmpegVersion:  "ffmpeg version 7.1.5 Copyright",
		Fixture: fixtureEvidence{
			SourceSHA256:       strings.Repeat("a", 64),
			FrameCount:         10,
			Columns:            5,
			Rows:               2,
			Width:              1600,
			Height:             360,
			DecodedPixelSHA256: strings.Repeat("a", 64),
		},
		CacheRepair: cacheRepairEvidence{
			InitialStatus:      200,
			MissingStatus:      202,
			RebuiltStatus:      200,
			DecodedPixelsMatch: true,
		},
		OriginalMediaUnchanged: true,
		ResourceLimit: resourceLimitEvidence{
			CPUs:        4,
			MemoryBytes: 4 * 1024 * 1024 * 1024,
		},
		SmokeSuite:         "tests/release/storyboard_vertical_smoke.sh",
		Result:             "passed",
		WorkflowRunID:      "123",
		WorkflowRunAttempt: 1,
		CreatedAt:          "2026-07-29T00:00:00Z",
	}
}

func writeEvidence(t *testing.T, directory string, evidence storyboardEvidence) {
	t.Helper()
	content, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(
		directory,
		"storyboard-evidence-"+evidence.Architecture+".json",
	)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}
