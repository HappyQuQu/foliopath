package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func TestFaceDiagnosticsBoundedSamplesLegacyReasonsAndAttemptReset(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	entries := []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}}
	for i := 0; i < 30; i++ {
		name := fmt.Sprintf("photo-%02d.jpg", i)
		entries = append(entries, scanner.CatalogEntry{Kind: scanner.CatalogEntryAsset, RelativePath: name, Name: name, AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg", MTimeNS: 1, SizeBytes: 10})
	}
	library := seedSearchCatalog(t, store, "diagnostics", "diagnostics", entries)
	now := time.Now().UTC()
	const generation = "face_generation_diagnostics"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, library, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	first, err := service.Request(ctx, library, generation, face.JobMissing, "diagnostic-attempt-one")
	if err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim=%v %v", found, err)
	}
	processor, _ := face.NewJobProcessor(store, store, &failingFaceAnalyzer{}, store, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(time.Second) }, 10)
	if err := processor.Process(ctx, claimed); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.GetFaceLibrarySettings(ctx, library)
	if err != nil {
		t.Fatal(err)
	}
	d := snapshot.Diagnostics
	if snapshot.Coverage.Failed != 30 || d.RecordedFailures != 30 || d.UnrecordedFailures != 0 || len(d.Samples) != 20 || len(d.Reasons) != 1 || d.Reasons[0].Code != "analysis_failed" || d.LatestTask.OperationID != first.Job.OperationID {
		t.Fatalf("snapshot=%+v diagnostics=%+v", snapshot, d)
	}
	for _, sample := range d.Samples {
		if sample.Name == "" || sample.Code != "analysis_failed" || sample.SizeBytes != 10 {
			t.Fatalf("sample=%+v", sample)
		}
	}
	if _, err := service.Request(ctx, library, generation, face.JobMissing, "diagnostic-attempt-one"); err != nil {
		t.Fatal(err)
	}
	replayed, _ := store.GetFaceLibrarySettings(ctx, library)
	if replayed.Diagnostics.RecordedFailures != 30 {
		t.Fatal("idempotent replay erased details")
	}
	// Historical coverage counts cannot be reverse-engineered into reasons.
	if _, err := store.db.Exec(`DELETE FROM face_asset_failures WHERE library_id=?`, library); err != nil {
		t.Fatal(err)
	}
	legacy, _ := store.GetFaceLibrarySettings(ctx, library)
	if legacy.Diagnostics.UnrecordedFailures != 30 || len(legacy.Diagnostics.Samples) != 0 {
		t.Fatalf("legacy=%+v", legacy.Diagnostics)
	}
	if _, err := service.Request(ctx, library, generation, face.JobMissing, "diagnostic-attempt-two"); err != nil {
		t.Fatal(err)
	}
	current, _ := store.GetFaceLibrarySettings(ctx, library)
	if current.Diagnostics.RecordedFailures != 0 || current.Diagnostics.UnrecordedFailures != 0 || current.Diagnostics.LatestTask.State != "queued" {
		t.Fatalf("new=%+v", current.Diagnostics)
	}
	if _, err := store.db.Exec(`UPDATE libraries SET status='scanning' WHERE id=?`, library); err != nil {
		t.Fatal(err)
	}
	if state, err := store.FaceJobLibraryState(ctx, library, generation); err != nil || state != face.JobLibraryScanning {
		t.Fatalf("scan=%s %v", state, err)
	}
	if _, err := store.db.Exec(`UPDATE face_library_settings SET enabled=0,state='disabled' WHERE library_id=?`, library); err != nil {
		t.Fatal(err)
	}
	if state, err := store.FaceJobLibraryState(ctx, library, generation); err != nil || state != face.JobLibraryDisabled {
		t.Fatalf("disabled while scan=%s %v", state, err)
	}
}

