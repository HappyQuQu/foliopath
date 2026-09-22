package face

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

var ErrDeploymentIdentity = errors.New("incompatible face deployment identity")

// DeploymentIdentity is compatibility, not attestation. Field order and JSON
// names are part of the version-1 fingerprint encoding.
type DeploymentIdentity struct {
	Version             int              `json:"version"`
	Protocol            string           `json:"protocol"`
	Model               string           `json:"model"`
	Dimension           int              `json:"dimension"`
	RuntimeSHA256       string           `json:"runtime_sha256"`
	CacheManifestSHA256 string           `json:"cache_manifest_sha256"`
	PreprocessingSHA256 string           `json:"preprocessing_sha256"`
	DetectorSHA256      string           `json:"detector_sha256"`
	EmbedderSHA256      string           `json:"embedder_sha256"`
	Threshold           ThresholdProfile `json:"threshold"`
}

func (i DeploymentIdentity) Fingerprint() (string, error) {
	if i.Version != 1 || i.Protocol != "foliopath-face-v1" || i.Model != "buffalo_l" || i.Dimension != 512 {
		return "", ErrDeploymentIdentity
	}
	for _, digest := range []string{i.RuntimeSHA256, i.CacheManifestSHA256, i.PreprocessingSHA256, i.DetectorSHA256, i.EmbedderSHA256} {
		if !thresholdSHA256Pattern.MatchString(digest) {
			return "", ErrDeploymentIdentity
		}
	}
	if _, err := i.thresholdHash(); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(i)
	if err != nil {
		return "", ErrDeploymentIdentity
	}
	return deploymentDigest(encoded), nil
}

func (i DeploymentIdentity) thresholdHash() (string, error) {
	encoded, err := json.Marshal(i.Threshold)
	if err != nil {
		return "", ErrDeploymentIdentity
	}
	if _, err := ParseThresholdProfile(encoded); err != nil {
		return "", ErrDeploymentIdentity
	}
	return deploymentDigest(encoded), nil
}

func deploymentDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (i DeploymentIdentity) Metadata() (aimodel.DeploymentFaceMetadata, error) {
	if _, err := i.Fingerprint(); err != nil {
		return aimodel.DeploymentFaceMetadata{}, err
	}
	hash, _ := i.thresholdHash()
	return aimodel.DeploymentFaceMetadata{FaceRuntimeMetadata: aimodel.FaceRuntimeMetadata{
		EmbeddingDimension: int64(i.Dimension), ThresholdProfile: i.Threshold.ProfileID},
		DetectorContentHash: i.DetectorSHA256, EmbedderContentHash: i.EmbedderSHA256, ThresholdProfileHash: hash}, nil
}

func (i DeploymentIdentity) MatchesGeneration(g aimodel.FaceGeneration) bool {
	metadata, err := i.Metadata()
	return err == nil && aimodel.ValidateFaceGeneration(g) == nil && g.PackageID == i.Model &&
		g.EmbeddingDimension == metadata.EmbeddingDimension && g.DetectorContentHash == metadata.DetectorContentHash &&
		g.EmbedderContentHash == metadata.EmbedderContentHash && g.ThresholdProfile == metadata.ThresholdProfile &&
		g.ThresholdProfileHash == metadata.ThresholdProfileHash
}

type DeploymentActivationCommit struct {
	Activation aimodel.FaceActivationCommit
	Identity   DeploymentIdentity
}

type DeploymentActivationRepository interface {
	CommitFaceDeploymentActivation(context.Context, DeploymentActivationCommit) (aimodel.Operation, error)
}
