package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	hashPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	runIDPattern  = regexp.MustCompile(`^[1-9][0-9]*$`)
)

type fixtureEvidence struct {
	SourceSHA256       string `json:"sourceSHA256"`
	FrameCount         int    `json:"frameCount"`
	Columns            int    `json:"columns"`
	Rows               int    `json:"rows"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	DecodedPixelSHA256 string `json:"decodedPixelSHA256"`
}

type cacheRepairEvidence struct {
	InitialStatus      int  `json:"initialStatus"`
	MissingStatus      int  `json:"missingStatus"`
	RebuiltStatus      int  `json:"rebuiltStatus"`
	DecodedPixelsMatch bool `json:"decodedPixelsMatch"`
}

type resourceLimitEvidence struct {
	CPUs        int   `json:"cpus"`
	MemoryBytes int64 `json:"memoryBytes"`
}

type storyboardEvidence struct {
	SchemaVersion          int                   `json:"schemaVersion"`
	Feature                string                `json:"feature"`
	SourceCommit           string                `json:"sourceCommit"`
	Architecture           string                `json:"architecture"`
	OS                     string                `json:"os"`
	ImageDigest            string                `json:"imageDigest"`
	ImageSizeBytes         int64                 `json:"imageSizeBytes"`
	FFmpegVersion          string                `json:"ffmpegVersion"`
	Fixture                fixtureEvidence       `json:"fixture"`
	CacheRepair            cacheRepairEvidence   `json:"cacheRepair"`
	OriginalMediaUnchanged bool                  `json:"originalMediaUnchanged"`
	ResourceLimit          resourceLimitEvidence `json:"resourceLimit"`
	SmokeSuite             string                `json:"smokeSuite"`
	Result                 string                `json:"result"`
	WorkflowRunID          string                `json:"workflowRunId"`
	WorkflowRunAttempt     int                   `json:"workflowRunAttempt"`
	CreatedAt              string                `json:"createdAt"`
}

type candidateSummary struct {
	Architecture   string `json:"architecture"`
	OS             string `json:"os"`
	ImageDigest    string `json:"imageDigest"`
	ImageSizeBytes int64  `json:"imageSizeBytes"`
	CreatedAt      string `json:"createdAt"`
}

type pairedEvidenceSummary struct {
	SchemaVersion      int                   `json:"schemaVersion"`
	Feature            string                `json:"feature"`
	SourceCommit       string                `json:"sourceCommit"`
	WorkflowRunID      string                `json:"workflowRunId"`
	WorkflowRunAttempt int                   `json:"workflowRunAttempt"`
	FFmpegVersion      string                `json:"ffmpegVersion"`
	Fixture            fixtureEvidence       `json:"fixture"`
	ResourceLimit      resourceLimitEvidence `json:"resourceLimit"`
	Candidates         []candidateSummary    `json:"candidates"`
	Checks             pairedChecks          `json:"checks"`
	Result             string                `json:"result"`
}

type pairedChecks struct {
	SourceCommitMatches    bool `json:"sourceCommitMatches"`
	WorkflowRunMatches     bool `json:"workflowRunMatches"`
	WorkflowAttemptMatches bool `json:"workflowAttemptMatches"`
	FFmpegVersionMatches   bool `json:"ffmpegVersionMatches"`
	FixtureMatches         bool `json:"fixtureMatches"`
	DecodedPixelsMatch     bool `json:"decodedPixelsMatch"`
	CacheRepairPassed      bool `json:"cacheRepairPassed"`
	OriginalMediaUnchanged bool `json:"originalMediaUnchanged"`
	ResourceLimitsMatch    bool `json:"resourceLimitsMatch"`
}

func main() {
	evidenceDirectory := flag.String("dir", "", "directory containing native storyboard evidence")
	sourceCommit := flag.String("commit", "", "source commit shared by both artifacts")
	output := flag.String("output", "", "optional path for the verified paired evidence summary")
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
	fmt.Printf("storyboard evidence verified for %s\n", *sourceCommit)
}

func verifyEvidence(directory, sourceCommit string) error {
	_, err := verifyEvidencePair(directory, sourceCommit)
	return err
}

func verifyEvidencePair(directory, sourceCommit string) (pairedEvidenceSummary, error) {
	if directory == "" {
		return pairedEvidenceSummary{}, errors.New("evidence directory is required")
	}
	if !commitPattern.MatchString(sourceCommit) {
		return pairedEvidenceSummary{}, fmt.Errorf("invalid source commit %q", sourceCommit)
	}

	evidences := make([]storyboardEvidence, 0, 2)
	for _, architecture := range []string{"amd64", "arm64"} {
		path := filepath.Join(directory, "storyboard-evidence-"+architecture+".json")
		content, err := os.ReadFile(path)
		if err != nil {
			return pairedEvidenceSummary{}, fmt.Errorf("%s evidence: %w", architecture, err)
		}
		var evidence storyboardEvidence
		if err := json.Unmarshal(content, &evidence); err != nil {
			return pairedEvidenceSummary{}, fmt.Errorf(
				"%s evidence JSON: %w",
				architecture,
				err,
			)
		}
		if err := validateEvidence(evidence, architecture, sourceCommit); err != nil {
			return pairedEvidenceSummary{}, fmt.Errorf("%s evidence: %w", architecture, err)
		}
		evidences = append(evidences, evidence)
	}

	baseline := evidences[0]
	for _, evidence := range evidences[1:] {
		if evidence.WorkflowRunID != baseline.WorkflowRunID {
			return pairedEvidenceSummary{}, fmt.Errorf(
				"architectures came from different workflow runs: %q and %q",
				baseline.WorkflowRunID,
				evidence.WorkflowRunID,
			)
		}
		if evidence.WorkflowRunAttempt != baseline.WorkflowRunAttempt {
			return pairedEvidenceSummary{}, fmt.Errorf(
				"architectures came from different workflow run attempts: %d and %d",
				baseline.WorkflowRunAttempt,
				evidence.WorkflowRunAttempt,
			)
		}
		if evidence.FFmpegVersion != baseline.FFmpegVersion {
			return pairedEvidenceSummary{}, errors.New(
				"FFmpeg versions differ across architectures",
			)
		}
		if evidence.Fixture.SourceSHA256 != baseline.Fixture.SourceSHA256 {
			return pairedEvidenceSummary{}, errors.New(
				"source fixture hashes differ across architectures",
			)
		}
		if evidence.Fixture.DecodedPixelSHA256 != baseline.Fixture.DecodedPixelSHA256 {
			return pairedEvidenceSummary{}, errors.New(
				"decoded storyboard pixels differ across architectures",
			)
		}
	}

	candidates := make([]candidateSummary, 0, len(evidences))
	for _, evidence := range evidences {
		candidates = append(candidates, candidateSummary{
			Architecture:   evidence.Architecture,
			OS:             evidence.OS,
			ImageDigest:    evidence.ImageDigest,
			ImageSizeBytes: evidence.ImageSizeBytes,
			CreatedAt:      evidence.CreatedAt,
		})
	}
	return pairedEvidenceSummary{
		SchemaVersion:      1,
		Feature:            "FTR-VID-001",
		SourceCommit:       sourceCommit,
		WorkflowRunID:      baseline.WorkflowRunID,
		WorkflowRunAttempt: baseline.WorkflowRunAttempt,
		FFmpegVersion:      baseline.FFmpegVersion,
		Fixture:            baseline.Fixture,
		ResourceLimit:      baseline.ResourceLimit,
		Candidates:         candidates,
		Checks: pairedChecks{
			SourceCommitMatches:    true,
			WorkflowRunMatches:     true,
			WorkflowAttemptMatches: true,
			FFmpegVersionMatches:   true,
			FixtureMatches:         true,
			DecodedPixelsMatch:     true,
			CacheRepairPassed:      true,
			OriginalMediaUnchanged: true,
			ResourceLimitsMatch:    true,
		},
		Result: "passed",
	}, nil
}

func writeSummary(path string, summary pairedEvidenceSummary) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create paired summary directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return fmt.Errorf("create paired summary: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect paired summary: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		temporary.Close()
		return fmt.Errorf("encode paired summary: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close paired summary: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish paired summary: %w", err)
	}
	return nil
}

func validateEvidence(
	evidence storyboardEvidence,
	architecture string,
	sourceCommit string,
) error {
	switch {
	case evidence.SchemaVersion != 1:
		return fmt.Errorf("schemaVersion = %d, want 1", evidence.SchemaVersion)
	case evidence.Feature != "FTR-VID-001":
		return fmt.Errorf("feature = %q, want FTR-VID-001", evidence.Feature)
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
	case !strings.HasPrefix(evidence.FFmpegVersion, "ffmpeg version 7.1.5"):
		return fmt.Errorf("unexpected FFmpeg version %q", evidence.FFmpegVersion)
	case !hashPattern.MatchString(evidence.Fixture.SourceSHA256):
		return fmt.Errorf("invalid source fixture hash %q", evidence.Fixture.SourceSHA256)
	case evidence.Fixture.FrameCount != 10 ||
		evidence.Fixture.Columns != 5 ||
		evidence.Fixture.Rows != 2 ||
		evidence.Fixture.Width != 1600 ||
		evidence.Fixture.Height != 360:
		return fmt.Errorf("unexpected fixture layout: %+v", evidence.Fixture)
	case !hashPattern.MatchString(evidence.Fixture.DecodedPixelSHA256):
		return fmt.Errorf(
			"invalid decoded storyboard hash %q",
			evidence.Fixture.DecodedPixelSHA256,
		)
	case evidence.CacheRepair.InitialStatus != 200 ||
		evidence.CacheRepair.MissingStatus != 202 ||
		evidence.CacheRepair.RebuiltStatus != 200 ||
		!evidence.CacheRepair.DecodedPixelsMatch:
		return fmt.Errorf("cache repair evidence is incomplete: %+v", evidence.CacheRepair)
	case !evidence.OriginalMediaUnchanged:
		return errors.New("originalMediaUnchanged must be true")
	case evidence.ResourceLimit.CPUs != 4 ||
		evidence.ResourceLimit.MemoryBytes != 4*1024*1024*1024:
		return fmt.Errorf("unexpected resource limit: %+v", evidence.ResourceLimit)
	case evidence.SmokeSuite != "tests/release/storyboard_vertical_smoke.sh":
		return fmt.Errorf("unexpected smokeSuite %q", evidence.SmokeSuite)
	case evidence.Result != "passed":
		return fmt.Errorf("result = %q, want passed", evidence.Result)
	case !runIDPattern.MatchString(evidence.WorkflowRunID):
		return fmt.Errorf("invalid workflowRunId %q", evidence.WorkflowRunID)
	case evidence.WorkflowRunAttempt < 1:
		return fmt.Errorf("workflowRunAttempt = %d, want positive", evidence.WorkflowRunAttempt)
	case !isRFC3339(evidence.CreatedAt):
		return fmt.Errorf("invalid createdAt %q", evidence.CreatedAt)
	}
	return nil
}

func isRFC3339(value string) bool {
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}
