package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type semanticContentRepositoryStub struct{ asset media.ContentAsset }

func (stub semanticContentRepositoryStub) GetContentAsset(context.Context, int64) (media.ContentAsset, error) {
	return stub.asset, nil
}

type semanticContentFileSourceStub struct{ path string }

func (stub semanticContentFileSourceStub) OpenContent(context.Context, string, string) (media.ContentFile, error) {
	return os.Open(stub.path)
}

func TestSemanticContentSourceUsesVerifiedContentBoundaryAndLibraryScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(path, []byte("bounded semantic input"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := media.NewSourceFingerprint(info.Size(), info.ModTime().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	content, err := media.NewContentService(semanticContentRepositoryStub{asset: media.ContentAsset{
		ID: 7, LibraryID: 3, RelativePath: "set/photo.jpg", Format: media.FormatJPEG,
		MIMEType: "image/jpeg", SizeBytes: info.Size(), ModifiedAtNS: info.ModTime().UnixNano(),
		SourceFingerprint: fingerprint,
	}}, semanticContentFileSourceStub{path: path})
	if err != nil {
		t.Fatal(err)
	}
	source := semanticContentSource{content: content}
	asset, err := source.OpenSemanticAsset(context.Background(), 3, 7)
	if err != nil {
		t.Fatal(err)
	}
	if asset.Format != media.FormatJPEG || asset.SourceFingerprint != fingerprint.String() || asset.File == nil {
		t.Fatalf("semantic asset = %#v", asset)
	}
	_ = asset.File.Close()
	if _, err := source.OpenSemanticAsset(context.Background(), 4, 7); !errors.Is(err, semantic.ErrSemanticSourceChanged) {
		t.Fatalf("cross-library error = %v", err)
	}
}
