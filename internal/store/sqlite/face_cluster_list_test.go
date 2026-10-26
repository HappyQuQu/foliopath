package sqlite

import (
	"context"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/face"
	"testing"
	"time"
)

func TestPendingClusterDirectoryUsesCurrentReviewStateAndSnapshot(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)
	generation := "face_generation_pending"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	observations := []face.ObservationItem{}
	for i, id := range []string{"face_pending_001", "face_pending_002", "face_pending_003", "face_pending_004", "face_pending_005"} {
		observations = append(observations, face.ObservationItem{ID: id, Box: face.Box{X: float32(i) * .2, Width: .1, Height: .1}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now})
	}
	if err := store.ReplaceFaceObservations(ctx, face.ObservationBatch{GenerationID: generation, LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: observations}); err != nil {
		t.Fatal(err)
	}
	groups := []face.Cluster{{ID: "cluster_pending_01", Role: "core", Members: []face.ClusterMember{{FaceID: observations[0].ID, Role: "core", Confidence: 1}, {FaceID: observations[1].ID, Role: "core", Confidence: 1}}}, {ID: "cluster_pending_02", Role: "core", Members: []face.ClusterMember{{FaceID: observations[2].ID, Role: "core", Confidence: 1}, {FaceID: observations[4].ID, Role: "core", Confidence: 1}}}, {ID: "cluster_pending_03", Role: "edge", Members: []face.ClusterMember{{FaceID: observations[3].ID, Role: "edge", Confidence: 0}}}}
	if err := store.ReplaceFaceClusters(ctx, face.ClusterBatch{GenerationID: generation, LibraryID: libraryID, UpdatedAt: now, Clusters: groups}); err != nil {
		t.Fatal(err)
	}
	service, _ := face.NewFaceClusterListService(store, []byte("0123456789abcdef0123456789abcdef"))
	list := func(role, review string, limit int) face.FaceClusterPage {
		t.Helper()
		p, err := service.List(ctx, face.FaceClusterListRequest{LibraryID: libraryID, Role: role, Review: review, Limit: limit})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	first := list("", "pending", 1)
	if first.Counts != (face.FaceClusterCounts{TotalGroups: 3, PendingGroups: 3, TotalFaces: 5, PendingFaces: 5}) || len(first.Items) != 1 || first.NextCursor == "" || first.Items[0].PendingMemberCount != 2 || len(first.Items[0].PreviewAssetIDs) != 1 {
		t.Fatalf("initial=%+v", first)
	}
	second, err := service.List(ctx, face.FaceClusterListRequest{LibraryID: libraryID, Review: "pending", Limit: 1, Cursor: first.NextCursor})
	if err != nil || second.Items[0].ID != groups[1].ID || second.Counts != first.Counts {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if _, err := store.CreatePerson(ctx, face.CreatePersonCommand{ID: "person_pending_01", Name: "测试", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignFace(ctx, face.AssignFaceCommand{EventID: "event_pending_assign", RequestHash: testFaceRequestHash, AnchorID: "anchor_pending_01", FaceID: observations[0].ID, PersonID: "person_pending_01", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Second)}); err != nil {
		t.Fatal(err)
	}
	partial := list("core", "pending", 10)
	if partial.Counts.PendingFaces != 3 || partial.Counts.PendingGroups != 2 || partial.Counts.TotalFaces != 4 || partial.Items[0].PendingMemberCount != 1 {
		t.Fatalf("partial=%+v", partial)
	}
	if _, err := service.List(ctx, face.FaceClusterListRequest{LibraryID: libraryID, Review: "pending", Limit: 1, Cursor: first.NextCursor}); !errors.Is(err, face.ErrFaceClusterCursorStale) {
		t.Fatalf("review cursor=%v", err)
	}
	if _, err := store.ExcludeFace(ctx, face.ExcludeFaceCommand{EventID: "event_pending_exclude", RequestHash: testFaceRequestHash, ExclusionID: "exclusion_pending_01", FaceID: observations[1].ID, ExpectedFaceRevision: 1, CreatedAt: now.Add(2 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	pending := list("core", "pending", 10)
	all := list("core", "all", 10)
	if len(pending.Items) != 1 || pending.Items[0].ID != groups[1].ID || pending.Counts != (face.FaceClusterCounts{TotalGroups: 2, PendingGroups: 1, TotalFaces: 4, PendingFaces: 2}) || len(all.Items) != 2 || all.Items[0].PendingMemberCount != 0 {
		t.Fatalf("pending=%+v all=%+v", pending, all)
	}
	detail, err := store.GetFaceClusterDetailSnapshot(ctx, libraryID, groups[0].ID)
	if err != nil || detail.Cluster.PendingMemberCount != 0 || detail.Cluster.MemberCount != 2 {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
	if _, err := store.SplitFace(ctx, face.SplitFaceCommand{EventID: "event_pending_split", RequestHash: testFaceRequestHash, FaceID: observations[0].ID, SourcePersonID: "person_pending_01", ExpectedFaceRevision: 1, ExpectedSourceRevision: 2, CreatedAt: now.Add(3 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	restored := list("core", "pending", 10)
	if len(restored.Items) != 2 || restored.Counts.PendingFaces != 3 {
		t.Fatalf("restored=%+v", restored)
	}
	edge := list("edge", "pending", 10)
	if edge.Counts.TotalGroups != 1 || edge.Counts.PendingFaces != 1 {
		t.Fatalf("edge=%+v", edge)
	}
	for i, id := range []string{observations[0].ID, observations[2].ID, observations[4].ID} {
		if _, err := store.ExcludeFace(ctx, face.ExcludeFaceCommand{EventID: fmt.Sprintf("event_pending_empty_%d", i), RequestHash: testFaceRequestHash, ExclusionID: fmt.Sprintf("exclusion_pending_empty_%d", i), FaceID: id, ExpectedFaceRevision: 1, CreatedAt: now.Add(4 * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	empty := list("core", "pending", 1)
	if len(empty.Items) != 0 || empty.NextCursor != "" || empty.Counts.PendingGroups != 0 || empty.Counts.PendingFaces != 0 || empty.Counts.TotalGroups != 2 {
		t.Fatalf("empty=%+v", empty)
	}
	if got := list("core", "all", 10); len(got.Items) != 2 {
		t.Fatalf("lost history=%+v", got)
	}
	if _, err := store.db.Exec(`UPDATE libraries SET status='offline' WHERE id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(ctx, face.FaceClusterListRequest{LibraryID: libraryID, Review: "pending", Limit: 1}); !errors.Is(err, face.ErrFaceNotReady) {
		t.Fatalf("offline=%v", err)
	}
}
