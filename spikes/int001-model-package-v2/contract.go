// Package modelpackagev2 contains isolated executable package proposals for
// ADR-0014 and ADR-0015.
// It is not imported by production packages.
package modelpackagev2

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path"
	"regexp"
	"strings"
)

const (
	maxManifestBytes = 64 << 10
	maxPackageBytes  = int64(4 << 30)

	ImagePreprocessContract = "siglip-rgb224-bicubic-v1"
	TextCanonicalContract   = "siglip-transformers-4.56.2-v1"
	TokenizerContract       = "sentencepiece-32k-unk2-eos1-pad1-seq64-v1"
	EmbeddingContract       = "siglip-768-l2-f16le-v1"
)

var errInvalid = errors.New("invalid proposed model package v2")
var hexSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Contracts struct {
	ImagePreprocess     string `json:"imagePreprocess"`
	TextCanonical       string `json:"textCanonicalization"`
	Tokenizer           string `json:"tokenizer"`
	EmbeddingAndStorage string `json:"embeddingAndStorage"`
}

type File struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Role   string `json:"role"`
}

type Manifest struct {
	FormatVersion int       `json:"formatVersion"`
	PackageID     string    `json:"packageId"`
	Purpose       string    `json:"purpose"`
	Version       string    `json:"version"`
	Architecture  string    `json:"architecture"`
	LicenseID     string    `json:"licenseId"`
	Contracts     Contracts `json:"contracts"`
	Files         []File    `json:"files"`
}

func Parse(data []byte) (Manifest, error) {
	if len(data) == 0 || len(data) > maxManifestBytes || duplicateKey(data) {
		return Manifest{}, errInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var result Manifest
	if decoder.Decode(&result) != nil || !jsonEOF(decoder) || !valid(result) {
		return Manifest{}, errInvalid
	}
	return result, nil
}

func valid(value Manifest) bool {
	if value.FormatVersion != 2 || value.PackageID == "" || len(value.PackageID) > 128 ||
		value.Purpose != "semantic_image_text" || value.Version == "" || len(value.Version) > 64 ||
		value.Architecture != "portable-onnx" || value.LicenseID == "" || len(value.LicenseID) > 128 ||
		value.Contracts.ImagePreprocess != ImagePreprocessContract ||
		value.Contracts.TextCanonical != TextCanonicalContract || value.Contracts.Tokenizer != TokenizerContract ||
		value.Contracts.EmbeddingAndStorage != EmbeddingContract || len(value.Files) != 3 {
		return false
	}
	required := map[string]bool{"image_encoder": false, "text_encoder": false, "sentencepiece_model": false}
	names := map[string]bool{}
	var total int64
	for _, file := range value.Files {
		if file.Name == "" || len(file.Name) > 255 || path.Base(file.Name) != file.Name ||
			strings.ContainsAny(file.Name, "\\\x00") || file.Name == "." || file.Name == ".." ||
			file.Size <= 0 || file.Size > maxPackageBytes || !hexSHA256.MatchString(file.SHA256) ||
			names[file.Name] || !requiredRole(required, file.Role) || required[file.Role] ||
			total > maxPackageBytes-file.Size {
			return false
		}
		names[file.Name] = true
		required[file.Role] = true
		total += file.Size
	}
	return required["image_encoder"] && required["text_encoder"] && required["sentencepiece_model"]
}

func requiredRole(roles map[string]bool, role string) bool {
	_, exists := roles[role]
	return exists
}

func jsonEOF(decoder *json.Decoder) bool {
	var extra any
	return errors.Is(decoder.Decode(&extra), io.EOF)
}

func duplicateKey(data []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() bool
	walk = func() bool {
		token, err := decoder.Token()
		if err != nil {
			return true
		}
		delimiter, object := token.(json.Delim)
		if !object {
			return false
		}
		if delimiter == '{' {
			seen := map[string]bool{}
			for decoder.More() {
				key, ok := mustToken(decoder).(string)
				if !ok || seen[key] {
					return true
				}
				seen[key] = true
				if walk() {
					return true
				}
			}
			_, err = decoder.Token()
			return err != nil
		}
		if delimiter == '[' {
			for decoder.More() {
				if walk() {
					return true
				}
			}
			_, err = decoder.Token()
			return err != nil
		}
		return true
	}
	return walk()
}

func mustToken(decoder *json.Decoder) json.Token {
	value, _ := decoder.Token()
	return value
}
