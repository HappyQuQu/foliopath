package remoteml

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func responseVector(n int, first string) string {
	items := make([]string, n)
	for i := range items {
		items[i] = "0"
	}
	if n > 0 {
		items[0] = first
	}
	data, _ := json.Marshal(map[string]string{"clip": "[" + strings.Join(items, ",") + "]"})
	return string(data)
}

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestMultipartAndNormalization(t *testing.T) {
	for _, kind := range []string{"textual", "visual"} {
		t.Run(kind, func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/predict" {
					t.Error("wrong endpoint")
				}
				if err := r.ParseMultipartForm(1024); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				want := `{"clip":{"` + kind + `":{"modelName":"` + ModelName + `"}}}`
				if r.FormValue("entries") != want {
					t.Error("wrong entries")
				}
				if kind == "textual" {
					if r.FormValue("text") != "中文, RED!" {
						t.Error("text was transformed")
					}
					if len(r.MultipartForm.File) != 0 {
						t.Error("unexpected file")
					}
				} else {
					f, header, err := r.FormFile("image")
					if err != nil {
						t.Error(err)
						return
					}
					defer f.Close()
					data, _ := io.ReadAll(f)
					if string(data) != "synthetic bytes" || header.Filename != "input" {
						t.Error("wrong image")
					}
				}
				_, _ = io.WriteString(w, responseVector(Dimension, "2"))
			})
			var vector []float32
			var err error
			if kind == "textual" {
				vector, err = c.EncodeText(context.Background(), "中文, RED!")
			} else {
				vector, err = c.EncodeImage(context.Background(), []byte("synthetic bytes"))
			}
			if err != nil || len(vector) != Dimension || vector[0] != 1 {
				t.Fatalf("invalid vector: %v", err)
			}
		})
	}
}

func TestInvalidResponses(t *testing.T) {
	cases := map[string]string{
		"missing": `{}`, "null": `null`, "array": `{"clip":[1]}`, "invalid json": `{`,
		"trailing":  responseVector(Dimension, "1") + `{}`,
		"dimension": responseVector(768, "1"), "zero": responseVector(Dimension, "0"),
		"null element": responseVector(Dimension, "null"), "overflow": responseVector(Dimension, "1e99"),
		"nan": responseVector(Dimension, "NaN"), "too large": strings.Repeat(" ", maxResponseBytes+1),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) })
			v, err := c.EncodeText(context.Background(), "test")
			if !errors.Is(err, ErrInvalidResponse) || v != nil {
				t.Fatalf("got %v, %v", v, err)
			}
		})
	}
}

func TestInputAndOriginValidation(t *testing.T) {
	for _, origin := range []string{"", "ftp://localhost", "http://u:p@localhost", "http://localhost/path",
		"http://localhost?", "http://localhost#", "http://localhost:0", "http://localhost:65536", "http://localhost:"} {
		if _, err := New(origin); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("accepted %q", origin)
		}
	}
	c := testClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid input made request") })
	for _, text := range []string{"", "  ", string([]byte{0xff}), strings.Repeat("字", 513)} {
		if _, err := c.EncodeText(context.Background(), text); !errors.Is(err, ErrInvalidInput) {
			t.Error(err)
		}
	}
	for _, image := range [][]byte{nil, make([]byte, MaxImageBytes+1)} {
		if _, err := c.EncodeImage(context.Background(), image); !errors.Is(err, ErrInvalidInput) {
			t.Error(err)
		}
	}
}

func TestNoProxyRedirectOrErrorLeak(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	var calls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer target.Close()
	for _, status := range []int{302, 307, 400, 500, 503} {
		c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", target.URL)
			w.WriteHeader(status)
			_, _ = io.WriteString(w, "/private/model/path secret")
		})
		if c.http.Transport.(*http.Transport).Proxy != nil {
			t.Fatal("ambient proxy enabled")
		}
		if _, err := c.EncodeText(context.Background(), "test"); err != ErrUnavailable {
			t.Fatalf("unexpected error %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("followed redirect")
	}
}

func TestCancellationTimeoutAndSlotRelease(t *testing.T) {
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
		}
		_, _ = io.WriteString(w, responseVector(Dimension, "1"))
	})
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := make(chan error, 2)
	for range 2 {
		go func() { _, err := c.EncodeText(ctx, "test"); results <- err }()
	}
	for range 2 {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("request did not enter")
		}
	}
	if _, err := c.EncodeText(context.Background(), "test"); err != ErrBusy {
		t.Fatal(err)
	}
	cancel()
	for range 2 {
		if err := <-results; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	if len(c.slots) != 0 {
		t.Fatal("slots leaked")
	}
	if _, err := c.EncodeText(ctx, "test"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	c.timeout = 20 * time.Millisecond
	if _, err := c.EncodeText(context.Background(), "test"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if len(c.slots) != 0 {
		t.Fatal("timeout slot leaked")
	}
}

func TestConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	c, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	server.Close()
	if v, err := c.EncodeText(context.Background(), "test"); v != nil || err != ErrUnavailable {
		t.Fatal(err)
	}
}

func TestResponseBodyTimeoutAndRecovery(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
			case <-release:
			}
			return
		}
		_, _ = io.WriteString(w, responseVector(Dimension, "1"))
	})
	defer close(release)
	c.timeout = 50 * time.Millisecond
	if v, err := c.EncodeText(context.Background(), "test"); v != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("body timeout: %v", err)
	}
	c.timeout = requestTimeout
	if v, err := c.EncodeText(context.Background(), "test"); err != nil || len(v) != Dimension {
		t.Fatalf("client did not recover: %v", err)
	}
	if calls.Load() != 2 || len(c.slots) != 0 {
		t.Fatal("unexpected retry or leaked request slot")
	}
}

func TestTruncatedResponseAndRecovery(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Length", "100")
			_, _ = io.WriteString(w, `{"clip":`)
			return
		}
		_, _ = io.WriteString(w, responseVector(Dimension, "1"))
	})
	if v, err := c.EncodeText(context.Background(), "test"); v != nil || err != ErrUnavailable {
		t.Fatalf("truncated body: %v", err)
	}
	if v, err := c.EncodeText(context.Background(), "test"); err != nil || len(v) != Dimension {
		t.Fatalf("client did not recover: %v", err)
	}
	if calls.Load() != 2 || len(c.slots) != 0 {
		t.Fatal("unexpected retry or leaked request slot")
	}
}
