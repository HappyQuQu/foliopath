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

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/media"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
	"github.com/HappyQuQu/foliopath/internal/thumbnail/cachefs"
)

type processingStub struct {
	result media.ProcessingResult
}

func TestMediaWorkerLifecycleClaimsAndCompletesDurableJob(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	original := []byte("durable media worker original")
	writeRuntimeFixture(t, mediaRoot, "family/photo.jpg", string(original))
	libraryID := seedRuntimeLibrary(t, dataRoot, mediaRoot)

	databaseComponent, database := newDatabaseComponent(
		dataRoot, newReadinessState(),
	)
	if err := databaseComponent.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer databaseComponent.stop(context.Background())
	source, mediaRootComponent, err := newMediaRootService(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := mediaRootComponent.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer mediaRootComponent.stop(context.Background())
	publisher, err := cachefs.New(filepath.Join(dataRoot, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	cacheManager, err := thumbnail.NewCacheManager(database, publisher)
	if err != nil {
		t.Fatal(err)
	}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 96, Height: 64, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{
			Bytes: []byte("worker-webp"), Width: 48, Height: 32,
		},
	}
	service, err := thumbnail.NewService(
		database, source, publisher, cacheManager,
		processingStub{result: result}, processingStub{},
		thumbnail.ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := thumbnail.NewClaimedProcessor(service, database)
	if err != nil {
		t.Fatal(err)
	}
	mediaSignal := jobs.NewSignal()
	cacheSignal := jobs.NewSignal()
	worker, err := jobs.NewWorkerPool(
		mediaJobQueue{database: database},
		processor,
		mediaSignal,
		jobs.WorkerOptions{
			Workers:           1,
			HeartbeatInterval: 20 * time.Millisecond,
			LeaseDuration:     200 * time.Millisecond,
			IdlePollInterval:  10 * time.Millisecond,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	component, err := newMediaWorkerComponent(worker, cacheManager, cacheSignal)
	if err != nil {
		t.Fatal(err)
	}
	if err := component.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer component.stop(context.Background())

	inspector, err := sql.Open(
		"sqlite", filepath.Join(dataRoot, databaseFilename),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer inspector.Close()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var status string
		err := inspector.QueryRow(`
            SELECT j.status
            FROM media_jobs j
            JOIN assets a ON a.id = j.asset_id
            WHERE a.library_id = ? AND a.relative_path = 'photo.jpg'`,
			libraryID,
		).Scan(&status)
		if err == nil && status == "succeeded" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("media job did not succeed: status %q error %v", status, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	after, err := os.ReadFile(filepath.Join(mediaRoot, "family", "photo.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("media worker modified original bytes")
	}
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
	cacheManager, err := thumbnail.NewCacheManager(store, publisher)
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
		cacheManager,
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
