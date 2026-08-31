package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testCommit     = "0123456789abcdef0123456789abcdef01234567"
	testRunID      = "123456"
	testRunAttempt = 2
)

func TestVerifyPairAcceptsMatchingNativeArtifacts(t *testing.T) {
	directory := t.TempDir()
	writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
	writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())

	summary, err := verifyPair(directory, testCommit, testRunID, testRunAttempt)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Result != "passed" || len(summary.Candidates) != 2 ||
		summary.Candidates[0].Architecture != "amd64" ||
		summary.Candidates[1].Architecture != "arm64" ||
		!summary.Checks["allStepsSucceeded"] || !summary.Checks["qemuRejected"] {
		t.Fatalf("unexpected paired summary: %+v", summary)
	}
}

func TestVerifyPairStrictModeAcceptsFinalModelEvidence(t *testing.T) {
	directory := t.TempDir()
	writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
	writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())
	writeModelArtifact(t, directory, validModelEvidence("amd64"))
	writeModelArtifact(t, directory, validModelEvidence("arm64"))

	summary, err := verifyPairWithOptions(directory, testCommit, testRunID, testRunAttempt, true)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Checks["finalModelEvidence"] || summary.Candidates[0].ModelPackageDigest == "" {
		t.Fatalf("unexpected strict summary: %+v", summary)
	}
}

func TestVerifyPairStrictModeRejectsMissingOrInvalidModelEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*modelEvidence)
		want   string
	}{
		{name: "rss", mutate: func(value *modelEvidence) { value.PeakRSSBytes = 3435973837 }, want: "peakRSSBytes"},
		{name: "numerical", mutate: func(value *modelEvidence) { value.MaxAbsError = 0.0011 }, want: "maxAbsError"},
		{name: "capacity", mutate: func(value *modelEvidence) { value.MediaCount = 99999 }, want: "100k media"},
		{name: "runtime", mutate: func(value *modelEvidence) { value.RuntimeFailureMatrixPassed = false }, want: "runtime failure"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
			writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())
			amd64 := validModelEvidence("amd64")
			test.mutate(&amd64)
			writeModelArtifact(t, directory, amd64)
			writeModelArtifact(t, directory, validModelEvidence("arm64"))
			_, err := verifyPairWithOptions(directory, testCommit, testRunID, testRunAttempt, true)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}

	t.Run("missing", func(t *testing.T) {
		directory := t.TempDir()
		writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
		writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())
		_, err := verifyPairWithOptions(directory, testCommit, testRunID, testRunAttempt, true)
		if err == nil || !strings.Contains(err.Error(), "model-evidence.json") {
			t.Fatalf("error = %v, want missing model evidence", err)
		}
	})
}

func TestVerifyPairStrictModeRejectsCrossArchitectureRankingMismatch(t *testing.T) {
	directory := t.TempDir()
	writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
	writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())
	writeModelArtifact(t, directory, validModelEvidence("amd64"))
	arm64 := validModelEvidence("arm64")
	writeModelArtifactWithRanking(t, directory, arm64, "different-ranking")
	_, err := verifyPairWithOptions(directory, testCommit, testRunID, testRunAttempt, true)
	if err == nil || !strings.Contains(err.Error(), "Top-20 sets differ") {
		t.Fatalf("error = %v, want ranking mismatch", err)
	}
}

func TestVerifyPairStrictModeRejectsTamperedOrSymlinkedReports(t *testing.T) {
	t.Run("tampered", func(t *testing.T) {
		directory := t.TempDir()
		writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
		writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())
		writeModelArtifact(t, directory, validModelEvidence("amd64"))
		writeModelArtifact(t, directory, validModelEvidence("arm64"))
		path := filepath.Join(directory, "artifact-amd64", "numeric-report.json")
		if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := verifyPairWithOptions(directory, testCommit, testRunID, testRunAttempt, true)
		if err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
			t.Fatalf("error = %v, want hash mismatch", err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		directory := t.TempDir()
		writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
		writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())
		writeModelArtifact(t, directory, validModelEvidence("amd64"))
		writeModelArtifact(t, directory, validModelEvidence("arm64"))
		artifactDirectory := filepath.Join(directory, "artifact-amd64")
		target := filepath.Join(artifactDirectory, "quality-target.json")
		if err := os.Rename(filepath.Join(artifactDirectory, "quality-summary.json"), target); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(artifactDirectory, "quality-summary.json")); err != nil {
			t.Fatal(err)
		}
		_, err := verifyPairWithOptions(directory, testCommit, testRunID, testRunAttempt, true)
		if err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("error = %v, want symlink rejection", err)
		}
	})
}

func TestVerifyPairRejectsFailedCapacity(t *testing.T) {
	directory := t.TempDir()
	failed := validOutcomes()
	failed.Capacity = "failure"
	failed.Complete = false
	writeArtifact(t, directory, validIdentity("amd64"), failed)
	writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())

	_, err := verifyPair(directory, testCommit, testRunID, testRunAttempt)
	if err == nil || !strings.Contains(err.Error(), "capacity outcome") {
		t.Fatalf("verifyPair() error = %v, want capacity failure", err)
	}
}

