package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

func deploymentConfigFixture() deploymentConfiguration {
	return deploymentConfiguration{Origin: "http://machine-learning:3003", ModelBytes: 123, LicenseID: "unreviewed-test", Identity: remoteml.ServiceIdentity{Version: 1, Protocol: "foliopath-ml-v1", Model: remoteml.ModelName, Dimension: 512, CacheManifestSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: strings.Repeat("b", 64)}}
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
	ctx := context.Background()
	config := deploymentConfigFixture()
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
		_ = json.NewEncoder(w).Encode(config.Identity)
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
	bootstrap := deploymentBootstrap{client, models, admission, config}
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
}
