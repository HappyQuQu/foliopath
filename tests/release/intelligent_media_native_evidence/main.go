package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
	runIDPattern  = regexp.MustCompile(`^[1-9][0-9]*$`)
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	hashPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type identityEvidence struct {
	SchemaVersion      int    `json:"schemaVersion"`
	SourceCommit       string `json:"sourceCommit"`
	WorkflowRunID      string `json:"workflowRunId"`
	WorkflowRunAttempt int    `json:"workflowRunAttempt"`
	RunnerLabel        string `json:"runnerLabel"`
	Machine            string `json:"machine"`
	GOOS               string `json:"goos"`
	GOARCH             string `json:"goarch"`
	Docker             string `json:"docker"`
	QEMUAllowed        bool   `json:"qemuAllowed"`
	CreatedAt          string `json:"createdAt"`
}

type outcomeEvidence struct {
	Identity     string `json:"identity"`
	Repository   string `json:"repository"`
	Libvips      string `json:"libvips"`
	SearchMatrix string `json:"searchMatrix"`
	Capacity     string `json:"capacity"`
	Complete     bool   `json:"complete"`
}

type artifactEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type modelEvidence struct {
	SchemaVersion              int              `json:"schemaVersion"`
	SourceCommit               string           `json:"sourceCommit"`
	Architecture               string           `json:"architecture"`
	ModelPackageDigest         string           `json:"modelPackageDigest"`
	FinalImageDigest           string           `json:"finalImageDigest"`
	QualityDatasetSHA256       string           `json:"qualityDatasetSHA256"`
	QualitySummary             artifactEvidence `json:"qualitySummary"`
	RankingFixtureSHA256       string           `json:"rankingFixtureSHA256"`
	RankingTop20               artifactEvidence `json:"rankingTop20"`
	TieFixtureSHA256           string           `json:"tieFixtureSHA256"`
	NumericReport              artifactEvidence `json:"numericReport"`
	RuntimeReport              artifactEvidence `json:"runtimeReport"`
	IndexReport                artifactEvidence `json:"indexReport"`
	MediaCount                 int              `json:"mediaCount"`
	DirectoryCount             int              `json:"directoryCount"`
	QualityPassed              bool             `json:"qualityPassed"`
	ReferenceRankingPassed     bool             `json:"referenceRankingPassed"`
	TieFixturePassed           bool             `json:"tieFixturePassed"`
	MaxAbsError                float64          `json:"maxAbsError"`
	PeakRSSBytes               int64            `json:"peakRSSBytes"`
	SemanticP95Millis          float64          `json:"semanticP95Millis"`
	SemanticP99Millis          float64          `json:"semanticP99Millis"`
	BrowseRegressionPercent    float64          `json:"browseRegressionPercent"`
	DerivedBytes               int64            `json:"derivedBytes"`
	IndexRebuildPassed         bool             `json:"indexRebuildPassed"`
	RestartRecoveryPassed      bool             `json:"restartRecoveryPassed"`
	RuntimeFailureMatrixPassed bool             `json:"runtimeFailureMatrixPassed"`
	Result                     string           `json:"result"`
}

type candidateSummary struct {
	Architecture       string `json:"architecture"`
	RunnerLabel        string `json:"runnerLabel"`
	Machine            string `json:"machine"`
	Docker             string `json:"docker"`
	CreatedAt          string `json:"createdAt"`
	ModelPackageDigest string `json:"modelPackageDigest,omitempty"`
	FinalImageDigest   string `json:"finalImageDigest,omitempty"`
	PeakRSSBytes       int64  `json:"peakRSSBytes,omitempty"`
}

type pairedSummary struct {
	SchemaVersion      int                `json:"schemaVersion"`
	Feature            string             `json:"feature"`
	SourceCommit       string             `json:"sourceCommit"`
	WorkflowRunID      string             `json:"workflowRunId"`
	WorkflowRunAttempt int                `json:"workflowRunAttempt"`
	Candidates         []candidateSummary `json:"candidates"`
	Checks             map[string]bool    `json:"checks"`
	Result             string             `json:"result"`
}

