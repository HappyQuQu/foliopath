package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceRebuildRebindsOnlyExactUniqueUnboundLineages(t *testing.T) {
	for _, scenario := range []string{"exact", "source_changed", "ambiguous", "old_visible_binding"} {
		t.Run(scenario, func(t *testing.T) {
			store, _ := openTestStore(t)
			ctx := context.Background()
			libraryID := seedBrowseCatalog(t, store)
			assetID := catalogAssetID(t, store, "photo-10.jpg")
			now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
			seedFaceGeneration(t, store, "face_rebind_generation", 2, "active", now)
			vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
			item := face.ObservationItem{ID: "face_rebind_original", Box: face.Box{X: .1, Y: .1, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "same-source", Vector: vector, CreatedAt: now}
			batch := face.ObservationBatch{GenerationID: "face_rebind_generation", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{item}}
			if err := store.ReplaceFaceObservations(ctx, batch); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreatePerson(ctx, face.CreatePersonCommand{ID: "person_rebind_001", Name: "Synthetic", CreatedAt: now}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.AssignFace(ctx, face.AssignFaceCommand{EventID: "event_rebind_001", RequestHash: testFaceRequestHash, AnchorID: "anchor_rebind_001", PersonID: "person_rebind_001", FaceID: item.ID, ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now}); err != nil {
				t.Fatal(err)
			}
			if scenario == "old_visible_binding" {
				seedFaceGeneration(t, store, "face_rebind_next_gen", 2, "ready", now)
				batch.GenerationID = "face_rebind_next_gen"
			} else {
				if _, err := store.db.Exec("DELETE FROM face_observations WHERE id=?", item.ID); err != nil {
					t.Fatal(err)
				}
			}
			item.ID = "face_rebind_new"
			if scenario == "source_changed" {
				item.SourceFingerprint = "changed-source"
			}
			batch.Items = []face.ObservationItem{item}
			if scenario == "ambiguous" {
				duplicate := item
				duplicate.ID = "face_rebind_duplicate"
				batch.Items = append(batch.Items, duplicate)
			}
			if err := store.ReplaceFaceObservations(ctx, batch); scenario == "ambiguous" {
				if err == nil {
					t.Fatal("ambiguous lineage must be rejected by unique schema")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var current string
			if err := store.db.QueryRow("SELECT COALESCE(current_face_id,'') FROM person_face_anchors WHERE id='anchor_rebind_001'").Scan(&current); err != nil {
				t.Fatal(err)
			}
			expected := ""
			if scenario == "exact" {
				expected = item.ID
			}
			if scenario == "old_visible_binding" {
				expected = "face_rebind_original"
			}
			if current != expected {
				t.Fatalf("binding=%s want=%s", current, expected)
			}
		})
	}
}
