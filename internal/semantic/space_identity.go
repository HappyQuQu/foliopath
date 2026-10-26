package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrInvalidSpaceIdentity = errors.New("invalid semantic space identity")
	ErrIncompatibleSpace    = errors.New("incompatible semantic space")
)

// SpaceIdentity describes an embedding space, not evidence of remote identity.
// The deployment admission owner must verify the digests before constructing it.
// Field order is part of the v1 fingerprint encoding; do not reorder these fields.
type SpaceIdentity struct {
	Version             int    `json:"version"`
	Protocol            string `json:"protocol"`
	Model               string `json:"model"`
	Dimension           int    `json:"dimension"`
	RuntimeSHA256       string `json:"runtime_sha256"`
	CacheManifestSHA256 string `json:"cache_manifest_sha256"`
	PreprocessingSHA256 string `json:"preprocessing_sha256"`
}

// Fingerprint provides deterministic compatibility identity, not attestation.
func (identity SpaceIdentity) Fingerprint() (string, error) {
	if identity.Version != 1 || !validSpaceToken(identity.Protocol) || !validSpaceToken(identity.Model) ||
		identity.Dimension < 1 || identity.Dimension > 65536 ||
		!validSpaceDigest(identity.RuntimeSHA256) || !validSpaceDigest(identity.CacheManifestSHA256) ||
		!validSpaceDigest(identity.PreprocessingSHA256) {
		return "", ErrInvalidSpaceIdentity
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		return "", ErrInvalidSpaceIdentity
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

// RequireSpace rejects legacy/unbound generations as well as mismatched spaces.
// It performs no activation or persistence and never falls back by dimension.
func RequireSpace(generationFingerprint string, runtime SpaceIdentity) error {
	fingerprint, err := runtime.Fingerprint()
	if err != nil {
		return err
	}
	if !validSpaceDigest(generationFingerprint) || generationFingerprint != fingerprint {
		return ErrIncompatibleSpace
	}
	return nil
}

func validSpaceDigest(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validSpaceToken(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.') {
			return false
		}
	}
	return true
}