func main() {
	directory := flag.String("dir", "", "directory containing downloaded native evidence artifacts")
	commit := flag.String("commit", "", "source commit shared by both artifacts")
	runID := flag.String("run-id", "", "GitHub workflow run ID")
	runAttempt := flag.Int("run-attempt", 0, "GitHub workflow run attempt")
	output := flag.String("output", "", "optional verified paired summary path")
	requireModel := flag.Bool("require-model", false, "require final-model quality, numerical, RSS and index evidence")
	flag.Parse()

	summary, err := verifyPairWithOptions(*directory, *commit, *runID, *runAttempt, *requireModel)
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
	fmt.Printf("intelligent media native evidence verified for %s\n", *commit)
}

func verifyPair(directory, commit, runID string, runAttempt int) (pairedSummary, error) {
	return verifyPairWithOptions(directory, commit, runID, runAttempt, false)
}

func verifyPairWithOptions(directory, commit, runID string, runAttempt int, requireModel bool) (pairedSummary, error) {
	if directory == "" {
		return pairedSummary{}, errors.New("evidence directory is required")
	}
	if !commitPattern.MatchString(commit) {
		return pairedSummary{}, fmt.Errorf("invalid source commit %q", commit)
	}
	if !runIDPattern.MatchString(runID) {
		return pairedSummary{}, fmt.Errorf("invalid workflow run ID %q", runID)
	}
	if runAttempt < 1 {
		return pairedSummary{}, fmt.Errorf("invalid workflow run attempt %d", runAttempt)
	}

	identities, err := findIdentities(directory)
	if err != nil {
		return pairedSummary{}, err
	}
	candidates := make([]candidateSummary, 0, 2)
	models := make(map[string]modelEvidence, 2)
	seen := map[string]bool{}
	for identityPath, identity := range identities {
		if err := validateIdentity(identity, commit, runID, runAttempt); err != nil {
			return pairedSummary{}, fmt.Errorf("%s: %w", identityPath, err)
		}
		if seen[identity.GOARCH] {
			return pairedSummary{}, fmt.Errorf("duplicate %s evidence", identity.GOARCH)
		}
		seen[identity.GOARCH] = true
		outcomesPath := filepath.Join(filepath.Dir(identityPath), "outcomes.json")
		outcomes, err := readOutcomes(outcomesPath)
		if err != nil {
			return pairedSummary{}, fmt.Errorf("%s outcomes: %w", identity.GOARCH, err)
		}
		if err := validateOutcomes(outcomes); err != nil {
			return pairedSummary{}, fmt.Errorf("%s outcomes: %w", identity.GOARCH, err)
		}
		if requireModel {
			modelPath := filepath.Join(filepath.Dir(identityPath), "model-evidence.json")
			model, err := readModelEvidence(modelPath)
			if err != nil {
				return pairedSummary{}, fmt.Errorf("%s model evidence: %w", identity.GOARCH, err)
			}
			if err := validateModelEvidence(filepath.Dir(modelPath), model, identity.GOARCH, commit); err != nil {
				return pairedSummary{}, fmt.Errorf("%s model evidence: %w", identity.GOARCH, err)
			}
			models[identity.GOARCH] = model
		}
		candidate := candidateSummary{
			Architecture: identity.GOARCH, RunnerLabel: identity.RunnerLabel,
			Machine: identity.Machine, Docker: identity.Docker, CreatedAt: identity.CreatedAt,
		}
		if requireModel {
			model := models[identity.GOARCH]
			candidate.ModelPackageDigest = model.ModelPackageDigest
			candidate.FinalImageDigest = model.FinalImageDigest
			candidate.PeakRSSBytes = model.PeakRSSBytes
		}
		candidates = append(candidates, candidate)
	}
	if requireModel {
		if err := validateModelPair(models["amd64"], models["arm64"]); err != nil {
			return pairedSummary{}, err
		}
	}
	for _, architecture := range []string{"amd64", "arm64"} {
		if !seen[architecture] {
			return pairedSummary{}, fmt.Errorf("missing %s evidence", architecture)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Architecture < candidates[j].Architecture })
	return pairedSummary{
		SchemaVersion: 1, Feature: "FTR-INT-001", SourceCommit: commit,
		WorkflowRunID: runID, WorkflowRunAttempt: runAttempt, Candidates: candidates,
		Checks: map[string]bool{
			"nativeIdentity": true, "sameSourceCommit": true, "sameWorkflowRun": true,
			"sameWorkflowAttempt": true, "allStepsSucceeded": true, "qemuRejected": true,
			"finalModelEvidence": requireModel,
		},
		Result: "passed",
	}, nil
}

