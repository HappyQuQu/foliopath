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
	result            media.ProcessingResult
	storyboardBytes   []byte
	storyboardFailure error
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
	storyboardService, err := thumbnail.NewStoryboardService(
		database,
		source,
		publisher,
		cacheManager,
		processingStub{},
		thumbnail.ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := thumbnail.NewClaimedProcessor(
		service,
		storyboardService,
		database,
	)
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

func (stub processingStub) Storyboard(
	_ context.Context,
	_ io.ReadSeeker,
	_ media.Format,
	request media.StoryboardRequest,
) (media.StoryboardResult, error) {
	if stub.storyboardFailure != nil {
		return media.StoryboardResult{}, stub.storyboardFailure
	}
	if len(stub.storyboardBytes) == 0 {
		return media.StoryboardResult{}, media.ErrProcessingFailed
	}
	return media.StoryboardResult{
		Bytes:      stub.storyboardBytes,
		FrameCount: len(request.TimestampsMS),
		Columns:    request.Columns,
		Rows:       request.Rows,
		CellWidth:  request.CellWidth,
		CellHeight: request.CellHeight,
	}, nil
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

func TestStoryboardWorkerRunsAfterGridAndPublishesIndependentReadyState(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	original := []byte("synthetic immutable video original")
	videoPath := filepath.Join(mediaRoot, "family", "clip.mp4")
	writeRuntimeFixture(t, mediaRoot, "family/clip.mp4", string(original))
	before, err := os.Stat(videoPath)
	if err != nil {
		t.Fatal(err)
	}
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
	durationMS := int64(10_000)
	videoResult := media.ProcessingResult{
		Metadata: media.Metadata{
			Width:          1920,
			Height:         1080,
			DurationMS:     &durationMS,
			PlaybackStatus: media.PlaybackPlayable,
		},
		Thumbnail: media.Thumbnail{
			Bytes: []byte("grid-webp"),
			Width: 320, Height: 180,
		},
	}
	gridService, err := thumbnail.NewService(
		database,
		source,
		publisher,
		cacheManager,
		processingStub{},
		processingStub{result: videoResult},
		thumbnail.ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	storyboardBytes := []byte("RIFFxxxxWEBP")
	storyboardService, err := thumbnail.NewStoryboardService(
		database,
		source,
		publisher,
		cacheManager,
		processingStub{storyboardBytes: storyboardBytes},
		thumbnail.ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := thumbnail.NewClaimedProcessor(
		gridService,
		storyboardService,
		database,
	)
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
			Workers:           2,
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
	var cacheRelative string
	deadline := time.Now().Add(3 * time.Second)
	for {
		var jobStatus, derivedStatus, gridStatus string
		var frameCount, columns, rows, cellWidth, cellHeight int
		err = inspector.QueryRow(`
            SELECT job.status, storyboard.status, grid.status,
                   storyboard.frame_count, storyboard.sprite_columns,
                   storyboard.sprite_rows, storyboard.cell_width,
                   storyboard.cell_height, storyboard.cache_rel_path
            FROM assets AS asset
            JOIN media_jobs AS job
              ON job.asset_id = asset.id AND job.variant = 'storyboard'
            JOIN thumbnails AS storyboard
              ON storyboard.asset_id = asset.id
             AND storyboard.variant = 'storyboard'
            JOIN thumbnails AS grid
              ON grid.asset_id = asset.id AND grid.variant = 'grid'
            WHERE asset.library_id = ? AND asset.relative_path = 'clip.mp4'`,
			libraryID,
		).Scan(
			&jobStatus,
			&derivedStatus,
			&gridStatus,
			&frameCount,
			&columns,
			&rows,
			&cellWidth,
			&cellHeight,
			&cacheRelative,
		)
		if err == nil && jobStatus == "succeeded" &&
			derivedStatus == "ready" {
			if gridStatus != "ready" ||
				frameCount != 10 || columns != 5 || rows != 2 ||
				cellWidth != 320 || cellHeight != 180 {
				t.Fatalf(
					"storyboard state = job %q derived %q grid %q layout %d/%dx%d/%dx%d",
					jobStatus,
					derivedStatus,
					gridStatus,
					frameCount,
					columns,
					rows,
					cellWidth,
					cellHeight,
				)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("storyboard did not become ready: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	afterBytes, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterBytes, original) ||
		before.Size() != after.Size() ||
		!before.ModTime().Equal(after.ModTime()) {
		t.Fatal("storyboard processing modified the original video")
	}
	cached, err := os.ReadFile(
		filepath.Join(dataRoot, "cache", filepath.FromSlash(cacheRelative)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cached, storyboardBytes) {
		t.Fatalf("cached storyboard bytes = %q", cached)
	}
}
