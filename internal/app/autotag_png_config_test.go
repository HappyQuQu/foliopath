package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestAutomaticTagPNGProfileChangesOnlyTagIdentity(t *testing.T) {
	legacy := deploymentConfigFixture()
	legacy.ImagePreprocessing = semantic.RemoteImagePNG
	legacy.Face = faceDeploymentConfigFixture()
	legacy.Tag = &tagDeploymentConfiguration{Identity: remoteml.TagIdentity{Version: 1, Protocol: "foliopath-tag-v1", Model: "JoyTag", ModelSHA256: strings.Repeat("a", 64), PolicySHA256: strings.Repeat("c", 64), RuntimeContractSHA256: legacy.Identity.RuntimeContractSHA256, CacheManifestSHA256: strings.Repeat("d", 64)}}
	changed := legacy
	tag := *legacy.Tag
	changed.Tag = &tag
	changed.Tag.ImagePreprocessing = semantic.RemoteImagePNG
	raw, err := json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)})
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := parsed.Tag.Identity.Fingerprint()
	want := sha256.Sum256([]byte("foliopath-autotag-input-v1\x00" + identity + "\x00" + semantic.RemotePNGPreprocessingContract))
	if parsed.Tag.fingerprint() != hex.EncodeToString(want[:]) || parsed.Tag.fingerprint() == legacy.Tag.fingerprint() {
		t.Fatal("tag PNG input does not bind a distinct identity")
	}
	if parsed.space() != legacy.space() || parsed.Face.space() != legacy.Face.space() || parsed.Tag.Identity != legacy.Tag.Identity {
		t.Fatal("tag input changed semantic/face/model identity")
	}
	// Changing semantic preprocessing independently must not change tag input.
	parsed.ImagePreprocessing = semantic.RemoteImageLegacy
	if parsed.Tag.fingerprint() != changed.Tag.fingerprint() {
		t.Fatal("tag inherited semantic profile")
	}
	for _, invalid := range []semantic.RemoteImageProfile{"v3", "latest", "PNG-v4", "png-v4 ", "png-v5"} {
		changed.Tag.ImagePreprocessing = invalid
		raw, _ = json.Marshal(changed)
		if _, err := deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)}); err == nil {
			t.Fatalf("invalid tag profile accepted: %q", invalid)
		}
	}
}
