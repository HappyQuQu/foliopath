package aimodel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
)

const (
	MaxManifestBytes = 64 << 10
	MaxPackageFiles  = 16
	MaxPackageBytes  = int64(4 << 30)
)

var ErrModelIncompatible = errors.New("AI model package is incompatible")

type ManifestFile struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Role   string `json:"role"`
}

type Manifest struct {
	FormatVersion int            `json:"formatVersion"`
	PackageID     string         `json:"packageId"`
	Purpose       string         `json:"purpose"`
	Version       string         `json:"version"`
	Architecture  string         `json:"architecture"`
	LicenseID     string         `json:"licenseId"`
	Files         []ManifestFile `json:"files"`
}

type CatalogEntry struct {
	Manifest             Manifest
	ContentHash          string
	RuntimeArchitectures []string
}

type FileFact struct {
	Name    string
	Size    int64
	SHA256  string
	Regular bool
}

type Catalog struct {
	entries       map[string]CatalogEntry
	byContentHash map[string]CatalogEntry
}

func NewCatalog(entries []CatalogEntry) (*Catalog, error) {
	result := &Catalog{
		entries:       make(map[string]CatalogEntry, len(entries)),
		byContentHash: make(map[string]CatalogEntry, len(entries)),
	}
	for _, entry := range entries {
		if validateManifest(entry.Manifest) != nil || !hexSHA256.MatchString(entry.ContentHash) ||
			len(entry.RuntimeArchitectures) == 0 {
			return nil, ErrInvalidModel
		}
		architectures := append([]string(nil), entry.RuntimeArchitectures...)
		slices.Sort(architectures)
		architectures = slices.Compact(architectures)
		if len(architectures) != len(entry.RuntimeArchitectures) {
			return nil, ErrInvalidModel
		}
		for _, architecture := range architectures {
			if architecture != "amd64" && architecture != "arm64" {
				return nil, ErrInvalidModel
			}
		}
		if _, exists := result.entries[entry.Manifest.PackageID]; exists {
			return nil, ErrInvalidModel
		}
		if _, exists := result.byContentHash[entry.ContentHash]; exists {
			return nil, ErrInvalidModel
		}
		entry.RuntimeArchitectures = architectures
		result.entries[entry.Manifest.PackageID] = entry
		result.byContentHash[entry.ContentHash] = entry
	}
	return result, nil
}

func (catalog *Catalog) PackageByContentHash(contentHash, runtimeArchitecture string) (VerifiedPackage, Manifest, bool) {
	if catalog == nil || !hexSHA256.MatchString(contentHash) ||
		(runtimeArchitecture != "amd64" && runtimeArchitecture != "arm64") {
		return VerifiedPackage{}, Manifest{}, false
	}
	entry, exists := catalog.byContentHash[contentHash]
	if !exists || !slices.Contains(entry.RuntimeArchitectures, runtimeArchitecture) {
		return VerifiedPackage{}, Manifest{}, false
	}
	manifest := entry.Manifest
	manifest.Files = append([]ManifestFile(nil), entry.Manifest.Files...)
	var total int64
	for _, file := range manifest.Files {
		total += file.Size
	}
	return VerifiedPackage{
		PackageID: manifest.PackageID, Purpose: manifest.Purpose, Version: manifest.Version,
		Architecture: runtimeArchitecture, ContentHash: contentHash, LicenseID: manifest.LicenseID,
		PackageSizeByte: total,
	}, manifest, true
}

