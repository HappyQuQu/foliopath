package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"runtime"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const deploymentEnvironment = "FOLIOPATH_AI_DEPLOYMENT"

type deploymentConfiguration struct {
	Origin     string                   `json:"origin"`
	Identity   remoteml.ServiceIdentity `json:"identity"`
	ModelBytes int64                    `json:"model_bytes"`
	LicenseID  string                   `json:"license_id"`
}

func deploymentFromEnvironment(environ []string) (*deploymentConfiguration, error) {
	var result *deploymentConfiguration
	for _, entry := range environ {
		name, value, ok := strings.Cut(entry, "=")
		if name != deploymentEnvironment {
			continue
		}
		if !ok || result != nil || len(value) > 8192 {
			return nil, errInvalidConfiguration
		}
		var config deploymentConfiguration
		decoder := json.NewDecoder(strings.NewReader(value))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&config) != nil || decoder.Decode(new(any)) != io.EOF {
			return nil, errInvalidConfiguration
		}
		client, err := remoteml.NewConfigured(config.Origin, config.Identity)
		if err != nil {
			return nil, errInvalidConfiguration
		}
		client.Close()
		if aimodel.ValidatePackage(config.model()) != nil {
			return nil, errInvalidConfiguration
		}
		result = &config
	}
	return result, nil
}

func (c deploymentConfiguration) space() semantic.SpaceIdentity {
	// Versioned algorithm contract, not a claim about the deployed main image.
	digest := sha256.Sum256([]byte("foliopath-remote-image-v1:jpeg,png,webp,gif:first-frame:autorotate:srgb:no-alpha:uchar:lanczos3:aspect-longedge1024:png-strip:source32MiB:pixels16000000:output8MiB"))
	return semantic.SpaceIdentity{Version: 1, Protocol: c.Identity.Protocol, Model: c.Identity.Model, Dimension: c.Identity.Dimension,
		RuntimeSHA256: c.Identity.RuntimeContractSHA256, CacheManifestSHA256: c.Identity.CacheManifestSHA256, PreprocessingSHA256: hex.EncodeToString(digest[:])}
}

func (c deploymentConfiguration) model() aimodel.VerifiedPackage {
	fingerprint, _ := c.space().Fingerprint()
	return aimodel.VerifiedPackage{PackageID: c.Identity.Model, Purpose: aimodel.PurposeSemanticImageText, Version: c.Identity.Protocol,
		Architecture: runtime.GOARCH, ContentHash: fingerprint, LicenseID: c.LicenseID, PackageSizeByte: c.ModelBytes}
}
