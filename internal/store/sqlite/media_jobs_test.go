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
	var attempts, thumbnails, deletions int
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT a.source_fingerprint, a.probe_status,
               j.status, j.attempt_count,
               (SELECT count(*) FROM thumbnails WHERE asset_id = a.id),
               (SELECT count(*) FROM cache_deletions
                WHERE cache_rel_path = 'libraries/lib_1/old.webp')
        FROM assets a JOIN media_jobs j ON j.asset_id = a.id
        WHERE a.id = ?`,
		assetID,
	).Scan(
		&fingerprint, &probe, &jobStatus, &attempts, &thumbnails, &deletions,
	); err != nil {
		t.Fatal(err)
	}
	if fingerprint == asset.SourceFingerprint.String() || probe != "pending" ||
		jobStatus != "queued" || attempts != 0 ||
		thumbnails != 0 || deletions != 1 {
		t.Fatalf(
			"state = fingerprint %q probe %q job %q/%d thumbnails %d deletions %d",
			fingerprint, probe, jobStatus, attempts, thumbnails, deletions,
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
