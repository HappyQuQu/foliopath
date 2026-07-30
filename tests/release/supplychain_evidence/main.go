package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const releaseName = "MVP-2026-07-23"

var (
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	hashPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type sbomEvidence struct {
	ImageSHA256  string `json:"imageSHA256"`
	NPMSHA256    string `json:"npmSHA256"`
	SourceSHA256 string `json:"sourceSHA256"`
}

type vulnerabilityEvidence struct {
	Policy        string `json:"policy"`
	Total         int    `json:"total"`
	Critical      int    `json:"critical"`
	High          int    `json:"high"`
	ReportSHA256  string `json:"reportSHA256"`
	SummarySHA256 string `json:"summarySHA256"`
}

type supplyChainEvidence struct {
	SchemaVersion      int                   `json:"schemaVersion"`
	Release            string                `json:"release"`
	SourceCommit       string                `json:"sourceCommit"`
	Architecture       string                `json:"architecture"`
	OS                 string                `json:"os"`
	ImageDigest        string                `json:"imageDigest"`
	ImageSizeBytes     int64                 `json:"imageSizeBytes"`
	SBOM               sbomEvidence          `json:"sbom"`
	VulnerabilityScan  vulnerabilityEvidence `json:"vulnerabilityScan"`
	NoticesSHA256      string                `json:"noticesSHA256"`
	GLibPackageVersion string                `json:"glibPackageVersion"`
	BannedPackageCount int                   `json:"bannedPackageCount"`
	WorkflowRunID      string                `json:"workflowRunId"`
	WorkflowRunAttempt int                   `json:"workflowRunAttempt"`
	CreatedAt          string                `json:"createdAt"`
	Result             string                `json:"result"`
}

type candidateSummary struct {
	Architecture   string `json:"architecture"`
	ImageDigest    string `json:"imageDigest"`
	ImageSizeBytes int64  `json:"imageSizeBytes"`
	ImageSBOM      string `json:"imageSBOMSHA256"`
	Notices        string `json:"noticesSHA256"`
}

type pairedSummary struct {
	SchemaVersion      int                `json:"schemaVersion"`
	Release            string             `json:"release"`
	SourceCommit       string             `json:"sourceCommit"`
	WorkflowRunID      string             `json:"workflowRunId"`
	WorkflowRunAttempt int                `json:"workflowRunAttempt"`
	Policy             string             `json:"policy"`
	Critical           int                `json:"critical"`
	High               int                `json:"high"`
	GLibPackageVersion string             `json:"glibPackageVersion"`
	Candidates         []candidateSummary `json:"candidates"`
	Result             string             `json:"result"`
}

func main() {
	evidenceDirectory := flag.String("dir", "", "directory containing native supply-chain evidence")
	sourceCommit := flag.String("commit", "", "source commit shared by both artifacts")
	output := flag.String("output", "", "optional verified paired summary path")
	flag.Parse()

	summary, err := verifyEvidencePair(*evidenceDirectory, *sourceCommit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *output != "" {
		if err := writeSummary(*output, summary); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Printf("supply-chain evidence verified for %s\n", *sourceCommit)
}

func verifyEvidencePair(directory, sourceCommit string) (pairedSummary, error) {
	if directory == "" {
		return pairedSummary{}, errors.New("evidence directory is required")
	}
	if !commitPattern.MatchString(sourceCommit) {
		return pairedSummary{}, fmt.Errorf("invalid source commit %q", sourceCommit)
	}

	evidences := make([]supplyChainEvidence, 0, 2)
	for _, architecture := range []string{"amd64", "arm64"} {
		path := filepath.Join(directory, "supply-chain-evidence-"+architecture+".json")
		content, err := os.ReadFile(path)
		if err != nil {
			return pairedSummary{}, fmt.Errorf("%s evidence: %w", architecture, err)
		}
		var evidence supplyChainEvidence
		if err := json.Unmarshal(content, &evidence); err != nil {
			return pairedSummary{}, fmt.Errorf("%s evidence JSON: %w", architecture, err)
		}
		if err := validateEvidence(evidence, architecture, sourceCommit); err != nil {
			return pairedSummary{}, fmt.Errorf("%s evidence: %w", architecture, err)
		}
		evidences = append(evidences, evidence)
	}

	baseline := evidences[0]
	for _, evidence := range evidences[1:] {
		switch {
		case evidence.WorkflowRunID != baseline.WorkflowRunID:
			return pairedSummary{}, errors.New("architectures came from different workflow runs")
		case evidence.WorkflowRunAttempt != baseline.WorkflowRunAttempt:
			return pairedSummary{}, errors.New("architectures came from different workflow run attempts")
		case evidence.SBOM.SourceSHA256 != baseline.SBOM.SourceSHA256:
			return pairedSummary{}, errors.New("source SPDX hashes differ across architectures")
		case evidence.SBOM.NPMSHA256 != baseline.SBOM.NPMSHA256:
			return pairedSummary{}, errors.New("npm SPDX hashes differ across architectures")
		case evidence.GLibPackageVersion != baseline.GLibPackageVersion:
			return pairedSummary{}, errors.New("GLib package versions differ across architectures")
		}
	}

	candidates := make([]candidateSummary, 0, len(evidences))
	for _, evidence := range evidences {
		candidates = append(candidates, candidateSummary{
			Architecture:   evidence.Architecture,
			ImageDigest:    evidence.ImageDigest,
			ImageSizeBytes: evidence.ImageSizeBytes,
			ImageSBOM:      evidence.SBOM.ImageSHA256,
			Notices:        evidence.NoticesSHA256,
		})
	}

	return pairedSummary{
		SchemaVersion:      1,
		Release:            releaseName,
		SourceCommit:       sourceCommit,
		WorkflowRunID:      baseline.WorkflowRunID,
		WorkflowRunAttempt: baseline.WorkflowRunAttempt,
		Policy:             "all",
		Critical:           0,
		High:               0,
		GLibPackageVersion: baseline.GLibPackageVersion,
		Candidates:         candidates,
		Result:             "passed",
	}, nil
}

func validateEvidence(evidence supplyChainEvidence, architecture, sourceCommit string) error {
	hashes := []string{
		evidence.SBOM.ImageSHA256,
		evidence.SBOM.NPMSHA256,
		evidence.SBOM.SourceSHA256,
		evidence.VulnerabilityScan.ReportSHA256,
		evidence.VulnerabilityScan.SummarySHA256,
		evidence.NoticesSHA256,
	}
	for _, hash := range hashes {
		if !hashPattern.MatchString(hash) {
			return fmt.Errorf("invalid SHA-256 %q", hash)
		}
	}

	switch {
	case evidence.SchemaVersion != 1:
		return fmt.Errorf("schemaVersion = %d, want 1", evidence.SchemaVersion)
	case evidence.Release != releaseName:
		return fmt.Errorf("release = %q, want %q", evidence.Release, releaseName)
	case evidence.SourceCommit != sourceCommit:
		return fmt.Errorf("sourceCommit = %q, want %q", evidence.SourceCommit, sourceCommit)
	case evidence.Architecture != architecture:
		return fmt.Errorf("architecture = %q, want %q", evidence.Architecture, architecture)
	case evidence.OS != "linux":
		return fmt.Errorf("os = %q, want linux", evidence.OS)
	case !digestPattern.MatchString(evidence.ImageDigest):
		return fmt.Errorf("invalid imageDigest %q", evidence.ImageDigest)
	case evidence.ImageSizeBytes <= 0:
		return fmt.Errorf("imageSizeBytes = %d, want positive", evidence.ImageSizeBytes)
	case evidence.VulnerabilityScan.Policy != "all":
		return fmt.Errorf("policy = %q, want all", evidence.VulnerabilityScan.Policy)
	case evidence.VulnerabilityScan.Total != 0 ||
		evidence.VulnerabilityScan.Critical != 0 ||
		evidence.VulnerabilityScan.High != 0:
		return errors.New("vulnerability counts must all be zero")
	case evidence.GLibPackageVersion != "2.88.3-1":
		return fmt.Errorf("glibPackageVersion = %q, want 2.88.3-1", evidence.GLibPackageVersion)
	case evidence.BannedPackageCount != 0:
		return fmt.Errorf("bannedPackageCount = %d, want 0", evidence.BannedPackageCount)
	case evidence.WorkflowRunID == "":
		return errors.New("workflowRunId is required")
	case evidence.WorkflowRunAttempt < 1:
		return errors.New("workflowRunAttempt must be positive")
	case evidence.CreatedAt == "":
		return errors.New("createdAt is required")
	case evidence.Result != "passed":
		return fmt.Errorf("result = %q, want passed", evidence.Result)
	}
	return nil
}

func writeSummary(path string, summary pairedSummary) error {
	content, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, 0o600)
}
