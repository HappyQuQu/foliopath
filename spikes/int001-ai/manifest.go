package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var packageSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var governanceReferencePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type LicenseEvidence struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type DatasetManifest struct {
	SchemaVersion   int                `json:"schema_version"`
	DatasetID       string             `json:"dataset_id"`
	Purpose         string             `json:"purpose"`
	LegalBasis      string             `json:"legal_basis"`
	License         LicenseEvidence    `json:"license"`
	Redistributable bool               `json:"redistributable"`
	Governance      *DatasetGovernance `json:"governance,omitempty"`
	Items           []DatasetItem      `json:"items"`
}

type DatasetGovernance struct {
	DataClass             string   `json:"data_class"`
	AllowedUses           []string `json:"allowed_uses"`
	AuthorizedRoles       []string `json:"authorized_roles"`
	RetentionUntil        string   `json:"retention_until,omitempty"`
	DeletionProcedure     string   `json:"deletion_procedure"`
	ConsentOrAuthorityRef string   `json:"consent_or_authority_ref,omitempty"`
	PrivacyReviewRef      string   `json:"privacy_review_ref,omitempty"`
	Redistribution        string   `json:"redistribution"`
}

type DatasetItem struct {
	ID           string              `json:"id"`
	RelativePath string              `json:"relative_path"`
	MediaType    string              `json:"media_type"`
	SHA256       string              `json:"sha256"`
	Queries      map[string][]string `json:"queries,omitempty"`
	IdentityID   string              `json:"identity_id,omitempty"`
}

type ModelCatalog struct {
	SchemaVersion int          `json:"schema_version"`
	Models        []ModelEntry `json:"models"`
}

type ModelEntry struct {
	ID            string          `json:"id"`
	Status        string          `json:"status"`
	Purpose       string          `json:"purpose"`
	Version       string          `json:"version"`
	Filename      string          `json:"filename"`
	SHA256        string          `json:"sha256"`
	SizeBytes     int64           `json:"size_bytes"`
	SourceURL     string          `json:"source_url"`
	CodeLicense   LicenseEvidence `json:"code_license"`
	WeightLicense LicenseEvidence `json:"weight_license"`
	Architectures []string        `json:"architectures"`
	Directory     string          `json:"directory,omitempty"`
	PackageSHA256 string          `json:"package_sha256,omitempty"`
	Artifacts     []ModelArtifact `json:"artifacts,omitempty"`
}

