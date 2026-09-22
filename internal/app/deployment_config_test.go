package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

func deploymentConfigFixture() deploymentConfiguration {
	return deploymentConfiguration{Origin: "http://machine-learning:3003", ModelBytes: 123, LicenseID: "unreviewed-test", Identity: remoteml.ServiceIdentity{Version: 1, Protocol: "foliopath-ml-v1", Model: remoteml.ModelName, Dimension: 512, CacheManifestSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: strings.Repeat("b", 64)}}
}

func TestSemanticPreprocessingUpgradeChangesGenerationIdentity(t *testing.T) {
	current := deploymentConfigFixture().space()
	expected := sha256.Sum256([]byte(semantic.RemoteImagePreprocessingContract))
	if current.PreprocessingSHA256 != hex.EncodeToString(expected[:]) {
		t.Fatal("deployment does not bind canonical preprocessing contract")
	}
	old := current
	v1 := sha256.Sum256([]byte("foliopath-remote-image-v1:jpeg,png,webp,gif:first-frame:autorotate:srgb:no-alpha:uchar:lanczos3:aspect-longedge1024:png-strip:source32MiB:pixels16000000:output8MiB"))
	old.PreprocessingSHA256 = hex.EncodeToString(v1[:])
	before, err := old.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	after, err := current.Fingerprint()
	if err != nil || before == after {
		t.Fatalf("old vectors could alias new preprocessing: %v", err)
	}
}

func TestDeploymentConfigurationIsOptionalStrictAndOffline(t *testing.T) {
	config := deploymentConfigFixture()
	raw, _ := json.Marshal(config)
	if got, err := loadConfiguration(Input{}); err != nil || got.deployment != nil {
		t.Fatalf("default=%+v %v", got, err)
	}
	value := deploymentEnvironment + "=" + string(raw)
	if got, err := loadConfiguration(Input{Environ: []string{value}}); err != nil || got.deployment == nil {
		t.Fatalf("configured=%+v %v", got, err)
	}
	for _, environ := range [][]string{{value, value}, {deploymentEnvironment + "="}, {deploymentEnvironment + "={}"}, {value + " trailing"}, {strings.TrimSuffix(value, "}") + `,"unknown":true}`}} {
		if _, err := loadConfiguration(Input{Environ: environ}); err == nil {
			t.Fatalf("accepted invalid config %v", environ)
		}
	}
}

func TestDeploymentBootstrapOfflineRecoveryAndIdempotency(t *testing.T) {
	t.Run("semantic", func(t *testing.T) { testDeploymentBootstrap(t, false) })
	t.Run("face", func(t *testing.T) { testDeploymentBootstrap(t, true) })
}

func faceDeploymentConfigFixture() *faceDeploymentConfiguration {
	i := deploymentConfigFixture().Identity
	i.Protocol = "foliopath-face-v1"
	i.Model = "buffalo_l"
	return &faceDeploymentConfiguration{Identity: i, DetectorSHA256: strings.Repeat("c", 64), EmbedderSHA256: strings.Repeat("d", 64),
		ModelBytes: 123, LicenseID: "test-only", Threshold: face.ThresholdProfile{SchemaVersion: 1, ProfileID: "test-only", CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2, QualitySummarySHA256: strings.Repeat("e", 64)}}
}

func TestFaceDeploymentConfigurationRejectsMismatches(t *testing.T) {
	config := deploymentConfigFixture()
	config.Face = faceDeploymentConfigFixture()
	raw, _ := json.Marshal(config)
	if _, err := deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)}); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*faceDeploymentConfiguration){
		func(c *faceDeploymentConfiguration) { c.Identity.CacheManifestSHA256 = strings.Repeat("0", 64) },
		func(c *faceDeploymentConfiguration) { c.Identity.Protocol = "foliopath-ml-v1" },
		func(c *faceDeploymentConfiguration) { c.DetectorSHA256 = "" },
		func(c *faceDeploymentConfiguration) { c.Threshold.QualitySummarySHA256 = "" },
		func(c *faceDeploymentConfiguration) { c.ModelBytes = 0 },
	} {
		config.Face = faceDeploymentConfigFixture()
		change(config.Face)
		raw, _ := json.Marshal(config)
		if _, err := deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)}); err == nil {
			t.Fatal("invalid face config accepted")
		}
	}
}

