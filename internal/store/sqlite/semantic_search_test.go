package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestSemanticVectorSearchRanksStablyAndUsesScoreAssetKeyset(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	page, err := store.ListSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobAll, 0, 10)
	if err != nil || len(page.Items) < 3 {
		t.Fatalf("candidates = %#v, %v", page, err)
	}
	vectors := [][]float32{{1, 0}, {1, 1}, {-1, 0}}
	now := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	items := make([]semantic.EmbeddingItem, 3)
	for index := range items {
		encoded, err := semantic.EncodeEmbedding(vectors[index], 2)
		if err != nil {
			t.Fatal(err)
		}
		items[index] = semantic.EmbeddingItem{AssetID: page.Items[index].AssetID, SourceFingerprint: page.Items[index].SourceFingerprint, Vector: encoded, CreatedAt: now}
	}
	if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: items}); err != nil {
		t.Fatal(err)
	}
	first, err := store.SearchSemanticVectors(context.Background(), semantic.VectorSearchRequest{GenerationID: generationID, LibraryID: libraryID, Query: []float32{1, 0}, Limit: 2})
	if err != nil || len(first) != 2 || first[0].AssetID != items[0].AssetID || first[1].AssetID != items[1].AssetID || first[0].Score <= first[1].Score {
		t.Fatalf("first page = %#v, err %v", first, err)
	}
	after := semantic.SearchPosition{Score: first[1].Score, AssetID: first[1].AssetID}
	second, err := store.SearchSemanticVectors(context.Background(), semantic.VectorSearchRequest{GenerationID: generationID, LibraryID: libraryID, Query: []float32{1, 0}, After: &after, Limit: 2})
	if err != nil || len(second) != 1 || second[0].AssetID != items[2].AssetID {
		t.Fatalf("second page = %#v, err %v", second, err)
	}
	if _, err := store.db.ExecContext(context.Background(), `UPDATE ai_library_settings SET enabled=0, state='disabled' WHERE library_id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	empty, err := store.SearchSemanticVectors(context.Background(), semantic.VectorSearchRequest{GenerationID: generationID, LibraryID: libraryID, Query: []float32{1, 0}, Limit: 2})
	if err != nil || len(empty) != 0 {
		t.Fatalf("disabled results = %#v, %v", empty, err)
	}
}

func TestSemanticSearchSnapshotValidatesScopeAndCapturesCoverage(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	if _, err := store.db.ExecContext(context.Background(), `
		UPDATE ai_model_state
		SET active_model_id=(SELECT model_id FROM semantic_generations WHERE id=?), active_generation_id=?, revision=revision+1
		WHERE singleton_key=1`, generationID, generationID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
		INSERT INTO semantic_library_progress(
			generation_id, library_id, eligible_count, completed_count, failed_count,
			stale_count, checkpoint_id, revision, updated_at_ms
		) VALUES(?, ?, 4, 3, 1, 0, 10, 6, ?)`, generationID, libraryID, time.Now().UnixMilli()); err != nil {
		t.Fatal(err)
	}
	var directoryID int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT id FROM directories WHERE library_id=? AND relative_path='Album 2'`, libraryID).Scan(&directoryID); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.GetSemanticSearchSnapshot(context.Background(), semantic.SearchScope{
		LibraryID: libraryID, DirectoryID: directoryID, Recursive: true,
	})
	if err != nil || snapshot.GenerationID != generationID || snapshot.CatalogRevision < 1 || len(snapshot.Members) != 1 ||
		snapshot.Members[0].Coverage.Completed != 3 || snapshot.Members[0].Coverage.Revision != 6 {
		t.Fatalf("snapshot = %#v, err=%v", snapshot, err)
	}
	if _, err := store.GetSemanticSearchSnapshot(context.Background(), semantic.SearchScope{LibraryID: libraryID, DirectoryID: 999999}); !errors.Is(err, semantic.ErrSemanticScopeNotFound) {
		t.Fatalf("unknown directory error = %v", err)
	}
	if _, err := store.db.ExecContext(context.Background(), `UPDATE libraries SET status='offline' WHERE id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetSemanticSearchSnapshot(context.Background(), semantic.SearchScope{LibraryID: libraryID}); !errors.Is(err, semantic.ErrSemanticLibraryOffline) {
		t.Fatalf("offline library error = %v", err)
	}
}