func readModelEvidence(path string) (modelEvidence, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return modelEvidence{}, err
	}
	var evidence modelEvidence
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&evidence); err != nil {
		return modelEvidence{}, err
	}
	return evidence, nil
}

func validateModelEvidence(base string, evidence modelEvidence, architecture, commit string) error {
	hashes := []string{
		evidence.QualityDatasetSHA256, evidence.RankingFixtureSHA256,
		evidence.TieFixtureSHA256,
	}
	for _, hash := range hashes {
		if !hashPattern.MatchString(hash) {
			return fmt.Errorf("invalid SHA-256 %q", hash)
		}
	}
	for label, artifact := range map[string]artifactEvidence{
		"quality summary": evidence.QualitySummary,
		"ranking Top-20":  evidence.RankingTop20,
		"numeric report":  evidence.NumericReport,
		"runtime report":  evidence.RuntimeReport,
		"index report":    evidence.IndexReport,
	} {
		if err := verifyArtifact(base, label, artifact); err != nil {
			return err
		}
	}
	numbers := []float64{
		evidence.MaxAbsError, evidence.SemanticP95Millis, evidence.SemanticP99Millis,
		evidence.BrowseRegressionPercent,
	}
	for _, number := range numbers {
		if math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
			return errors.New("model metrics must be finite and non-negative")
		}
	}
	switch {
	case evidence.SchemaVersion != 1:
		return fmt.Errorf("schemaVersion = %d, want 1", evidence.SchemaVersion)
	case evidence.SourceCommit != commit:
		return errors.New("source commit mismatch")
	case evidence.Architecture != architecture:
		return errors.New("architecture mismatch")
	case !digestPattern.MatchString(evidence.ModelPackageDigest):
		return errors.New("invalid model package digest")
	case !digestPattern.MatchString(evidence.FinalImageDigest):
		return errors.New("invalid final image digest")
	case evidence.MediaCount < 100000 || evidence.DirectoryCount < 10000:
		return errors.New("final 100k media / 10k directory tier is required")
	case !evidence.QualityPassed || !evidence.ReferenceRankingPassed || !evidence.TieFixturePassed:
		return errors.New("quality, ranking and tie fixtures must pass")
	case evidence.MaxAbsError > 0.001:
		return fmt.Errorf("maxAbsError = %g, want <= 0.001", evidence.MaxAbsError)
	case evidence.PeakRSSBytes <= 0 || evidence.PeakRSSBytes > 3435973836:
		return fmt.Errorf("peakRSSBytes = %d, want 1..3435973836", evidence.PeakRSSBytes)
	case evidence.SemanticP95Millis > 750:
		return fmt.Errorf("semanticP95Millis = %g, want <= 750", evidence.SemanticP95Millis)
	case evidence.SemanticP99Millis > 1500:
		return fmt.Errorf("semanticP99Millis = %g, want <= 1500", evidence.SemanticP99Millis)
	case evidence.BrowseRegressionPercent > 20:
		return fmt.Errorf("browseRegressionPercent = %g, want <= 20", evidence.BrowseRegressionPercent)
	case evidence.DerivedBytes < 0 || evidence.DerivedBytes > 524288000:
		return fmt.Errorf("derivedBytes = %d, want 0..524288000", evidence.DerivedBytes)
	case !evidence.IndexRebuildPassed || !evidence.RestartRecoveryPassed || !evidence.RuntimeFailureMatrixPassed:
		return errors.New("index rebuild, restart recovery and runtime failure matrix must pass")
	case evidence.Result != "passed":
		return fmt.Errorf("result = %q, want passed", evidence.Result)
	}
	return nil
}

