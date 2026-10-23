package remoteml

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMultilingualModelUsesBoundModelAnd640Dimensions(t *testing.T) {
	expected := identityFixture()
	expected.Model, expected.Dimension = MultilingualModelName, MultilingualDimension
	wrongDimension := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/identity" {
			_ = json.NewEncoder(w).Encode(expected)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		want := `{"clip":{"textual":{"modelName":"` + MultilingualModelName + `"}}}`
		if r.FormValue("entries") != want || r.FormValue("text") != "海边的汽车" {
			t.Error("bound model/query not preserved")
		}
		w.Header().Set("X-FolioPath-Runtime", expected.RuntimeContractSHA256)
		w.Header().Set("X-FolioPath-Cache", expected.CacheManifestSHA256)
		dimension := MultilingualDimension
		if wrongDimension {
			dimension = Dimension
		}
		_, _ = w.Write([]byte(responseVector(dimension, "1")))
	}))
	defer server.Close()
	client, err := NewConfigured(server.URL, expected)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	vector, err := client.EncodeText(context.Background(), "海边的汽车")
	if err != nil || len(vector) != 640 {
		t.Fatalf("dimension=%d err=%v", len(vector), err)
	}
	wrongDimension = true
	if _, err := client.EncodeText(context.Background(), "海边的汽车"); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal(err)
	}
	for _, value := range []ServiceIdentity{
		{Version: 1, Protocol: expected.Protocol, Model: MultilingualModelName, Dimension: 512, CacheManifestSHA256: expected.CacheManifestSHA256, RuntimeContractSHA256: expected.RuntimeContractSHA256},
		{Version: 1, Protocol: expected.Protocol, Model: "arbitrary-model", Dimension: 640, CacheManifestSHA256: expected.CacheManifestSHA256, RuntimeContractSHA256: expected.RuntimeContractSHA256},
	} {
		if _, err := NewConfigured(server.URL, value); !errors.Is(err, ErrIdentityMismatch) {
			t.Fatal("unknown model/dimension accepted", err)
		}
	}
}

func TestForegroundTextWaitsWithinExistingDeadlineAndReleasesSlot(t *testing.T) {
	entered := make(chan struct{}, 1)
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		_, _ = w.Write([]byte(responseVector(Dimension, "1")))
	})
	client.slots <- struct{}{}
	client.slots <- struct{}{}
	client.timeout = 20 * time.Millisecond
	if _, err := client.EncodeText(context.Background(), "汽车"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("foreground wait did not respect total deadline", err)
	}
	if len(client.slots) != 2 || len(entered) != 0 {
		t.Fatal("wait consumed background capacity or sent input")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.EncodeText(ctx, "汽车"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	client.timeout = time.Second
	result := make(chan error, 1)
	go func() { _, err := client.EncodeText(context.Background(), "汽车"); result <- err }()
	<-client.slots
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("text did not acquire a released slot")
	}
	if len(client.slots) != 1 {
		t.Fatal("foreground leaked a slot")
	}
	<-client.slots
}

func TestMultilingualBindingKeepsIndependentFaceAndTagIdentities(t *testing.T) {
	expected := identityFixture()
	expected.Model, expected.Dimension = MultilingualModelName, MultilingualDimension
	client, err := NewConfigured("http://machine-learning:3003", expected)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	face := identityFixture()
	face.Protocol, face.Model = "foliopath-face-v1", "buffalo_l"
	face.RuntimeContractSHA256 = strings.Repeat("c", 64)
	face.CacheManifestSHA256 = strings.Repeat("d", 64)
	bound, err := client.BindFace(face)
	if err != nil || bound.Identity() != face {
		t.Fatal("face identity changed", err)
	}
	tag := TagIdentity{Version: 1, Protocol: "foliopath-tag-v1", Model: "JoyTag", ModelSHA256: strings.Repeat("a", 64), PolicySHA256: strings.Repeat("b", 64), RuntimeContractSHA256: face.RuntimeContractSHA256, CacheManifestSHA256: strings.Repeat("e", 64)}
	boundTag, err := client.BindTag(tag)
	if err != nil || boundTag.Identity() != tag {
		t.Fatal("tag identity changed", err)
	}
	face.Model = "arbitrary"
	if _, err := client.BindFace(face); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal("unrecognized face model accepted")
	}
}
