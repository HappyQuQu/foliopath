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
)

type imageEvidence struct {
	SchemaVersion      int    `json:"schemaVersion"`
	Release            string `json:"release"`
	SourceCommit       string `json:"sourceCommit"`
	Architecture       string `json:"architecture"`
	OS                 string `json:"os"`
	ImageDigest        string `json:"imageDigest"`
	ImageSizeBytes     int64  `json:"imageSizeBytes"`
	SmokeSuite         string `json:"smokeSuite"`
	Result             string `json:"result"`
	WorkflowRunID      string `json:"workflowRunId"`
	WorkflowRunAttempt int    `json:"workflowRunAttempt"`
	CreatedAt          string `json:"createdAt"`
}

func main() {
	evidenceDirectory := flag.String("dir", "", "directory containing native image evidence")
	sourceCommit := flag.String("commit", "", "source commit shared by both artifacts")
	flag.Parse()

	if err := verifyEvidence(*evidenceDirectory, *sourceCommit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("release image evidence verified for %s\n", *sourceCommit)
}

func verifyEvidence(directory, sourceCommit string) error {
	if directory == "" {
		return errors.New("evidence directory is required")
	}
	if !commitPattern.MatchString(sourceCommit) {
		return fmt.Errorf("invalid source commit %q", sourceCommit)
	}

	architectures := []string{"amd64", "arm64"}
	runIDs := make([]string, 0, len(architectures))
	for _, architecture := range architectures {
		path := filepath.Join(directory, "release-image-"+architecture+".json")
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s evidence: %w", architecture, err)
		}
		var evidence imageEvidence
		if err := json.Unmarshal(content, &evidence); err != nil {
			return fmt.Errorf("%s evidence JSON: %w", architecture, err)
		}
		if err := validateEvidence(evidence, architecture, sourceCommit); err != nil {
			return fmt.Errorf("%s evidence: %w", architecture, err)
		}
		runIDs = append(runIDs, evidence.WorkflowRunID)
	}
	if runIDs[0] != runIDs[1] {
		return fmt.Errorf("architectures came from different workflow runs: %v", runIDs)
	}
	return nil
}

func validateEvidence(evidence imageEvidence, architecture, sourceCommit string) error {
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
	case evidence.SmokeSuite != "tests/release/image_smoke.sh":
		return fmt.Errorf("unexpected smokeSuite %q", evidence.SmokeSuite)
	case evidence.Result != "passed":
		return fmt.Errorf("result = %q, want passed", evidence.Result)
	case evidence.WorkflowRunID == "":
		return errors.New("workflowRunId is required")
	case evidence.WorkflowRunAttempt < 1:
		return fmt.Errorf("workflowRunAttempt = %d, want positive", evidence.WorkflowRunAttempt)
	case evidence.CreatedAt == "":
		return errors.New("createdAt is required")
	}
	return nil
}
