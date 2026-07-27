package app

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
	"github.com/HappyQuQu/foliopath/internal/thumbnail/cachefs"
)

type processingStub struct {
	result media.ProcessingResult
}

func (stub processingStub) Process(
	context.Context,
	io.ReadSeeker,
	media.Format,
) (media.ProcessingResult, error) {
	return stub.result, nil
}

func TestMediaDerivationCompositionPreservesOriginalAndPublishesBeforeReady(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	original := []byte("synthetic immutable original")
	writeRuntimeFixture(t, mediaRoot, "family/photo.jpg", string(original))
	libraryID := seedRuntimeLibrary(t, dataRoot, mediaRoot)

	store, err := sqlitestore.Open(
		context.Background(),
		filepath.Join(dataRoot, databaseFilename),
		sqlitestore.Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	inspector, err := sql.Open(
		"sqlite",
		filepath.Join(dataRoot, databaseFilename),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer inspector.Close()
	var assetID int64
	if err := inspector.QueryRow(`
        SELECT id FROM assets
        WHERE library_id = ? AND relative_path = 'photo.jpg'`,
		libraryID,
	).Scan(&assetID); err != nil {
		t.Fatal(err)
	}

	source, lifecycle, err := newMediaRootService(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer lifecycle.stop(context.Background())
	publisher, err := cachefs.New(filepath.Join(dataRoot, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 96, Height: 64, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{
			Bytes: []byte("synthetic-webp"), Width: 48, Height: 32,
		},
	}
	service, err := thumbnail.NewService(
		store,
		source,
		publisher,
		processingStub{result: result},
		processingStub{},
		thumbnail.ServiceOptions{
			Now: func() time.Time { return time.UnixMilli(1000) },
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(context.Background(), assetID); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(filepath.Join(mediaRoot, "family", "photo.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("media derivation modified original bytes")
	}
	var probeStatus, thumbnailStatus, cacheRelative string
	if err := inspector.QueryRow(`
        SELECT a.probe_status, t.status, t.cache_rel_path
        FROM assets a JOIN thumbnails t ON t.asset_id = a.id
        WHERE a.id = ?`,
		assetID,
	).Scan(&probeStatus, &thumbnailStatus, &cacheRelative); err != nil {
		t.Fatal(err)
	}
	if probeStatus != "ready" || thumbnailStatus != "ready" {
		t.Fatalf("derived state = %q, %q", probeStatus, thumbnailStatus)
	}
	cached, err := os.ReadFile(
		filepath.Join(dataRoot, "cache", filepath.FromSlash(cacheRelative)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(cached) != "synthetic-webp" {
		t.Fatalf("cached bytes = %q", cached)
	}
}
