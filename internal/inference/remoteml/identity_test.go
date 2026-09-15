package remoteml

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func identityFixture() ServiceIdentity {
	return ServiceIdentity{Version: 1, Protocol: "foliopath-ml-v1", Model: ModelName, Dimension: Dimension,
		CacheManifestSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: strings.Repeat("b", 64)}
}

func TestConfiguredClientChecksIdentityBeforeSendingInput(t *testing.T) {
	expected := identityFixture()
	var requests, predictions atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path == "/predict" {
			predictions.Add(1)
		}
		actual := expected
		actual.Dimension = 768
		_ = json.NewEncoder(w).Encode(actual)
	}))
	defer server.Close()
	client, err := NewConfigured(server.URL, expected)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if requests.Load() != 0 {
		t.Fatal("construction contacted service")
	}
	if _, err := client.EncodeText(context.Background(), "private test input"); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	if requests.Load() != 1 || predictions.Load() != 0 {
		t.Fatal("input sent before identity verification")
	}
}

func TestVerifiedIdentityRejectsChangedHeaders(t *testing.T) {
	expected := identityFixture()
	var changed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/identity" {
			_ = json.NewEncoder(w).Encode(expected)
			return
		}
		w.Header().Set("X-FolioPath-Runtime", expected.RuntimeContractSHA256)
		if !changed.Load() {
			w.Header().Set("X-FolioPath-Cache", expected.CacheManifestSHA256)
		}
		vector := make([]float32, Dimension)
		vector[0] = 1
		data, _ := json.Marshal(vector)
		_ = json.NewEncoder(w).Encode(map[string]string{"clip": string(data)})
	}))
	defer server.Close()
	client, err := NewVerified(context.Background(), server.URL, expected)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.EncodeText(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	changed.Store(true)
	if result, err := client.EncodeText(context.Background(), "hello"); !errors.Is(err, ErrIdentityMismatch) || result != nil {
		t.Fatalf("result=%v err=%v", result, err)
	}
}

func TestVerifiedIdentityRejectsMismatchAndInvalidExpected(t *testing.T) {
	expected := identityFixture()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actual := expected
		actual.Dimension = 768
		_ = json.NewEncoder(w).Encode(actual)
	}))
	defer server.Close()
	if _, err := NewVerified(context.Background(), server.URL, expected); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	expected.Protocol = "unknown"
	if _, err := NewVerified(context.Background(), server.URL, expected); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
}
