package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestVerifyEvidenceBindsAllFourPassedSummaries(t *testing.T) {
	directory := t.TempDir()
	qualityPath, faceQualityPath, nativePath, supplyPath := writeValidSummaries(t, directory)
	summary, err := verifyEvidence(qualityPath, faceQualityPath, nativePath, supplyPath, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Result != "passed" || !summary.Checks["sameModelPackage"] || !summary.Checks["faceQualityPassed"] || len(summary.Inputs) != 4 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestVerifyEvidenceRejectsMismatchedOrIncompleteEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*qualitySummary, *faceQualitySummary, *nativeSummary, *supplySummary)
		want   string
	}{
		{name: "model", mutate: func(_ *qualitySummary, _ *faceQualitySummary, native *nativeSummary, _ *supplySummary) {
			native.Candidates[0].ModelPackageDigest = "sha256:" + strings.Repeat("b", 64)
		}, want: "model package digests differ"},
		{name: "face model", mutate: func(_ *qualitySummary, faceQuality *faceQualitySummary, _ *nativeSummary, _ *supplySummary) {
			faceQuality.ModelPackageDigest = "sha256:" + strings.Repeat("b", 64)
		}, want: "model package digests differ"},
		{name: "image", mutate: func(_ *qualitySummary, _ *faceQualitySummary, _ *nativeSummary, supply *supplySummary) {
			supply.Images[0].ImageDigest = "sha256:" + strings.Repeat("b", 64)
		}, want: "final image digest differs"},
		{name: "quality", mutate: func(quality *qualitySummary, _ *faceQualitySummary, _ *nativeSummary, _ *supplySummary) {
			quality.GatePass = false
		}, want: "quality gate did not pass"},
		{name: "face quality", mutate: func(_ *qualitySummary, faceQuality *faceQualitySummary, _ *nativeSummary, _ *supplySummary) {
			faceQuality.GatePass = false
		}, want: "face quality gate did not pass"},
		{name: "face group assignment", mutate: func(_ *qualitySummary, faceQuality *faceQualitySummary, _ *nativeSummary, _ *supplySummary) {
			faceQuality.GroupAssignmentAllowed = false
		}, want: "group assignment"},
		{name: "baseline native", mutate: func(_ *qualitySummary, _ *faceQualitySummary, native *nativeSummary, _ *supplySummary) {
			native.Checks["finalModelEvidence"] = false
		}, want: "finalModelEvidence"},
		{name: "missing face capacity", mutate: func(_ *qualitySummary, _ *faceQualitySummary, native *nativeSummary, _ *supplySummary) {
			native.Checks["faceSyntheticCapacity"] = false
		}, want: "faceSyntheticCapacity"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			quality, faceQuality, native, supply := validSummaries()
			test.mutate(&quality, &faceQuality, &native, &supply)
			qualityPath := writeJSON(t, directory, "quality.json", quality)
			faceQualityPath := writeJSON(t, directory, "face-quality.json", faceQuality)
			nativePath := writeJSON(t, directory, "native.json", native)
			supplyPath := writeJSON(t, directory, "supply.json", supply)
			_, err := verifyEvidence(qualityPath, faceQualityPath, nativePath, supplyPath, testCommit)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestReadSummaryRejectsAmbiguousJSON(t *testing.T) {
	quality, _, _, _ := validSummaries()
	content, err := json.Marshal(quality)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		content []byte
		want    string
	}{
		{name: "unknown field", content: []byte(strings.Replace(string(content), "{", `{"unknown":true,`, 1)), want: "unknown field"},
		{name: "duplicate key", content: []byte(strings.Replace(string(content), `"gate_pass":true`, `"gate_pass":true,"gate_pass":false`, 1)), want: "duplicate JSON key"},
		{name: "trailing value", content: append(append([]byte{}, content...), []byte(` {}`)...), want: "trailing JSON value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "summary.json")
			if err := os.WriteFile(path, test.content, 0o600); err != nil {
				t.Fatal(err)
			}
			var decoded qualitySummary
			if _, err := readSummary(path, &decoded); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v want=%q", err, test.want)
			}
		})
	}
}

func writeValidSummaries(t *testing.T, directory string) (string, string, string, string) {
	t.Helper()
	quality, faceQuality, native, supply := validSummaries()
	return writeJSON(t, directory, "quality.json", quality),
		writeJSON(t, directory, "face-quality.json", faceQuality),
		writeJSON(t, directory, "native.json", native),
		writeJSON(t, directory, "supply.json", supply)
}

func validSummaries() (qualitySummary, faceQualitySummary, nativeSummary, supplySummary) {
	digest := "sha256:" + strings.Repeat("a", 64)
	hash := strings.Repeat("a", 64)
	quality := qualitySummary{
		SchemaVersion: 1, SourceCommit: testCommit, DatasetManifestSHA256: hash,
		ModelPackageDigest: digest, GatePass: true,
		Approvals: qualityApprovals{Product: "PRODUCT-APPROVAL-001", ML: "ML-APPROVAL-001", QA: "QA-APPROVAL-001"},
	}
	faceQuality := faceQualitySummary{
		SchemaVersion: 1, SourceCommit: testCommit, DatasetManifestSHA256: strings.Repeat("b", 64),
		ModelPackageDigest: digest, GatePass: true, GroupAssignmentAllowed: true,
		Approvals: faceQualityApprovals{Product: "PRODUCT-APPROVAL-001", ML: "ML-APPROVAL-001", QA: "QA-APPROVAL-001", Privacy: "PRIVACY-APPROVAL-001", Compliance: "COMPLIANCE-APPROVAL-001"},
	}
	native := nativeSummary{
		SchemaVersion: 1, Feature: "FTR-INT-001", SourceCommit: testCommit,
		WorkflowRunID: "123456", WorkflowRunAttempt: 1, Result: "passed",
		Checks: map[string]bool{
			"nativeIdentity": true, "sameSourceCommit": true, "sameWorkflowRun": true,
			"sameWorkflowAttempt": true, "allStepsSucceeded": true, "qemuRejected": true,
			"faceCandidateNativePreflight": true, "faceSyntheticCapacity": true,
			"finalModelEvidence": true,
		},
		Candidates: []nativeCandidate{
			{Architecture: "amd64", ModelPackageDigest: digest, FinalImageDigest: digest, PeakRSSBytes: 1, FaceCandidateCount: 3, FaceCandidateSHA256: hash},
			{Architecture: "arm64", ModelPackageDigest: digest, FinalImageDigest: digest, PeakRSSBytes: 1, FaceCandidateCount: 3, FaceCandidateSHA256: hash},
		},
	}
	supply := supplySummary{
		SchemaVersion: 1, Release: "POST-MVP-5-r2", SourceCommit: testCommit,
		ModelPackageDigest: digest, Architectures: []string{"amd64", "arm64"}, Result: "passed",
		Images: []supplyImage{{Architecture: "amd64", ImageDigest: digest}, {Architecture: "arm64", ImageDigest: digest}},
	}
	return quality, faceQuality, native, supply
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