func testDeploymentBootstrap(t *testing.T, withFace bool) {
	ctx := context.Background()
	config := deploymentConfigFixture()
	if withFace {
		config.Face = faceDeploymentConfigFixture()
	}
	var online atomic.Bool
	var block atomic.Bool
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if block.Load() {
			entered <- struct{}{}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		if !online.Load() {
			w.WriteHeader(503)
			return
		}
		if withFace && r.URL.Path == "/api/v1/face/identity" {
			_ = json.NewEncoder(w).Encode(config.Face.Identity)
		} else {
			_ = json.NewEncoder(w).Encode(config.Identity)
		}
	}))
	defer server.Close()
	config.Origin = server.URL
	client, err := remoteml.NewConfigured(config.Origin, config.Identity)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "deployment.db"), sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	models, err := aimodel.NewService(store, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := aimodel.NewActivationAdmissionService(store, jobs.NewSignal(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := deploymentBootstrap{client: client, models: models, admission: admission, config: config}
	if withFace {
		bootstrap.faceClient, err = client.BindFace(config.Face.Identity)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := bootstrap.reconcile(ctx); err == nil {
		t.Fatal("offline service accepted")
	}
	snapshot, err := models.List(ctx)
	if err != nil || len(snapshot.Items) != 0 {
		t.Fatal("offline first run registered model")
	}
	online.Store(true)
	if err := bootstrap.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	snapshot, err = models.List(ctx)
	if err != nil || len(snapshot.Items) != 1 {
		t.Fatal("online service not registered")
	}
	id := snapshot.Items[0].ID
	if err := bootstrap.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	again, err := models.List(ctx)
	if err != nil || len(again.Items) != 1 || again.Items[0].ID != id || again.Revision != snapshot.Revision {
		t.Fatal("reconciliation not idempotent")
	}
	block.Store(true)
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- client.VerifyIdentity(ctx) }()
	}
	<-entered
	<-entered
	busyErr := bootstrap.reconcile(ctx)
	busySnapshot, busyListErr := models.List(ctx)
	block.Store(false)
	close(release)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if !errors.Is(busyErr, remoteml.ErrBusy) || busyListErr != nil || busySnapshot.Revision != again.Revision {
		t.Fatal("local request saturation changed model availability")
	}
	online.Store(false)
	if err := bootstrap.reconcile(ctx); err == nil {
		t.Fatal("offline recovery state missed")
	}
	model, err := models.Get(ctx, id)
	if err != nil || model.State != aimodel.StateUnavailable {
		t.Fatal("offline model not retained/unavailable")
	}
	online.Store(true)
	if err := bootstrap.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	model, err = models.Get(ctx, id)
	if err != nil || model.State != aimodel.StateAvailable {
		t.Fatal("service did not recover")
	}
	changed := config
	changed.Identity.RuntimeContractSHA256 = strings.Repeat("0", 64)
	if withFace {
		faceConfig := *config.Face
		faceConfig.Identity.RuntimeContractSHA256 = changed.Identity.RuntimeContractSHA256
		changed.Face = &faceConfig
	}
	replacementClient, err := remoteml.NewConfigured(changed.Origin, changed.Identity)
	if err != nil {
		t.Fatal(err)
	}
	defer replacementClient.Close()
	replacement := deploymentBootstrap{client: replacementClient, models: models, admission: admission, config: changed}
	if withFace {
		replacement.faceClient, err = replacementClient.BindFace(changed.Face.Identity)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := replacement.reconcile(ctx); err == nil {
		t.Fatal("wrong replacement identity accepted")
	}
	model, err = models.Get(ctx, id)
	if err != nil || model.State != aimodel.StateUnavailable {
		t.Fatal("unconfigured previous identity falsely available", model, err)
	}
	if err := bootstrap.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	model, err = models.Get(ctx, id)
	if err != nil || model.State != aimodel.StateAvailable {
		t.Fatal("restoring original deployment did not recover", model, err)
	}
}