type ModelArtifact struct {
	Filename  string `json:"filename"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

func ModelPackageDigest(artifacts []ModelArtifact) string {
	ordered := append([]ModelArtifact(nil), artifacts...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Filename < ordered[j].Filename })
	hash := sha256.New()
	hash.Write([]byte("foliopath-model-package-v1\n"))
	for _, artifact := range ordered {
		hash.Write([]byte(artifact.Filename))
		hash.Write([]byte{'\n'})
		hash.Write([]byte(strconv.FormatInt(artifact.SizeBytes, 10)))
		hash.Write([]byte{'\n'})
		hash.Write([]byte(artifact.SHA256))
		hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func ReadDatasetManifest(filename string) (DatasetManifest, error) {
	var manifest DatasetManifest
	if err := decodeStrict(filename, &manifest); err != nil {
		return manifest, err
	}
	return manifest, manifest.Validate()
}

func ReadModelCatalog(filename string) (ModelCatalog, error) {
	var catalog ModelCatalog
	if err := decodeStrict(filename, &catalog); err != nil {
		return catalog, err
	}
	return catalog, catalog.Validate()
}

func decodeStrict(filename string, target any) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", filename, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return fmt.Errorf("decode trailing JSON in %s: %w", filename, err)
	}
	return nil
}

func (manifest DatasetManifest) Validate() error {
	if (manifest.SchemaVersion != 1 && manifest.SchemaVersion != 2) || manifest.DatasetID == "" || manifest.Purpose == "" {
		return errors.New("dataset requires schema_version 1 or 2, dataset_id, and purpose")
	}
	if manifest.LegalBasis != "synthetic" && manifest.LegalBasis != "public-license" && manifest.LegalBasis != "written-authorization" {
		return fmt.Errorf("unsupported legal_basis %q", manifest.LegalBasis)
	}
	if err := validateLicense("dataset", manifest.License); err != nil {
		return err
	}
	if manifest.SchemaVersion == 1 && manifest.Governance != nil {
		return errors.New("dataset schema_version 1 must not contain governance")
	}
	if manifest.SchemaVersion == 1 && manifest.LegalBasis != "synthetic" {
		return errors.New("non-synthetic datasets require schema_version 2 governance")
	}
	if manifest.SchemaVersion == 2 {
		if manifest.Governance == nil {
			return errors.New("dataset schema_version 2 requires governance")
		}
		if err := validateDatasetGovernance(manifest); err != nil {
			return err
		}
	}
	if len(manifest.Items) == 0 {
		return errors.New("dataset contains no items")
	}
	seen := make(map[string]struct{}, len(manifest.Items))
	for i, item := range manifest.Items {
		if item.ID == "" || item.RelativePath == "" {
			return fmt.Errorf("item %d requires id and relative_path", i)
		}
		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("duplicate item id %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		if path.IsAbs(item.RelativePath) || path.Clean(item.RelativePath) != item.RelativePath || strings.HasPrefix(item.RelativePath, "../") {
			return fmt.Errorf("item %q has unsafe relative_path", item.ID)
		}
		if item.MediaType != "image" && item.MediaType != "video" {
			return fmt.Errorf("item %q has unsupported media_type", item.ID)
		}
		if !sha256Pattern.MatchString(item.SHA256) {
			return fmt.Errorf("item %q has invalid sha256", item.ID)
		}
		if manifest.SchemaVersion == 2 && item.IdentityID != "" {
			if manifest.Governance.DataClass != "biometric-ground-truth" {
				return fmt.Errorf("item %q has identity_id outside biometric ground truth", item.ID)
			}
			if !governanceReferencePattern.MatchString(item.IdentityID) {
				return fmt.Errorf("item %q identity_id must be an opaque safe reference", item.ID)
			}
		}
	}
	if manifest.SchemaVersion == 2 && manifest.Governance.DataClass == "biometric-ground-truth" &&
		(containsString(manifest.Governance.AllowedUses, "face-verification-evaluation") ||
			containsString(manifest.Governance.AllowedUses, "face-clustering-evaluation")) {
		for _, item := range manifest.Items {
			if item.IdentityID == "" {
				return fmt.Errorf("item %q requires identity_id for face verification or clustering evaluation", item.ID)
			}
		}
	}
	return nil
}

func validateDatasetGovernance(manifest DatasetManifest) error {
	governance := manifest.Governance
	if governance.DataClass != "ordinary-media" && governance.DataClass != "biometric-ground-truth" {
		return fmt.Errorf("unsupported dataset data_class %q", governance.DataClass)
	}
	allowedUses := map[string]struct{}{
		"semantic-evaluation":          {},
		"video-evaluation":             {},
		"face-detection-evaluation":    {},
		"face-verification-evaluation": {},
		"face-clustering-evaluation":   {},
	}
	if err := validateUniqueEnum("allowed_use", governance.AllowedUses, allowedUses); err != nil {
		return err
	}
	authorizedRoles := map[string]struct{}{
		"evaluation-team":  {},
		"privacy-reviewer": {},
	}
	if err := validateUniqueEnum("authorized_role", governance.AuthorizedRoles, authorizedRoles); err != nil {
		return err
	}
	if governance.DeletionProcedure != "delete-fixtures-and-derived-evidence" {
		return fmt.Errorf("unsupported deletion_procedure %q", governance.DeletionProcedure)
	}
	if governance.Redistribution != "allowed" && governance.Redistribution != "prohibited" {
		return fmt.Errorf("unsupported redistribution %q", governance.Redistribution)
	}
	if manifest.Redistributable != (governance.Redistribution == "allowed") {
		return errors.New("redistributable must agree with governance.redistribution")
	}
	if governance.RetentionUntil == "" || !validRetentionDate(governance.RetentionUntil) {
		return errors.New("dataset governance requires retention_until as YYYY-MM-DD or RFC3339")
	}
	for _, reference := range []struct {
		name  string
		value string
	}{
		{name: "consent_or_authority_ref", value: governance.ConsentOrAuthorityRef},
		{name: "privacy_review_ref", value: governance.PrivacyReviewRef},
	} {
		if reference.value != "" && !governanceReferencePattern.MatchString(reference.value) {
			return fmt.Errorf("%s must be an opaque safe reference", reference.name)
		}
	}

	faceUse := false
	for _, allowedUse := range governance.AllowedUses {
		if strings.HasPrefix(allowedUse, "face-") {
			faceUse = true
		}
	}
	if governance.DataClass == "ordinary-media" && faceUse {
		return errors.New("face evaluation requires biometric-ground-truth data_class")
	}
	if governance.DataClass == "biometric-ground-truth" {
		if !faceUse {
			return errors.New("biometric ground truth requires at least one face evaluation use")
		}
		if manifest.LegalBasis == "public-license" {
			return errors.New("public media license is not authority for biometric ground truth")
		}
		if governance.Redistribution != "prohibited" {
			return errors.New("biometric ground truth redistribution must be prohibited")
		}
		if governance.PrivacyReviewRef == "" {
			return errors.New("biometric ground truth requires privacy_review_ref")
		}
		if !containsString(governance.AuthorizedRoles, "privacy-reviewer") {
			return errors.New("biometric ground truth requires privacy-reviewer access role")
		}
	}
	if manifest.LegalBasis == "written-authorization" && governance.ConsentOrAuthorityRef == "" {
		return errors.New("written authorization requires consent_or_authority_ref")
	}
	return nil
}

func validateUniqueEnum(name string, values []string, allowed map[string]struct{}) error {
	if len(values) == 0 {
		return fmt.Errorf("dataset governance requires at least one %s", name)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("unsupported %s %q", name, value)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate %s %q", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validRetentionDate(value string) bool {
	if _, err := time.Parse("2006-01-02", value); err == nil {
		return true
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func (catalog ModelCatalog) Validate() error {
	if (catalog.SchemaVersion != 1 && catalog.SchemaVersion != 2) || len(catalog.Models) == 0 {
		return errors.New("model catalog requires schema_version 1 or 2 and at least one model")
	}
	ids := make(map[string]struct{}, len(catalog.Models))
	files := make(map[string]struct{}, len(catalog.Models))
	for i, model := range catalog.Models {
		if model.ID == "" || model.Purpose == "" || model.Version == "" {
			return fmt.Errorf("model %d requires id, purpose, and version", i)
		}
		if model.Status != "candidate" && model.Status != "approved" && model.Status != "rejected" {
			return fmt.Errorf("model %q has unsupported status %q", model.ID, model.Status)
		}
		if _, exists := ids[model.ID]; exists {
			return fmt.Errorf("duplicate model id %q", model.ID)
		}
		ids[model.ID] = struct{}{}
		if catalog.SchemaVersion == 1 {
			if len(model.Artifacts) != 0 || model.Directory != "" || model.PackageSHA256 != "" {
				return fmt.Errorf("model %q mixes schema_version 1 file and package fields", model.ID)
			}
			if err := validateArtifact(model.ID, ModelArtifact{Filename: model.Filename, SHA256: model.SHA256, SizeBytes: model.SizeBytes}); err != nil {
				return err
			}
			if _, exists := files[model.Filename]; exists {
				return fmt.Errorf("duplicate model filename %q", model.Filename)
			}
			files[model.Filename] = struct{}{}
		} else {
			if model.Filename != "" || model.SHA256 != "" || model.SizeBytes != 0 {
				return fmt.Errorf("model %q mixes schema_version 2 package and legacy file fields", model.ID)
			}
			if !packageSegmentPattern.MatchString(model.Directory) {
				return fmt.Errorf("model %q directory must be one safe path segment", model.ID)
			}
			if _, exists := files[model.Directory]; exists {
				return fmt.Errorf("duplicate model directory %q", model.Directory)
			}
			files[model.Directory] = struct{}{}
			if len(model.Artifacts) == 0 || len(model.Artifacts) > 128 || !sha256Pattern.MatchString(model.PackageSHA256) {
				return fmt.Errorf("model %q requires 1..128 artifacts and a valid package_sha256", model.ID)
			}
			artifactNames := make(map[string]struct{}, len(model.Artifacts))
			for _, artifact := range model.Artifacts {
				if err := validateArtifact(model.ID, artifact); err != nil {
					return err
				}
				if _, exists := artifactNames[artifact.Filename]; exists {
					return fmt.Errorf("model %q has duplicate artifact %q", model.ID, artifact.Filename)
				}
				artifactNames[artifact.Filename] = struct{}{}
			}
			if actual := ModelPackageDigest(model.Artifacts); actual != model.PackageSHA256 {
				return fmt.Errorf("model %q package_sha256 does not match artifact manifest", model.ID)
			}
		}
		if !strings.HasPrefix(model.SourceURL, "https://") {
			return fmt.Errorf("model %q source_url must use https", model.ID)
		}
		if err := validateLicense(model.ID+" code", model.CodeLicense); err != nil {
			return err
		}
		if err := validateLicense(model.ID+" weights", model.WeightLicense); err != nil {
			return err
		}
		if len(model.Architectures) == 0 {
			return fmt.Errorf("model %q has no architecture evidence", model.ID)
		}
	}
	return nil
}

func validateArtifact(modelID string, artifact ModelArtifact) error {
	if !packageSegmentPattern.MatchString(artifact.Filename) || path.Base(artifact.Filename) != artifact.Filename {
		return fmt.Errorf("model %q artifact filename must be one safe path segment", modelID)
	}
	if !sha256Pattern.MatchString(artifact.SHA256) || artifact.SizeBytes <= 0 {
		return fmt.Errorf("model %q artifact %q requires valid sha256 and positive size_bytes", modelID, artifact.Filename)
	}
	return nil
}

func validateLicense(subject string, license LicenseEvidence) error {
	if license.ID == "" || !strings.HasPrefix(license.URL, "https://") {
		return fmt.Errorf("%s license requires id and https evidence URL", subject)
	}
	return nil
}
