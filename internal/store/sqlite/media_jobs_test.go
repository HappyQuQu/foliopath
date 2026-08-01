package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

func TestMediaJobClaimIsFairAcrossLibraries(t *testing.T) {
	store, _ := openTestStore(t)
	firstLibraryID := seedBrowseCatalog(t, store)
	libraries, err := library.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	second, err := libraries.Create(context.Background(), "Videos", "videos")
	if err != nil {
		t.Fatal(err)
	}
	run, err := store.BeginFullScan(
		context.Background(), second.ID, scanner.TriggerManual,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(context.Background(), run.ID, []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "second.jpg",
			Name: "second.jpg", AssetKind: scanner.AssetKindImage,
			MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg",
			SizeBytes: 1, MTimeNS: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}
	first, found, err := store.ClaimNextMediaJob(context.Background(), time.Minute)
	if err != nil || !found || first.LibraryID != firstLibraryID {
		t.Fatalf("first claim = %#v, %t, %v", first, found, err)
	}
	if err := store.FinishMediaJob(context.Background(), first, thumbnail.JobResult{
		Outcome: thumbnail.JobSucceeded,
	}); err != nil {
		t.Fatal(err)
	}
	next, found, err := store.ClaimNextMediaJob(context.Background(), time.Minute)
	if err != nil || !found || next.LibraryID != second.ID {
		t.Fatalf("fair claim = %#v, %t, %v", next, found, err)
	}
}

func TestMediaProcessingProgressAggregatesCurrentLibraryJobs(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO media_jobs(
			library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, attempt_count,
			created_at_ms, started_at_ms, heartbeat_at_ms, lease_expires_at_ms
		) VALUES
			(1, 1, 'grid', 0, 1, 'v1:42:100', 'running', 0, 1, 1,
			 1, 1, 100);
	`); err != nil {
		t.Fatal(err)
	}

	progress, found, err := store.GetMediaProcessingProgress(ctx, 1)
	if err != nil || !found {
		t.Fatalf("progress = %#v, found %t, error %v", progress, found, err)
	}
	if progress.Grid.Running != 1 || progress.Grid.Total() != 1 ||
		progress.Storyboard.Total() != 0 ||
		progress.StoryboardPendingEligibility != 1 || !progress.Active() {
		t.Fatalf("progress = %#v", progress)
	}

	empty, found, err := store.GetMediaProcessingProgress(ctx, 999)
	if err != nil || found || empty != (thumbnail.ProcessingProgress{}) {
		t.Fatalf("missing progress = %#v, found %t, error %v", empty, found, err)
	}
}

func TestMediaJobClaimStrictlyPrioritizesGridBeforeStoryboard(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	ctx := context.Background()

	var assetID int64
	var fingerprint string
	if err := store.db.QueryRowContext(ctx, `
		SELECT id, source_fingerprint FROM assets ORDER BY id LIMIT 1
	`).Scan(&assetID, &fingerprint); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		UPDATE media_jobs
		SET status = 'succeeded', finished_at_ms = 1
		WHERE variant = 'grid';
		UPDATE media_jobs
		SET status = 'queued', finished_at_ms = NULL
		WHERE asset_id = ? AND variant = 'grid';
		INSERT INTO media_jobs(
			library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, created_at_ms
		)
		SELECT library_id, id, 'storyboard', 100, ?, source_fingerprint,
		       'queued', 0, 0
		FROM assets WHERE id = ?;
	`, assetID, thumbnail.StoryboardTransformVersion, assetID); err != nil {
		t.Fatal(err)
	}

	grid, found, err := store.ClaimNextMediaJob(ctx, time.Minute)
	if err != nil || !found || grid.Variant != thumbnail.VariantGrid {
		t.Fatalf("first claim = %#v, %t, %v", grid, found, err)
	}
	if err := store.FinishMediaJob(ctx, grid, thumbnail.JobResult{
		Outcome: thumbnail.JobSucceeded,
	}); err != nil {
		t.Fatal(err)
	}
	storyboard, found, err := store.ClaimNextMediaJob(ctx, time.Minute)
	if err != nil || !found ||
		storyboard.Variant != thumbnail.VariantStoryboard ||
		storyboard.SourceFingerprint.String() != fingerprint {
		t.Fatalf("second claim = %#v, %t, %v", storyboard, found, err)
	}
}

