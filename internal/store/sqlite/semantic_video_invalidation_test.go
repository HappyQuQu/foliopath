package sqlite

import (
	"context"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"testing"
	"time"
)

func TestVideoRemovalPreservesCurrentPlanWithoutCreditingFullRebuild(t *testing.T) {
	for _, ready := range []int{0, 1} {
		t.Run(fmt.Sprint(ready), func(t *testing.T) {
			ctx := context.Background()
			store, _ := openTestStore(t)
			seedStoryboardAsset(t, store.db)
			generation := seedEmbeddingGeneration(t, store, 2)
			now := time.Now().UnixMilli()
			vector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
			if _, err := store.db.Exec(`INSERT INTO thumbnails(library_id,asset_id,variant,source_fingerprint,transform_version,cache_rel_path,status,width,height,byte_size,created_at_ms,last_accessed_at_ms,frame_count,sprite_columns,sprite_rows,cell_width,cell_height) VALUES(1,1,'storyboard','v1:42:100',1,'libraries/lib_1/test.webp','ready',1280,180,1000,?,?,4,4,1,320,180)`, now, now); err != nil {
				t.Fatal(err)
			}
			for ordinal := 0; ordinal < 4; ordinal++ {
				if _, err := store.db.Exec(`INSERT INTO semantic_video_frames(generation_id,library_id,asset_id,source_fingerprint,storyboard_fingerprint,storyboard_transform_version,plan_size,ordinal,timestamp_ms,vector,created_at_ms) VALUES(?,1,1,'v1:42:100','test-plan-fingerprint',1,4,?,?,?,?)`, generation, ordinal, ordinal*1000, vector, now); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := store.db.Exec(`INSERT INTO semantic_video_progress(generation_id,library_id,eligible_count,ready_count,revision,updated_at_ms) VALUES(?,1,1,?,1,?)`, generation, ready, now); err != nil {
				t.Fatal(err)
			}
			entries := []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}, {Kind: scanner.CatalogEntryAsset, RelativePath: "clip.mp4", Name: "clip.mp4", AssetKind: scanner.AssetKindVideo, MediaFormat: scanner.MediaFormatMP4, MIMEType: "video/mp4", SizeBytes: 42, MTimeNS: 100}}
			extra := entries[1]
			extra.RelativePath = "other.mp4"
			extra.Name = "other.mp4"
			run, err := store.BeginFullScan(ctx, 1, scanner.TriggerManual)
			if err != nil {
				t.Fatal(err)
			}
			if err = store.UpsertCatalogBatch(ctx, run.ID, append(entries, extra)); err != nil {
				t.Fatal(err)
			}
			if _, err = store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
				t.Fatal(err)
			}
			run, err = store.BeginFullScan(ctx, 1, scanner.TriggerManual)
			if err != nil {
				t.Fatal(err)
			}
			if err = store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
				t.Fatal(err)
			}
			if _, err = store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
				t.Fatal(err)
			}
			p, found, err := store.GetVideoJobProgress(ctx, generation, 1)
			if err != nil || !found || p.Eligible != 1 || p.Ready != int64(ready) {
				t.Fatalf("progress=%+v %v", p, err)
			}
		})
	}
}

