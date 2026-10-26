package remoteml

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/ocr"
)

func ocrIdentityFixture() ocr.Identity {
	return ocr.Identity{Version: 1, Protocol: "foliopath-ocr-v1", Model: ocr.Model, DetectorSHA256: strings.Repeat("a", 64), RecognizerSHA256: strings.Repeat("b", 64), InputContractSHA256: strings.Repeat("c", 64), OutputContractSHA256: strings.Repeat("d", 64), RuntimeContractSHA256: strings.Repeat("e", 64), CacheManifestSHA256: strings.Repeat("f", 64)}
}
func ocrIdentityJSON() string { data, _ := json.Marshal(ocrIdentityFixture()); return string(data) }
func ocrFingerprint() string  { value, _ := ocrIdentityFixture().Fingerprint(); return value }
func ocrLineJSON() string {
	return `{"text":"中文 Cafe\u0301 <b>😀</b>","box":[0.1,0.1,0.9,0.1,0.9,0.3,0.1,0.3],"detection_score":0.9,"recognition_score":0.95}`
}
func ocrResultJSON(lines string) string {
	return `{"identity":"` + ocrFingerprint() + `","lines":` + lines + `}`
}

func ocrTestServer(t *testing.T, identity, result string, status int) (*OCRClient, *atomic.Int64) {
	t.Helper()
	var predictions atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/ocr/identity":
			if r.Method != http.MethodGet || r.ContentLength > 0 {
				t.Error("identity request must contain no media")
			}
			_, _ = io.WriteString(w, identity)
		case "/api/v1/ocr/predict":
			predictions.Add(1)
			if r.Method != http.MethodPost {
				t.Error(r.Method)
			}
			if err := r.ParseMultipartForm(ocr.MaxImageBytes); err != nil {
				t.Error(err)
				return
			}
			defer r.MultipartForm.RemoveAll()
			if len(r.MultipartForm.Value) != 0 || len(r.MultipartForm.File) != 1 {
				t.Error("unexpected fields")
			}
			source, header, err := r.FormFile("image")
			if err != nil {
				t.Error(err)
				return
			}
			defer source.Close()
			data, _ := io.ReadAll(source)
			if string(data) != "synthetic" || header.Filename != "input" {
				t.Error("must send opaque bytes and fixed filename")
			}
			w.WriteHeader(status)
			_, _ = io.WriteString(w, result)
		default:
			t.Error(r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	client, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	bound, err := client.BindOCR(ocrIdentityFixture())
	if err != nil {
		t.Fatal(err)
	}
	return bound, &predictions
}

func TestOCRIdentityIndependentAndNoTOFU(t *testing.T) {
	var absent *Client
	if _, err := absent.BindOCR(ocrIdentityFixture()); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	semantic := identityFixture()
	client, _ := NewConfigured("http://localhost:3003", semantic)
	defer client.Close()
	bound, err := client.BindOCR(ocrIdentityFixture())
	if err != nil || bound.Identity() != ocrIdentityFixture() {
		t.Fatal(err)
	}
	invalid := ocrIdentityFixture()
	invalid.Model = "JoyTag"
	if _, err := client.BindOCR(invalid); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"wrongModel":       strings.Replace(ocrIdentityJSON(), ocr.Model, "JoyTag", 1),
		"wrongInput":       strings.Replace(ocrIdentityJSON(), strings.Repeat("c", 64), strings.Repeat("0", 64), 1),
		"wrongRuntime":     strings.Replace(ocrIdentityJSON(), strings.Repeat("e", 64), strings.Repeat("0", 64), 1),
		"unknown":          strings.Replace(ocrIdentityJSON(), `{`, `{"extra":1,`, 1),
		"duplicate":        strings.Replace(ocrIdentityJSON(), `{`, `{"version":2,`, 1),
		"escapedDuplicate": strings.Replace(ocrIdentityJSON(), `{`, `{"\u0076ersion":2,`, 1),
		"case":             strings.Replace(ocrIdentityJSON(), `"model"`, `"Model"`, 1),
		"null":             strings.Replace(ocrIdentityJSON(), `"version":1`, `"version":null`, 1),
		"trailing":         ocrIdentityJSON() + `{}`, "empty": `{}`, "array": `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			o, count := ocrTestServer(t, body, ocrResultJSON(`[]`), 200)
			result, err := o.Recognize(context.Background(), []byte("synthetic"))
			if !errors.Is(err, ErrIdentityMismatch) || result.Lines != nil || count.Load() != 0 {
				t.Fatal(result, err, count.Load())
			}
		})
	}
}

func TestOCRActualMultipartNFCPlaintextAndZeroLines(t *testing.T) {
	for _, lines := range []string{`[]`, `[` + ocrLineJSON() + `]`, strings.Replace(`[`+ocrLineJSON()+`]`, `😀`, `\ud83d\ude00`, 1)} {
		o, count := ocrTestServer(t, ocrIdentityJSON(), ocrResultJSON(lines), 200)
		result, err := o.Recognize(context.Background(), []byte("synthetic"))
		if err != nil || result.Lines == nil || count.Load() != 1 {
			t.Fatal(result, err, count.Load())
		}
		if lines != `[]` && (len(result.Lines) != 1 || result.Lines[0].Text != "中文 Café <b>😀</b>") {
			t.Fatal(result)
		}
	}
}

func TestOCRMalformedWireNeverReturnsPartialText(t *testing.T) {
	good := ocrLineJSON()
	cases := map[string]string{
		"missing": `{}`, "nullLines": ocrResultJSON(`null`), "nullEntry": ocrResultJSON(`[null]`),
		"unknown":          strings.Replace(ocrResultJSON(`[]`), `{`, `{"extra":1,`, 1),
		"duplicate":        strings.Replace(ocrResultJSON(`[]`), `{`, `{"lines":[`+good+`],`, 1),
		"escapedDuplicate": strings.Replace(ocrResultJSON(`[]`), `{`, `{"\u006cines":[],`, 1),
		"case":             strings.Replace(ocrResultJSON(`[]`), `"lines"`, `"Lines"`, 1),
		"nullIdentity":     strings.Replace(ocrResultJSON(`[]`), `"`+ocrFingerprint()+`"`, `null`, 1),
		"trailing":         ocrResultJSON(`[]`) + `{}`, "array": `[]`, "invalidUTF8": ocrResultJSON(`[` + strings.Replace(good, "中文", string([]byte{0xff}), 1) + `]`),
		"oversize": ocrResultJSON(`[]`) + strings.Repeat(" ", maxOCRResponseBytes),
		"tooMany":  ocrResultJSON(`[` + strings.TrimSuffix(strings.Repeat(good+",", ocr.MaxLines+1), ",") + `]`),
	}
	for name, line := range map[string]string{
		"unknownLine":     strings.Replace(good, `{`, `{"extra":1,`, 1),
		"duplicateText":   strings.Replace(good, `{`, `{"text":"old",`, 1),
		"nullScore":       strings.Replace(good, `0.95`, `null`, 1),
		"missingScore":    strings.Replace(good, `,"recognition_score":0.95`, ``, 1),
		"lowScore":        strings.Replace(good, `0.95`, `0.79`, 1),
		"badNumber":       strings.Replace(good, `0.95`, `1e999`, 1),
		"booleanScore":    strings.Replace(good, `0.95`, `true`, 1),
		"nullCoordinate":  strings.Replace(good, `[0.1,`, `[null,`, 1),
		"extraCoordinate": strings.Replace(good, `0.1,0.3]`, `0.1,0.3,0.1]`, 1),
		"outside":         strings.Replace(good, `[0.1,`, `[-0.1,`, 1),
		"wrongShape":      strings.Replace(good, `[0.1,0.1,0.9,0.1,0.9,0.3,0.1,0.3]`, `[0,0,0,0,0,0,0,0]`, 1),
		"unpairedHigh":    strings.Replace(good, "中文", `\ud800`, 1),
		"unpairedLow":     strings.Replace(good, "中文", `\udfff`, 1),
		"wrongPair":       strings.Replace(good, "中文", `\ud800\u0041`, 1),
		"control":         strings.Replace(good, "中文", `\u0000`, 1),
		"expandedNFC":     strings.Replace(good, `中文 Cafe\u0301 <b>😀</b>`, strings.Repeat(`\u0344`, ocr.MaxLineRunes), 1),
	} {
		cases[name] = ocrResultJSON(`[` + good + `,` + line + `]`)
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			o, _ := ocrTestServer(t, ocrIdentityJSON(), body, 200)
			result, err := o.Recognize(context.Background(), []byte("synthetic"))
			if !errors.Is(err, ErrInvalidResponse) || result.Lines != nil {
				t.Fatal(result, err)
			}
		})
	}
}

func TestOCRErrorClassificationAndNoRawLeaks(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
		want   error
	}{
		{400, "invalid_input", ErrInvalidInput}, {413, "input_limit", ocr.ErrInputLimit}, {422, "invalid_media", ocr.ErrInvalidMedia},
		{422, "output_invalid", ErrInvalidResponse}, {503, "busy", ErrBusy}, {503, "unavailable", ErrUnavailable}, {503, "model_unavailable", ErrUnavailable},
		{500, "inference_failed", ErrUnavailable}, {500, "invalid_media", ErrInvalidResponse}, {422, "/private/path secret", ErrInvalidResponse},
	} {
		t.Run(tc.code, func(t *testing.T) {
			body := `{"identity":"` + ocrFingerprint() + `","error":"` + tc.code + `"}`
			o, count := ocrTestServer(t, ocrIdentityJSON(), body, tc.status)
			result, err := o.Recognize(context.Background(), []byte("synthetic"))
			if !errors.Is(err, tc.want) || result.Lines != nil || count.Load() != 1 || strings.Contains(err.Error(), "secret") {
				t.Fatal(result, err, count.Load())
			}
		})
	}
	for _, status := range []int{200, 422} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			body := ocrResultJSON(`[]`)
			if status != 200 {
				body = `{"identity":"` + ocrFingerprint() + `","error":"invalid_media"}`
			}
			body = strings.Replace(body, ocrFingerprint(), strings.Repeat("0", 64), 1)
			o, _ := ocrTestServer(t, ocrIdentityJSON(), body, status)
			result, err := o.Recognize(context.Background(), []byte("synthetic"))
			if !errors.Is(err, ErrIdentityMismatch) || result.Lines != nil {
				t.Fatal(result, err)
			}
		})
	}
}

func TestOCRInputBoundsAndCancelledParentSendNoRequests(t *testing.T) {
	o, count := ocrTestServer(t, ocrIdentityJSON(), ocrResultJSON(`[]`), 200)
	for _, data := range [][]byte{nil, {}, make([]byte, ocr.MaxImageBytes+1)} {
		if _, err := o.Recognize(context.Background(), data); !errors.Is(err, ErrInvalidInput) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := o.Recognize(ctx, []byte("synthetic")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := o.VerifyIdentity(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if count.Load() != 0 {
		t.Fatal(count.Load())
	}
}

func TestOCRSharesAdmissionWithTagsAndReleasesAfterCancellation(t *testing.T) {
	entered := make(chan struct{}, 2)
	gate := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ocr/identity" {
			t.Error(r.URL.Path)
		}
		entered <- struct{}{}
		select {
		case <-r.Context().Done():
			return
		case <-gate:
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, ocrIdentityJSON())
	}))
	defer server.Close()
	c, _ := New(server.URL)
	defer c.Close()
	o, _ := c.BindOCR(ocrIdentityFixture())
	tag, _ := c.BindTag(tagIdentityFixture())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 2)
	for range 2 {
		go func() { done <- o.VerifyIdentity(ctx) }()
	}
	for range 2 {
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("requests did not arrive")
		}
	}
	if err := o.VerifyIdentity(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if err := tag.VerifyIdentity(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	cancel()
	for range 2 {
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("slot leaked")
		}
	}
	close(gate)
	if err := o.VerifyIdentity(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestOCRDeadlineIncludesIdentityAndPredictionAndRecovers(t *testing.T) {
	var slow atomic.Bool
	slow.Store(true)
	var predictions atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/ocr/identity" {
			_, _ = io.WriteString(w, ocrIdentityJSON())
			return
		}
		predictions.Add(1)
		if slow.Load() {
			w.WriteHeader(200)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		_, _ = io.WriteString(w, ocrResultJSON(`[]`))
	}))
	defer server.Close()
	c, _ := New(server.URL)
	defer c.Close()
	c.timeout = 50 * time.Millisecond
	o, _ := c.BindOCR(ocrIdentityFixture())
	if _, err := o.Recognize(context.Background(), []byte("synthetic")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	slow.Store(false)
	c.timeout = time.Second
	if _, err := o.Recognize(context.Background(), []byte("synthetic")); err != nil || predictions.Load() != 2 {
		t.Fatal(err, predictions.Load())
	}
}

func TestOCRRedirectsContentTypeAndIdentityResponseBound(t *testing.T) {
	var redirected atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1) }))
	defer target.Close()
	for _, tc := range []struct {
		name              string
		status            int
		contentType, body string
		want              error
	}{
		{"redirect", 302, "application/json", "", ErrUnavailable}, {"missingService", 404, "application/json", "{}", ErrUnavailable},
		{"wrongContentType", 200, "text/plain", ocrIdentityJSON(), ErrInvalidResponse},
		{"oversize", 200, "application/json", ocrIdentityJSON() + strings.Repeat(" ", 4096), ErrInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.Header().Set("Location", target.URL)
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			c, _ := New(server.URL)
			defer c.Close()
			o, _ := c.BindOCR(ocrIdentityFixture())
			if err := o.VerifyIdentity(context.Background()); !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
		})
	}
	if redirected.Load() != 0 {
		t.Fatal("followed redirect")
	}
}

func TestOCRDoesNotInheritProxy(t *testing.T) {
	old := http.DefaultTransport
	configured := http.DefaultTransport.(*http.Transport).Clone()
	var calls atomic.Int64
	configured.Proxy = func(*http.Request) (*url.URL, error) { calls.Add(1); return url.Parse("http://127.0.0.1:1") }
	http.DefaultTransport = configured
	defer func() { http.DefaultTransport = old }()
	o, _ := ocrTestServer(t, ocrIdentityJSON(), ocrResultJSON(`[]`), 200)
	if _, err := o.Recognize(context.Background(), []byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("inherited default proxy")
	}
}

type ocrRoundTrip func(*http.Request) (*http.Response, error)

func (f ocrRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOCRUsesOneDeadlineForIdentityAndPrediction(t *testing.T) {
	o, _ := ocrTestServer(t, ocrIdentityJSON(), ocrResultJSON(`[]`), 200)
	original := o.client.http.Transport
	var deadlines []time.Time
	o.client.http.Transport = ocrRoundTrip(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("missing bounded deadline")
		}
		deadlines = append(deadlines, deadline)
		return original.RoundTrip(r)
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := o.Recognize(ctx, []byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	if len(deadlines) != 2 || !deadlines[0].Equal(deadlines[1]) {
		t.Fatal(deadlines)
	}
	// The HTTP transport is still owned by the shared client; restore it so Close
	// can close the original pool rather than a test wrapper.
	o.client.http.Transport = original
}

func TestOCRResponseTruncationIsServiceFailureNotBadMedia(t *testing.T) {
	o, _ := ocrTestServer(t, ocrIdentityJSON(), ocrResultJSON(`[]`), 200)
	original := o.client.http.Transport
	o.client.http.Transport = ocrRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost {
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: ocrBrokenBody{}}, nil
		}
		return original.RoundTrip(r)
	})
	result, err := o.Recognize(context.Background(), []byte("synthetic"))
	if !errors.Is(err, ErrUnavailable) || errors.Is(err, ocr.ErrInvalidMedia) || result.Lines != nil {
		t.Fatal(result, err)
	}
	o.client.http.Transport = original
}

type ocrBrokenBody struct{}

func (ocrBrokenBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (ocrBrokenBody) Close() error             { return nil }