func TestStoryboardAdmissionIsEligibleIdempotentAndBounded(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO thumbnails(
			library_id, asset_id, variant, source_fingerprint,
			transform_version, cache_rel_path, status, width, height,
			byte_size, created_at_ms, last_accessed_at_ms
		) VALUES (
			1, 1, 'grid', 'v1:42:100', 1,
			'libraries/lib_1/aa/grid.webp', 'ready', 512, 288, 1024, 1, 1
		)
	`); err != nil {
		t.Fatal(err)
	}
	admitted, err := store.AdmitStoryboardJobs(ctx, 1)
	if err != nil || admitted != 1 {
		t.Fatalf("first admission = %d, %v", admitted, err)
	}
	admitted, err = store.AdmitStoryboardJobs(ctx, 1)
	if err != nil || admitted != 0 {
		t.Fatalf("idempotent admission = %d, %v", admitted, err)
	}
	if _, err := store.AdmitStoryboardJobs(
		ctx,
		MaxStoryboardAdmissionBatch+1,
	); err == nil {
		t.Fatal("unbounded storyboard admission unexpectedly accepted")
	}
	var variant string
	var priority, transform int
	if err := store.db.QueryRowContext(ctx, `
		SELECT variant, priority, transform_version
		FROM media_jobs
		WHERE asset_id = 1 AND variant = 'storyboard'
	`).Scan(&variant, &priority, &transform); err != nil {
		t.Fatal(err)
	}
	if variant != "storyboard" || priority != 100 ||
		transform != thumbnail.StoryboardTransformVersion {
		t.Fatalf("admitted job = %q/%d/v%d", variant, priority, transform)
	}
}

func TestStoryboardClaimConcurrencyIsOneWithoutBlockingGrid(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO assets(
			id, library_id, directory_id, relative_path, name, kind,
			media_format, mime_type, size_bytes, mtime_ns, last_seen_generation,
			source_fingerprint, width, height, duration_ms, probe_status,
			playback_status
		) VALUES (
			2, 1, 1, 'second.mp4', 'second.mp4', 'video',
			'mp4', 'video/mp4', 43, 101, 1,
			'v1:43:101', 1920, 1080, 10000, 'ready', 'playable'
		);
		INSERT INTO media_jobs(
			library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, created_at_ms
		) VALUES
			(1, 1, 'storyboard', 100, 1, 'v1:42:100', 'queued', 0, 0),
			(1, 2, 'storyboard', 100, 1, 'v1:43:101', 'queued', 0, 0);
	`); err != nil {
		t.Fatal(err)
	}
	first, found, err := store.ClaimNextMediaJob(ctx, time.Minute)
	if err != nil || !found || first.Variant != thumbnail.VariantStoryboard {
		t.Fatalf("first storyboard claim = %#v, %t, %v", first, found, err)
	}
	if second, found, err := store.ClaimNextMediaJob(
		ctx,
		time.Minute,
	); err != nil || found {
		t.Fatalf("parallel storyboard claim = %#v, %t, %v", second, found, err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO media_jobs(
			library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, created_at_ms
		) VALUES (
			1, 2, 'grid', 0, 1, 'v1:43:101', 'queued', 0, 0
		)
	`); err != nil {
		t.Fatal(err)
	}
	grid, found, err := store.ClaimNextMediaJob(ctx, time.Minute)
	if err != nil || !found || grid.Variant != thumbnail.VariantGrid {
		t.Fatalf("grid claim beside storyboard = %#v, %t, %v", grid, found, err)
	}
}

func TestExhaustedStoryboardJobDoesNotFailVideoOrPosterMetadata(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO media_jobs(
			library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, attempt_count,
			created_at_ms
		) VALUES (
			1, 1, 'storyboard', 100, 1, 'v1:42:100', 'queued', 0, 2, 0
		)
	`); err != nil {
		t.Fatal(err)
	}
	job, found, err := store.ClaimNextMediaJob(ctx, time.Minute)
	if err != nil || !found || job.Variant != thumbnail.VariantStoryboard ||
		job.Attempt != thumbnail.MaxJobAttempts {
		t.Fatalf("claim = %#v, %t, %v", job, found, err)
	}
	if err := store.FinishMediaJob(ctx, job, thumbnail.JobResult{
		Outcome:    thumbnail.JobRetry,
		Code:       thumbnail.JobErrorProcessing,
		RetryDelay: time.Second,
	}); err != nil {
		t.Fatal(err)
	}

	var probe, playback, storyboardStatus, storyboardError string
	var width, height, duration int64
	if err := store.db.QueryRowContext(ctx, `
		SELECT
			a.probe_status, a.playback_status, a.width, a.height, a.duration_ms,
			t.status, t.error_code
		FROM assets AS a
		JOIN thumbnails AS t
		  ON t.asset_id = a.id AND t.variant = 'storyboard'
		WHERE a.id = 1
	`).Scan(
		&probe,
		&playback,
		&width,
		&height,
		&duration,
		&storyboardStatus,
		&storyboardError,
	); err != nil {
		t.Fatal(err)
	}
	if probe != "ready" || playback != "playable" ||
		width != 1920 || height != 1080 || duration != 10_000 ||
		storyboardStatus != "failed" ||
		storyboardError != string(media.ErrorProcessingFailed) {
		t.Fatalf(
			"asset/storyboard = %q %q %dx%d/%d %q/%q",
			probe,
			playback,
			width,
			height,
			duration,
			storyboardStatus,
			storyboardError,
		)
	}
}