func TestScanInvalidatesVideoCoveragePerAsset(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	generation := seedEmbeddingGeneration(t, store, 2)
	vector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	now := time.Now().UnixMilli()
	if _, err := store.db.Exec(`INSERT INTO thumbnails(library_id,asset_id,variant,source_fingerprint,transform_version,cache_rel_path,status,width,height,byte_size,created_at_ms,last_accessed_at_ms,frame_count,sprite_columns,sprite_rows,cell_width,cell_height)
 VALUES(1,1,'storyboard','v1:42:100',1,'libraries/lib_1/test.webp','ready',1280,180,1000,?,?,4,4,1,320,180)`, now, now); err != nil {
		t.Fatal(err)
	}
	for ordinal := 0; ordinal < 4; ordinal++ {
		if _, err := store.db.Exec(`INSERT INTO semantic_video_frames(generation_id,library_id,asset_id,source_fingerprint,storyboard_fingerprint,storyboard_transform_version,plan_size,ordinal,timestamp_ms,vector,created_at_ms) VALUES(?,1,1,'v1:42:100','test-plan-fingerprint',1,4,?,?,?,?)`, generation, ordinal, ordinal*1000, vector, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.db.Exec(`INSERT INTO semantic_video_progress(generation_id,library_id,eligible_count,ready_count,revision,updated_at_ms) VALUES(?,1,1,1,1,?)`, generation, now); err != nil {
		t.Fatal(err)
	}
	counts, err := store.CountVideoJobCandidates(ctx, 1, generation, semantic.JobMissing)
	if err != nil || counts.Eligible != 1 || counts.Pending != 0 {
		t.Fatalf("initial complete plan=%+v %v", counts, err)
	}
	run, err := store.BeginFullScan(ctx, 1, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	entries := []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}, {Kind: scanner.CatalogEntryAsset, RelativePath: "clip.mp4", Name: "clip.mp4", AssetKind: scanner.AssetKindVideo, MediaFormat: scanner.MediaFormatMP4, MIMEType: "video/mp4", SizeBytes: 42, MTimeNS: 101}}
	check := func(eligible, ready, stale int64) {
		t.Helper()
		var e, r, s int64
		if err := store.db.QueryRow(`SELECT eligible_count,ready_count,stale_count FROM semantic_video_progress WHERE generation_id=? AND library_id=1`, generation).Scan(&e, &r, &s); err != nil {
			t.Fatal(err)
		}
		if e != eligible || r != ready || s != stale {
			t.Fatalf("coverage=%d/%d/%d want %d/%d/%d", e, r, s, eligible, ready, stale)
		}
	}
	for _, mtime := range []int64{101, 101, 102} {
		entries[1].MTimeNS = mtime
		if err = store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
			t.Fatal(err)
		}
		check(1, 0, 1)
	}
	entries[1].RelativePath = "new.mp4"
	entries[1].Name = "new.mp4"
	for range 2 {
		if err = store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
			t.Fatal(err)
		}
		check(2, 0, 1)
	}
	if _, err = store.FailFullScan(ctx, run.ID, scanner.SkipCounts{}, "partial_tree_unreadable"); err != nil {
		t.Fatal(err)
	}
	check(2, 0, 1)
	var frames int
	if err = store.db.QueryRow(`SELECT COUNT(*) FROM semantic_video_frames`).Scan(&frames); err != nil || frames != 4 {
		t.Fatalf("frames=%d err=%v", frames, err)
	}
	// Without per-asset failure attribution, deletion returns unknown survivors
	// to pending rather than guessing which failure/degradation disappeared.
	if _, err = store.db.Exec(`UPDATE semantic_video_progress SET stale_count=0,degraded_count=1,failed_count=1 WHERE library_id=1`); err != nil {
		t.Fatal(err)
	}
	assertUnknown := func(eligible, degraded, failed int64) {
		t.Helper()
		p, found, err := store.GetVideoJobProgress(ctx, generation, 1)
		if err != nil || !found || p.Eligible != eligible || p.Ready != 0 || p.Stale != 0 || p.Degraded != degraded || p.Failed != failed {
			t.Fatalf("progress=%+v %v", p, err)
		}
	}
	run, err = store.BeginFullScan(ctx, 1, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
		t.Fatal(err)
	}
	if _, err = store.FailFullScan(ctx, run.ID, scanner.SkipCounts{}, "partial_tree_unreadable"); err != nil {
		t.Fatal(err)
	}
	assertUnknown(2, 1, 1)
	run, err = store.BeginFullScan(ctx, 1, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
		t.Fatal(err)
	}
	if _, err = store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}
	assertUnknown(1, 0, 0)
	for range 2 {
		run, err = store.BeginFullScan(ctx, 1, scanner.TriggerManual)
		if err != nil {
			t.Fatal(err)
		}
		if err = store.UpsertCatalogBatch(ctx, run.ID, entries[:1]); err != nil {
			t.Fatal(err)
		}
		if _, err = store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
			t.Fatal(err)
		}
		assertUnknown(0, 0, 0)
	}
}
