package sqlite

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"testing"
	"time"
)

func TestScanInvalidatesSemanticCoverageAtomicallyAndOnce(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	library := createTestLibrary(t, store)
	run, err := store.BeginFullScan(ctx, library.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}
	entries := catalogFixture("album/photo.jpg")
	if err = store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
		t.Fatal(err)
	}
	if _, err = store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}
	generation := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, library.ID)
	var asset int64
	if err = store.db.QueryRow(`SELECT id FROM assets WHERE library_id=?`, library.ID).Scan(&asset); err != nil {
		t.Fatal(err)
	}
	vector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	if err = store.PutSemanticEmbeddingBatch(ctx, semantic.EmbeddingBatch{GenerationID: generation, LibraryID: library.ID, Items: []semantic.EmbeddingItem{{AssetID: asset, SourceFingerprint: "v1:12:2", Vector: vector, CreatedAt: time.Now()}}}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.db.Exec(`UPDATE ai_model_state SET active_model_id=(SELECT model_id FROM semantic_generations WHERE id=?),active_generation_id=?,revision=revision+1 WHERE singleton_key=1`, generation, generation); err != nil {
		t.Fatal(err)
	}
	if _, err = store.db.Exec(`INSERT INTO semantic_library_progress(generation_id,library_id,eligible_count,completed_count,failed_count,stale_count,checkpoint_id,revision,updated_at_ms) VALUES(?,?,1,1,0,0,0,1,?)`, generation, library.ID, time.Now().UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err = store.db.Exec(`UPDATE ai_library_settings SET state='ready' WHERE library_id=?`, library.ID); err != nil {
		t.Fatal(err)
	}
	run, err = store.BeginFullScan(ctx, library.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	for _, mtime := range []int64{3, 3, 4} {
		entries[2].MTimeNS = mtime
		if err = store.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
			t.Fatal(err)
		}
		settings, err := store.GetSemanticLibrarySettings(ctx, library.ID)
		if err != nil || settings.Coverage.Completed != 0 || settings.Coverage.Stale != 1 || settings.Coverage.Revision != 2 || settings.State != semantic.LibraryDegraded || settings.Coverage.Complete() {
			t.Fatalf("settings=%+v err=%v", settings, err)
		}
		snapshot, err := store.GetSemanticSearchSnapshot(ctx, semantic.SearchScope{LibraryID: library.ID})
		if err != nil || snapshot.Members[0].Coverage != settings.Coverage {
			t.Fatalf("snapshot=%+v err=%v", snapshot, err)
		}
		matches, err := store.SearchSemanticVectors(ctx, semantic.VectorSearchRequest{GenerationID: generation, LibraryID: library.ID, Query: []float32{1, 0}, Limit: 10})
		if err != nil || len(matches) != 0 {
			t.Fatalf("matches=%v err=%v", matches, err)
		}
	}
	if _, found, err := store.GetSemanticEmbedding(ctx, generation, library.ID, asset); err != nil || !found {
		t.Fatal("old embedding deleted", err)
	}
	if _, err = store.FailFullScan(ctx, run.ID, scanner.SkipCounts{}, "partial_tree_unreadable"); err != nil {
		t.Fatal(err)
	}
	settings, err := store.GetSemanticLibrarySettings(ctx, library.ID)
	if err != nil || settings.Coverage.Stale != 1 {
		t.Fatalf("failed scan lost invalidation: %+v %v", settings, err)
	}
	// New candidates count immediately, but missing old files are retained until
	// a complete scan succeeds. Repeated upserts must not inflate eligibility.
	run, err = store.BeginFullScan(ctx, library.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	added := catalogFixture("album/new.jpg")
	added[2].Name = "new.jpg"
	for range 2 {
		if err = store.UpsertCatalogBatch(ctx, run.ID, added); err != nil {
			t.Fatal(err)
		}
	}
	settings, err = store.GetSemanticLibrarySettings(ctx, library.ID)
	if err != nil || settings.Coverage.Eligible != 2 || settings.Coverage.Stale != 1 || settings.Coverage.Complete() {
		t.Fatalf("addition coverage: %+v %v", settings, err)
	}
	if _, err = store.FailFullScan(ctx, run.ID, scanner.SkipCounts{}, "partial_tree_unreadable"); err != nil {
		t.Fatal(err)
	}
	settings, err = store.GetSemanticLibrarySettings(ctx, library.ID)
	if err != nil || settings.Coverage.Eligible != 2 {
		t.Fatalf("failed scan removed candidate: %+v %v", settings, err)
	}
	run, err = store.BeginFullScan(ctx, library.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.UpsertCatalogBatch(ctx, run.ID, added); err != nil {
		t.Fatal(err)
	}
	if _, err = store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}
	settings, err = store.GetSemanticLibrarySettings(ctx, library.ID)
	if err != nil || settings.Coverage.Eligible != 1 || settings.Coverage.Stale != 0 || settings.Coverage.Completed != 0 || settings.Coverage.Complete() {
		t.Fatalf("successful removal coverage: %+v %v", settings, err)
	}
	var newAsset int64
	if err = store.db.QueryRow(`SELECT id FROM assets WHERE library_id=?`, library.ID).Scan(&newAsset); err != nil {
		t.Fatal(err)
	}
	if err = store.PutSemanticEmbeddingBatch(ctx, semantic.EmbeddingBatch{GenerationID: generation, LibraryID: library.ID, Items: []semantic.EmbeddingItem{{AssetID: newAsset, SourceFingerprint: "v1:12:2", Vector: vector, CreatedAt: time.Now()}}}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.db.Exec(`UPDATE semantic_library_progress SET completed_count=1 WHERE library_id=?`, library.ID); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		run, err = store.BeginFullScan(ctx, library.ID, scanner.TriggerManual)
		if err != nil {
			t.Fatal(err)
		}
		if err = store.UpsertCatalogBatch(ctx, run.ID, catalogFixture("")); err != nil {
			t.Fatal(err)
		}
		if _, err = store.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
			t.Fatal(err)
		}
		settings, err = store.GetSemanticLibrarySettings(ctx, library.ID)
		if err != nil || settings.Coverage.Eligible != 0 || settings.Coverage.Completed != 0 || settings.Coverage.Stale != 0 {
			t.Fatalf("completed removal coverage: %+v %v", settings, err)
		}
	}
}