func validateModelPair(amd64, arm64 modelEvidence) error {
	switch {
	case amd64.ModelPackageDigest != arm64.ModelPackageDigest:
		return errors.New("model package digests differ across architectures")
	case amd64.QualityDatasetSHA256 != arm64.QualityDatasetSHA256:
		return errors.New("quality dataset hashes differ across architectures")
	case amd64.RankingFixtureSHA256 != arm64.RankingFixtureSHA256:
		return errors.New("ranking fixture hashes differ across architectures")
	case amd64.RankingTop20.SHA256 != arm64.RankingTop20.SHA256:
		return errors.New("ranking Top-20 sets differ across architectures")
	case amd64.TieFixtureSHA256 != arm64.TieFixtureSHA256:
		return errors.New("tie fixture hashes differ across architectures")
	}
	return nil
}

func verifyArtifact(base, label string, artifact artifactEvidence) error {
	if artifact.Path == "" || filepath.IsAbs(artifact.Path) || !hashPattern.MatchString(artifact.SHA256) {
		return fmt.Errorf("%s has invalid path or SHA-256", label)
	}
	clean := filepath.Clean(artifact.Path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s path escapes evidence directory", label)
	}
	current := base
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s path contains a symlink", label)
		}
	}
	file, err := os.Open(filepath.Join(base, clean))
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", label)
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if actual := hex.EncodeToString(hasher.Sum(nil)); actual != artifact.SHA256 {
		return fmt.Errorf("%s SHA-256 mismatch", label)
	}
	return nil
}

func findIdentities(directory string) (map[string]identityEvidence, error) {
	identities := map[string]identityEvidence{}
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "identity.json" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var identity identityEvidence
		if err := json.Unmarshal(content, &identity); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
		identities[path] = identity
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk evidence directory: %w", err)
	}
	if len(identities) != 2 {
		return nil, fmt.Errorf("identity artifact count = %d, want 2", len(identities))
	}
	return identities, nil
}

func validateIdentity(identity identityEvidence, commit, runID string, runAttempt int) error {
	expected := map[string]struct {
		runner, machine, docker string
	}{
		"amd64": {runner: "ubuntu-24.04", machine: "x86_64", docker: "linux/x86_64"},
		"arm64": {runner: "ubuntu-24.04-arm", machine: "aarch64", docker: "linux/aarch64"},
	}
	want, ok := expected[identity.GOARCH]
	switch {
	case identity.SchemaVersion != 1:
		return fmt.Errorf("schemaVersion = %d, want 1", identity.SchemaVersion)
	case !ok:
		return fmt.Errorf("unsupported goarch %q", identity.GOARCH)
	case identity.SourceCommit != commit:
		return errors.New("source commit mismatch")
	case identity.WorkflowRunID != runID:
		return errors.New("workflow run mismatch")
	case identity.WorkflowRunAttempt != runAttempt:
		return errors.New("workflow run attempt mismatch")
	case identity.GOOS != "linux":
		return fmt.Errorf("goos = %q, want linux", identity.GOOS)
	case identity.RunnerLabel != want.runner || identity.Machine != want.machine || identity.Docker != want.docker:
		return errors.New("native runner identity mismatch")
	case identity.QEMUAllowed:
		return errors.New("QEMU evidence is forbidden")
	}
	if _, err := time.Parse(time.RFC3339, identity.CreatedAt); err != nil {
		return fmt.Errorf("invalid createdAt: %w", err)
	}
	return nil
}

func readOutcomes(path string) (outcomeEvidence, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return outcomeEvidence{}, err
	}
	var outcomes outcomeEvidence
	if err := json.Unmarshal(content, &outcomes); err != nil {
		return outcomeEvidence{}, err
	}
	return outcomes, nil
}

func validateOutcomes(outcomes outcomeEvidence) error {
	for name, value := range map[string]string{
		"identity": outcomes.Identity, "repository": outcomes.Repository,
		"libvips": outcomes.Libvips, "searchMatrix": outcomes.SearchMatrix,
		"capacity": outcomes.Capacity,
	} {
		if value != "success" {
			return fmt.Errorf("%s outcome = %q, want success", name, value)
		}
	}
	if !outcomes.Complete {
		return errors.New("complete = false")
	}
	return nil
}

func writeSummary(path string, summary pairedSummary) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".native-summary-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}
