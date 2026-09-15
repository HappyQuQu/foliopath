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
)

func faceIdentityFixture() ServiceIdentity {
	i := identityFixture()
	i.Protocol, i.Model = "foliopath-face-v1", "buffalo_l"
	return i
}

func TestFaceClientIdentityAndSharedAdmission(t *testing.T) {
	expected := faceIdentityFixture()
	predictions := 0
	actual := expected
	headers := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/face/identity" {
			_ = json.NewEncoder(w).Encode(actual)
			return
		}
		predictions++
		if r.URL.Path != "/api/v1/face/predict" {
			t.Error(r.URL.Path)
		}
		if err := r.ParseMultipartForm(MaxImageBytes); err != nil {
			t.Error(err)
			return
		}
		defer r.MultipartForm.RemoveAll()
		if len(r.MultipartForm.Value) != 0 || len(r.MultipartForm.File) != 1 {
			t.Error("unexpected fields")
		}
		file, _, err := r.FormFile("image")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		content, _ := io.ReadAll(file)
		if string(content) != "synthetic" {
			t.Error("wrong payload")
		}
		if headers {
			w.Header().Set("X-FolioPath-Runtime", expected.RuntimeContractSHA256)
			w.Header().Set("X-FolioPath-Cache", expected.CacheManifestSHA256)
		}
		_, _ = io.WriteString(w, `{"faces":[]}`)
	}))
	defer server.Close()
	c, err := NewConfigured(server.URL, identityFixture())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	f, err := c.BindFace(expected)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.BindFace(identityFixture()); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	wrong := expected
	wrong.CacheManifestSHA256 = strings.Repeat("c", 64)
	if _, err := c.BindFace(wrong); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	if result, err := f.Detect(context.Background(), []byte("synthetic")); err != nil || len(result) != 0 {
		t.Fatal(result, err)
	}
	actual.Protocol = "foliopath-ml-v1"
	if _, err := f.Detect(context.Background(), []byte("synthetic")); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	if predictions != 1 {
		t.Fatal("sent input to wrong identity")
	}
	actual = expected
	headers = false
	if _, err := f.Detect(context.Background(), []byte("synthetic")); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatal(err)
	}
	for i := 0; i < cap(c.slots); i++ {
		c.slots <- struct{}{}
	}
	if _, err := f.Detect(context.Background(), []byte("synthetic")); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	for i := 0; i < cap(c.slots); i++ {
		<-c.slots
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.Detect(ctx, []byte("synthetic")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestDecodeFacesStrictContract(t *testing.T) {
	vector := make([]any, 512)
	for i := range vector {
		vector[i] = 0
	}
	vector[0] = 2
	valid := map[string]any{"box": map[string]any{"x": 0, "y": 0, "width": 0.5, "height": 0.5}, "detection": 0.9, "embedding": vector}
	data, _ := json.Marshal(map[string]any{"faces": []any{valid}})
	faces, err := decodeFaces(data)
	if err != nil || len(faces) != 1 || faces[0].Embedding[0] != 1 {
		t.Fatal(faces, err)
	}
	for _, raw := range []string{`{}`, `{"faces":null}`, `{"faces":[null]}`, `{"faces":[{}]}`, strings.Replace(string(data), `"x":0`, `"x":null`, 1), strings.Replace(string(data), `"width":0.5`, `"width":2`, 1), strings.Replace(string(data), `"detection":0.9`, `"detection":0.1`, 1)} {
		if _, err := decodeFaces([]byte(raw)); !errors.Is(err, ErrInvalidResponse) {
			t.Fatalf("accepted malformed result: %v", err)
		}
	}
	vector[0] = nil
	data, _ = json.Marshal(map[string]any{"faces": []any{valid}})
	if _, err := decodeFaces(data); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal(err)
	}
	vector[0] = 0
	data, _ = json.Marshal(map[string]any{"faces": []any{valid}})
	if _, err := decodeFaces(data); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal(err)
	}
	tooMany := make([]any, 65)
	for i := range tooMany {
		tooMany[i] = valid
	}
	data, _ = json.Marshal(map[string]any{"faces": tooMany})
	if _, err := decodeFaces(data); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal(err)
	}
}
