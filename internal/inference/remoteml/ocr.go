package remoteml

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/HappyQuQu/foliopath/internal/ocr"
)

const maxOCRResponseBytes = 128 << 10

// OCRClient is an independently deployment-bound transport consumer. It shares
// the original HTTP budget; app owns explicit opt-in OCR composition (ADR-0036).
type OCRClient struct {
	client      *Client
	identity    ocr.Identity
	fingerprint string
}

func (c *Client) BindOCR(expected ocr.Identity) (*OCRClient, error) {
	fingerprint, err := expected.Fingerprint()
	if c == nil || err != nil {
		return nil, ErrIdentityMismatch
	}
	// OCR's reviewed runtime/cache pins are independent of CLIP, face and tags.
	return &OCRClient{client: c, identity: expected, fingerprint: fingerprint}, nil
}

func (o *OCRClient) Identity() ocr.Identity { return o.identity }

func (o *OCRClient) admit(parent context.Context) (context.Context, func(), error) {
	if err := parent.Err(); err != nil {
		return nil, nil, err
	}
	select {
	case o.client.slots <- struct{}{}:
	default:
		return nil, nil, ErrBusy
	}
	ctx, cancel := context.WithTimeout(parent, o.client.timeout)
	return ctx, func() { cancel(); <-o.client.slots }, nil
}

func (o *OCRClient) VerifyIdentity(parent context.Context) error {
	ctx, release, err := o.admit(parent)
	if err != nil {
		return err
	}
	defer release()
	return o.verify(ctx)
}

func (o *OCRClient) endpoint(path string) string {
	return strings.TrimSuffix(o.client.endpoint, "/predict") + "/api/v1/ocr/" + path
}

func (o *OCRClient) verify(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.endpoint("identity"), nil)
	if err != nil {
		return ErrInvalidInput
	}
	response, err := o.client.http.Do(req)
	if err != nil {
		return ocrTransportError(ctx)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ocrContextError(ctx, ErrUnavailable)
	}
	data, err := readOCRJSON(ctx, response, 4096)
	if err != nil {
		return err
	}
	object, ok := decodeOCRObject(data, "version", "protocol", "model", "detector_sha256", "recognizer_sha256",
		"input_contract_sha256", "output_contract_sha256", "runtime_contract_sha256", "cache_manifest_sha256")
	if !ok || len(object) != 9 {
		return ErrIdentityMismatch
	}
	var actual ocr.Identity
	if json.Unmarshal(data, &actual) != nil || actual != o.identity {
		return ErrIdentityMismatch
	}
	if _, err := actual.Fingerprint(); err != nil {
		return ErrIdentityMismatch
	}
	return nil
}

// Recognize accepts only bounded caller-provided encoded bytes. Safe source open,
// pixel/profile checks, source fingerprint and persistence are caller capabilities
// not implemented or bypassed here. There is deliberately no retry or fallback.
func (o *OCRClient) Recognize(parent context.Context, encoded []byte) (ocr.Result, error) {
	if len(encoded) == 0 || len(encoded) > ocr.MaxImageBytes {
		return ocr.Result{}, ErrInvalidInput
	}
	ctx, release, err := o.admit(parent)
	if err != nil {
		return ocr.Result{}, err
	}
	defer release()
	if err := o.verify(ctx); err != nil {
		return ocr.Result{}, err
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("image", "input")
	if err != nil {
		return ocr.Result{}, ErrInvalidInput
	}
	if _, err := part.Write(encoded); err != nil {
		return ocr.Result{}, ErrInvalidInput
	}
	if err := form.Close(); err != nil {
		return ocr.Result{}, ErrInvalidInput
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.endpoint("predict"), &body)
	if err != nil {
		return ocr.Result{}, ErrInvalidInput
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	response, err := o.client.http.Do(req)
	if err != nil {
		return ocr.Result{}, ocrTransportError(ctx)
	}
	defer response.Body.Close()
	limit := maxOCRResponseBytes
	if response.StatusCode != http.StatusOK {
		limit = 1024
	}
	data, err := readOCRJSON(ctx, response, int64(limit))
	if err != nil {
		return ocr.Result{}, err
	}
	if response.StatusCode != http.StatusOK {
		return ocr.Result{}, o.decodeFailure(data, response.StatusCode)
	}
	result, err := o.decodeResult(data)
	if cancelled := ctx.Err(); cancelled != nil {
		return ocr.Result{}, cancelled
	}
	return result, err
}

func ocrContextError(ctx context.Context, fallback error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fallback
}

func ocrTransportError(ctx context.Context) error { return ocrContextError(ctx, ErrUnavailable) }

func readOCRJSON(ctx context.Context, response *http.Response, limit int64) ([]byte, error) {
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, ocrContextError(ctx, ErrInvalidResponse)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, ocrTransportError(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(data)) > limit || !utf8.Valid(data) {
		return nil, ErrInvalidResponse
	}
	return data, nil
}

// decodeOCRObject preserves duplicate-key visibility (including escaped keys),
// exact case and required fields, which struct Unmarshal alone cannot enforce.
// Nested objects are decoded separately by the same OCR wire-contract owner.
func decodeOCRObject(data []byte, keys ...string) (map[string]json.RawMessage, bool) {
	if !utf8.Valid(data) || !validOCRUnicode(data) {
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return nil, false
	}
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
	}
	object := make(map[string]json.RawMessage, len(keys))
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || !allowed[key] {
			return nil, false
		}
		if _, duplicate := object[key]; duplicate {
			return nil, false
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, false
		}
		object[key] = value
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') || decoder.Decode(new(any)) != io.EOF || len(object) != len(keys) {
		return nil, false
	}
	return object, true
}