func TestFaceSettingsGenerationFreshnessIsReadOnly(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	library := seedBrowseCatalog(t, store)
	now := time.Now().UTC()
	seedFaceGeneration(t, store, "face_generation_old", 2, "retired", now)
	seedFaceGeneration(t, store, "face_generation_current", 2, "active", now)
	seedFaceReadySettings(t, store, library, "face_generation_old", now)
	old, err := store.GetFaceLibrarySettings(ctx, library)
	if err != nil || old.GenerationCurrent == nil || *old.GenerationCurrent {
		t.Fatalf("historical=%+v err=%v", old, err)
	}
	var bound string
	var revision int64
	if err := store.db.QueryRow(`SELECT active_generation_id,revision FROM face_library_settings WHERE library_id=?`, library).Scan(&bound, &revision); err != nil {
		t.Fatal(err)
	}
	if bound != "face_generation_old" || revision != old.Revision {
		t.Fatal("reading freshness mutated the library")
	}
	if _, err := store.db.Exec(`UPDATE face_library_settings SET active_generation_id='face_generation_current' WHERE library_id=?`, library); err != nil {
		t.Fatal(err)
	}
	current, err := store.GetFaceLibrarySettings(ctx, library)
	if err != nil || current.GenerationCurrent == nil || !*current.GenerationCurrent {
		t.Fatalf("current=%+v err=%v", current, err)
	}
	if _, err := store.db.Exec(`UPDATE face_generations SET state='retired' WHERE state='active'`); err != nil {
		t.Fatal(err)
	}
	unknown, err := store.GetFaceLibrarySettings(ctx, library)
	if err != nil || unknown.GenerationCurrent != nil {
		t.Fatalf("unknown=%+v err=%v", unknown, err)
	}
}

func TestFaceFailureCheckpointConflictDoesNotPersistDiagnostic(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	library := seedBrowseCatalog(t, store)
	now := time.Now().UTC()
	const generation = "face_generation_diagnostic_atomic"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, library, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(ctx, library, generation, face.JobMissing, "diagnostic-atomic-one"); err != nil {
		t.Fatal(err)
	}
	j, _, _ := store.ClaimFaceJob(ctx, now, time.Minute)
	candidates, _ := store.ListFaceJobCandidates(ctx, library, generation, face.JobMissing, 0, 1)
	asset := candidates.Items[0]
	progress, _, _ := store.GetFaceJobProgress(ctx, generation, library)
	commit := face.JobProgressCommit{JobID: j.ID, ClaimedRevision: j.ClaimedRevision, ExpectedProgressRevision: progress.Revision + 1, NextCheckpointID: asset.AssetID, SourceFingerprint: asset.SourceFingerprint, FailedCount: 1, FailureCode: "media_unreadable", UpdatedAt: now, Batch: face.ObservationBatch{GenerationID: generation, LibraryID: library, AssetID: asset.AssetID}}
	if _, err := store.CommitFaceJobProgress(ctx, commit); err == nil {
		t.Fatal("accepted stale checkpoint")
	}
	var count int
	store.db.QueryRow(`SELECT COUNT(*) FROM face_asset_failures`).Scan(&count)
	if count != 0 {
		t.Fatal("failure escaped transaction")
	}
	commit.ExpectedProgressRevision = progress.Revision
	if _, err := store.CommitFaceJobProgress(ctx, commit); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := store.GetFaceLibrarySettings(ctx, library)
	if snapshot.Coverage.Failed != 1 || snapshot.Diagnostics.RecordedFailures != 1 || snapshot.Diagnostics.Samples[0].Code != "media_unreadable" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	// A changed source must not expose this sample as a current-file error.
	if _, err := store.db.Exec(`UPDATE assets SET source_fingerprint='v1:100:9999' WHERE library_id=? AND id=?`, library, asset.AssetID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = store.GetFaceLibrarySettings(ctx, library)
	if snapshot.Diagnostics.RecordedFailures != 0 || snapshot.Diagnostics.UnrecordedFailures != 1 {
		t.Fatalf("stale=%+v", snapshot.Diagnostics)
	}
}
