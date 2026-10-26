package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/ocr"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func persistenceIdentity() ocr.Identity {
	return ocr.Identity{Version: 1, Protocol: "foliopath-ocr-v1", Model: ocr.Model, DetectorSHA256: strings.Repeat("a", 64), RecognizerSHA256: strings.Repeat("b", 64), InputContractSHA256: ocr.InputContractSHA256(), OutputContractSHA256: ocr.OutputContractSHA256(), RuntimeContractSHA256: strings.Repeat("e", 64), CacheManifestSHA256: strings.Repeat("f", 64)}
}
func setupOCR(t *testing.T) (*Store, int64, ocr.Settings, time.Time) {
	t.Helper()
	s, _ := openTestStore(t)
	id := seedBrowseCatalog(t, s)
	now := time.Now().UTC()
	v, err := s.GetOCRSettings(context.Background(), id)
	if err != nil || v.Enabled || v.Revision != 1 || v.Epoch != 1 {
		t.Fatal(v, err)
	}
	v, err = s.ConfigureOCR(context.Background(), id, 1, true, persistenceIdentity(), now)
	if err != nil {
		t.Fatal(err)
	}
	return s, id, v, now
}
func admitOCR(t *testing.T, s *Store, v ocr.Settings, mode ocr.Mode, key string, now time.Time) ocr.Job {
	t.Helper()
	j, replayed, err := s.AdmitOCRJob(context.Background(), v.LibraryID, v.Revision, mode, key, "ocrjob_"+key, now)
	if err != nil || replayed {
		t.Fatal(j, replayed, err)
	}
	return j
}
func claimOCR(t *testing.T, s *Store, now time.Time) ocr.Job {
	t.Helper()
	j, found, err := s.ClaimOCRJob(context.Background(), now, time.Minute)
	if err != nil || !found {
		t.Fatal(j, found, err)
	}
	return j
}
func resultOCR(t *testing.T, s *Store, j ocr.Job, now time.Time) ocr.Commit {
	t.Helper()
	c, found, err := s.NextOCRCandidate(context.Background(), j, now)
	if err != nil || !found {
		t.Fatal(c, found, err)
	}
	return ocr.Commit{Job: j, Value: ocr.StoredResult{Candidate: c, Fingerprint: j.Fingerprint, Outcome: "ready", Width: 1000, Height: 700, Result: ocr.Result{Lines: []ocr.Line{{Text: "中文 Cafe\u0301", Box: [8]float64{.1, .1, .9, .1, .9, .3, .1, .3}, DetectionScore: .9, RecognitionScore: .95}}}}}
}
func reloadOCR(t *testing.T, s *Store, j ocr.Job) ocr.Job {
	t.Helper()
	v, err := s.GetOCRJob(context.Background(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestOCRPersistenceCommitEmptyFailureAndMissing(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "missing001", now)
	j := claimOCR(t, s, now)
	if err := s.FinishOCRJob(ctx, j, "succeeded", "", now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal("premature finish", err)
	}
	c := resultOCR(t, s, j, now)
	if err := s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	first := c.Value.AssetID
	stored, found, err := s.GetCurrentOCRResult(ctx, id, first)
	if err != nil || !found || stored.Result.Lines[0].Text != "中文 Café" {
		t.Fatal(stored, found, err)
	}
	if err = s.CommitOCRResult(ctx, c, now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal("replay", err)
	}
	// Heartbeat owns the claim, not the mutable checkpoint snapshot.
	if err = s.RefreshOCRLease(ctx, j, now.Add(time.Second), time.Minute); err != nil {
		t.Fatal("heartbeat after progress", err)
	}
	j = reloadOCR(t, s, j)
	c = resultOCR(t, s, j, now)
	c.Value.Result.Lines = []ocr.Line{}
	if err = s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	emptyID := c.Value.AssetID
	stored, found, err = s.GetCurrentOCRResult(ctx, id, emptyID)
	if err != nil || !found || stored.Result.Lines == nil || len(stored.Result.Lines) != 0 {
		t.Fatal(stored, found, err)
	}
	j = reloadOCR(t, s, j)
	c = resultOCR(t, s, j, now)
	c.Value.Outcome = "failed"
	c.Value.ErrorCode = "invalid_media"
	c.Value.Width, c.Value.Height = 0, 0
	c.Value.Result.Lines = nil
	if err = s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	failedID := c.Value.AssetID
	j = reloadOCR(t, s, j)
	if j.Completed != 3 || j.Failed != 1 {
		t.Fatal(j)
	}
	if err = s.FinishOCRJob(ctx, j, "succeeded", "", now); err != nil {
		t.Fatal(err)
	}
	original := reloadOCR(t, s, j)
	replay, replayed, err := s.AdmitOCRJob(ctx, id, v.Revision, ocr.Missing, "missing001", "unused_id", now)
	if err != nil || !replayed || replay != original {
		t.Fatal(replay, replayed, err)
	}
	if _, _, err = s.AdmitOCRJob(ctx, id, v.Revision, ocr.RetryFailed, "missing001", "unused_id", now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal(err)
	}
	admitOCR(t, s, v, ocr.Missing, "missing002", now)
	j = claimOCR(t, s, now)
	if _, found, err = s.NextOCRCandidate(ctx, j, now); err != nil || found {
		t.Fatal("poison source loop", found, err)
	}
	if err = s.FinishOCRJob(ctx, j, "succeeded", "", now); err != nil {
		t.Fatal(err)
	}
	admitOCR(t, s, v, ocr.RetryFailed, "retry00001", now)
	j = claimOCR(t, s, now)
	c = resultOCR(t, s, j, now)
	if c.Value.AssetID != failedID {
		t.Fatal(c)
	}
	if err = s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	j = reloadOCR(t, s, j)
	if err = s.FinishOCRJob(ctx, j, "succeeded", "", now); err != nil {
		t.Fatal(err)
	}
	if _, found, err = s.GetCurrentOCRResult(ctx, id+100, first); err != nil || found {
		t.Fatal("cross library", found, err)
	}
}
func TestOCRPersistenceSourceAndGenerationFences(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "source0001", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	if _, err := s.db.Exec(`UPDATE assets SET source_fingerprint='v1:50:99',size_bytes=50,mtime_ns=99 WHERE id=?`, c.Value.AssetID); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitOCRResult(ctx, c, now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal("stale source", err)
	}
	c = resultOCR(t, s, j, now)
	if c.Value.SourceFingerprint != "v1:50:99" {
		t.Fatal(c)
	}
	if err := s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	j = reloadOCR(t, s, j)
	next := resultOCR(t, s, j, now)
	changed := persistenceIdentity()
	changed.RuntimeContractSHA256 = strings.Repeat("c", 64)
	v2, err := s.ConfigureOCR(ctx, id, v.Revision, true, changed, now)
	if err != nil || v2.Fingerprint == v.Fingerprint {
		t.Fatal(v2, err)
	}
	if err = s.CommitOCRResult(ctx, next, now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal("old generation", err)
	}
	if _, found, err := s.GetCurrentOCRResult(ctx, id, c.Value.AssetID); err != nil || found {
		t.Fatal(found, err)
	}
	if _, _, err := s.AdmitOCRJob(ctx, id, v2.Revision, ocr.Missing, "source0002", "job_source02", now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal("old child still owns slot", err)
	}
	if err = s.FinishOCRJob(ctx, j, "cancelled", "", now); err != nil {
		t.Fatal(err)
	}
	admitOCR(t, s, v2, ocr.Missing, "source0003", now)
	newJob := claimOCR(t, s, now)
	newResult := resultOCR(t, s, newJob, now)
	if newResult.Value.AssetID != c.Value.AssetID {
		t.Fatal(newResult)
	}
	if err = s.CommitOCRResult(ctx, newResult, now); err != nil {
		t.Fatal(err)
	}
	// Old identity result retained, never relabelled or overwritten.
	var count int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_asset_results WHERE asset_id=?`, c.Value.AssetID).Scan(&count); err != nil || count != 2 {
		t.Fatal(count, err)
	}
	if _, err = s.db.Exec(`UPDATE assets SET source_fingerprint='v1:60:100' WHERE id=?`, c.Value.AssetID); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.GetCurrentOCRResult(ctx, id, c.Value.AssetID); err != nil || found {
		t.Fatal("stale visible", found, err)
	}
}
func TestOCRPersistenceRecoveryReopenAndAttemptLimit(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "recover001", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	if err := s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	j = reloadOCR(t, s, j)
	var filename string
	if err := s.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&filename); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s.db = reopened.db
	s.queries = reopened.queries
	for attempt := int64(1); attempt <= 3; attempt++ {
		if err = s.RefreshOCRLease(ctx, j, now.Add(time.Minute), time.Minute); !errors.Is(err, ocr.ErrConflict) {
			t.Fatal("resurrect lease", err)
		}
		summary, e := s.RecoverOCRJobs(ctx, now.Add(time.Minute))
		if e != nil {
			t.Fatal(e)
		}
		if e = s.CommitOCRResult(ctx, c, now.Add(time.Minute)); !errors.Is(e, ocr.ErrConflict) {
			t.Fatal("expired commit", e)
		}
		if attempt == 3 {
			if summary.Interrupted != 1 {
				t.Fatal(summary)
			}
			break
		}
		if summary.Requeued != 1 {
			t.Fatal(summary)
		}
		now = now.Add(time.Minute)
		j = claimOCR(t, s, now)
		if j.Attempts != attempt+1 || j.CheckpointID != c.Value.AssetID || j.Completed != 1 {
			t.Fatal(j)
		}
	}
	terminal := reloadOCR(t, s, j)
	if terminal.State != "failed" || terminal.ErrorCode != "interrupted" {
		t.Fatal(terminal)
	}
	if _, found, e := s.GetCurrentOCRResult(ctx, id, c.Value.AssetID); e != nil || !found {
		t.Fatal(found, e)
	}
	if _, found, e := s.ClaimOCRJob(ctx, now.Add(2*time.Minute), time.Minute); e != nil || found {
		t.Fatal(found, e)
	}
}
func TestOCRPersistenceCancelDisableOfflineAndServiceFailure(t *testing.T) {
	for _, kind := range []string{"cancel", "disable", "offline", "clear", "model_unavailable"} {
		t.Run(kind, func(t *testing.T) {
			s, id, v, now := setupOCR(t)
			ctx := context.Background()
			admitOCR(t, s, v, ocr.Missing, "fencing001", now)
			j := claimOCR(t, s, now)
			c := resultOCR(t, s, j, now)
			var err error
			switch kind {
			case "cancel":
				err = s.CancelOCRJob(ctx, j.ID, j.ClaimRevision, now)
			case "disable":
				_, err = s.ConfigureOCR(ctx, id, v.Revision, false, ocr.Identity{}, now)
			case "offline":
				_, err = s.db.Exec(`UPDATE libraries SET status='offline' WHERE id=?`, id)
			case "clear":
				_, err = s.BeginOCRClear(ctx, id, v.Revision, now)
			case "model_unavailable":
				err = s.FinishOCRJob(ctx, j, "failed", kind, now)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = s.CommitOCRResult(ctx, c, now); !errors.Is(err, ocr.ErrConflict) {
				t.Fatal(err)
			}
			if _, found, e := s.GetCurrentOCRResult(ctx, id, c.Value.AssetID); e != nil || found {
				t.Fatal(found, e)
			}
			var n int
			if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_asset_results`).Scan(&n); err != nil || n != 0 {
				t.Fatal(n, err)
			}
		})
	}
}
func TestOCRPersistenceClearBatchesResumeAndIsolation(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "clear00001", now)
	j := claimOCR(t, s, now)
	for range 3 {
		c := resultOCR(t, s, j, now)
		if err := s.CommitOCRResult(ctx, c, now); err != nil {
			t.Fatal(err)
		}
		j = reloadOCR(t, s, j)
	}
	clear, err := s.BeginOCRClear(ctx, id, v.Revision, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConfigureOCR(ctx, id, clear.Revision, true, persistenceIdentity(), now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal(err)
	}
	for index := 0; index < 3; index++ {
		done, e := s.DrainOCRClear(ctx, id, clear.Epoch, 1, now)
		if e != nil || done != (index == 2) {
			t.Fatal(index, done, e)
		}
		var count int
		if e = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_asset_results`).Scan(&count); e != nil || count != 2-index {
			t.Fatal(count, e)
		}
		if index == 0 {
			var filename string
			if e = s.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&filename); e != nil {
				t.Fatal(e)
			}
			if e = s.Close(); e != nil {
				t.Fatal(e)
			}
			reopened, e := Open(ctx, filename, Options{})
			if e != nil {
				t.Fatal(e)
			}
			s.db, s.queries = reopened.db, reopened.queries
		}
	}
	if done, e := s.DrainOCRClear(ctx, id, clear.Epoch, 1, now); e != nil || !done {
		t.Fatal(done, e)
	}
	final, err := s.GetOCRSettings(ctx, id)
	if err != nil || final.Clearing || final.Enabled || final.Revision != clear.Revision+1 {
		t.Fatal(final, err)
	}
	if _, err = s.BeginOCRClear(ctx, id, v.Revision, now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal(err)
	}
	if _, err = s.ConfigureOCR(ctx, id, final.Revision, true, persistenceIdentity(), now); err != nil {
		t.Fatal(err)
	}
	if err = s.CommitOCRResult(ctx, resultWithCandidate(j, 1, "v1:1:1"), now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal(err)
	}
}
func resultWithCandidate(j ocr.Job, id int64, source string) ocr.Commit {
	return ocr.Commit{Job: j, Value: ocr.StoredResult{Candidate: ocr.Candidate{AssetID: id, SourceFingerprint: source}, Fingerprint: j.Fingerprint, Outcome: "ready", Width: 32, Height: 32, Result: ocr.Result{Lines: []ocr.Line{}}}}
}
func TestOCRPersistenceAtomicRollbackAndConcurrentCommit(t *testing.T) {
	s, _, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "atomic0001", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	if _, err := s.db.Exec(`CREATE TRIGGER ocr_fail_checkpoint BEFORE UPDATE OF checkpoint_id ON ocr_jobs BEGIN SELECT RAISE(ABORT,'synthetic checkpoint fault'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitOCRResult(ctx, c, now); err == nil {
		t.Fatal("fault did not fire")
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM ocr_asset_results`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if reloadOCR(t, s, j) != j {
		t.Fatal("partial checkpoint")
	}
	if _, err := s.db.Exec(`DROP TRIGGER ocr_fail_checkpoint`); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- s.CommitOCRResult(ctx, c, now) }()
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ocr.ErrConflict):
			conflicts++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal(success, conflicts)
	}
}
func TestOCRPersistenceValidationAndForgedTokens(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	bad := persistenceIdentity()
	bad.InputContractSHA256 = strings.Repeat("0", 64)
	if _, err := s.ConfigureOCR(ctx, id, v.Revision, true, bad, now); !errors.Is(err, ocr.ErrInvalidIdentity) {
		t.Fatal(err)
	}
	if _, err := s.ConfigureOCR(ctx, id, v.Revision-1, true, persistenceIdentity(), now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal(err)
	}
	admitOCR(t, s, v, ocr.Missing, "validate01", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	for _, mutate := range []func(*ocr.Commit){func(c *ocr.Commit) { c.Job.LibraryID++ }, func(c *ocr.Commit) { c.Job.Epoch++ }, func(c *ocr.Commit) { c.Job.ClaimRevision++ }, func(c *ocr.Commit) { c.Job.CheckpointID++ }, func(c *ocr.Commit) { c.Value.AssetID++ }, func(c *ocr.Commit) { c.Value.SourceFingerprint = "v1:999:999" }} {
		changed := c
		mutate(&changed)
		if err := s.CommitOCRResult(ctx, changed, now); !errors.Is(err, ocr.ErrConflict) {
			t.Fatal(changed, err)
		}
	}
	for _, code := range []string{"model_unavailable", "arbitrary exception /private/path", ""} {
		bad := c
		bad.Value.Outcome = "failed"
		bad.Value.ErrorCode = code
		bad.Value.Width, bad.Value.Height = 0, 0
		bad.Value.Result.Lines = nil
		if err := s.CommitOCRResult(ctx, bad, now); !errors.Is(err, ocr.ErrInvalidPersistence) {
			t.Fatal(err)
		}
	}
	if err := s.FinishOCRJob(ctx, j, "failed", "raw exception", now); !errors.Is(err, ocr.ErrInvalidPersistence) {
		t.Fatal(err)
	}
	if err := s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
}
func TestOCRPersistenceLibraryAssetCascadeAndProtection(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "protect001", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	manual := newCurationServiceForStore(t, s, now)
	tag, _, err := manual.CreateTag(ctx, "人工中文")
	if err != nil {
		t.Fatal(err)
	}
	before, err := manual.GetAssetState(ctx, c.Value.AssetID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = manual.ReplaceAssetTags(ctx, c.Value.AssetID, before.Revision, []int64{tag.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO asset_tag_suppressions VALUES(?,?,?,?,?)`, id, c.Value.AssetID, "tree", now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if err = s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	clear, err := s.BeginOCRClear(ctx, id, v.Revision, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DrainOCRClear(ctx, id, clear.Epoch, 500, now); err != nil {
		t.Fatal(err)
	}
	after, err := manual.GetAssetState(ctx, c.Value.AssetID)
	if err != nil || len(after.Tags) != 1 || after.Tags[0].ID != tag.ID {
		t.Fatal(after, err)
	}
	var n int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM asset_tag_suppressions`).Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if _, err = s.db.Exec(`DELETE FROM libraries WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"ocr_library_settings", "ocr_jobs", "ocr_job_requests", "ocr_asset_results"} {
		if err = s.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Fatal(table, n, err)
		}
	}
	rows, err := s.db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("FK violation")
	}
}
func TestOCRPersistenceUpperBoundAndNewSourcesDeferred(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "bound00001", now)
	j := claimOCR(t, s, now)
	for range 3 {
		c := resultOCR(t, s, j, now)
		if err := s.CommitOCRResult(ctx, c, now); err != nil {
			t.Fatal(err)
		}
		j = reloadOCR(t, s, j)
	}
	// Add a new asset via the catalog owner, not the OCR adapter.
	run, err := s.BeginFullScan(ctx, id, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	err = s.UpsertCatalogBatch(ctx, run.ID, []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}, {Kind: scanner.CatalogEntryAsset, RelativePath: "new.jpg", Name: "new.jpg", AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg", SizeBytes: 10, MTimeNS: 10}})
	if err != nil {
		t.Fatal(err)
	}
	// Do not complete this synthetic scan: it would correctly remove the old assets.
	if _, err = s.db.Exec(`UPDATE libraries SET status='ready' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if _, found, e := s.NextOCRCandidate(ctx, j, now); e != nil || found {
		t.Fatal(found, e)
	}
	if err = s.FinishOCRJob(ctx, j, "succeeded", "", now); err != nil {
		t.Fatal(err)
	}
	admitOCR(t, s, v, ocr.Missing, "bound00002", now)
	j = claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	if c.Value.AssetID <= j.CheckpointID || c.Value.AssetID != j.UpperAssetID {
		t.Fatal(c, j)
	}
}

// Keep test imports exercised by explicit on-disk migration/reopen evidence.
func TestOCRPersistenceFreshAndDetachedMigration(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "isolated.db")
	ctx := context.Background()
	s, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO ocr_generations VALUES(?,?,1)`, strings.Repeat("a", 64), "{}"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, filename, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var count int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_generations`).Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	var integrity string
	if err = s.db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatal(integrity, err)
	}
	var version sql.NullInt64
	if err = s.db.QueryRow(`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1`).Scan(&version); err != nil || version.Int64 != 47 {
		t.Fatal(fmt.Sprint(version), err)
	}
}

func TestOCRPersistenceClearCommitRaceCannotResurrect(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "raceclear1", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	start := make(chan struct{})
	results := make(chan error, 2)
	var clear ocr.Settings
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); <-start; results <- s.CommitOCRResult(ctx, c, now) }()
	go func() {
		defer wg.Done()
		<-start
		var err error
		clear, err = s.BeginOCRClear(ctx, id, v.Revision, now)
		results <- err
	}()
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil && !errors.Is(err, ocr.ErrConflict) {
			t.Fatal(err)
		}
	}
	if !clear.Clearing {
		t.Fatal(clear)
	}
	if _, err := s.DrainOCRClear(ctx, id, clear.Epoch, 500, now); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM ocr_asset_results`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if err := s.CommitOCRResult(ctx, c, now); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal("resurrected", err)
	}
}
func TestOCRPersistenceQueuedCancellationAndDisablePreserveResults(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	queued := admitOCR(t, s, v, ocr.Missing, "queued0001", now)
	if err := s.CancelOCRJob(ctx, queued.ID, 0, now); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.ClaimOCRJob(ctx, now, time.Minute); err != nil || found {
		t.Fatal(found, err)
	}
	admitOCR(t, s, v, ocr.Missing, "queued0002", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	if err := s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	off, err := s.ConfigureOCR(ctx, id, v.Revision, false, ocr.Identity{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.GetCurrentOCRResult(ctx, id, c.Value.AssetID); err != nil || found {
		t.Fatal(found, err)
	}
	on, err := s.ConfigureOCR(ctx, id, off.Revision, true, persistenceIdentity(), now)
	if err != nil {
		t.Fatal(err)
	}
	if on.Epoch <= off.Epoch {
		t.Fatal(on)
	}
	if _, found, err := s.GetCurrentOCRResult(ctx, id, c.Value.AssetID); err != nil || !found {
		t.Fatal("lost completed same-model result", found, err)
	}
	// Changing media kind without a source fingerprint change also hides OCR.
	if _, err = s.db.Exec(`UPDATE assets SET kind='video' WHERE id=?`, c.Value.AssetID); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.GetCurrentOCRResult(ctx, id, c.Value.AssetID); err != nil || found {
		t.Fatal(found, err)
	}
}
func TestOCRPersistenceGlobalAdmissionBound(t *testing.T) {
	s, _ := openTestStore(t)
	ctx := context.Background()
	now := time.Now()
	for index := 0; index <= ocr.MaximumActiveJobs; index++ {
		id := seedSearchCatalog(t, s, fmt.Sprintf("OCR %d", index), fmt.Sprintf("ocr-%d", index), []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}})
		v, err := s.ConfigureOCR(ctx, id, 1, true, persistenceIdentity(), now)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = s.AdmitOCRJob(ctx, id, v.Revision, ocr.Missing, fmt.Sprintf("capacity%03d", index), fmt.Sprintf("jobcapacity%03d", index), now)
		if index == ocr.MaximumActiveJobs {
			if !errors.Is(err, ocr.ErrConflict) {
				t.Fatal("unbounded jobs", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
}

func ocrProtectedDigest(t *testing.T, s *Store) string {
	t.Helper()
	h := sha256.New()
	for _, table := range []string{"assets", "directories", "tags", "asset_tags", "asset_tag_suppressions", "curation_state", "people", "person_face_anchors", "face_observations", "face_generations", "face_library_settings", "face_audit_events", "semantic_generations", "semantic_embeddings", "ai_model_operations", "autotag_asset_runs", "autotag_library_settings"} {
		rows, err := s.db.Query(`SELECT * FROM ` + table + ` ORDER BY rowid`)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		_, _ = h.Write([]byte(table))
		for rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err = rows.Scan(pointers...); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = h.Write(raw)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		if err = rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
func TestOCRPersistenceMigration44AndNonemptyOtherCapabilities(t *testing.T) {
	s, _ := openTestStore(t)
	ctx := context.Background()
	id, assetID := seedFaceLifecycleState(t, s)
	now := time.Now()
	generation := seedEmbeddingGeneration(t, s, 2)
	vector, err := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.PutSemanticEmbeddingBatch(ctx, semantic.EmbeddingBatch{GenerationID: generation, LibraryID: id, Items: []semantic.EmbeddingItem{{AssetID: assetID, SourceFingerprint: "v1:10:10", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	manual := newCurationServiceForStore(t, s, now)
	tag, _, err := manual.CreateTag(ctx, "人工中文")
	if err != nil {
		t.Fatal(err)
	}
	before, err := manual.GetAssetState(ctx, assetID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = manual.ReplaceAssetTags(ctx, assetID, before.Revision, []int64{tag.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO asset_tag_suppressions VALUES(?,?,?,?,?)`, id, assetID, "tree", now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	// Strip only this dormant empty migration to construct a genuine v44 baseline.
	provider, err := goose.NewProvider(goose.DialectSQLite3, s.db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.DownTo(ctx, 44); err != nil {
		t.Fatal(err)
	}
	digest := ocrProtectedDigest(t, s)
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if ocrProtectedDigest(t, s) != digest {
		t.Fatal("migration changed protected state")
	}
	settings, err := s.GetOCRSettings(ctx, id)
	if err != nil || settings.Enabled {
		t.Fatal(settings, err)
	}
	var jobs int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_jobs`).Scan(&jobs); err != nil || jobs != 0 {
		t.Fatal("migration enabled work", jobs, err)
	}
	v, err := s.ConfigureOCR(ctx, id, 1, true, persistenceIdentity(), now)
	if err != nil {
		t.Fatal(err)
	}
	admitOCR(t, s, v, ocr.Missing, "migrate001", now)
	j := claimOCR(t, s, now)
	for range 3 {
		c := resultOCR(t, s, j, now)
		if err = s.CommitOCRResult(ctx, c, now); err != nil {
			t.Fatal(err)
		}
		j = reloadOCR(t, s, j)
	}
	clear, err := s.BeginOCRClear(ctx, id, v.Revision, now)
	if err != nil {
		t.Fatal(err)
	}
	for {
		done, err := s.DrainOCRClear(ctx, id, clear.Epoch, 1, now)
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
	}
	if ocrProtectedDigest(t, s) != digest {
		t.Fatal("OCR modified tags/suppression/people/semantic/catalog")
	}
	var filename string
	if err = s.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&filename); err != nil {
		t.Fatal(err)
	}
	if err = CheckIntegrity(ctx, filename); err != nil {
		t.Fatal(err)
	}
	t.Logf("13+ protected table digest unchanged: %s; integrity passed", digest)
}

func TestOCRPersistenceOfflineFinishAndCancellationAfterProgress(t *testing.T) {
	for _, kind := range []string{"offline", "cancel-after-progress"} {
		t.Run(kind, func(t *testing.T) {
			s, id, v, now := setupOCR(t)
			ctx := context.Background()
			admitOCR(t, s, v, ocr.Missing, "terminal01", now)
			original := claimOCR(t, s, now)
			c := resultOCR(t, s, original, now)
			if err := s.CommitOCRResult(ctx, c, now); err != nil {
				t.Fatal(err)
			}
			if kind == "offline" {
				if _, err := s.db.Exec(`UPDATE libraries SET status='offline' WHERE id=?`, id); err != nil {
					t.Fatal(err)
				}
				current := reloadOCR(t, s, original)
				if err := s.FinishOCRJob(ctx, current, "failed", "source_unavailable", now); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := s.CancelOCRJob(ctx, original.ID, original.ClaimRevision, now); err != nil {
					t.Fatal(err)
				}
				// The worker's original claim can acknowledge cancellation after already
				// committed progress; acknowledgement never overwrites those counters.
				if err := s.FinishOCRJob(ctx, original, "cancelled", "", now); err != nil {
					t.Fatal(err)
				}
			}
			current := reloadOCR(t, s, original)
			if current.Completed != 1 || current.Failed != 0 {
				t.Fatal(current)
			}
			var n int
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM ocr_asset_results`).Scan(&n); err != nil || n != 1 {
				t.Fatal(n, err)
			}
		})
	}
}
func TestOCRPersistenceTwoLibrariesClearAndAssetDeleteIsolation(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "twolib0001", now)
	j := claimOCR(t, s, now)
	first := resultOCR(t, s, j, now)
	if err := s.CommitOCRResult(ctx, first, now); err != nil {
		t.Fatal(err)
	}
	other := seedSearchCatalog(t, s, "Other OCR", "other-ocr", []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}, {Kind: scanner.CatalogEntryAsset, RelativePath: "other.jpg", Name: "other.jpg", AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg", SizeBytes: 1, MTimeNS: 1}})
	secondSettings, err := s.ConfigureOCR(ctx, other, 1, true, persistenceIdentity(), now)
	if err != nil {
		t.Fatal(err)
	}
	admitOCR(t, s, secondSettings, ocr.Missing, "twolib0002", now)
	secondJob := claimOCR(t, s, now)
	second := resultOCR(t, s, secondJob, now)
	if err = s.CommitOCRResult(ctx, second, now); err != nil {
		t.Fatal(err)
	}
	clear, err := s.BeginOCRClear(ctx, id, v.Revision, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DrainOCRClear(ctx, id, clear.Epoch, 500, now); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.GetCurrentOCRResult(ctx, other, second.Value.AssetID); err != nil || !found {
		t.Fatal("other library lost", found, err)
	}
	if _, found, err := s.GetCurrentOCRResult(ctx, id, second.Value.AssetID); err != nil || found {
		t.Fatal("cross-library result", found, err)
	}
	if _, err = s.db.Exec(`DELETE FROM assets WHERE library_id=? AND id=?`, other, second.Value.AssetID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_asset_results`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
}
