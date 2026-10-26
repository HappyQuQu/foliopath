package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceProjectionsPagePersonAssetsAndExposeOnlyCoarseMultiFaceState(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	firstAsset := catalogAssetID(t, store, "photo-10.jpg")
	secondAsset := catalogAssetID(t, store, "photo-2.jpg")
	now := time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)
	const generation = "face_generation_projection"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	first := face.ObservationBatch{GenerationID: generation, LibraryID: libraryID, AssetID: firstAsset, UpdatedAt: now, Items: []face.ObservationItem{
		{ID: "face_projection_01", Box: face.Box{X: .1234, Y: .2345, Width: .201, Height: .202}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		{ID: "face_projection_02", Box: face.Box{X: .5, Y: .2, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		{ID: "face_projection_03", Box: face.Box{X: .1, Y: .6, Width: .15, Height: .15}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		{ID: "face_projection_04", Box: face.Box{X: .4, Y: .6, Width: .15, Height: .15}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
	}}
	if err := store.ReplaceFaceObservations(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := face.ObservationBatch{GenerationID: generation, LibraryID: libraryID, AssetID: secondAsset, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_projection_05", Box: face.Box{X: .2, Y: .2, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v2", Vector: vector, CreatedAt: now}}}
	if err := store.ReplaceFaceObservations(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceFaceClusters(context.Background(), face.ClusterBatch{GenerationID: generation, LibraryID: libraryID, UpdatedAt: now, Clusters: []face.Cluster{{ID: "face_projection_group", Role: "core", Members: []face.ClusterMember{{FaceID: "face_projection_03", Role: "core", Confidence: 1}, {FaceID: "face_projection_04", Role: "core", Confidence: 1}}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_projection_1", Name: "多人", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	revision := int64(1)
	for index, item := range []struct {
		faceID  string
		assetID int64
	}{{"face_projection_01", firstAsset}, {"face_projection_02", firstAsset}, {"face_projection_05", secondAsset}} {
		_ = item.assetID
		if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: []string{"event_projection_1", "event_projection_2", "event_projection_3"}[index], RequestHash: testFaceRequestHash, AnchorID: []string{"anchor_projection_1", "anchor_projection_2", "anchor_projection_3"}[index], FaceID: item.faceID, PersonID: "person_projection_1", ExpectedFaceRevision: 1, ExpectedPersonRevision: revision, CreatedAt: now.Add(time.Duration(index+1) * time.Minute)}); err != nil {
			t.Fatal(err)
		}
		revision++
	}
	faces, err := store.ListAssetFaceViews(context.Background(), firstAsset)
	if err != nil || len(faces) != 4 {
		t.Fatalf("faces=%+v err=%v", faces, err)
	}
	states := map[string]string{}
	for _, item := range faces {
		states[item.FaceID] = item.State
		if item.Region.XPercent < 0 || item.Region.XPercent > 100 || item.Region.WidthPercent < 1 || item.Region.WidthPercent > 100 {
			t.Fatalf("unsafe region=%+v", item.Region)
		}
	}
	if states["face_projection_01"] != "assigned" || states["face_projection_02"] != "assigned" || states["face_projection_03"] != "anonymous_core" || states["face_projection_04"] != "anonymous_core" {
		t.Fatalf("states=%v", states)
	}
	service, err := face.NewPersonAssetService(store, []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	page1, err := service.List(context.Background(), face.PersonAssetRequest{PersonID: "person_projection_1", Limit: 1})
	if err != nil || len(page1.Items) != 1 || page1.NextCursor == "" || page1.Items[0].LibraryID != libraryID || page1.Items[0].AssetID != secondAsset || len(page1.Items[0].FaceIDs) != 1 {
		t.Fatalf("page1=%+v err=%v", page1, err)
	}
	page2, err := service.List(context.Background(), face.PersonAssetRequest{PersonID: "person_projection_1", Limit: 1, Cursor: page1.NextCursor})
	if err != nil || len(page2.Items) != 1 || page2.Items[0].AssetID != firstAsset || len(page2.Items[0].FaceIDs) != 2 {
		t.Fatalf("page2=%+v err=%v", page2, err)
	}
	if _, err := store.db.Exec(`UPDATE libraries SET status='offline',revision=revision+1 WHERE id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), face.PersonAssetRequest{PersonID: "person_projection_1", Limit: 1}); !errors.Is(err, face.ErrFaceNotReady) {
		t.Fatalf("offline first page err=%v", err)
	}
	if _, err := service.List(context.Background(), face.PersonAssetRequest{PersonID: "person_projection_1", Limit: 1, Cursor: page1.NextCursor}); !errors.Is(err, face.ErrFaceNotReady) {
		t.Fatalf("offline cursor err=%v", err)
	}
	if _, err := store.db.Exec(`UPDATE libraries SET status='ready',revision=revision+1 WHERE id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), face.PersonAssetRequest{PersonID: "person_projection_1", Limit: 1, Cursor: page1.NextCursor}); !errors.Is(err, face.ErrFaceProjectionStale) {
		t.Fatalf("source revision stale err=%v", err)
	}
	fresh, err := service.List(context.Background(), face.PersonAssetRequest{PersonID: "person_projection_1", Limit: 1})
	if err != nil || fresh.NextCursor == "" {
		t.Fatalf("fresh=%+v err=%v", fresh, err)
	}
	if _, err := store.RenamePerson(context.Background(), face.RenamePersonCommand{ID: "person_projection_1", Name: "变更", ExpectedRevision: revision, UpdatedAt: now.Add(5 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), face.PersonAssetRequest{PersonID: "person_projection_1", Limit: 1, Cursor: fresh.NextCursor}); !errors.Is(err, face.ErrFaceProjectionStale) {
		t.Fatalf("stale err=%v", err)
	}
}

func TestFaceClusterDetailUsesActiveBuildAndRejectsStaleCursor(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 18, 0, 0, 0, time.UTC)
	const generation = "face_generation_detail"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{
		GenerationID: generation,
		LibraryID:    libraryID,
		AssetID:      assetID,
		UpdatedAt:    now,
		Items: []face.ObservationItem{
			{ID: "face_detail_0001", Box: face.Box{X: .89, Y: .88, Width: .1, Height: .1}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
			{ID: "face_detail_0002", Box: face.Box{X: .2, Y: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceFaceClusters(context.Background(), face.ClusterBatch{GenerationID: generation, LibraryID: libraryID, UpdatedAt: now, Clusters: []face.Cluster{{ID: "face_cluster_detail", Role: "core", Members: []face.ClusterMember{{FaceID: "face_detail_0001", Role: "core", Confidence: 1}, {FaceID: "face_detail_0002", Role: "core", Confidence: .8}}}}}); err != nil {
		t.Fatal(err)
	}
	service, err := face.NewFaceClusterDetailService(store, []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.List(context.Background(), face.FaceClusterDetailRequest{LibraryID: libraryID, ClusterID: "face_cluster_detail", Limit: 1})
	if err != nil || len(page.Items) != 1 || page.NextCursor == "" || page.Cluster.MemberCount != 2 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	if page.Items[0].Region.XPercent+page.Items[0].Region.WidthPercent > 100 || page.Items[0].Region.YPercent+page.Items[0].Region.HeightPercent > 100 {
		t.Fatalf("region escaped bounds: %+v", page.Items[0].Region)
	}
	page2, err := service.List(context.Background(), face.FaceClusterDetailRequest{LibraryID: libraryID, ClusterID: "face_cluster_detail", Limit: 1, Cursor: page.NextCursor})
	if err != nil || len(page2.Items) != 1 || page2.NextCursor != "" {
		t.Fatalf("page2=%+v err=%v", page2, err)
	}
	if err := store.ReplaceFaceClusters(context.Background(), face.ClusterBatch{GenerationID: generation, LibraryID: libraryID, UpdatedAt: now.Add(time.Minute), Clusters: []face.Cluster{{ID: "face_cluster_detail", Role: "core", Members: []face.ClusterMember{{FaceID: "face_detail_0001", Role: "core", Confidence: 1}, {FaceID: "face_detail_0002", Role: "core", Confidence: .8}}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), face.FaceClusterDetailRequest{LibraryID: libraryID, ClusterID: "face_cluster_detail", Limit: 1, Cursor: page.NextCursor}); !errors.Is(err, face.ErrFaceProjectionStale) {
		t.Fatalf("stale err=%v", err)
	}
	if _, err := service.List(context.Background(), face.FaceClusterDetailRequest{LibraryID: libraryID, ClusterID: "face_cluster_detail", Limit: 1, Cursor: page.NextCursor + "tampered"}); !errors.Is(err, face.ErrInvalidFaceClusterCursor) {
		t.Fatalf("tamper err=%v", err)
	}
}
