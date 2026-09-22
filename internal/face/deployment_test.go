package face

import (
	"strings"
	"testing"
)

func TestDeploymentIdentityBindsEveryCompatibilityField(t *testing.T) {
	base := DeploymentIdentity{Version: 1, Protocol: "foliopath-face-v1", Model: "buffalo_l", Dimension: 512,
		RuntimeSHA256: strings.Repeat("a", 64), CacheManifestSHA256: strings.Repeat("b", 64), PreprocessingSHA256: strings.Repeat("c", 64),
		DetectorSHA256: strings.Repeat("d", 64), EmbedderSHA256: strings.Repeat("e", 64),
		Threshold: ThresholdProfile{SchemaVersion: 1, ProfileID: "test-only", CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2, QualitySummarySHA256: strings.Repeat("f", 64)}}
	fingerprint, err := base.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	again, _ := base.Fingerprint()
	if again != fingerprint {
		t.Fatal("not deterministic")
	}
	for _, change := range []func(*DeploymentIdentity){
		func(i *DeploymentIdentity) { i.RuntimeSHA256 = strings.Repeat("0", 64) },
		func(i *DeploymentIdentity) { i.CacheManifestSHA256 = strings.Repeat("0", 64) },
		func(i *DeploymentIdentity) { i.PreprocessingSHA256 = strings.Repeat("0", 64) },
		func(i *DeploymentIdentity) { i.DetectorSHA256 = strings.Repeat("0", 64) },
		func(i *DeploymentIdentity) { i.EmbedderSHA256 = strings.Repeat("0", 64) },
		func(i *DeploymentIdentity) { i.Threshold.CoreSimilarity = .9 },
		func(i *DeploymentIdentity) { i.Threshold.QualitySummarySHA256 = strings.Repeat("0", 64) },
	} {
		changed := base
		change(&changed)
		got, err := changed.Fingerprint()
		if err != nil || got == fingerprint {
			t.Fatal("unbound field", err)
		}
	}
	for _, change := range []func(*DeploymentIdentity){
		func(i *DeploymentIdentity) { i.Protocol = "foliopath-ml-v1" }, func(i *DeploymentIdentity) { i.Dimension = 768 },
		func(i *DeploymentIdentity) { i.DetectorSHA256 = "" }, func(i *DeploymentIdentity) { i.Threshold.EdgeSimilarity = .99 },
	} {
		changed := base
		change(&changed)
		if _, err := changed.Fingerprint(); err == nil {
			t.Fatal("invalid identity accepted")
		}
	}
}