func TestSemanticVectorSearchScopesToDirectAndRecursiveDirectory(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	page, err := store.ListSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobAll, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	for _, candidate := range page.Items {
		encoded, encodeErr := semantic.EncodeEmbedding([]float32{1, 0}, 2)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{
			GenerationID: generationID,
			LibraryID:    libraryID,
			Items: []semantic.EmbeddingItem{{
				AssetID: candidate.AssetID, SourceFingerprint: candidate.SourceFingerprint,
				Vector: encoded, CreatedAt: now,
			}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	var albumID, nestedID int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT id FROM directories WHERE library_id=? AND relative_path='Album 2'`, libraryID).Scan(&albumID); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT id FROM directories WHERE library_id=? AND relative_path='Album 2/Nested'`, libraryID).Scan(&nestedID); err != nil {
		t.Fatal(err)
	}
	assetPaths := func(matches []semantic.VectorMatch) []string {
		t.Helper()
		paths := make([]string, 0, len(matches))
		for _, match := range matches {
			var path string
			if err := store.db.QueryRowContext(context.Background(), `SELECT relative_path FROM assets WHERE id=?`, match.AssetID).Scan(&path); err != nil {
				t.Fatal(err)
			}
			paths = append(paths, path)
		}
		return paths
	}
	direct, err := store.SearchSemanticVectors(context.Background(), semantic.VectorSearchRequest{
		GenerationID: generationID, LibraryID: libraryID, DirectoryID: albumID,
		Query: []float32{1, 0}, Limit: 10,
	})
	if err != nil || len(direct) != 0 {
		t.Fatalf("direct matches = %#v paths=%v err=%v", direct, assetPaths(direct), err)
	}
	recursive, err := store.SearchSemanticVectors(context.Background(), semantic.VectorSearchRequest{
		GenerationID: generationID, LibraryID: libraryID, DirectoryID: albumID, Recursive: true,
		Query: []float32{1, 0}, Limit: 10,
	})
	paths := assetPaths(recursive)
	if err != nil || len(recursive) != 1 || paths[0] != "Album 2/Nested/photo.jpg" {
		t.Fatalf("recursive matches = %#v paths=%v err=%v", recursive, paths, err)
	}
	nested, err := store.SearchSemanticVectors(context.Background(), semantic.VectorSearchRequest{
		GenerationID: generationID, LibraryID: libraryID, DirectoryID: nestedID, Recursive: true,
		Query: []float32{1, 0}, Limit: 10,
	})
	if err != nil || len(nested) != 1 || assetPaths(nested)[0] != "Album 2/Nested/photo.jpg" {
		t.Fatalf("nested matches = %#v paths=%v err=%v", nested, assetPaths(nested), err)
	}
}

func TestSemanticVectorSearchFailsOnCorruptVectorAndExcludesStaleSource(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	page, err := store.ListSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobAll, 0, 10)
	if err != nil || len(page.Items) < 2 {
		t.Fatalf("candidates=%#v err=%v", page, err)
	}
	vector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	now := time.Date(2026, 8, 28, 17, 0, 0, 0, time.UTC)
	items := []semantic.EmbeddingItem{
		{AssetID: page.Items[0].AssetID, SourceFingerprint: page.Items[0].SourceFingerprint, Vector: vector, CreatedAt: now},
		{AssetID: page.Items[1].AssetID, SourceFingerprint: page.Items[1].SourceFingerprint, Vector: vector, CreatedAt: now},
	}
	if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: items}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `UPDATE assets SET source_fingerprint='v1:changed' WHERE id=?`, items[0].AssetID); err != nil {
		t.Fatal(err)
	}
	matches, err := store.SearchSemanticVectors(context.Background(), semantic.VectorSearchRequest{
		GenerationID: generationID, LibraryID: libraryID, Query: []float32{1, 0}, Limit: 10,
	})
	if err != nil || len(matches) != 1 || matches[0].AssetID != items[1].AssetID {
		t.Fatalf("stale-source matches=%#v err=%v", matches, err)
	}
	if _, err := store.db.ExecContext(context.Background(), `UPDATE semantic_embeddings SET vector=x'0000' WHERE asset_id=?`, items[1].AssetID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SearchSemanticVectors(context.Background(), semantic.VectorSearchRequest{
		GenerationID: generationID, LibraryID: libraryID, Query: []float32{1, 0}, Limit: 10,
	}); !errors.Is(err, semantic.ErrInvalidEmbeddingRecord) {
		t.Fatalf("corrupt vector error = %v", err)
	}
}
