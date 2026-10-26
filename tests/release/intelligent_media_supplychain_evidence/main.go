package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/HappyQuQu/foliopath/tests/release/evidencejson"
)

const releaseName = "POST-MVP-5-r2"

var (
	commitPattern   = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
	digestPattern   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	hashPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	approvalPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{7,255}$`)
)

type artifactEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type componentEvidence struct {
	Name                   string           `json:"name"`
	Role                   string           `json:"role"`
	Version                string           `json:"version"`
	License                string           `json:"license"`
	RedistributionApproved bool             `json:"redistributionApproved"`
	ApprovalRef            string           `json:"approvalRef"`
	Notices                artifactEvidence `json:"notices"`
}

type vulnerabilityEvidence struct {
	Report      artifactEvidence  `json:"report"`
	Critical    int               `json:"critical"`
	High        int               `json:"high"`
	VEX         *artifactEvidence `json:"vex,omitempty"`
	SecurityRef string            `json:"securityApprovalRef,omitempty"`
}

type architectureEvidence struct {
	Architecture      string                `json:"architecture"`
	OS                string                `json:"os"`
	Native            bool                  `json:"native"`
	ImageDigest       string                `json:"imageDigest"`
	SBOM              artifactEvidence      `json:"sbom"`
	SBOMComplete      bool                  `json:"sbomComplete"`
	Provenance        artifactEvidence      `json:"provenance"`
	SignatureVerify   artifactEvidence      `json:"signatureVerification"`
	SignatureVerified bool                  `json:"signatureVerified"`
	Vulnerabilities   vulnerabilityEvidence `json:"vulnerabilities"`
}

type approvals struct {
	Security   string `json:"security"`
	Compliance string `json:"compliance"`
	Release    string `json:"release"`
	Inference  string `json:"inference"`
}

type supplyChainEvidence struct {
	SchemaVersion      int                    `json:"schemaVersion"`
	Release            string                 `json:"release"`
	SourceCommit       string                 `json:"sourceCommit"`
	Catalog            artifactEvidence       `json:"catalog"`
	ModelPackage       artifactEvidence       `json:"modelPackage"`
	ModelPackageDigest string                 `json:"modelPackageDigest"`
	Components         []componentEvidence    `json:"components"`
	Architectures      []architectureEvidence `json:"architectures"`
	Approvals          approvals              `json:"approvals"`
	Result             string                 `json:"result"`
}

type verifiedSummary struct {
	SchemaVersion      int            `json:"schemaVersion"`
	Release            string         `json:"release"`
	SourceCommit       string         `json:"sourceCommit"`
	ModelPackageDigest string         `json:"modelPackageDigest"`
	Architectures      []string       `json:"architectures"`
	Images             []imageSummary `json:"images"`
	Components         []string       `json:"components"`
	Result             string         `json:"result"`
}

type imageSummary struct {
	Architecture string `json:"architecture"`
	ImageDigest  string `json:"imageDigest"`
}

func main() {
	input := flag.String("input", "", "intelligent-media supply-chain evidence manifest")
	commit := flag.String("commit", "", "expected source commit")
	output := flag.String("output", "", "optional verified summary path")
	flag.Parse()

	summary, err := verifyManifest(*input, *commit)
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
	fmt.Printf("intelligent-media supply-chain evidence verified for %s\n", *commit)
}

func verifyManifest(path, expectedCommit string) (verifiedSummary, error) {
	if path == "" {
		return verifiedSummary{}, errors.New("input manifest is required")
	}
	if !commitPattern.MatchString(expectedCommit) {
		return verifiedSummary{}, fmt.Errorf("invalid source commit %q", expectedCommit)
	}
	content, err := evidencejson.ReadRegularFile(path)
	if err != nil {
		return verifiedSummary{}, err
	}
	var evidence supplyChainEvidence
	if err := evidencejson.Decode(content, &evidence); err != nil {
		return verifiedSummary{}, fmt.Errorf("decode evidence: %w", err)
	}
	base := filepath.Dir(path)
	if err := validateEvidence(base, evidence, expectedCommit); err != nil {
		return verifiedSummary{}, err
	}

	architectures := make([]string, 0, len(evidence.Architectures))
	images := make([]imageSummary, 0, len(evidence.Architectures))
	for _, item := range evidence.Architectures {
		architectures = append(architectures, item.Architecture)
		images = append(images, imageSummary{Architecture: item.Architecture, ImageDigest: item.ImageDigest})
	}
	sort.Strings(architectures)
	sort.Slice(images, func(i, j int) bool { return images[i].Architecture < images[j].Architecture })
	components := make([]string, 0, len(evidence.Components))
	for _, item := range evidence.Components {
		components = append(components, item.Role+":"+item.Name+"@"+item.Version)
	}
	sort.Strings(components)
	return verifiedSummary{
		SchemaVersion:      1,
		Release:            releaseName,
		SourceCommit:       expectedCommit,
		ModelPackageDigest: evidence.ModelPackageDigest,
		Architectures:      architectures, Images: images,
		Components: components,
		Result:     "passed",
	}, nil
}

func validateEvidence(base string, evidence supplyChainEvidence, expectedCommit string) error {
	switch {
	case evidence.SchemaVersion != 2:
		return fmt.Errorf("schemaVersion = %d, want 2", evidence.SchemaVersion)
	case evidence.Release != releaseName:
		return fmt.Errorf("release = %q, want %q", evidence.Release, releaseName)
	case evidence.SourceCommit != expectedCommit:
		return fmt.Errorf("sourceCommit = %q, want %q", evidence.SourceCommit, expectedCommit)
	case !digestPattern.MatchString(evidence.ModelPackageDigest):
		return fmt.Errorf("invalid modelPackageDigest %q", evidence.ModelPackageDigest)
	case evidence.ModelPackageDigest != "sha256:"+evidence.ModelPackage.SHA256:
		return errors.New("modelPackageDigest does not match modelPackage SHA-256")
	case evidence.Result != "passed":
		return fmt.Errorf("result = %q, want passed", evidence.Result)
	}
	for name, ref := range map[string]string{
		"security": evidence.Approvals.Security, "compliance": evidence.Approvals.Compliance,
		"release": evidence.Approvals.Release, "inference": evidence.Approvals.Inference,
	} {
		if !approvalPattern.MatchString(ref) {
			return fmt.Errorf("%s approval must be an opaque safe reference", name)
		}
	}
	if err := verifyArtifact(base, "catalog", evidence.Catalog); err != nil {
		return err
	}
	if err := verifyArtifact(base, "modelPackage", evidence.ModelPackage); err != nil {
		return err
	}

	required := map[string]bool{
		"inference_runtime":  false,
		"semantic_tokenizer": false,
		"semantic_model":     false,
		"face_detector":      false,
		"face_embedder":      false,
	}
	seenComponents := make(map[string]struct{}, len(evidence.Components))
	seenRoles := make(map[string]struct{}, len(evidence.Components))
	for _, component := range evidence.Components {
		key := strings.ToLower(component.Name)
		if _, duplicate := seenComponents[key]; duplicate {
			return fmt.Errorf("duplicate component %q", component.Name)
		}
		seenComponents[key] = struct{}{}
		if !approvalPattern.MatchString(component.Role) {
			return fmt.Errorf("component %q has invalid role", component.Name)
		}
		if _, duplicate := seenRoles[component.Role]; duplicate {
			return fmt.Errorf("duplicate component role %q", component.Role)
		}
		seenRoles[component.Role] = struct{}{}
		if _, ok := required[component.Role]; ok {
			required[component.Role] = true
		}
		if component.Name == "" || component.Version == "" || component.License == "" {
			return errors.New("component name, version and license are required")
		}
		if !component.RedistributionApproved || !approvalPattern.MatchString(component.ApprovalRef) {
			return fmt.Errorf("component %q lacks redistribution approval", component.Name)
		}
		if err := verifyArtifact(base, component.Name+" notices", component.Notices); err != nil {
			return err
		}
	}
	for role, present := range required {
		if !present {
			return fmt.Errorf("required component role %q is missing", role)
		}
	}

	seenArchitectures := map[string]bool{"amd64": false, "arm64": false}
	if len(evidence.Architectures) != 2 {
		return errors.New("exactly native amd64 and arm64 evidence is required")
	}
	for _, architecture := range evidence.Architectures {
		if _, ok := seenArchitectures[architecture.Architecture]; !ok || seenArchitectures[architecture.Architecture] {
			return fmt.Errorf("unexpected or duplicate architecture %q", architecture.Architecture)
		}
		seenArchitectures[architecture.Architecture] = true
		switch {
		case architecture.OS != "linux":
			return fmt.Errorf("%s os = %q, want linux", architecture.Architecture, architecture.OS)
		case !architecture.Native:
			return fmt.Errorf("%s evidence is not native", architecture.Architecture)
		case !digestPattern.MatchString(architecture.ImageDigest):
			return fmt.Errorf("%s has invalid image digest", architecture.Architecture)
		case !architecture.SBOMComplete:
			return fmt.Errorf("%s SBOM is incomplete", architecture.Architecture)
		case !architecture.SignatureVerified:
			return fmt.Errorf("%s provenance signature is not verified", architecture.Architecture)
		}
		for label, artifact := range map[string]artifactEvidence{
			"SBOM": architecture.SBOM, "provenance": architecture.Provenance,
			"signature verification": architecture.SignatureVerify,
			"vulnerability report":   architecture.Vulnerabilities.Report,
		} {
			if err := verifyArtifact(base, architecture.Architecture+" "+label, artifact); err != nil {
				return err
			}
		}
		vulnerabilities := architecture.Vulnerabilities
		if vulnerabilities.Critical < 0 || vulnerabilities.High < 0 {
			return errors.New("vulnerability counts cannot be negative")
		}
		if vulnerabilities.Critical > 0 || vulnerabilities.High > 0 {
			if vulnerabilities.VEX == nil || !approvalPattern.MatchString(vulnerabilities.SecurityRef) {
				return fmt.Errorf("%s blocking findings require signed VEX and security approval", architecture.Architecture)
			}
			if err := verifyArtifact(base, architecture.Architecture+" VEX", *vulnerabilities.VEX); err != nil {
				return err
			}
		}
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
	path := filepath.Join(base, clean)
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
	file, err := os.Open(path)
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

func writeSummary(path string, summary verifiedSummary) error {
	content, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}
