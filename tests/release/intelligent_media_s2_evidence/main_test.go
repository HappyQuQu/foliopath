package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestVerifyEvidenceBindsAllThreePassedSummaries(t *testing.T) {
	directory := t.TempDir()
	qualityPath, nativePath, supplyPath := writeValidSummaries(t, directory)
	summary, err := verifyEvidence(qualityPath, nativePath, supplyPath, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Result != "passed" || !summary.Checks["sameModelPackage"] || len(summary.Inputs) != 3 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestVerifyEvidenceRejectsMismatchedOrIncompleteEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*qualitySummary, *nativeSummary, *supplySummary)
		want   string
	}{
		{name: "model", mutate: func(_ *qualitySummary, native *nativeSummary, _ *supplySummary) {
			native.Candidates[0].ModelPackageDigest = "sha256:" + strings.Repeat("b", 64)
		}, want: "model package digests differ"},
		{name: "image", mutate: func(_ *qualitySummary, _ *nativeSummary, supply *supplySummary) {
			supply.Images[0].ImageDigest = "sha256:" + strings.Repeat("b", 64)
		}, want: "final image digest differs"},
		{name: "quality", mutate: func(quality *qualitySummary, _ *nativeSummary, _ *supplySummary) {
			quality.GatePass = false
		}, want: "quality gate did not pass"},
		{name: "baseline native", mutate: func(_ *qualitySummary, native *nativeSummary, _ *supplySummary) {
			native.Checks["finalModelEvidence"] = false
		}, want: "finalModelEvidence"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			quality, native, supply := validSummaries()
			test.mutate(&quality, &native, &supply)
			qualityPath := writeJSON(t, directory, "quality.json", quality)
			nativePath := writeJSON(t, directory, "native.json", native)
			supplyPath := writeJSON(t, directory, "supply.json", supply)
			_, err := verifyEvidence(qualityPath, nativePath, supplyPath, testCommit)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func writeValidSummaries(t *testing.T, directory string) (string, string, string) {
	t.Helper()
	quality, native, supply := validSummaries()
	return writeJSON(t, directory, "quality.json", quality),
		writeJSON(t, directory, "native.json", native),
		writeJSON(t, directory, "supply.json", supply)
}

func validSummaries() (qualitySummary, nativeSummary, supplySummary) {
	digest := "sha256:" + strings.Repeat("a", 64)
	hash := strings.Repeat("a", 64)
	quality := qualitySummary{
		SchemaVersion: 1, SourceCommit: testCommit, DatasetManifestSHA256: hash,
		ModelPackageDigest: digest, GatePass: true,
		Approvals: qualityApprovals{Product: "PRODUCT-APPROVAL-001", ML: "ML-APPROVAL-001", QA: "QA-APPROVAL-001"},
	}
	native := nativeSummary{
		SchemaVersion: 1, Feature: "FTR-INT-001", SourceCommit: testCommit, Result: "passed",
		Checks: map[string]bool{
			"nativeIdentity": true, "sameSourceCommit": true, "sameWorkflowRun": true,
			"sameWorkflowAttempt": true, "allStepsSucceeded": true, "qemuRejected": true,
			"finalModelEvidence": true,
		},
		Candidates: []nativeCandidate{
			{Architecture: "amd64", ModelPackageDigest: digest, FinalImageDigest: digest, PeakRSSBytes: 1},
			{Architecture: "arm64", ModelPackageDigest: digest, FinalImageDigest: digest, PeakRSSBytes: 1},
		},
	}
	supply := supplySummary{
		SchemaVersion: 1, Release: "POST-MVP-5-r2", SourceCommit: testCommit,
		ModelPackageDigest: digest, Architectures: []string{"amd64", "arm64"}, Result: "passed",
		Images: []supplyImage{{Architecture: "amd64", ImageDigest: digest}, {Architecture: "arm64", ImageDigest: digest}},
	}
	return quality, native, supply
}

func writeJSON(t *testing.T, directory, name string, value any) string {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
