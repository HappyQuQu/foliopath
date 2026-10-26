package remoteml

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTagRecoveryClassifiesOnlyServiceOutages(t *testing.T) {
	identity := tagIdentityFixture()
	status, body, headers := http.StatusServiceUnavailable, `{"error":"unavailable"}`, true
	identityStatus := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tag/identity" {
			w.WriteHeader(identityStatus)
			_ = json.NewEncoder(w).Encode(identity)
			return
		}
		if headers {
			w.Header().Set("X-FolioPath-Runtime", identity.RuntimeContractSHA256)
			w.Header().Set("X-FolioPath-Tag-Model", identity.ModelSHA256)
			w.Header().Set("X-FolioPath-Tag-Policy", identity.PolicySHA256)
			w.Header().Set("X-FolioPath-Tag-Cache", identity.CacheManifestSHA256)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	c, _ := NewConfigured(server.URL, identityFixture())
	defer c.Close()
	tags, _ := c.BindTag(identity)
	check := func(want error) {
		t.Helper()
		_, err := tags.Analyze(context.Background(), []byte("synthetic"))
		if !errors.Is(err, want) {
			t.Fatalf("%v want %v", err, want)
		}
	}
	check(ErrTagTemporarilyUnavailable)
	body = `{"error":"unavailable","extra":true}`
	check(ErrUnavailable)
	_, err := tags.Analyze(context.Background(), []byte("synthetic"))
	if errors.Is(err, ErrTagTemporarilyUnavailable) {
		t.Fatal("malformed error retried")
	}
	headers = false
	check(ErrIdentityMismatch)
	headers = true
	status = 200
	body = `{"labels":null}`
	check(ErrInvalidResponse)
	identityStatus = 503
	check(ErrTagTemporarilyUnavailable)
	identityStatus = 302
	if err := tags.VerifyIdentity(context.Background()); !errors.Is(err, ErrUnavailable) || errors.Is(err, ErrTagTemporarilyUnavailable) {
		t.Fatalf("redirect retried: %v", err)
	}
	server.Close()
	if err := tags.VerifyIdentity(context.Background()); !errors.Is(err, ErrTagTemporarilyUnavailable) {
		t.Fatal(err)
	}
}