// encoding/json replaces unpaired JSON surrogate escapes with U+FFFD. Reject
// those escapes before decoding so malformed wire data cannot become plausible
// recognized text. Valid surrogate pairs (including emoji) remain supported.
func validOCRUnicode(data []byte) bool {
	inString := false
	for n := 0; n < len(data); n++ {
		if data[n] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[n] != '\\' {
			continue
		}
		n++
		if n >= len(data) {
			return false
		}
		if data[n] != 'u' {
			continue
		}
		if n+4 >= len(data) {
			return false
		}
		code, err := strconv.ParseUint(string(data[n+1:n+5]), 16, 16)
		if err != nil {
			return false
		}
		n += 4
		if code >= 0xdc00 && code <= 0xdfff {
			return false
		}
		if code < 0xd800 || code > 0xdbff {
			continue
		}
		if n+6 >= len(data) || data[n+1] != '\\' || data[n+2] != 'u' {
			return false
		}
		low, err := strconv.ParseUint(string(data[n+3:n+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return false
		}
		n += 6
	}
	return true
}

func (o *OCRClient) matches(object map[string]json.RawMessage) bool {
	var actual string
	return json.Unmarshal(object["identity"], &actual) == nil && actual == o.fingerprint
}

func (o *OCRClient) decodeResult(data []byte) (ocr.Result, error) {
	object, ok := decodeOCRObject(data, "identity", "lines")
	if !ok {
		return ocr.Result{}, ErrInvalidResponse
	}
	if !o.matches(object) {
		return ocr.Result{}, ErrIdentityMismatch
	}
	var items []json.RawMessage
	if json.Unmarshal(object["lines"], &items) != nil || len(items) > ocr.MaxLines || items == nil {
		return ocr.Result{}, ErrInvalidResponse
	}
	lines := make([]ocr.Line, 0, len(items))
	for _, item := range items {
		fields, ok := decodeOCRObject(item, "text", "box", "detection_score", "recognition_score")
		if !ok {
			return ocr.Result{}, ErrInvalidResponse
		}
		var line ocr.Line
		var box []*float64
		if json.Unmarshal(fields["text"], &line.Text) != nil || json.Unmarshal(fields["box"], &box) != nil || len(box) != 8 ||
			json.Unmarshal(fields["detection_score"], &line.DetectionScore) != nil || json.Unmarshal(fields["recognition_score"], &line.RecognitionScore) != nil {
			return ocr.Result{}, ErrInvalidResponse
		}
		for i, coordinate := range box {
			if coordinate == nil {
				return ocr.Result{}, ErrInvalidResponse
			}
			line.Box[i] = *coordinate
		}
		lines = append(lines, line)
	}
	result, err := ocr.Normalize(lines)
	if err != nil {
		return ocr.Result{}, errors.Join(ErrInvalidResponse, err)
	}
	return result, nil
}

func (o *OCRClient) decodeFailure(data []byte, status int) error {
	object, ok := decodeOCRObject(data, "identity", "error")
	if !ok {
		return ErrInvalidResponse
	}
	if !o.matches(object) {
		return ErrIdentityMismatch
	}
	var code string
	if json.Unmarshal(object["error"], &code) != nil {
		return ErrInvalidResponse
	}
	switch {
	case status == 400 && code == "invalid_input":
		return ErrInvalidInput
	case status == 413 && code == "input_limit":
		return ocr.ErrInputLimit
	case status == 422 && code == "invalid_media":
		return ocr.ErrInvalidMedia
	case status == 422 && code == "output_invalid":
		return ErrInvalidResponse
	case status == 503 && code == "busy":
		return ErrBusy
	case status == 503 && (code == "unavailable" || code == "model_unavailable"):
		return ErrUnavailable
	case status == 500 && code == "inference_failed":
		return ErrUnavailable
	default:
		return ErrInvalidResponse
	}
}
