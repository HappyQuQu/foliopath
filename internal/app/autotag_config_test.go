package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"strings"
	"testing"
)

func TestAutomaticTagDeploymentConfigurationPinsIndependentModelAndPreprocessing(t *testing.T) {
	config := deploymentConfigFixture()
	config.Tag = &tagDeploymentConfiguration{Identity: remoteml.TagIdentity{Version: 1, Protocol: "foliopath-tag-v1", Model: "JoyTag", ModelSHA256: strings.Repeat("a", 64), PolicySHA256: strings.Repeat("c", 64), RuntimeContractSHA256: config.Identity.RuntimeContractSHA256, CacheManifestSHA256: strings.Repeat("d", 64)}}
	raw, _ := json.Marshal(config)
	parsed, err := deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)})
	if err != nil || parsed.Tag == nil {
		t.Fatalf("valid config %v", err)
	}
	identity, _ := config.Tag.Identity.Fingerprint()
	expected := sha256.Sum256([]byte("foliopath-autotag-input-v1\x00" + identity + "\x00" + semantic.RemoteImagePreprocessingContract))
	if parsed.Tag.fingerprint() != hex.EncodeToString(expected[:]) {
		t.Fatal("preprocessing unbound")
	}
	for _, change := range []func(*remoteml.TagIdentity){func(v *remoteml.TagIdentity) { v.ModelSHA256 = "" }, func(v *remoteml.TagIdentity) { v.PolicySHA256 = "" }, func(v *remoteml.TagIdentity) { v.RuntimeContractSHA256 = strings.Repeat("e", 64) }, func(v *remoteml.TagIdentity) { v.Protocol = "foliopath-ml-v1" }} {
		invalid := config
		tag := *config.Tag
		invalid.Tag = &tag
		change(&invalid.Tag.Identity)
		raw, _ = json.Marshal(invalid)
		if _, err = deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)}); err == nil {
			t.Fatal("invalid tag identity accepted")
		}
	}
}
