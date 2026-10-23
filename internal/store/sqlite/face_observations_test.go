package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceGenerationContractDoesNotMisclassifyStorageFailure(t *testing.T) {
	store, _ := openTestStore(t)
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	_, _, err := store.faceGenerationContract(context.Background(), store.db, "face_generation_storage_failure")
	if err == nil || errors.Is(err, face.ErrFaceGenerationUnavailable) {
		t.Fatalf("storage failure=%v", err)
	}
}

func TestReplaceFaceObservationsIsAtomicAndIdempotent(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 17, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_repo", 2, "active", now)
	vector, err := face.EncodeEmbedding([]float32{3, 4}, 2)
	if err != nil {
		t.Fatal(err)
	}
	batch := face.ObservationBatch{GenerationID: "face_generation_repo", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now,
		Items: []face.ObservationItem{
			{ID: "face_repo_0001", Box: face.Box{X: .1, Y: .2, Width: .2, Height: .3}, Detection: .9, Quality: .8, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
			{ID: "face_repo_0002", Box: face.Box{X: .6, Y: .1, Width: .2, Height: .2}, Detection: .8, Quality: .7, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		}}
	if err := store.ReplaceFaceObservations(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceFaceObservations(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListFaceObservations(context.Background(), batch.GenerationID, libraryID, assetID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Revision != 1 || items[1].Revision != 1 {
		t.Fatalf("idempotent items=%+v", items)
	}

	batch.Items = batch.Items[:1]
	batch.UpdatedAt = now.Add(time.Minute)
	if err := store.ReplaceFaceObservations(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListFaceObservations(context.Background(), batch.GenerationID, libraryID, assetID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "face_repo_0001" {
		t.Fatalf("replacement=%+v", items)
	}
}

func TestFaceObservationsRejectDimensionMismatchWithoutPartialWrite(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 17, 30, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_dims", 2, "ready", now)
	vector, err := face.EncodeEmbedding([]float32{1, 1, 1}, 3)
	if err != nil {
		t.Fatal(err)
	}
	err = store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{
		GenerationID: "face_generation_dims", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now,
		Items: []face.ObservationItem{{ID: "face_wrong_dim", Box: face.Box{Width: .5, Height: .5}, Detection: 1, Quality: 1,
			SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}},
	})
	if err == nil {
		t.Fatal("dimension mismatch succeeded")
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_observations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partial rows=%d", count)
	}
}

func TestFaceObservationLifecycleClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	createdAt := time.Date(2026, 8, 31, 18, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_clock", 2, "active", createdAt)
	vector, err := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	batch := face.ObservationBatch{
		GenerationID: "face_generation_clock", LibraryID: libraryID, AssetID: assetID,
		UpdatedAt: createdAt.Add(-time.Hour), Items: []face.ObservationItem{{
			ID: "face_clock_0001", Box: face.Box{Width: .5, Height: .5}, Detection: 1, Quality: 1,
			SourceFingerprint: "source-v1", Vector: vector, CreatedAt: createdAt,
		}},
	}
	if err := store.ReplaceFaceObservations(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	batch.Items[0].Quality = .9
	batch.UpdatedAt = createdAt.Add(-2 * time.Hour)
	if err := store.ReplaceFaceObservations(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListFaceObservations(context.Background(), batch.GenerationID, libraryID, assetID)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].UpdatedAt.Before(items[0].CreatedAt) || items[0].Revision != 2 {
		t.Fatalf("observation=%+v", items[0])
	}
}

func TestDeleteFaceObservationsOnSourceChangePreservesManualAnchor(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 18, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_src", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	batch := face.ObservationBatch{GenerationID: "face_generation_src", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now,
		Items: []face.ObservationItem{{ID: "face_source_001", Box: face.Box{Width: .5, Height: .5}, Detection: 1, Quality: 1,
			SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}
	if err := store.ReplaceFaceObservations(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO people(id,name,created_at_ms,updated_at_ms) VALUES('person_source_1','P',?,?)`, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO person_face_anchors(id,person_id,library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm,current_face_id,state,created_at_ms,updated_at_ms)
		VALUES('anchor_source_1','person_source_1',?,?, 'source-v1',0,0,500000,500000,'face_source_001','bound',?,?)`, libraryID, assetID, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	deleted, err := store.DeleteFaceObservationsIfSourceChanged(context.Background(), batch.GenerationID, libraryID, assetID, "source-v2", now.Add(time.Minute))
	if err != nil || !deleted {
		t.Fatalf("deleted=%v err=%v", deleted, err)
	}
	var state string
	var currentFace any
	if err := store.db.QueryRow(`SELECT state,current_face_id FROM person_face_anchors WHERE id='anchor_source_1'`).Scan(&state, &currentFace); err != nil {
		t.Fatal(err)
	}
	if state != "needs_review" || currentFace != nil {
		t.Fatalf("anchor state=%s current=%v", state, currentFace)
	}
}

func seedFaceGeneration(t *testing.T, store *Store, id string, dimension int, state string, now time.Time) {
	t.Helper()
	hash := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	_, err := store.db.Exec(`INSERT INTO face_generations(id,detector_package_id,detector_content_hash,embedder_package_id,embedder_content_hash,embedding_dimension,transform_version,threshold_profile,state,created_at_ms,activated_at_ms,updated_at_ms)
		VALUES(?, 'yunet', ?, 'sface', ?, ?, 1, 'test-v1', ?, ?, CASE WHEN ?='active' THEN ? ELSE NULL END, ?)`,
		id, hash, hash, dimension, state, now.UnixMilli(), state, now.UnixMilli(), now.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
}