func TestMediaJobRetriesAreLeasedBoundedAndTerminal(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.UnixMilli(1000)
	store.now = func() time.Time { return now }
	seedBrowseCatalog(t, store)

	job, found, err := store.ClaimNextMediaJob(context.Background(), time.Minute)
	if err != nil || !found {
		t.Fatalf("claim = %#v, %t, %v", job, found, err)
	}
	if job.Attempt != 1 {
		t.Fatalf("attempt = %d", job.Attempt)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        DELETE FROM media_jobs WHERE id <> ?`,
		job.ID,
	); err != nil {
		t.Fatal(err)
	}
	if err := store.RefreshMediaJobLease(
		context.Background(), job, time.Minute,
	); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishMediaJob(context.Background(), job, thumbnail.JobResult{
		Outcome:    thumbnail.JobRetry,
		Code:       thumbnail.JobErrorSource,
		RetryDelay: 5 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.ClaimNextMediaJob(
		context.Background(), time.Minute,
	); err != nil || found {
		t.Fatalf("early retry claim = %t, %v", found, err)
	}
	now = now.Add(5 * time.Second)
	retry, found, err := store.ClaimNextMediaJob(context.Background(), time.Minute)
	if err != nil || !found || retry.ID != job.ID || retry.Attempt != 2 {
		t.Fatalf("retry = %#v, %t, %v", retry, found, err)
	}
	if err := store.FinishMediaJob(context.Background(), retry, thumbnail.JobResult{
		Outcome:    thumbnail.JobRetry,
		Code:       thumbnail.JobErrorCache,
		RetryDelay: 5 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(9 * time.Second)
	if _, found, err := store.ClaimNextMediaJob(
		context.Background(), time.Minute,
	); err != nil || found {
		t.Fatalf("early exponential retry claim = %t, %v", found, err)
	}
	now = now.Add(time.Second)
	final, found, err := store.ClaimNextMediaJob(context.Background(), time.Minute)
	if err != nil || !found || final.ID != job.ID || final.Attempt != 3 {
		t.Fatalf("final retry = %#v, %t, %v", final, found, err)
	}
	if err := store.FinishMediaJob(context.Background(), final, thumbnail.JobResult{
		Outcome: thumbnail.JobSucceeded,
	}); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT status FROM media_jobs WHERE id = ?`,
		job.ID,
	).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "succeeded" {
		t.Fatalf("status = %q", status)
	}
}