func (catalog *Catalog) Verify(manifestBytes []byte, facts []FileFact, runtimeArchitecture string) (VerifiedPackage, error) {
	if catalog == nil || len(manifestBytes) == 0 || len(manifestBytes) > MaxManifestBytes ||
		(runtimeArchitecture != "amd64" && runtimeArchitecture != "arm64") {
		return VerifiedPackage{}, ErrModelIncompatible
	}
	if err := rejectDuplicateJSONKeys(manifestBytes); err != nil {
		return VerifiedPackage{}, ErrModelIncompatible
	}
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return VerifiedPackage{}, ErrModelIncompatible
	}
	if err := ensureJSONEOF(decoder); err != nil || validateManifest(manifest) != nil {
		return VerifiedPackage{}, ErrModelIncompatible
	}
	entry, exists := catalog.entries[manifest.PackageID]
	if !exists || !equalManifest(manifest, entry.Manifest) ||
		!slices.Contains(entry.RuntimeArchitectures, runtimeArchitecture) {
		return VerifiedPackage{}, ErrModelIncompatible
	}
	if err := verifyFileFacts(manifest.Files, facts); err != nil {
		return VerifiedPackage{}, err
	}
	var total int64
	for _, file := range manifest.Files {
		total += file.Size
	}
	verified := VerifiedPackage{
		PackageID:       manifest.PackageID,
		Purpose:         manifest.Purpose,
		Version:         manifest.Version,
		Architecture:    runtimeArchitecture,
		ContentHash:     entry.ContentHash,
		LicenseID:       manifest.LicenseID,
		PackageSizeByte: total,
	}
	if err := ValidatePackage(verified); err != nil {
		return VerifiedPackage{}, ErrModelIncompatible
	}
	return verified, nil
}

func (catalog *Catalog) Manifest(packageID string) (Manifest, bool) {
	if catalog == nil {
		return Manifest{}, false
	}
	entry, exists := catalog.entries[packageID]
	if !exists {
		return Manifest{}, false
	}
	manifest := entry.Manifest
	manifest.Files = append([]ManifestFile(nil), entry.Manifest.Files...)
	return manifest, true
}

func validateManifest(manifest Manifest) error {
	if manifest.FormatVersion != 1 || manifest.PackageID == "" || len(manifest.PackageID) > 128 ||
		manifest.Purpose != PurposeSemanticImageText || manifest.Version == "" || len(manifest.Version) > 64 ||
		manifest.Architecture != "portable-onnx" || manifest.LicenseID == "" || len(manifest.LicenseID) > 128 ||
		len(manifest.Files) == 0 || len(manifest.Files) > MaxPackageFiles {
		return ErrInvalidModel
	}
	names := make(map[string]struct{}, len(manifest.Files))
	roles := make(map[string]struct{}, len(manifest.Files))
	var total int64
	for _, file := range manifest.Files {
		if file.Name == "" || len(file.Name) > 255 || path.Base(file.Name) != file.Name ||
			strings.ContainsAny(file.Name, "\\\x00") || file.Name == "." || file.Name == ".." ||
			file.Size <= 0 || !hexSHA256.MatchString(file.SHA256) ||
			(file.Role != "image_encoder" && file.Role != "text_encoder" && file.Role != "tokenizer") {
			return ErrInvalidModel
		}
		if _, exists := names[file.Name]; exists {
			return ErrInvalidModel
		}
		if _, exists := roles[file.Role]; exists {
			return ErrInvalidModel
		}
		names[file.Name] = struct{}{}
		roles[file.Role] = struct{}{}
		if total > MaxPackageBytes-file.Size {
			return ErrInvalidModel
		}
		total += file.Size
	}
	for _, required := range []string{"image_encoder", "text_encoder", "tokenizer"} {
		if _, exists := roles[required]; !exists {
			return ErrInvalidModel
		}
	}
	return nil
}

func equalManifest(left, right Manifest) bool {
	return left.FormatVersion == right.FormatVersion && left.PackageID == right.PackageID &&
		left.Purpose == right.Purpose && left.Version == right.Version &&
		left.Architecture == right.Architecture && left.LicenseID == right.LicenseID &&
		slices.Equal(left.Files, right.Files)
}

func verifyFileFacts(expected []ManifestFile, actual []FileFact) error {
	if len(actual) != len(expected) {
		return ErrModelIncompatible
	}
	byName := make(map[string]FileFact, len(actual))
	for _, fact := range actual {
		if !fact.Regular || fact.Name == "" {
			return ErrModelIncompatible
		}
		if _, exists := byName[fact.Name]; exists {
			return ErrModelIncompatible
		}
		byName[fact.Name] = fact
	}
	for _, file := range expected {
		fact, exists := byName[file.Name]
		if !exists || fact.Size != file.Size || fact.SHA256 != file.SHA256 {
			return ErrModelIncompatible
		}
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("trailing JSON value")
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("invalid JSON object key")
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return errors.New("unexpected JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	return ensureJSONEOF(decoder)
}
