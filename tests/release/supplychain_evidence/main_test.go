package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestVerifyEvidencePairRequiresMatchingNativeEvidence(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	writeEvidence(t, directory, validEvidence("arm64"))

	summary, err := verifyEvidencePair(directory, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Result != "passed" || len(summary.Candidates) != 2 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestVerifyEvidencePairRejectsDifferentRun(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	arm64 := validEvidence("arm64")
	arm64.WorkflowRunID = "9002"
	writeEvidence(t, directory, arm64)

	_, err := verifyEvidencePair(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "different workflow runs") {
		t.Fatalf("error = %v, want workflow run mismatch", err)
	}
}

func TestVerifyEvidencePairRejectsNonZeroFindings(t *testing.T) {
	directory := t.TempDir()
	amd64 := validEvidence("amd64")
	amd64.VulnerabilityScan.High = 1
	writeEvidence(t, directory, amd64)
	writeEvidence(t, directory, validEvidence("arm64"))

	_, err := verifyEvidencePair(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "counts must all be zero") {
		t.Fatalf("error = %v, want vulnerability count failure", err)
	}
}

func TestVerifyEvidencePairRejectsBannedPackage(t *testing.T) {
	directory := t.TempDir()
	amd64 := validEvidence("amd64")
	amd64.BannedPackageCount = 1
	writeEvidence(t, directory, amd64)
	writeEvidence(t, directory, validEvidence("arm64"))

	_, err := verifyEvidencePair(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "bannedPackageCount") {
		t.Fatalf("error = %v, want banned package failure", err)
	}
}

func TestVerifyEvidencePairRejectsSourceSPDXMismatch(t *testing.T) {
	directory := t.TempDir()
	writeEvidence(t, directory, validEvidence("amd64"))
	arm64 := validEvidence("arm64")
	arm64.SBOM.SourceSHA256 = strings.Repeat("b", 64)
	writeEvidence(t, directory, arm64)

	_, err := verifyEvidencePair(directory, testCommit)
	if err == nil || !strings.Contains(err.Error(), "source SPDX") {
		t.Fatalf("error = %v, want source SPDX mismatch", err)
	}
}

func validEvidence(architecture string) supplyChainEvidence {
	hash := strings.Repeat("a", 64)
	return supplyChainEvidence{
		SchemaVersion:  1,
		Release:        releaseName,
		SourceCommit:   testCommit,
		Architecture:   architecture,
		OS:             "linux",
		ImageDigest:    "sha256:" + hash,
		ImageSizeBytes: 1234,
		SBOM: sbomEvidence{
			ImageSHA256:  hash,
			NPMSHA256:    hash,
			SourceSHA256: hash,
		},
		VulnerabilityScan: vulnerabilityEvidence{
			Policy:        "all",
			ReportSHA256:  hash,
			SummarySHA256: hash,
		},
		NoticesSHA256:      hash,
		GLibPackageVersion: "2.88.3-1",
		WorkflowRunID:      "9001",
		WorkflowRunAttempt: 1,
		CreatedAt:          "2026-07-30T00:00:00Z",
		Result:             "passed",
	}
}

func writeEvidence(t *testing.T, directory string, evidence supplyChainEvidence) {
	t.Helper()
	content, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(
		directory,
		"supply-chain-evidence-"+evidence.Architecture+".json",
	)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}
