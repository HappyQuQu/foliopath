package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestAutomaticTagInferenceBusyMappingPreservesOtherErrors(t *testing.T) {
	if err := automaticTagInferenceError(remoteml.ErrTagTemporarilyUnavailable); !errors.Is(err, autotag.ErrTemporarilyUnavailable) {
		t.Fatal(err)
	}
	if err := automaticTagInferenceError(fmt.Errorf("wrapped: %w", remoteml.ErrBusy)); !errors.Is(err, autotag.ErrBusy) {
		t.Fatal(err)
	}
	for _, err := range []error{nil, context.Canceled, remoteml.ErrIdentityMismatch, remoteml.ErrUnavailable, remoteml.ErrInvalidResponse} {
		if got := automaticTagInferenceError(err); got != err {
			t.Fatalf("%v became %v", err, got)
		}
	}
}

type automaticPreprocessorStub struct{ calls int }

func (p *automaticPreprocessorStub) PrepareRemoteSemanticImage(context.Context, io.ReadSeeker, media.Format) ([]byte, error) {
	p.calls++
	return []byte("synthetic encoded image"), nil
}
func TestAutomaticTagRuntimeVerifiesLibraryAndReopensSourceAfterInference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.jpg")
	if err := os.WriteFile(path, []byte("synthetic original"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, _ := media.NewSourceFingerprint(info.Size(), info.ModTime().UnixNano())
	content, err := media.NewContentService(semanticContentRepositoryStub{asset: media.ContentAsset{ID: 7, LibraryID: 3, RelativePath: "fixture.jpg", Format: media.FormatJPEG, MIMEType: "image/jpeg", SizeBytes: info.Size(), ModifiedAtNS: info.ModTime().UnixNano(), SourceFingerprint: fingerprint}}, semanticContentFileSourceStub{path: path})
	if err != nil {
		t.Fatal(err)
	}
	deployment := deploymentConfigFixture()
	identity := remoteml.TagIdentity{Version: 1, Protocol: "foliopath-tag-v1", Model: "JoyTag", ModelSHA256: strings.Repeat("a", 64), PolicySHA256: strings.Repeat("c", 64), RuntimeContractSHA256: deployment.Identity.RuntimeContractSHA256, CacheManifestSHA256: strings.Repeat("d", 64)}
	replace := false
	predictions := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tag/identity" {
			_ = json.NewEncoder(w).Encode(identity)
			return
		}
		predictions++
		if replace {
			if err := os.WriteFile(path, []byte("synthetic changed source is longer"), 0600); err != nil {
				t.Error(err)
			}
		}
		w.Header().Set("X-FolioPath-Runtime", identity.RuntimeContractSHA256)
		w.Header().Set("X-FolioPath-Tag-Model", identity.ModelSHA256)
		w.Header().Set("X-FolioPath-Tag-Policy", identity.PolicySHA256)
		w.Header().Set("X-FolioPath-Tag-Cache", identity.CacheManifestSHA256)
		_, _ = io.WriteString(w, `{"labels":[{"name":"tree","confidence":0.9}]}`)
	}))
	defer server.Close()
	client, err := remoteml.NewConfigured(server.URL, deployment.Identity)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	tag, err := client.BindTag(identity)
	if err != nil {
		t.Fatal(err)
	}
	prep := &automaticPreprocessorStub{}
	runtime := automaticTagRuntime{content: content, preprocessor: prep, client: tag}
	asset := autotag.Candidate{ID: 7, LibraryID: 3, SourceFingerprint: fingerprint.String()}
	wrong := asset
	wrong.LibraryID = 4
	if _, err = runtime.Analyze(context.Background(), wrong); !errors.Is(err, autotag.ErrSourceChanged) || predictions != 0 || prep.calls != 0 {
		t.Fatalf("cross-library leaked to model: %v", err)
	}
	labels, err := runtime.Analyze(context.Background(), asset)
	if err != nil || len(labels) != 1 || labels[0].Name != "tree" || predictions != 1 {
		t.Fatalf("valid labels %+v %v", labels, err)
	}
	replace = true
	if _, err = runtime.Analyze(context.Background(), asset); !errors.Is(err, autotag.ErrSourceChanged) {
		t.Fatalf("replacement not checked after inference: %v", err)
	}
	if _, err = runtime.Analyze(context.Background(), asset); !errors.Is(err, autotag.ErrSourceChanged) || predictions != 2 {
		t.Fatalf("known stale input reached model: %v count%d", err, predictions)
	}
}