func TestFingerprintChangeInvalidatesThumbnailAndRequeuesSameAsset(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var assetID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM assets
        WHERE library_id = ? AND relative_path = 'photo-2.jpg'`,
		libraryID,
	).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	asset, err := store.GetAssetForDerivation(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 20, Height: 10, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{
			Bytes: []byte("webp"), Width: 20, Height: 10,
		},
	}
	if err := store.CommitReady(context.Background(), thumbnail.Ready{
		AssetID: assetID, SourceFingerprint: asset.SourceFingerprint,
		Result: result, CacheRelativePath: "libraries/lib_1/old.webp",
		ByteSize: 4, CreatedAtMS: 100,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
		INSERT INTO media_jobs(
			library_id, asset_id, variant, priority, transform_version,
			source_fingerprint, status, available_at_ms, created_at_ms
		)
		SELECT library_id, id, 'storyboard', 100, ?,
		       source_fingerprint, 'queued', 0, 0
		FROM assets WHERE id = ?
	`, thumbnail.StoryboardTransformVersion, assetID); err != nil {
		t.Fatal(err)
	}

	run, err := store.BeginFullScan(
		context.Background(), libraryID, scanner.TriggerManual,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(context.Background(), run.ID, []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "photo-2.jpg",
			Name: "photo-2.jpg", AssetKind: scanner.AssetKindImage,
			MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg",
			SizeBytes: 21, MTimeNS: 200,
		},
	}); err != nil {
		t.Fatal(err)
	}
	var fingerprint, probe, jobStatus string
	var attempts, thumbnails, deletions, storyboardJobs int
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT a.source_fingerprint, a.probe_status,
               j.status, j.attempt_count,
               (SELECT count(*) FROM thumbnails WHERE asset_id = a.id),
               (SELECT count(*) FROM cache_deletions
                WHERE cache_rel_path = 'libraries/lib_1/old.webp'),
               (SELECT count(*) FROM media_jobs
                WHERE asset_id = a.id AND variant = 'storyboard')
        FROM assets a JOIN media_jobs j ON j.asset_id = a.id
        WHERE a.id = ?`,
		assetID,
	).Scan(
		&fingerprint, &probe, &jobStatus, &attempts,
		&thumbnails, &deletions, &storyboardJobs,
	); err != nil {
		t.Fatal(err)
	}
	if fingerprint == asset.SourceFingerprint.String() || probe != "pending" ||
		jobStatus != "queued" || attempts != 0 ||
		thumbnails != 0 || deletions != 1 || storyboardJobs != 0 {
		t.Fatalf(
			"state = fingerprint %q probe %q job %q/%d thumbnails %d deletions %d storyboards %d",
			fingerprint, probe, jobStatus, attempts, thumbnails, deletions,
			storyboardJobs,
		)
	}
}

func TestUnchangedFingerprintPreservesReadyDerivedState(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var assetID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM assets
        WHERE library_id = ? AND relative_path = 'photo-2.jpg'`,
		libraryID,
	).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	asset, err := store.GetAssetForDerivation(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CommitReady(context.Background(), thumbnail.Ready{
		AssetID: assetID, SourceFingerprint: asset.SourceFingerprint,
		Result: media.ProcessingResult{
			Metadata: media.Metadata{
				Width: 20, Height: 10,
				PlaybackStatus: media.PlaybackNotApplicable,
			},
			Thumbnail: media.Thumbnail{
				Bytes: []byte("webp"), Width: 20, Height: 10,
			},
		},
		CacheRelativePath: "libraries/lib_1/ready.webp",
		ByteSize:          4,
		CreatedAtMS:       100,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        UPDATE media_jobs
        SET status = 'succeeded', attempt_count = 2,
            created_at_ms = 50, finished_at_ms = 100
        WHERE asset_id = ?`,
		assetID,
	); err != nil {
		t.Fatal(err)
	}

	run, err := store.BeginFullScan(
		context.Background(), libraryID, scanner.TriggerManual,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(context.Background(), run.ID, []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "photo-2.jpg",
			Name: "photo-2.jpg", AssetKind: scanner.AssetKindImage,
			MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg",
			SizeBytes: 20, MTimeNS: 20,
		},
	}); err != nil {
		t.Fatal(err)
	}

	var status string
	var attempts, createdAt, thumbnails, deletions int
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT status, attempt_count, created_at_ms,
               (SELECT count(*) FROM thumbnails WHERE asset_id = ?),
               (SELECT count(*) FROM cache_deletions
                WHERE cache_rel_path = 'libraries/lib_1/ready.webp')
        FROM media_jobs WHERE asset_id = ?`,
		assetID, assetID,
	).Scan(&status, &attempts, &createdAt, &thumbnails, &deletions); err != nil {
		t.Fatal(err)
	}
	if status != "succeeded" || attempts != 2 || createdAt != 50 ||
		thumbnails != 1 || deletions != 0 {
		t.Fatalf(
			"unchanged state = job %q/%d created %d thumbnails %d deletions %d",
			status, attempts, createdAt, thumbnails, deletions,
		)
	}
}

func TestTransformVersionReconciliationIsBoundedAndRequeues(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	var assetID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM assets ORDER BY id LIMIT 1`,
	).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	asset, err := store.GetAssetForDerivation(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 10, Height: 10, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{
			Bytes: []byte("old"), Width: 10, Height: 10,
		},
	}
	if err := store.CommitReady(context.Background(), thumbnail.Ready{
		AssetID: assetID, SourceFingerprint: asset.SourceFingerprint,
		Result: result, CacheRelativePath: "libraries/lib_1/old-transform.webp",
		ByteSize: 3, CreatedAtMS: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        UPDATE media_jobs
        SET transform_version = 99, status = 'succeeded',
            finished_at_ms = 1
        WHERE asset_id = ?`,
		assetID,
	); err != nil {
		t.Fatal(err)
	}
	changed, err := store.ReconcileMediaJobTransform(
		context.Background(), thumbnail.GridTransformVersion, 1,
	)
	if err != nil || changed != 1 {
		t.Fatalf("reconcile = %d, %v", changed, err)
	}
	var version, attempts, thumbnails, deletions int
	var status string
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT transform_version, status, attempt_count,
               (SELECT count(*) FROM thumbnails WHERE asset_id = ?),
               (SELECT count(*) FROM cache_deletions
                WHERE cache_rel_path = 'libraries/lib_1/old-transform.webp')
        FROM media_jobs WHERE asset_id = ?`,
		assetID, assetID,
	).Scan(&version, &status, &attempts, &thumbnails, &deletions); err != nil {
		t.Fatal(err)
	}
	if version != thumbnail.GridTransformVersion || status != "queued" ||
		attempts != 0 || thumbnails != 0 || deletions != 1 {
		t.Fatalf(
			"reconciled state = version %d status %q attempts %d thumbnails %d deletions %d",
			version, status, attempts, thumbnails, deletions,
		)
	}
}

func TestExpiredMediaJobsRecoverWithoutExceedingGlobalBound(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.UnixMilli(10_000)
	store.now = func() time.Time { return now }
	seedBrowseCatalog(t, store)
	for index := 0; index < thumbnail.MediaWorkerCount; index++ {
		job, found, err := store.ClaimNextMediaJob(
			context.Background(), time.Millisecond,
		)
		if err != nil || !found {
			t.Fatalf("claim %d = %t, %v", index, found, err)
		}
		if _, err := store.db.ExecContext(context.Background(), `
            UPDATE media_jobs
            SET attempt_count = ?, lease_expires_at_ms = ?
            WHERE id = ?`,
			thumbnail.MaxJobAttempts, now.Add(-time.Second).UnixMilli(), job.ID,
		); err != nil {
			t.Fatal(err)
		}
	}
	summary, err := store.RecoverExpiredMediaJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary != (jobs.RecoverySummary{Interrupted: thumbnail.MediaWorkerCount}) {
		t.Fatalf("recovery = %#v", summary)
	}
}
