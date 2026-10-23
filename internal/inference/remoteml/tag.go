package remoteml

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strings"
)

const maxTagResponseBytes = 8192

// TagIdentity is independently configured, never trusted on first use. Tag cache
// and policy identities are distinct from the CLIP/face vector-space contract.
type TagIdentity struct {
	Version               int    `json:"version"`
	Protocol              string `json:"protocol"`
	Model                 string `json:"model"`
	ModelSHA256           string `json:"model_sha256"`
	PolicySHA256          string `json:"policy_sha256"`
	RuntimeContractSHA256 string `json:"runtime_contract_sha256"`
	CacheManifestSHA256   string `json:"cache_manifest_sha256"`
}

func (i TagIdentity) valid() bool {
	return i.Version == 1 && i.Protocol == "foliopath-tag-v1" && i.Model == "JoyTag" &&
		identityDigest(i.ModelSHA256) && identityDigest(i.PolicySHA256) &&
		identityDigest(i.RuntimeContractSHA256) && identityDigest(i.CacheManifestSHA256)
}

func (i TagIdentity) Fingerprint() (string, error) {
	if !i.valid() {
		return "", ErrIdentityMismatch
	}
	data, _ := json.Marshal(i)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

// TagLabel confidence is an uncalibrated model score, not a probability.
type TagLabel struct {
	Name       string
	Confidence float64
}

// TagClient shares transport and admission with search and faces. The caller owns
// safe image decoding and manual-tag policy; no paths or user vocabulary cross HTTP.
type TagClient struct {
	client   *Client
	identity TagIdentity
}

func (c *Client) BindTag(expected TagIdentity) (*TagClient, error) {
	if c == nil || !expected.valid() || (c.identity != nil && c.identity.Model != MultilingualModelName &&
		c.identity.RuntimeContractSHA256 != expected.RuntimeContractSHA256) {
		return nil, ErrIdentityMismatch
	}
	return &TagClient{client: c, identity: expected}, nil
}

func (t *TagClient) Identity() TagIdentity { return t.identity }

func (t *TagClient) VerifyIdentity(parent context.Context) error {
	if err := parent.Err(); err != nil {
		return err
	}
	select {
	case t.client.slots <- struct{}{}:
		defer func() { <-t.client.slots }()
	default:
		return ErrBusy
	}
	ctx, cancel := context.WithTimeout(parent, t.client.timeout)
	defer cancel()
	return t.verify(ctx)
}

func (t *TagClient) verify(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		strings.TrimSuffix(t.client.endpoint, "/predict")+"/api/v1/tag/identity", nil)
	if err != nil {
		return ErrInvalidInput
	}
	response, err := t.client.http.Do(req)
	if err != nil {
		return tagTransportError(ctx)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil {
		return tagTransportError(ctx)
	}
	var actual TagIdentity
	if len(data) > 4096 || !decodeTagJSON(data, &actual) || actual != t.identity {
		return ErrIdentityMismatch
	}
	return nil
}

func (t *TagClient) validHeaders(h http.Header) bool {
	return h.Get("X-FolioPath-Runtime") == t.identity.RuntimeContractSHA256 &&
		h.Get("X-FolioPath-Tag-Model") == t.identity.ModelSHA256 &&
		h.Get("X-FolioPath-Tag-Policy") == t.identity.PolicySHA256 &&
		h.Get("X-FolioPath-Tag-Cache") == t.identity.CacheManifestSHA256
}

func (t *TagClient) Analyze(parent context.Context, encoded []byte) ([]TagLabel, error) {
	if len(encoded) == 0 || len(encoded) > MaxImageBytes {
		return nil, ErrInvalidInput
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	select {
	case t.client.slots <- struct{}{}:
		defer func() { <-t.client.slots }()
	default:
		return nil, ErrBusy
	}
	ctx, cancel := context.WithTimeout(parent, t.client.timeout)
	defer cancel()
	if err := t.verify(ctx); err != nil {
		return nil, err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("image", "input")
	if err != nil {
		return nil, ErrInvalidInput
	}
	if _, err := part.Write(encoded); err != nil {
		return nil, ErrInvalidInput
	}
	if err := w.Close(); err != nil {
		return nil, ErrInvalidInput
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(t.client.endpoint, "/predict")+"/api/v1/tag/predict", &body)
	if err != nil {
		return nil, ErrInvalidInput
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	response, err := t.client.http.Do(req)
	if err != nil {
		return nil, tagTransportError(ctx)
	}
	defer response.Body.Close()
	if !t.validHeaders(response.Header) {
		return nil, ErrIdentityMismatch
	}
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusServiceUnavailable {
			data, err := io.ReadAll(io.LimitReader(response.Body, 1025))
			var failure struct {
				Error string `json:"error"`
			}
			if err == nil && len(data) <= 1024 && decodeTagJSON(data, &failure) && failure.Error == "busy" {
				return nil, ErrBusy
			}
		}
		return nil, tagTransportError(ctx)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxTagResponseBytes+1))
	if err != nil {
		return nil, tagTransportError(ctx)
	}
	if len(data) > maxTagResponseBytes {
		return nil, ErrInvalidResponse
	}
	return decodeTagLabels(data)
}

func tagTransportError(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return ErrUnavailable
}

func decodeTagJSON(data []byte, target any) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(target) == nil && d.Decode(new(any)) == io.EOF
}

func decodeTagLabels(data []byte) ([]TagLabel, error) {
	var wire struct {
		Labels *[]struct {
			Name       string   `json:"name"`
			Confidence *float64 `json:"confidence"`
		} `json:"labels"`
	}
	if !decodeTagJSON(data, &wire) || wire.Labels == nil || len(*wire.Labels) > 20 {
		return nil, ErrInvalidResponse
	}
	result := make([]TagLabel, 0, len(*wire.Labels))
	seen := make(map[string]bool, len(*wire.Labels))
	for _, item := range *wire.Labels {
		if item.Name == "" || len(item.Name) > 32 || seen[item.Name] || item.Confidence == nil ||
			math.IsNaN(*item.Confidence) || math.IsInf(*item.Confidence, 0) || *item.Confidence < 0 || *item.Confidence > 1 {
			return nil, ErrInvalidResponse
		}
		// tag-v1 canonical labels are lowercase ASCII words separated by underscores.
		for _, r := range item.Name {
			if (r < 'a' || r > 'z') && r != '_' {
				return nil, ErrInvalidResponse
			}
		}
		if strings.HasPrefix(item.Name, "_") || strings.HasSuffix(item.Name, "_") || strings.Contains(item.Name, "__") {
			return nil, ErrInvalidResponse
		}
		seen[item.Name] = true
		result = append(result, TagLabel{Name: item.Name, Confidence: *item.Confidence})
	}
	return result, nil
}
