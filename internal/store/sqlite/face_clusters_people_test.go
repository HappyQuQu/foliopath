package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestReplaceAndIncrementFaceClusters(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 18, 30, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_clusters", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	observations := face.ObservationBatch{GenerationID: "face_generation_clusters", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now,
		Items: []face.ObservationItem{
			{ID: "face_cluster_001", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
			{ID: "face_cluster_002", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		}}
	if err := store.ReplaceFaceObservations(context.Background(), observations); err != nil {
		t.Fatal(err)
	}
	batch := face.ClusterBatch{GenerationID: observations.GenerationID, LibraryID: libraryID, UpdatedAt: now,
		Clusters: []face.Cluster{{ID: "face_group_0001", Role: "core", Members: []face.ClusterMember{
			{FaceID: "face_cluster_001", Role: "core", Confidence: 1}, {FaceID: "face_cluster_002", Role: "core", Confidence: .99},
		}}}}
	if err := store.ReplaceFaceClusters(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListFaceClusters(context.Background(), observations.GenerationID, libraryID)
	if err != nil || len(items) != 1 || len(items[0].Members) != 2 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	batch.UpdatedAt = now.Add(time.Minute)
	if err := store.UpsertFaceClusters(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	var revision int
	if err := store.db.QueryRow(`SELECT revision FROM face_clusters WHERE id='face_group_0001'`).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if revision != 2 {
		t.Fatalf("revision=%d", revision)
	}
	batch.Clusters = nil
	if err := store.ReplaceFaceClusters(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListFaceClusters(context.Background(), observations.GenerationID, libraryID)
	if err != nil || len(items) != 0 {
		t.Fatalf("remaining=%+v err=%v", items, err)
	}
}

func TestFaceClusterActivationClampsSettingsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	createdAt := time.Date(2026, 8, 31, 18, 30, 0, 0, time.UTC)
	const generationID = "face_generation_cluster_clock"
	seedFaceGeneration(t, store, generationID, 2, "active", createdAt)
	seedFaceReadySettings(t, store, libraryID, generationID, createdAt)
	if err := store.ReplaceFaceClusters(context.Background(), face.ClusterBatch{
		GenerationID: generationID, LibraryID: libraryID, UpdatedAt: createdAt.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	var settingsCreatedAt, settingsUpdatedAt int64
	var buildID string
	if err := store.db.QueryRow(`SELECT created_at_ms,updated_at_ms,active_cluster_build_id FROM face_library_settings WHERE library_id=?`, libraryID).
		Scan(&settingsCreatedAt, &settingsUpdatedAt, &buildID); err != nil {
		t.Fatal(err)
	}
	if settingsUpdatedAt < settingsCreatedAt || buildID == "" {
		t.Fatalf("settings times/build=%d/%d/%q", settingsCreatedAt, settingsUpdatedAt, buildID)
	}
	var buildCreatedAt, activatedAt int64
	if err := store.db.QueryRow(`SELECT created_at_ms,activated_at_ms FROM face_cluster_builds WHERE id=?`, buildID).
		Scan(&buildCreatedAt, &activatedAt); err != nil {
		t.Fatal(err)
	}
	if activatedAt < buildCreatedAt {
		t.Fatalf("build times=%d/%d", buildCreatedAt, activatedAt)
	}
}

func TestRebuildFaceClustersDoesNotMisclassifyStorageFailure(t *testing.T) {
	store, _ := openTestStore(t)
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	err := store.RebuildFaceClusters(context.Background(), "face_generation_storage_failure", 1,
		"face_job_storage_failure", 1, face.ClusterProfile{}, time.Now().UTC())
	if err == nil || errors.Is(err, face.ErrFaceGenerationUnavailable) {
		t.Fatalf("storage failure=%v", err)
	}
}

func TestFaceClusterBuildIsInvisibleUntilAtomicActivation(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 18, 45, 0, 0, time.UTC)
	const generation = "face_generation_atomic"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	observations := face.ObservationBatch{GenerationID: generation, LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{
		{ID: "face_atomic_001", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		{ID: "face_atomic_002", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
	}}
	if err := store.ReplaceFaceObservations(context.Background(), observations); err != nil {
		t.Fatal(err)
	}
	oldBatch := face.ClusterBatch{GenerationID: generation, LibraryID: libraryID, UpdatedAt: now, Clusters: []face.Cluster{{ID: "face_atomic_old", Role: "core", Members: []face.ClusterMember{{FaceID: "face_atomic_001", Role: "core", Confidence: 1}, {FaceID: "face_atomic_002", Role: "core", Confidence: 1}}}}}
	if err := store.ReplaceFaceClusters(context.Background(), oldBatch); err != nil {
		t.Fatal(err)
	}
	const buildID = "facebuild_atomic_pending"
	if _, err := store.db.Exec(`INSERT INTO face_cluster_builds(id,generation_id,library_id,state,created_at_ms) VALUES(?,?,?,'building',?)`, buildID, generation, libraryID, now.Add(time.Minute).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	newBatch := face.ClusterBatch{GenerationID: generation, LibraryID: libraryID, UpdatedAt: now.Add(time.Minute), Clusters: []face.Cluster{
		{ID: "face_atomic_new_1", Role: "edge", Members: []face.ClusterMember{{FaceID: "face_atomic_001", Role: "edge", Confidence: 0}}},
		{ID: "face_atomic_new_2", Role: "edge", Members: []face.ClusterMember{{FaceID: "face_atomic_002", Role: "edge", Confidence: 0}}},
	}}
	if err := store.writeFaceClustersToBuild(context.Background(), buildID, newBatch); err != nil {
		t.Fatal(err)
	}
	visible, err := store.ListFaceClusters(context.Background(), generation, libraryID)
	if err != nil || len(visible) != 1 || visible[0].ID != "face_atomic_old" {
		t.Fatalf("before=%+v err=%v", visible, err)
	}
	if err := store.withWriteTx(context.Background(), func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE face_cluster_builds SET state='stale' WHERE library_id=? AND state='active'`, libraryID); err != nil {
			return err
		}
		_, err := tx.Exec(`UPDATE face_cluster_builds SET state='active',activated_at_ms=? WHERE id=?`, now.Add(time.Minute).UnixMilli(), buildID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	visible, err = store.ListFaceClusters(context.Background(), generation, libraryID)
	if err != nil || len(visible) != 2 || visible[0].ID != "face_atomic_new_1" {
		t.Fatalf("after=%+v err=%v", visible, err)
	}
}

func TestRunnableFaceClusterActivationFailsClosedWhenLibraryStops(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 2, 17, 0, 0, 0, time.UTC)
	const generation = "face_generation_activation_guard"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{
		GenerationID: generation, LibraryID: libraryID, AssetID: assetID, UpdatedAt: now,
		Items: []face.ObservationItem{
			{ID: "face_activation_guard_1", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
			{ID: "face_activation_guard_2", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		},
	}); err != nil {
		t.Fatal(err)
	}
	oldBatch := face.ClusterBatch{GenerationID: generation, LibraryID: libraryID, UpdatedAt: now, Clusters: []face.Cluster{{
		ID: "face_activation_old", Role: "core", Members: []face.ClusterMember{
			{FaceID: "face_activation_guard_1", Role: "core", Confidence: 1},
			{FaceID: "face_activation_guard_2", Role: "core", Confidence: 1},
		},
	}}}
	if err := store.ReplaceFaceClusters(context.Background(), oldBatch); err != nil {
		t.Fatal(err)
	}
	newBatch := oldBatch
	newBatch.UpdatedAt = now.Add(time.Minute)
	newBatch.Clusters = []face.Cluster{{ID: "face_activation_new", Role: "edge", Members: []face.ClusterMember{{FaceID: "face_activation_guard_1", Role: "edge", Confidence: 0}}}}
	if _, err := store.db.Exec(`UPDATE libraries SET status='offline',revision=revision+1 WHERE id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	if err := store.replaceFaceClusters(context.Background(), newBatch, true, "", 0); !errors.Is(err, face.ErrFaceLibraryOffline) {
		t.Fatalf("expected offline activation rejection, got %v", err)
	}
	assertOnlyActiveFaceCluster(t, store, libraryID, "face_activation_old")

	if _, err := store.db.Exec(`UPDATE libraries SET status='ready',revision=revision+1 WHERE id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE face_library_settings SET enabled=0,state='disabled',revision=revision+1 WHERE library_id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	if err := store.replaceFaceClusters(context.Background(), newBatch, true, "", 0); !errors.Is(err, face.ErrFaceDisabled) {
		t.Fatalf("expected disabled activation rejection, got %v", err)
	}
	assertOnlyActiveFaceCluster(t, store, libraryID, "face_activation_old")
}

func assertOnlyActiveFaceCluster(t *testing.T, store *Store, libraryID int64, expected string) {
	t.Helper()
	var activeID string
	if err := store.db.QueryRow(`SELECT cluster.id FROM face_clusters cluster JOIN face_cluster_builds build ON build.id=cluster.build_id WHERE build.library_id=? AND build.state='active'`, libraryID).Scan(&activeID); err != nil {
		t.Fatal(err)
	}
	if activeID != expected {
		t.Fatalf("active cluster=%s, want %s", activeID, expected)
	}
	var inactiveBuilds int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_cluster_builds WHERE library_id=? AND state<>'active'`, libraryID).Scan(&inactiveBuilds); err != nil {
		t.Fatal(err)
	}
	if inactiveBuilds != 0 {
		t.Fatalf("orphan inactive builds=%d", inactiveBuilds)
	}
}

func TestPeopleAllowDuplicateNormalizedNamesAndEmptyPerson(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 31, 19, 0, 0, 0, time.UTC)
	first, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_duplicate_1", Name: "  Cafe\u0301  ", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_duplicate_2", Name: "Caf\u00e9", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Caf\u00e9" || second.Name != first.Name || first.ConfirmedFaceCount != 0 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	items, err := store.ListPeople(context.Background(), 10)
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	renamed, err := store.RenamePerson(context.Background(), face.RenamePersonCommand{ID: first.ID, Name: "新名字", ExpectedRevision: 1, UpdatedAt: now.Add(time.Minute)})
	if err != nil || renamed.Revision != 2 {
		t.Fatalf("renamed=%+v err=%v", renamed, err)
	}
	_, err = store.RenamePerson(context.Background(), face.RenamePersonCommand{ID: first.ID, Name: "冲突", ExpectedRevision: 1, UpdatedAt: now.Add(2 * time.Minute)})
	if !errors.Is(err, face.ErrPersonConflict) {
		t.Fatalf("err=%v", err)
	}
	if err := store.DeletePerson(context.Background(), face.DeletePersonCommand{ID: second.ID, ExpectedRevision: 1, DeletedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPerson(context.Background(), second.ID); !errors.Is(err, face.ErrPersonNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestPersonLifecycleClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	createdAt := time.Date(2026, 8, 31, 19, 0, 0, 0, time.UTC)
	person, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{
		ID: "person_clock_rollback", Name: "时钟回拨", CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	person, err = store.RenamePerson(context.Background(), face.RenamePersonCommand{
		ID: person.ID, Name: "回拨后改名", ExpectedRevision: person.Revision, UpdatedAt: createdAt.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !person.UpdatedAt.Equal(createdAt) {
		t.Fatalf("renamed updated at=%v, want %v", person.UpdatedAt, createdAt)
	}
	if err := store.DeletePerson(context.Background(), face.DeletePersonCommand{
		ID: person.ID, ExpectedRevision: person.Revision, DeletedAt: createdAt.Add(-2 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	var storedCreatedAt, storedUpdatedAt, tombstonedAt int64
	if err := store.db.QueryRow(`SELECT created_at_ms,updated_at_ms,tombstoned_at_ms FROM people WHERE id=?`, person.ID).
		Scan(&storedCreatedAt, &storedUpdatedAt, &tombstonedAt); err != nil {
		t.Fatal(err)
	}
	if storedUpdatedAt < storedCreatedAt || tombstonedAt < storedCreatedAt {
		t.Fatalf("person times=%d/%d/%d", storedCreatedAt, storedUpdatedAt, tombstonedAt)
	}
}

func TestCreatePersonFromCoreClusterIsSingleTransaction(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 19, 30, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_create", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	observations := face.ObservationBatch{GenerationID: "face_generation_create", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{
		{ID: "face_create_001", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		{ID: "face_create_002", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}
	if err := store.ReplaceFaceObservations(context.Background(), observations); err != nil {
		t.Fatal(err)
	}
	clusters := face.ClusterBatch{GenerationID: observations.GenerationID, LibraryID: libraryID, UpdatedAt: now, Clusters: []face.Cluster{{ID: "face_create_group", Role: "core", Members: []face.ClusterMember{{FaceID: "face_create_001", Role: "core", Confidence: 1}, {FaceID: "face_create_002", Role: "core", Confidence: 1}}}}}
	if err := store.ReplaceFaceClusters(context.Background(), clusters); err != nil {
		t.Fatal(err)
	}
	result, err := store.CreatePersonFromCluster(context.Background(), face.CreatePersonFromClusterCommand{EventID: "event_create_001", RequestHash: testFaceRequestHash, PersonID: "person_create_01", Name: "新人物", ClusterID: "face_create_group", AnchorIDs: []string{"anchor_create_01", "anchor_create_02"}, ExpectedClusterRevision: 1, CreatedAt: now.Add(time.Minute)})
	if err != nil || result.Revision != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	person, err := store.GetPerson(context.Background(), "person_create_01")
	if err != nil || person.Revision != 1 || person.ConfirmedFaceCount != 2 {
		t.Fatalf("person=%+v err=%v", person, err)
	}
	_, err = store.CreatePersonFromCluster(context.Background(), face.CreatePersonFromClusterCommand{EventID: "event_create_bad", RequestHash: testFaceRequestHash, PersonID: "person_create_02", Name: "失败人物", ClusterID: "face_create_group", AnchorIDs: []string{"anchor_bad_only"}, ExpectedClusterRevision: 1, CreatedAt: now.Add(2 * time.Minute)})
	if !errors.Is(err, face.ErrInvalidReview) {
		t.Fatalf("err=%v", err)
	}
	if _, err := store.GetPerson(context.Background(), "person_create_02"); !errors.Is(err, face.ErrPersonNotFound) {
		t.Fatalf("partial person err=%v", err)
	}
}

func TestPersonMutationServiceCreatesClusterAndEmptyPeopleIdempotently(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 2, 16, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_person_service", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_person_service", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{
		{ID: "face_person_service_1", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		{ID: "face_person_service_2", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceFaceClusters(context.Background(), face.ClusterBatch{GenerationID: "face_generation_person_service", LibraryID: libraryID, UpdatedAt: now, Clusters: []face.Cluster{{ID: "face_person_group_1", Role: "core", Members: []face.ClusterMember{{FaceID: "face_person_service_1", Role: "core", Confidence: 1}, {FaceID: "face_person_service_2", Role: "core", Confidence: 1}}}}}); err != nil {
		t.Fatal(err)
	}
	service, err := face.NewPersonMutationService(store, store, store, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	revision := int64(1)
	created, replayed, err := service.Create(context.Background(), "person-service-key-001", "  新人物  ", "face_person_group_1", &revision)
	if err != nil || replayed || created.Name != "新人物" || created.ConfirmedFaceCount != 2 {
		t.Fatalf("created=%+v replayed=%v err=%v", created, replayed, err)
	}
	again, replayed, err := service.Create(context.Background(), "person-service-key-001", "新人物", "face_person_group_1", &revision)
	if err != nil || !replayed || again.ID != created.ID {
		t.Fatalf("again=%+v replayed=%v err=%v", again, replayed, err)
	}
	empty, replayed, err := service.Create(context.Background(), "person-service-key-002", "空人物", "", nil)
	if err != nil || replayed || empty.ConfirmedFaceCount != 0 {
		t.Fatalf("empty=%+v replayed=%v err=%v", empty, replayed, err)
	}
	if _, replayed, err := service.Create(context.Background(), "person-service-key-002", "空人物", "", nil); err != nil || !replayed {
		t.Fatalf("empty replayed=%v err=%v", replayed, err)
	}
	renamed, err := service.Rename(context.Background(), empty.ID, "空人物改名", empty.Revision)
	if err != nil || renamed.Name != "空人物改名" || renamed.Revision != empty.Revision+1 {
		t.Fatalf("renamed=%+v err=%v", renamed, err)
	}
	if err := service.Delete(context.Background(), renamed.ID, renamed.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPerson(context.Background(), renamed.ID); !errors.Is(err, face.ErrPersonNotFound) {
		t.Fatalf("deleted person err=%v", err)
	}
}
