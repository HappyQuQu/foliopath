package remoteml

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func tagIdentityFixture() TagIdentity {
	return TagIdentity{Version: 1, Protocol: "foliopath-tag-v1", Model: "JoyTag", ModelSHA256: strings.Repeat("a", 64),
		PolicySHA256: strings.Repeat("b", 64), RuntimeContractSHA256: identityFixture().RuntimeContractSHA256,
		CacheManifestSHA256: strings.Repeat("c", 64)}
}

func TestTagIdentityIsSeparateAndComplete(t *testing.T) {
	i := tagIdentityFixture()
	fp, err := i.Fingerprint()
	if err != nil || len(fp) != 64 {
		t.Fatal(fp, err)
	}
	changed := i
	changed.PolicySHA256 = strings.Repeat("d", 64)
	other, _ := changed.Fingerprint()
	if other == fp {
		t.Fatal("policy change must invalidate old results")
	}
	c, err := NewConfigured("http://localhost:3003", identityFixture())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.BindTag(i); err != nil {
		t.Fatal("tag has independent cache", err)
	}
	changed.RuntimeContractSHA256 = strings.Repeat("e", 64)
	if _, err := c.BindTag(changed); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	for _, mutate := range []func(*TagIdentity){
		func(i *TagIdentity) { i.Version = 2 }, func(i *TagIdentity) { i.Protocol = "foliopath-ml-v1" },
		func(i *TagIdentity) { i.Model = "other" }, func(i *TagIdentity) { i.PolicySHA256 = "" },
		func(i *TagIdentity) { i.ModelSHA256 = strings.Repeat("A", 64) },
	} {
		bad := i
		mutate(&bad)
		if _, err := bad.Fingerprint(); !errors.Is(err, ErrIdentityMismatch) {
			t.Fatal(err)
		}
	}
}

func TestTagClientStrictIdentityAndSharedBudget(t *testing.T) {
	expected, actual := tagIdentityFixture(), tagIdentityFixture()
	predictions, status := 0, 200
	body, headers := `{"labels":[{"name":"flower","confidence":0.8}]}`, true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tag/identity" {
			_ = json.NewEncoder(w).Encode(actual)
			return
		}
		predictions++
		if r.URL.Path != "/api/v1/tag/predict" {
			t.Error(r.URL.Path)
		}
		if err := r.ParseMultipartForm(MaxImageBytes); err != nil {
			t.Error(err)
			return
		}
		defer r.MultipartForm.RemoveAll()
		if len(r.MultipartForm.Value) != 0 || len(r.MultipartForm.File) != 1 {
			t.Error("must not send candidate words or paths")
		}
		file, _, err := r.FormFile("image")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "synthetic" {
			t.Error("wrong image")
		}
		if headers {
			w.Header().Set("X-FolioPath-Runtime", expected.RuntimeContractSHA256)
			w.Header().Set("X-FolioPath-Tag-Model", expected.ModelSHA256)
			w.Header().Set("X-FolioPath-Tag-Policy", expected.PolicySHA256)
			w.Header().Set("X-FolioPath-Tag-Cache", expected.CacheManifestSHA256)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()
	c, _ := NewConfigured(server.URL, identityFixture())
	defer c.Close()
	tags, err := c.BindTag(expected)
	if err != nil {
		t.Fatal(err)
	}
	if err := tags.VerifyIdentity(context.Background()); err != nil {
		t.Fatal(err)
	}
	labels, err := tags.Analyze(context.Background(), []byte("synthetic"))
	if err != nil || len(labels) != 1 || labels[0].Name != "flower" {
		t.Fatal(labels, err)
	}
	actual.ModelSHA256 = strings.Repeat("f", 64)
	if _, err := tags.Analyze(context.Background(), []byte("synthetic")); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	if predictions != 1 {
		t.Fatal("sent image to mismatched service")
	}
	actual = expected
	headers = false
	if _, err := tags.Analyze(context.Background(), []byte("synthetic")); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	headers, status, body = true, 503, `{"error":"busy"}`
	if _, err := tags.Analyze(context.Background(), []byte("synthetic")); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	body = `{"error":"busy","extra":true}`
	if _, err := tags.Analyze(context.Background(), []byte("synthetic")); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	status, body = 200, strings.Repeat(" ", maxTagResponseBytes+1)
	if _, err := tags.Analyze(context.Background(), []byte("synthetic")); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal(err)
	}
	for n := 0; n < cap(c.slots); n++ {
		c.slots <- struct{}{}
	}
	if _, err := tags.Analyze(context.Background(), []byte("synthetic")); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if err := tags.VerifyIdentity(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	for n := 0; n < cap(c.slots); n++ {
		<-c.slots
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tags.Analyze(ctx, []byte("synthetic")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := tags.Analyze(context.Background(), nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
}

func TestTagClientIdentityTimeoutAndRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	c, _ := New(server.URL)
	c.timeout = 10 * time.Millisecond
	tags, _ := c.BindTag(tagIdentityFixture())
	if err := tags.VerifyIdentity(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	c.Close()
	server.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/elsewhere", 302) }))
	defer redirect.Close()
	c, _ = New(redirect.URL)
	defer c.Close()
	tags, _ = c.BindTag(tagIdentityFixture())
	if err := tags.VerifyIdentity(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestDecodeAutomaticTagLabels(t *testing.T) {
	for _, raw := range []string{`{}`, `null`, `{"labels":null}`, `{"labels":[null]}`, `{"labels":[{}]}`,
		`{"labels":[],"other":1}`, `{"labels":[]} {}`, `{"labels":[{"name":"flower"}]}`,
		`{"labels":[{"name":"Flower","confidence":0.8}]}`, `{"labels":[{"name":"_flower","confidence":0.8}]}`,
		`{"labels":[{"name":"flower","confidence":1.1}]}`, `{"labels":[{"name":"flower","confidence":null}]}`,
		`{"labels":[{"name":"flower","confidence":0.8},{"name":"flower","confidence":0.9}]}`,
	} {
		if _, err := decodeTagLabels([]byte(raw)); !errors.Is(err, ErrInvalidResponse) {
			t.Fatalf("accepted %s: %v", raw, err)
		}
	}
	if labels, err := decodeTagLabels([]byte(`{"labels":[]}`)); err != nil || labels == nil || len(labels) != 0 {
		t.Fatal(labels, err)
	}
	if labels, err := decodeTagLabels([]byte(`{"labels":[{"name":"cherry_blossoms","confidence":0.7}]}`)); err != nil || len(labels) != 1 {
		t.Fatal(labels, err)
	}
}