func TestVerifyPairRejectsQEMUOrWrongRunner(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*identityEvidence)
		want   string
	}{
		{name: "qemu", mutate: func(identity *identityEvidence) { identity.QEMUAllowed = true }, want: "QEMU"},
		{name: "runner", mutate: func(identity *identityEvidence) { identity.Machine = "aarch64" }, want: "native runner"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			amd64 := validIdentity("amd64")
			test.mutate(&amd64)
			writeArtifact(t, directory, amd64, validOutcomes())
			writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())
			_, err := verifyPair(directory, testCommit, testRunID, testRunAttempt)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("verifyPair() error = %v, want %s failure", err, test.want)
			}
		})
	}
}

func TestVerifyPairRejectsCrossRunAndMissingArchitecture(t *testing.T) {
	t.Run("cross run", func(t *testing.T) {
		directory := t.TempDir()
		writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
		arm64 := validIdentity("arm64")
		arm64.WorkflowRunID = "999"
		writeArtifact(t, directory, arm64, validOutcomes())
		_, err := verifyPair(directory, testCommit, testRunID, testRunAttempt)
		if err == nil || !strings.Contains(err.Error(), "workflow run mismatch") {
			t.Fatalf("verifyPair() error = %v, want cross-run failure", err)
		}
	})
	t.Run("missing architecture", func(t *testing.T) {
		directory := t.TempDir()
		writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
		_, err := verifyPair(directory, testCommit, testRunID, testRunAttempt)
		if err == nil || !strings.Contains(err.Error(), "artifact count") {
			t.Fatalf("verifyPair() error = %v, want missing artifact failure", err)
		}
	})
}

func TestWriteSummaryPublishesValidatedPair(t *testing.T) {
	directory := t.TempDir()
	writeArtifact(t, directory, validIdentity("amd64"), validOutcomes())
	writeArtifact(t, directory, validIdentity("arm64"), validOutcomes())
	summary, err := verifyPair(directory, testCommit, testRunID, testRunAttempt)
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
	var decoded pairedSummary
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SourceCommit != testCommit || decoded.WorkflowRunID != testRunID || decoded.Result != "passed" {
		t.Fatalf("unexpected written summary: %+v", decoded)
	}
}

func validIdentity(architecture string) identityEvidence {
	identity := identityEvidence{
		SchemaVersion: 1, SourceCommit: testCommit, WorkflowRunID: testRunID,
		WorkflowRunAttempt: testRunAttempt, GOOS: "linux", GOARCH: architecture,
		QEMUAllowed: false, CreatedAt: "2026-08-31T00:00:00Z",
	}
	if architecture == "amd64" {
		identity.RunnerLabel = "ubuntu-24.04"
		identity.Machine = "x86_64"
		identity.Docker = "linux/x86_64"
	} else {
		identity.RunnerLabel = "ubuntu-24.04-arm"
		identity.Machine = "aarch64"
		identity.Docker = "linux/aarch64"
	}
	return identity
}

func validOutcomes() outcomeEvidence {
	return outcomeEvidence{
		Identity: "success", Repository: "success", Libvips: "success",
		SearchMatrix: "success", Capacity: "success", Complete: true,
	}
}

func validModelEvidence(architecture string) modelEvidence {
	hash := strings.Repeat("a", 64)
	return modelEvidence{
		SchemaVersion: 1, SourceCommit: testCommit, Architecture: architecture,
		ModelPackageDigest: "sha256:" + hash, FinalImageDigest: "sha256:" + hash,
		QualityDatasetSHA256: hash, RankingFixtureSHA256: hash,
		TieFixtureSHA256: hash,
		MediaCount:       100000, DirectoryCount: 10000,
		QualityPassed: true, ReferenceRankingPassed: true, TieFixturePassed: true,
		MaxAbsError: 0.001, PeakRSSBytes: 3_000_000_000,
		SemanticP95Millis: 700, SemanticP99Millis: 1400,
		BrowseRegressionPercent: 20, DerivedBytes: 500_000_000,
		IndexRebuildPassed: true, RestartRecoveryPassed: true,
		RuntimeFailureMatrixPassed: true, Result: "passed",
	}
}

func writeArtifact(t *testing.T, root string, identity identityEvidence, outcomes outcomeEvidence) {
	t.Helper()
	directory := filepath.Join(root, "artifact-"+identity.GOARCH)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]any{"identity.json": identity, "outcomes.json": outcomes} {
		content, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func writeModelArtifact(t *testing.T, root string, evidence modelEvidence) {
	t.Helper()
	writeModelArtifactWithRanking(t, root, evidence, "ranking")
}

func writeModelArtifactWithRanking(t *testing.T, root string, evidence modelEvidence, ranking string) {
	t.Helper()
	directory := filepath.Join(root, "artifact-"+evidence.Architecture)
	evidence.QualitySummary = writeEvidenceReport(t, directory, "quality-summary.json", "quality")
	evidence.RankingTop20 = writeEvidenceReport(t, directory, "ranking-top20.json", ranking)
	evidence.NumericReport = writeEvidenceReport(t, directory, "numeric-report.json", "numeric-"+evidence.Architecture)
	evidence.RuntimeReport = writeEvidenceReport(t, directory, "runtime-report.json", "runtime-"+evidence.Architecture)
	evidence.IndexReport = writeEvidenceReport(t, directory, "index-report.json", "index-"+evidence.Architecture)
	content, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "model-evidence.json"), content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeEvidenceReport(t *testing.T, directory, name, content string) artifactEvidence {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(content))
	return artifactEvidence{Path: name, SHA256: hex.EncodeToString(sum[:])}
}
