package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

func ocrDeploymentFixture() ocr.Identity {
	return ocr.Identity{Version: 1, Protocol: "foliopath-ocr-v1", Model: ocr.Model, DetectorSHA256: strings.Repeat("a", 64), RecognizerSHA256: strings.Repeat("b", 64), InputContractSHA256: ocr.InputContractSHA256(), OutputContractSHA256: ocr.OutputContractSHA256(), RuntimeContractSHA256: strings.Repeat("e", 64), CacheManifestSHA256: strings.Repeat("f", 64)}
}
func TestOCRDeploymentIsExplicitAndDoesNotChangeOtherIdentities(t *testing.T) {
	baseline := deploymentConfigFixture()
	with := baseline
	with.OCR = &ocrDeploymentConfiguration{Identity: ocrDeploymentFixture()}
	raw, _ := json.Marshal(with)
	parsed, err := deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)})
	if err != nil || parsed.OCR == nil {
		t.Fatal(parsed, err)
	}
	if baseline.space() != parsed.space() || baseline.model() != parsed.model() {
		t.Fatal("OCR modified semantic identity")
	}
	for _, mutate := range []func(*ocr.Identity){func(i *ocr.Identity) { i.InputContractSHA256 = strings.Repeat("a", 64) }, func(i *ocr.Identity) { i.OutputContractSHA256 = strings.Repeat("b", 64) }, func(i *ocr.Identity) { i.DetectorSHA256 = "" }} {
		bad := ocrDeploymentFixture()
		mutate(&bad)
		with.OCR = &ocrDeploymentConfiguration{Identity: bad}
		raw, _ := json.Marshal(with)
		if _, err = deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)}); err == nil {
			t.Fatal("accepted incompatible OCR")
		}
	}
	raw, _ = json.Marshal(baseline)
	parsed, err = deploymentFromEnvironment([]string{deploymentEnvironment + "=" + string(raw)})
	if err != nil || parsed.OCR != nil {
		t.Fatal(parsed, err)
	}
}
func TestOCRDatabaseLifecycleMethodsFailClosedOutsideOwner(t *testing.T) {
	_, database := newDatabaseComponent(t.TempDir(), newReadinessState())
	ctx := context.Background()
	if _, err := database.GetOCRSettings(ctx, 1); err == nil {
		t.Fatal("unstarted db")
	}
	if _, err := database.GetOCRCoverage(ctx, 1); err == nil {
		t.Fatal("unstarted db")
	}
	if err := database.HandoffOCRJob(ctx, ocr.Job{}, time.Now()); err == nil {
		t.Fatal("unstarted db")
	}
}

// Verify adapter error classes using real HTTP transport tests elsewhere; known
// OCR transport errors retain fixed classes and never become original failures.
func TestOCRRuntimeInterfaceAndIndependentExpectedIdentity(t *testing.T) {
	client, err := remoteml.NewConfigured("http://127.0.0.1:9", deploymentConfigFixture().Identity)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	bound, err := client.BindOCR(ocrDeploymentFixture())
	if err != nil {
		t.Fatal(err)
	}
	runtime := ocrRuntime{client: bound}
	var _ ocr.Runtime = runtime
	if runtime.Identity() != ocrDeploymentFixture() {
		t.Fatal("expected identity replaced")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = runtime.Recognize(ctx, []byte("png")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestOCRUnconfiguredApplicationDoesNotRegisterManagement(t *testing.T) {
	a, err := composeConfiguration(Input{Version: "ocr-no-config"}, configuration{listenAddress: "127.0.0.1:0", mediaRoot: t.TempDir(), dataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.run(ctx) }()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(10 * time.Second):
			t.Error("app stop")
		}
	}()
	address := waitForListenAddress(t, a.http)
	client := &http.Client{Timeout: 5 * time.Second}
	setup := runtimeAuthenticationRequest(t, client, address, "POST", "/api/v1/auth/setup", `{"username":"noconfig","displayName":"Synthetic","password":"synthetic fixture passphrase"}`, "", "")
	if setup.StatusCode != 201 {
		t.Fatal(setup.StatusCode)
	}
	r := runtimeAuthenticationRequest(t, client, address, "GET", "/api/v1/libraries/lib_1/ai/ocr", "", setup.Cookie, "")
	if r.StatusCode != 404 {
		t.Fatal("unconfigured OCR route registered", r.StatusCode, r.Body)
	}
}
