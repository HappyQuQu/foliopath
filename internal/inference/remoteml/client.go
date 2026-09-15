// Package remoteml implements the isolated ADR-0019 HTTP transport. A successful
// response is not model identity attestation or permission to activate a generation.
package remoteml

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const (
	ModelName        = "ViT-B-32__openai"
	Dimension        = 512
	MaxImageBytes    = semantic.RemoteImageMaxBytes
	maxResponseBytes = 64 << 10
	requestTimeout   = 30 * time.Second
)

var (
	ErrInvalidInput    = errors.New("invalid ML transport input")
	ErrBusy            = errors.New("ML transport busy")
	ErrUnavailable     = errors.New("ML transport unavailable")
	ErrInvalidResponse = errors.New("invalid ML transport response")
)

type Client struct {
	endpoint string
	http     *http.Client
	slots    chan struct{}
	timeout  time.Duration
	identity *ServiceIdentity
}

// New accepts only a deployment-owned origin. The operator must enforce the
// same-host private network boundary; URL validation does not enforce that boundary.
func New(origin string) (*Client, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") ||
		u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery ||
		u.Fragment != "" || strings.Contains(origin, "#") || u.Opaque != "" {
		return nil, ErrInvalidInput
	}
	if strings.HasSuffix(u.Host, ":") {
		return nil, ErrInvalidInput
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return nil, ErrInvalidInput
		}
	}
	u.Path = "/predict"
	u.RawPath = ""
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil // Never send local inference through ambient proxies.
	transport.MaxConnsPerHost = 2
	transport.MaxIdleConnsPerHost = 2
	transport.MaxResponseHeaderBytes = 16 << 10
	transport.DisableCompression = true
	return &Client{endpoint: u.String(), slots: make(chan struct{}, 2), timeout: requestTimeout,
		http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) EncodeText(ctx context.Context, text string) ([]float32, error) {
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > semantic.MaxQueryRunes || strings.TrimSpace(text) == "" {
		return nil, ErrInvalidInput
	}
	return c.predict(ctx, "textual", text, nil)
}

// EncodeImage accepts bounded encoded media, not a SigLIP-normalized tensor.
// The caller owns safe decoding/pixel limits and obtains media via internal/files.
func (c *Client) EncodeImage(ctx context.Context, encoded []byte) ([]float32, error) {
	if len(encoded) == 0 || len(encoded) > MaxImageBytes {
		return nil, ErrInvalidInput
	}
	return c.predict(ctx, "visual", "", encoded)
}

func (c *Client) predict(parent context.Context, kind, text string, encoded []byte) ([]float32, error) {
	if err := parent.Err(); err != nil {
		return nil, err
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	default:
		return nil, ErrBusy
	}
	ctx, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	if c.identity != nil {
		if err := c.verifyIdentity(ctx, *c.identity); err != nil {
			return nil, err
		}
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	// Fixed keys and model, never interpolated user input.
	entries := `{"clip":{"` + kind + `":{"modelName":"` + ModelName + `"}}}`
	if err := w.WriteField("entries", entries); err != nil {
		return nil, ErrInvalidInput
	}
	if kind == "textual" {
		if err := w.WriteField("text", text); err != nil {
			return nil, ErrInvalidInput
		}
	} else {
		part, err := w.CreateFormFile("image", "input")
		if err != nil {
			return nil, ErrInvalidInput
		}
		if _, err := part.Write(encoded); err != nil {
			return nil, ErrInvalidInput
		}
	}
	if err := w.Close(); err != nil {
		return nil, ErrInvalidInput
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, &body)
	if err != nil {
		return nil, ErrInvalidInput
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrUnavailable
	}
	if c.identity != nil && (resp.Header.Get("X-FolioPath-Runtime") != c.identity.RuntimeContractSHA256 ||
		resp.Header.Get("X-FolioPath-Cache") != c.identity.CacheManifestSHA256) {
		return nil, ErrIdentityMismatch
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrUnavailable
	}
	if len(data) > maxResponseBytes {
		return nil, ErrInvalidResponse
	}
	var response struct {
		Clip string `json:"clip"`
	}
	if json.Unmarshal(data, &response) != nil {
		return nil, ErrInvalidResponse
	}
	// Pointers reject JSON null elements, which float32 unmarshalling treats as zero.
	var values []*float32
	if json.Unmarshal([]byte(response.Clip), &values) != nil || len(values) != Dimension {
		return nil, ErrInvalidResponse
	}
	vector := make([]float32, Dimension)
	for i, value := range values {
		if value == nil {
			return nil, ErrInvalidResponse
		}
		vector[i] = *value
	}
	normalized, err := semantic.NormalizeEmbedding(vector, Dimension)
	if err != nil {
		return nil, ErrInvalidResponse
	}
	return normalized, nil
}
