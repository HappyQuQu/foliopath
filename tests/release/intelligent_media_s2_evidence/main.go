package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var (
	commitPattern   = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
	digestPattern   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	hashPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	approvalPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{7,255}$`)
)

type qualityApprovals struct {
	Product string `json:"product"`
	ML      string `json:"ml"`
	QA      string `json:"qa"`
}

type qualitySummary struct {
	SchemaVersion         int              `json:"schema_version"`
	SourceCommit          string           `json:"source_commit"`
	DatasetManifestSHA256 string           `json:"dataset_manifest_sha256"`
	ModelPackageDigest    string           `json:"model_package_digest"`
	Approvals             qualityApprovals `json:"approvals"`
	GatePass              bool             `json:"gate_pass"`
}

type nativeCandidate struct {
	Architecture       string `json:"architecture"`
	ModelPackageDigest string `json:"modelPackageDigest"`
	FinalImageDigest   string `json:"finalImageDigest"`
	PeakRSSBytes       int64  `json:"peakRSSBytes"`
}

type nativeSummary struct {
	SchemaVersion int               `json:"schemaVersion"`
	Feature       string            `json:"feature"`
	SourceCommit  string            `json:"sourceCommit"`
	Candidates    []nativeCandidate `json:"candidates"`
	Checks        map[string]bool   `json:"checks"`
	Result        string            `json:"result"`
}

type supplyImage struct {
	Architecture string `json:"architecture"`
	ImageDigest  string `json:"imageDigest"`
}

type supplySummary struct {
	SchemaVersion      int           `json:"schemaVersion"`
	Release            string        `json:"release"`
	SourceCommit       string        `json:"sourceCommit"`
	ModelPackageDigest string        `json:"modelPackageDigest"`
	Architectures      []string      `json:"architectures"`
	Images             []supplyImage `json:"images"`
	Result             string        `json:"result"`
}

type inputHash struct {
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
}

type verifiedSummary struct {
	SchemaVersion         int             `json:"schemaVersion"`
	Release               string          `json:"release"`
	SourceCommit          string          `json:"sourceCommit"`
	ModelPackageDigest    string          `json:"modelPackageDigest"`
	DatasetManifestSHA256 string          `json:"datasetManifestSHA256"`
	Architectures         []string        `json:"architectures"`
	Inputs                []inputHash     `json:"inputs"`
	Checks                map[string]bool `json:"checks"`
	Result                string          `json:"result"`
}

func main() {
	qualityPath := flag.String("quality", "", "verified quality summary")
	nativePath := flag.String("native", "", "verified strict native-model summary")
	supplyPath := flag.String("supply-chain", "", "verified supply-chain summary")
	commit := flag.String("commit", "", "expected source commit")
	output := flag.String("output", "", "optional aggregate summary path")
	flag.Parse()

	summary, err := verifyEvidence(*qualityPath, *nativePath, *supplyPath, *commit)
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
	fmt.Printf("POST-MVP-5 S2 evidence bound for %s\n", *commit)
}

func verifyEvidence(qualityPath, nativePath, supplyPath, commit string) (verifiedSummary, error) {
	if !commitPattern.MatchString(commit) {
		return verifiedSummary{}, fmt.Errorf("invalid source commit %q", commit)
	}
	var quality qualitySummary
	qualityHash, err := readSummary(qualityPath, &quality)
	if err != nil {
		return verifiedSummary{}, fmt.Errorf("quality summary: %w", err)
	}
	var native nativeSummary
	nativeHash, err := readSummary(nativePath, &native)
	if err != nil {
		return verifiedSummary{}, fmt.Errorf("native summary: %w", err)
	}
	var supply supplySummary
	supplyHash, err := readSummary(supplyPath, &supply)
	if err != nil {
		return verifiedSummary{}, fmt.Errorf("supply-chain summary: %w", err)
	}
	if err := validateQuality(quality, commit); err != nil {
		return verifiedSummary{}, err
	}
	if err := validateNative(native, commit); err != nil {
		return verifiedSummary{}, err
	}
	if err := validateSupply(supply, commit); err != nil {
		return verifiedSummary{}, err
	}
	if quality.ModelPackageDigest != native.Candidates[0].ModelPackageDigest ||
		quality.ModelPackageDigest != native.Candidates[1].ModelPackageDigest ||
		quality.ModelPackageDigest != supply.ModelPackageDigest {
		return verifiedSummary{}, errors.New("quality, native and supply-chain model package digests differ")
	}
	nativeImages := map[string]string{}
	for _, candidate := range native.Candidates {
		nativeImages[candidate.Architecture] = candidate.FinalImageDigest
	}
	for _, image := range supply.Images {
		if nativeImages[image.Architecture] != image.ImageDigest {
			return verifiedSummary{}, fmt.Errorf("%s final image digest differs between native and supply-chain evidence", image.Architecture)
		}
	}

	return verifiedSummary{
		SchemaVersion: 1, Release: "POST-MVP-5-r2", SourceCommit: commit,
		ModelPackageDigest:    quality.ModelPackageDigest,
		DatasetManifestSHA256: quality.DatasetManifestSHA256,
		Architectures:         []string{"amd64", "arm64"},
		Inputs: []inputHash{
			{Kind: "quality", SHA256: qualityHash},
			{Kind: "native", SHA256: nativeHash},
			{Kind: "supply-chain", SHA256: supplyHash},
		},
		Checks: map[string]bool{
			"qualityPassed": true, "strictNativePassed": true, "supplyChainPassed": true,
			"sameSourceCommit": true, "sameModelPackage": true, "sameFinalImages": true,
		},
		Result: "passed",
	}, nil
}

func validateQuality(summary qualitySummary, commit string) error {
	switch {
	case summary.SchemaVersion != 1:
		return errors.New("quality schemaVersion must be 1")
	case summary.SourceCommit != commit:
		return errors.New("quality source commit mismatch")
	case !hashPattern.MatchString(summary.DatasetManifestSHA256):
		return errors.New("quality dataset manifest hash is invalid")
	case !digestPattern.MatchString(summary.ModelPackageDigest):
		return errors.New("quality model package digest is invalid")
	case !summary.GatePass:
		return errors.New("quality gate did not pass")
	}
	for name, value := range map[string]string{
		"product": summary.Approvals.Product, "ml": summary.Approvals.ML, "qa": summary.Approvals.QA,
	} {
		if !approvalPattern.MatchString(value) {
			return fmt.Errorf("quality %s approval is invalid", name)
		}
	}
	return nil
}

func validateNative(summary nativeSummary, commit string) error {
	if summary.SchemaVersion != 1 || summary.Feature != "FTR-INT-001" ||
		summary.SourceCommit != commit || summary.Result != "passed" {
		return errors.New("strict native summary identity or result is invalid")
	}
	for _, check := range []string{
		"nativeIdentity", "sameSourceCommit", "sameWorkflowRun", "sameWorkflowAttempt",
		"allStepsSucceeded", "qemuRejected", "finalModelEvidence",
	} {
		if !summary.Checks[check] {
			return fmt.Errorf("strict native check %q did not pass", check)
		}
	}
	if len(summary.Candidates) != 2 {
		return errors.New("strict native summary requires two candidates")
	}
	seen := map[string]bool{}
	for _, candidate := range summary.Candidates {
		if (candidate.Architecture != "amd64" && candidate.Architecture != "arm64") || seen[candidate.Architecture] {
			return errors.New("strict native architectures are invalid")
		}
		seen[candidate.Architecture] = true
		if !digestPattern.MatchString(candidate.ModelPackageDigest) ||
			!digestPattern.MatchString(candidate.FinalImageDigest) || candidate.PeakRSSBytes <= 0 {
			return fmt.Errorf("%s strict native candidate is incomplete", candidate.Architecture)
		}
	}
	return nil
}

func validateSupply(summary supplySummary, commit string) error {
	if summary.SchemaVersion != 1 || summary.Release != "POST-MVP-5-r2" ||
		summary.SourceCommit != commit || summary.Result != "passed" ||
		!digestPattern.MatchString(summary.ModelPackageDigest) {
		return errors.New("supply-chain summary identity or result is invalid")
	}
	if len(summary.Architectures) != 2 || len(summary.Images) != 2 {
		return errors.New("supply-chain summary requires amd64 and arm64 images")
	}
	sort.Strings(summary.Architectures)
	if summary.Architectures[0] != "amd64" || summary.Architectures[1] != "arm64" {
		return errors.New("supply-chain architectures are invalid")
	}
	seen := map[string]bool{}
	for _, image := range summary.Images {
		if (image.Architecture != "amd64" && image.Architecture != "arm64") || seen[image.Architecture] ||
			!digestPattern.MatchString(image.ImageDigest) {
			return errors.New("supply-chain images are invalid")
		}
		seen[image.Architecture] = true
	}
	return nil
}

func readSummary(path string, target any) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New("summary must be a non-symlink regular file")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(content, target); err != nil {
		return "", err
	}
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:]), nil
}

func writeSummary(path string, summary verifiedSummary) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}
