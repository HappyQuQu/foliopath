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

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestVerifyManifestAcceptsCompleteNativeEvidence(t *testing.T) {
	directory := t.TempDir()
	evidence := validEvidence(t, directory)
	path := writeManifest(t, directory, evidence)
	summary, err := verifyManifest(path, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Result != "passed" || len(summary.Architectures) != 2 || len(summary.Components) != 5 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestVerifyManifestRejectsMissingOrDuplicateFaceComponentRole(t *testing.T) {
	directory := t.TempDir()
	evidence := validEvidence(t, directory)
	evidence.Components = evidence.Components[:len(evidence.Components)-1]
	if _, err := verifyManifest(writeManifest(t, directory, evidence), testCommit); err == nil ||
		!strings.Contains(err.Error(), `required component role "face_embedder"`) {
		t.Fatalf("missing role error=%v", err)
	}
	evidence = validEvidence(t, directory)
	evidence.Components[len(evidence.Components)-1].Role = "face_detector"
	if _, err := verifyManifest(writeManifest(t, directory, evidence), testCommit); err == nil ||
		!strings.Contains(err.Error(), "duplicate component role") {
		t.Fatalf("duplicate role error=%v", err)
	}
}

func TestVerifyManifestRejectsIncompleteSBOM(t *testing.T) {
	directory := t.TempDir()
	evidence := validEvidence(t, directory)
	evidence.Architectures[0].SBOMComplete = false
	_, err := verifyManifest(writeManifest(t, directory, evidence), testCommit)
	if err == nil || !strings.Contains(err.Error(), "SBOM is incomplete") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyManifestRejectsUnsignedBlockingFindings(t *testing.T) {
	directory := t.TempDir()
	evidence := validEvidence(t, directory)
	evidence.Architectures[0].Vulnerabilities.High = 1
	_, err := verifyManifest(writeManifest(t, directory, evidence), testCommit)
	if err == nil || !strings.Contains(err.Error(), "require signed VEX") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyManifestAcceptsSignedVEX(t *testing.T) {
	directory := t.TempDir()
	evidence := validEvidence(t, directory)
	vex := artifact(t, directory, "amd64.vex.json", "scoped-vex")
	evidence.Architectures[0].Vulnerabilities.High = 1
	evidence.Architectures[0].Vulnerabilities.VEX = &vex
	evidence.Architectures[0].Vulnerabilities.SecurityRef = "SEC-APPROVAL/2026-0001"
	if _, err := verifyManifest(writeManifest(t, directory, evidence), testCommit); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyManifestRejectsArtifactHashMismatch(t *testing.T) {
	directory := t.TempDir()
	evidence := validEvidence(t, directory)
	evidence.ModelPackage.SHA256 = strings.Repeat("b", 64)
	evidence.ModelPackageDigest = "sha256:" + evidence.ModelPackage.SHA256
	_, err := verifyManifest(writeManifest(t, directory, evidence), testCommit)
	if err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyManifestRejectsSymlinkArtifact(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	evidence := validEvidence(t, directory)
	sum := sha256.Sum256([]byte("target"))
	evidence.Catalog = artifactEvidence{Path: "link", SHA256: hex.EncodeToString(sum[:])}
	_, err := verifyManifest(writeManifest(t, directory, evidence), testCommit)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyManifestRejectsAmbiguousJSON(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{name: "unknown", mutate: func(value string) string { return strings.Replace(value, "{", `{"unknown":true,`, 1) }, want: "unknown field"},
		{name: "duplicate", mutate: func(value string) string {
			return strings.Replace(value, `"result":"passed"`, `"result":"passed","result":"failed"`, 1)
		}, want: "duplicate JSON key"},
		{name: "trailing", mutate: func(value string) string { return value + ` {}` }, want: "trailing JSON value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := writeManifest(t, directory, validEvidence(t, directory))
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(test.mutate(string(content))), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := verifyManifest(path, testCommit); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v want=%q", err, test.want)
			}
		})
	}
}

func TestVerifyManifestRejectsSymlinkInputDocument(t *testing.T) {
	directory := t.TempDir()
	target := writeManifest(t, directory, validEvidence(t, directory))
	link := filepath.Join(t.TempDir(), "evidence.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyManifest(link, testCommit); err == nil || !strings.Contains(err.Error(), "non-symlink regular file") {
		t.Fatalf("error=%v", err)
	}
}

func validEvidence(t *testing.T, directory string) supplyChainEvidence {
	t.Helper()
	model := artifact(t, directory, "model.foliomodel", "model")
	componentRoles := []struct{ name, role string }{
		{name: "onnxruntime", role: "inference_runtime"},
		{name: "sentencepiece", role: "semantic_tokenizer"},
		{name: "siglip", role: "semantic_model"},
		{name: "reviewed-detector", role: "face_detector"},
		{name: "reviewed-embedder", role: "face_embedder"},
	}
	components := make([]componentEvidence, 0, len(componentRoles))
	for _, value := range componentRoles {
		components = append(components, componentEvidence{
			Name: value.name, Role: value.role, Version: "1.0.0", License: "Apache-2.0",
			RedistributionApproved: true, ApprovalRef: "LEGAL-APPROVAL/2026-" + value.name,
			Notices: artifact(t, directory, value.name+".NOTICE", value.name+" notice"),
		})
	}
	architectures := make([]architectureEvidence, 0, 2)
	for _, architecture := range []string{"amd64", "arm64"} {
		architectures = append(architectures, architectureEvidence{
			Architecture: architecture, OS: "linux", Native: true,
			ImageDigest:       "sha256:" + strings.Repeat("a", 64),
			SBOM:              artifact(t, directory, architecture+".cdx.json", "sbom "+architecture),
			SBOMComplete:      true,
			Provenance:        artifact(t, directory, architecture+".provenance.json", "provenance "+architecture),
			SignatureVerify:   artifact(t, directory, architecture+".verify.json", "verified "+architecture),
			SignatureVerified: true,
			Vulnerabilities: vulnerabilityEvidence{
				Report: artifact(t, directory, architecture+".vulnerabilities.json", "scan "+architecture),
			},
		})
	}
	return supplyChainEvidence{
		SchemaVersion: 2, Release: releaseName, SourceCommit: testCommit,
		Catalog:      artifact(t, directory, "catalog.json", "catalog"),
		ModelPackage: model, ModelPackageDigest: "sha256:" + model.SHA256,
		Components: components, Architectures: architectures,
		Approvals: approvals{
			Security: "SEC-APPROVAL/2026-0001", Compliance: "LEGAL-APPROVAL/2026-0001",
			Release: "REL-APPROVAL/2026-0001", Inference: "ML-APPROVAL/2026-0001",
		},
		Result: "passed",
	}
}

func artifact(t *testing.T, directory, name, content string) artifactEvidence {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(content))
	return artifactEvidence{Path: name, SHA256: hex.EncodeToString(sum[:])}
}

func writeManifest(t *testing.T, directory string, evidence supplyChainEvidence) string {
	t.Helper()
	content, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "evidence.json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
