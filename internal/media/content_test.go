package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type contentRepositoryStub struct {
	asset ContentAsset
	err   error
}

func (stub contentRepositoryStub) GetContentAsset(context.Context, int64) (ContentAsset, error) {
	return stub.asset, stub.err
}

type contentSourceStub struct {
	path string
	err  error
}

func (stub contentSourceStub) OpenContent(context.Context, string, string) (ContentFile, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return os.Open(stub.path)
}

func TestContentServiceOpensVerifiedSupportedSource(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(sourcePath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := NewSourceFingerprint(info.Size(), info.ModTime().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewContentService(contentRepositoryStub{asset: ContentAsset{
		ID:                9,
		LibraryRoot:       "family",
		RelativePath:      "2026/photo.jpg",
		Format:            FormatJPEG,
		MIMEType:          "image/jpeg",
		SizeBytes:         info.Size(),
		ModifiedAtNS:      info.ModTime().UnixNano(),
		SourceFingerprint: fingerprint,
	}}, contentSourceStub{path: sourcePath})
	if err != nil {
		t.Fatal(err)
	}

	content, err := service.Open(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	defer content.File.Close()
	if content.MIMEType != "image/jpeg" || content.SizeBytes != info.Size() {
		t.Fatalf("content = %#v", content)
	}
	if content.ETag == "" || content.ETag[0] != '"' {
		t.Fatalf("ETag = %q", content.ETag)
	}
}

func TestContentServiceRejectsOfflineChangedAndInvalidAssets(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(sourcePath, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	base := ContentAsset{
		ID:                1,
		RelativePath:      "photo.jpg",
		Format:            FormatJPEG,
		MIMEType:          "image/jpeg",
		SizeBytes:         1,
		ModifiedAtNS:      1,
		SourceFingerprint: "v1:1:1",
	}
	tests := []struct {
		name  string
		asset ContentAsset
		want  error
	}{
		{name: "offline", asset: func() ContentAsset {
			asset := base
			asset.LibraryOffline = true
			return asset
		}(), want: ErrContentSourceOffline},
		{name: "source changed", asset: base, want: ErrContentSourceChanged},
		{name: "MIME mismatch", asset: func() ContentAsset {
			asset := base
			asset.MIMEType = "text/plain"
			return asset
		}(), want: ErrInvalidContentState},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewContentService(
				contentRepositoryStub{asset: test.asset},
				contentSourceStub{path: sourcePath},
			)
			if err != nil {
				t.Fatal(err)
			}
			content, err := service.Open(context.Background(), 1)
			if content.File != nil {
				_ = content.File.Close()
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("Open() error = %v, want %v", err, test.want)
			}
		})
	}
}
