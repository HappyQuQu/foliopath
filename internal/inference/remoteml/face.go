package remoteml

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const maxFaceResponseBytes = 1 << 20

// FaceClient shares the CLIP transport and admission budget, but never its model
// identity. It does not own quality estimation, generation activation or storage.
type FaceClient struct {
	client   *Client
	identity ServiceIdentity
}

type DetectedFace struct {
	Box       face.Box
	Detection float32
	Embedding []float32
}

func (f *FaceClient) Identity() ServiceIdentity { return f.identity }

// VerifyIdentity uses the same admission budget as CLIP and face predictions.
func (f *FaceClient) VerifyIdentity(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case f.client.slots <- struct{}{}:
		defer func() { <-f.client.slots }()
	default:
		return ErrBusy
	}
	return f.client.verifyIdentityAt(ctx, f.identity, "/api/v1/face/identity")
}

// WaitVerifyIdentity is for the single activation worker. Waiting for the shared
// request budget is not an activation failure. The caller owns cancellation;
// the HTTP identity request retains the client's bounded timeout.
func (f *FaceClient) WaitVerifyIdentity(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case f.client.slots <- struct{}{}:
		defer func() { <-f.client.slots }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return f.client.verifyIdentityAt(ctx, f.identity, "/api/v1/face/identity")
}

// BindFace requires an independently configured face identity. Fetching identity
// from the service and trusting it on first use is not an activation mechanism.
func (c *Client) BindFace(expected ServiceIdentity) (*FaceClient, error) {
	if c == nil || expected.Version != 1 || expected.Protocol != "foliopath-face-v1" ||
		expected.Model != "buffalo_l" || expected.Dimension != 512 ||
		!identityDigest(expected.CacheManifestSHA256) || !identityDigest(expected.RuntimeContractSHA256) {
		return nil, ErrIdentityMismatch
	}
	if c.identity != nil && (c.identity.CacheManifestSHA256 != expected.CacheManifestSHA256 ||
		c.identity.RuntimeContractSHA256 != expected.RuntimeContractSHA256) {
		return nil, ErrIdentityMismatch
	}
	return &FaceClient{client: c, identity: expected}, nil
}

// Detect accepts only caller-decoded, bounded encoded image bytes. The service
// enforces the face-v1 format/pixel contract; original paths never cross HTTP.
func (f *FaceClient) Detect(parent context.Context, encoded []byte) ([]DetectedFace, error) {
	if len(encoded) == 0 || len(encoded) > MaxImageBytes {
		return nil, ErrInvalidInput
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	c := f.client
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	default:
		return nil, ErrBusy
	}
	ctx, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	if err := c.verifyIdentityAt(ctx, f.identity, "/api/v1/face/identity"); err != nil {
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
		strings.TrimSuffix(c.endpoint, "/predict")+"/api/v1/face/predict", &body)
	if err != nil {
		return nil, ErrInvalidInput
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	response, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusServiceUnavailable &&
			response.Header.Get("X-FolioPath-Runtime") == f.identity.RuntimeContractSHA256 &&
			response.Header.Get("X-FolioPath-Cache") == f.identity.CacheManifestSHA256 {
			// Only the fixed, bounded service error contract can request retry.
			data, err := io.ReadAll(io.LimitReader(response.Body, 1025))
			if err == nil && len(data) <= 1024 {
				var body struct {
					Error string `json:"error"`
				}
				decoder := json.NewDecoder(bytes.NewReader(data))
				decoder.DisallowUnknownFields()
				if decoder.Decode(&body) == nil && decoder.Decode(new(any)) == io.EOF && body.Error == "busy" {
					return nil, ErrBusy
				}
			}
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrUnavailable
	}
	if response.Header.Get("X-FolioPath-Runtime") != f.identity.RuntimeContractSHA256 ||
		response.Header.Get("X-FolioPath-Cache") != f.identity.CacheManifestSHA256 {
		return nil, ErrIdentityMismatch
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxFaceResponseBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrUnavailable
	}
	if len(data) > maxFaceResponseBytes {
		return nil, ErrInvalidResponse
	}
	return decodeFaces(data)
}

func decodeFaces(data []byte) ([]DetectedFace, error) {
	var wire struct {
		Faces *[]struct {
			Box       *struct{ X, Y, Width, Height *float32 }
			Detection *float32
			Embedding []*float32
		}
	}
	if json.Unmarshal(data, &wire) != nil || wire.Faces == nil || len(*wire.Faces) > face.MaxCandidatesPerAsset {
		return nil, ErrInvalidResponse
	}
	result := make([]DetectedFace, 0, len(*wire.Faces))
	for _, item := range *wire.Faces {
		if item.Box == nil || item.Detection == nil || len(item.Embedding) != 512 {
			return nil, ErrInvalidResponse
		}
		for _, v := range []*float32{item.Box.X, item.Box.Y, item.Box.Width, item.Box.Height, item.Detection} {
			if v == nil || math.IsNaN(float64(*v)) || math.IsInf(float64(*v), 0) || *v < 0 || *v > 1 {
				return nil, ErrInvalidResponse
			}
		}
		box := face.Box{X: *item.Box.X, Y: *item.Box.Y, Width: *item.Box.Width, Height: *item.Box.Height}
		if box.Width <= 0 || box.Height <= 0 || box.X+box.Width > 1 || box.Y+box.Height > 1 || *item.Detection < 0.7 {
			return nil, ErrInvalidResponse
		}
		values := make([]float32, 512)
		for i, v := range item.Embedding {
			if v == nil {
				return nil, ErrInvalidResponse
			}
			values[i] = *v
		}
		vector, err := semantic.NormalizeEmbedding(values, 512)
		if err != nil {
			return nil, ErrInvalidResponse
		}
		result = append(result, DetectedFace{Box: box, Detection: *item.Detection, Embedding: vector})
	}
	return result, nil
}
