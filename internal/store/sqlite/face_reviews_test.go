package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

const testFaceRequestHash = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"

func TestFaceReviewStatePreservesStorageFailure(t *testing.T) {
	storageErr := errors.New("storage failure")
	if err := requireFaceReviewState(storageErr, false); !errors.Is(err, storageErr) {
		t.Fatalf("storage failure=%v", err)
	}
	if err := requireFaceReviewState(sql.ErrNoRows, true); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("missing row=%v", err)
	}
	if err := requireFaceReviewState(nil, false); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("state mismatch=%v", err)
	}
}

func TestFaceReviewAssignSplitExcludeTransactions(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 20, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_review", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_review", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now,
		Items: []face.ObservationItem{{ID: "face_review_0001", Box: face.Box{X: .1, Y: .1, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_review_01", Name: "测试", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	assignCommand := face.AssignFaceCommand{EventID: "event_assign_001", RequestHash: testFaceRequestHash, AnchorID: "anchor_review_001", FaceID: "face_review_0001", PersonID: "person_review_01", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Minute)}
	assigned, err := store.AssignFace(context.Background(), assignCommand)
	if err != nil || assigned.Revision != 2 {
		t.Fatalf("assigned=%+v err=%v", assigned, err)
	}
	replayed, err := store.AssignFace(context.Background(), assignCommand)
	if err != nil || !replayed.Replayed || replayed.Revision != 2 {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
	conflicting := assignCommand
	conflicting.RequestHash = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	if _, err := store.AssignFace(context.Background(), conflicting); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("idempotency conflict err=%v", err)
	}
	person, err := store.GetPerson(context.Background(), "person_review_01")
	if err != nil || person.ConfirmedFaceCount != 1 || person.AssetCount != 1 {
		t.Fatalf("person=%+v err=%v", person, err)
	}
	if _, err := store.SplitFace(context.Background(), face.SplitFaceCommand{EventID: "event_split_0001", RequestHash: testFaceRequestHash, FaceID: "face_review_0001", SourcePersonID: "person_review_01", ExpectedFaceRevision: 1, ExpectedSourceRevision: 2, CreatedAt: now.Add(2 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	person, _ = store.GetPerson(context.Background(), "person_review_01")
	if person.ConfirmedFaceCount != 0 || person.Revision != 3 {
		t.Fatalf("empty person=%+v", person)
	}
	if _, err := store.ExcludeFace(context.Background(), face.ExcludeFaceCommand{EventID: "event_exclude_01", RequestHash: testFaceRequestHash, ExclusionID: "exclude_review_1", FaceID: "face_review_0001", ExpectedFaceRevision: 1, CreatedAt: now.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	var exclusions int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_exclusions WHERE current_face_id='face_review_0001'`).Scan(&exclusions); err != nil || exclusions != 1 {
		t.Fatalf("exclusions=%d err=%v", exclusions, err)
	}
	if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: "event_reassign_01", RequestHash: testFaceRequestHash, AnchorID: "anchor_unused_001", FaceID: "face_review_0001", PersonID: "person_review_01", ExpectedFaceRevision: 1, ExpectedPersonRevision: 3, CreatedAt: now.Add(4 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_exclusions`).Scan(&exclusions); err != nil || exclusions != 0 {
		t.Fatalf("exclusions=%d err=%v", exclusions, err)
	}
}

func TestCannotLinkPreventsPeopleMergeAtomically(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 21, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_merge", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	batch := face.ObservationBatch{GenerationID: "face_generation_merge", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now,
		Items: []face.ObservationItem{
			{ID: "face_merge_0001", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
			{ID: "face_merge_0002", Box: face.Box{X: .4, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
		}}
	if err := store.ReplaceFaceObservations(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ id, name string }{{"person_merge_01", "A"}, {"person_merge_02", "B"}} {
		if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: item.id, Name: item.name, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct{ event, anchor, faceID, person string }{{"event_merge_a01", "anchor_merge_01", "face_merge_0001", "person_merge_01"}, {"event_merge_a02", "anchor_merge_02", "face_merge_0002", "person_merge_02"}} {
		if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: item.event, RequestHash: testFaceRequestHash, AnchorID: item.anchor, FaceID: item.faceID, PersonID: item.person, ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CannotLinkFaces(context.Background(), face.CannotLinkCommand{EventID: "event_cannot_01", RequestHash: testFaceRequestHash, LeftFaceID: "face_merge_0001", RightFaceID: "face_merge_0002", ExpectedLeftRevision: 1, ExpectedRightRevision: 1, CreatedAt: now.Add(2 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	_, err := store.MergePeople(context.Background(), face.MergePeopleCommand{EventID: "event_merge_fail", RequestHash: testFaceRequestHash, SourcePersonID: "person_merge_01", TargetPersonID: "person_merge_02", ExpectedSourceRevision: 2, ExpectedTargetRevision: 2, ConflictsAcknowledged: true, CreatedAt: now.Add(3 * time.Minute)})
	if !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("err=%v", err)
	}
	first, _ := store.GetPerson(context.Background(), "person_merge_01")
	second, _ := store.GetPerson(context.Background(), "person_merge_02")
	if first.Revision != 2 || second.Revision != 2 || first.ConfirmedFaceCount != 1 || second.ConfirmedFaceCount != 1 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}

func TestFaceAnchorReconciliationPreservesManualAssignmentAcrossGeneration(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 22, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_old", 2, "active", now)
	seedFaceGeneration(t, store, "face_generation_new", 2, "ready", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	oldBatch := face.ObservationBatch{GenerationID: "face_generation_old", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_upgrade_old", Box: face.Box{X: .1, Y: .1, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-stable", Vector: vector, CreatedAt: now}}}
	if err := store.ReplaceFaceObservations(context.Background(), oldBatch); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_upgrade_1", Name: "保留", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: "event_upgrade_01", RequestHash: testFaceRequestHash, AnchorID: "anchor_upgrade_1", FaceID: "face_upgrade_old", PersonID: "person_upgrade_1", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	newBatch := oldBatch
	newBatch.GenerationID = "face_generation_new"
	newBatch.UpdatedAt = now.Add(2 * time.Minute)
	newBatch.Items = []face.ObservationItem{{ID: "face_upgrade_new", Box: oldBatch.Items[0].Box, Detection: 1, Quality: 1, SourceFingerprint: "source-stable", Vector: vector, CreatedAt: now.Add(2 * time.Minute)}}
	if err := store.ReplaceFaceObservations(context.Background(), newBatch); err != nil {
		t.Fatal(err)
	}
	result, err := store.ReconcileFaceAnchors(context.Background(), "face_generation_new", libraryID, now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if result.Bound != 1 || result.NeedsReview != 0 {
		t.Fatalf("result=%+v", result)
	}
	var personID, state, currentFace string
	if err := store.db.QueryRow(`SELECT person_id,state,current_face_id FROM person_face_anchors WHERE id='anchor_upgrade_1'`).Scan(&personID, &state, &currentFace); err != nil {
		t.Fatal(err)
	}
	if personID != "person_upgrade_1" || state != "bound" || currentFace != "face_upgrade_new" {
		t.Fatalf("person=%s state=%s face=%s", personID, state, currentFace)
	}
	var cannotLinks int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_cannot_links`).Scan(&cannotLinks); err != nil {
		t.Fatal(err)
	}
}

func TestFaceReviewAndReconciliationClampTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	createdAt := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_clock_old", 2, "active", createdAt)
	seedFaceGeneration(t, store, "face_generation_clock_new", 2, "ready", createdAt)
	vector, err := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	box := face.Box{Width: .2, Height: .2}
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{
		GenerationID: "face_generation_clock_old", LibraryID: libraryID, AssetID: assetID, UpdatedAt: createdAt,
		Items: []face.ObservationItem{{ID: "face_review_clock_old", Box: box, Detection: 1, Quality: 1,
			SourceFingerprint: "source-clock", Vector: vector, CreatedAt: createdAt}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{
		ID: "person_review_clock", Name: "回拨复核", CreatedAt: createdAt,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{
		EventID: "event_clock_assign", RequestHash: testFaceRequestHash, AnchorID: "anchor_review_clock",
		FaceID: "face_review_clock_old", PersonID: "person_review_clock", ExpectedFaceRevision: 1,
		ExpectedPersonRevision: 1, CreatedAt: createdAt.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	split, err := store.SplitFace(context.Background(), face.SplitFaceCommand{
		EventID: "event_clock_split", RequestHash: testFaceRequestHash, FaceID: "face_review_clock_old",
		SourcePersonID: "person_review_clock", ExpectedFaceRevision: 1, ExpectedSourceRevision: 2,
		CreatedAt: createdAt.Add(-2 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UndoFaceReview(context.Background(), face.UndoReviewCommand{
		EventID: "event_clock_undo", RequestHash: testFaceRequestHash, ReviewID: split.EventID,
		ExpectedRevision: split.Revision, CreatedAt: createdAt.Add(-3 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{
		GenerationID: "face_generation_clock_new", LibraryID: libraryID, AssetID: assetID,
		UpdatedAt: createdAt.Add(-4 * time.Hour), Items: []face.ObservationItem{{
			ID: "face_review_clock_new", Box: box, Detection: 1, Quality: 1,
			SourceFingerprint: "source-clock", Vector: vector, CreatedAt: createdAt,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileFaceAnchors(context.Background(), "face_generation_clock_new", libraryID, createdAt.Add(-5*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		query string
		id    string
	}{
		{query: `SELECT created_at_ms,updated_at_ms FROM people WHERE id=?`, id: "person_review_clock"},
		{query: `SELECT created_at_ms,updated_at_ms FROM person_face_anchors WHERE id=?`, id: "anchor_review_clock"},
	} {
		var rowCreatedAt, rowUpdatedAt int64
		if err := store.db.QueryRow(check.query, check.id).Scan(&rowCreatedAt, &rowUpdatedAt); err != nil {
			t.Fatal(err)
		}
		if rowUpdatedAt < rowCreatedAt {
			t.Fatalf("row times=%d/%d for %q", rowCreatedAt, rowUpdatedAt, check.id)
		}
	}
}

func TestGuardedUndoRestoresSplitAndRejectsSecondUndo(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_undo", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_undo", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_undo_split_1", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_undo_01", Name: "撤销", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: "event_undo_assign", RequestHash: testFaceRequestHash, AnchorID: "anchor_undo_001", FaceID: "face_undo_split_1", PersonID: "person_undo_01", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	split, err := store.SplitFace(context.Background(), face.SplitFaceCommand{EventID: "event_undo_split", RequestHash: testFaceRequestHash, FaceID: "face_undo_split_1", SourcePersonID: "person_undo_01", ExpectedFaceRevision: 1, ExpectedSourceRevision: 2, CreatedAt: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	command := face.UndoReviewCommand{EventID: "event_undo_apply", RequestHash: testFaceRequestHash, ReviewID: split.EventID, ExpectedRevision: split.Revision, CreatedAt: now.Add(3 * time.Minute)}
	undone, err := store.UndoFaceReview(context.Background(), command)
	if err != nil || undone.Action != "undo" || undone.Revision != 4 {
		t.Fatalf("undone=%+v err=%v", undone, err)
	}
	person, err := store.GetPerson(context.Background(), "person_undo_01")
	if err != nil || person.Revision != 4 || person.ConfirmedFaceCount != 1 {
		t.Fatalf("person=%+v err=%v", person, err)
	}
	replay, err := store.UndoFaceReview(context.Background(), command)
	if err != nil || !replay.Replayed {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	second := command
	second.EventID = "event_undo_again"
	if _, err := store.UndoFaceReview(context.Background(), second); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("second undo err=%v", err)
	}
}

func TestGuardedUndoRemovesOnlyNewCannotLink(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_undo_link", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	items := []face.ObservationItem{{ID: "face_undo_link_1", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}, {ID: "face_undo_link_2", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_undo_link", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: items}); err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"person_undo_link_1", "person_undo_link_2"} {
		if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: id, Name: id, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: fmt.Sprintf("event_undo_link_assign_%d", i), RequestHash: testFaceRequestHash, AnchorID: fmt.Sprintf("anchor_undo_link_%d", i), FaceID: items[i].ID, PersonID: id, ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	linked, err := store.CannotLinkFaces(context.Background(), face.CannotLinkCommand{EventID: "event_undo_link", RequestHash: testFaceRequestHash, LeftFaceID: items[0].ID, RightFaceID: items[1].ID, ExpectedLeftRevision: 1, ExpectedRightRevision: 1, CreatedAt: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UndoFaceReview(context.Background(), face.UndoReviewCommand{EventID: "event_undo_link_apply", RequestHash: testFaceRequestHash, ReviewID: linked.EventID, ExpectedRevision: linked.Revision, CreatedAt: now.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	var links int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_cannot_links`).Scan(&links); err != nil || links != 0 {
		t.Fatalf("links=%d err=%v", links, err)
	}
}

func TestGuardedUndoAssignmentRestoresPriorExclusion(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_undo_assign", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_undo_assign", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_undo_assign_1", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ExcludeFace(context.Background(), face.ExcludeFaceCommand{EventID: "event_undo_prior_exclude", RequestHash: testFaceRequestHash, ExclusionID: "exclude_undo_prior", FaceID: "face_undo_assign_1", ExpectedFaceRevision: 1, CreatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_undo_assign", Name: "归类撤销", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	assigned, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: "event_undo_assignment", RequestHash: testFaceRequestHash, AnchorID: "anchor_undo_assign", FaceID: "face_undo_assign_1", PersonID: "person_undo_assign", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UndoFaceReview(context.Background(), face.UndoReviewCommand{EventID: "event_undo_assignment_apply", RequestHash: testFaceRequestHash, ReviewID: assigned.EventID, ExpectedRevision: assigned.Revision, CreatedAt: now.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	var exclusions, anchors int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_exclusions WHERE id='exclude_undo_prior' AND current_face_id='face_undo_assign_1'`).Scan(&exclusions); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM person_face_anchors WHERE id='anchor_undo_assign'`).Scan(&anchors); err != nil {
		t.Fatal(err)
	}
	person, err := store.GetPerson(context.Background(), "person_undo_assign")
	if err != nil || exclusions != 1 || anchors != 0 || person.Revision != 3 || person.ConfirmedFaceCount != 0 {
		t.Fatalf("exclusions=%d anchors=%d person=%+v err=%v", exclusions, anchors, person, err)
	}
}

func TestGuardedUndoExclusionRestoresPriorAssignment(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 13, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_undo_exclude", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_undo_exclude", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_undo_exclude_1", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_undo_exclude", Name: "排除撤销", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: "event_undo_exclude_assign", RequestHash: testFaceRequestHash, AnchorID: "anchor_undo_exclude", FaceID: "face_undo_exclude_1", PersonID: "person_undo_exclude", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	excluded, err := store.ExcludeFace(context.Background(), face.ExcludeFaceCommand{EventID: "event_undo_exclusion", RequestHash: testFaceRequestHash, ExclusionID: "exclude_undo_action", FaceID: "face_undo_exclude_1", ExpectedFaceRevision: 1, CreatedAt: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UndoFaceReview(context.Background(), face.UndoReviewCommand{EventID: "event_undo_exclusion_apply", RequestHash: testFaceRequestHash, ReviewID: excluded.EventID, ExpectedRevision: excluded.Revision, CreatedAt: now.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	var anchors, exclusions int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM person_face_anchors WHERE id='anchor_undo_exclude' AND current_face_id='face_undo_exclude_1' AND state='bound'`).Scan(&anchors); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_exclusions WHERE id='exclude_undo_action'`).Scan(&exclusions); err != nil {
		t.Fatal(err)
	}
	person, err := store.GetPerson(context.Background(), "person_undo_exclude")
	if err != nil || anchors != 1 || exclusions != 0 || person.Revision != 4 || person.ConfirmedFaceCount != 1 {
		t.Fatalf("anchors=%d exclusions=%d person=%+v err=%v", anchors, exclusions, person, err)
	}
}

func TestGuardedUndoRestoresAtomicPeopleMerge(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 14, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_undo_merge", 2, "active", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	items := []face.ObservationItem{{ID: "face_undo_merge_1", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}, {ID: "face_undo_merge_2", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_undo_merge", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: items}); err != nil {
		t.Fatal(err)
	}
	people := []string{"person_undo_merge_1", "person_undo_merge_2"}
	for i, id := range people {
		if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: id, Name: id, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: fmt.Sprintf("event_undo_merge_assign_%d", i), RequestHash: testFaceRequestHash, AnchorID: fmt.Sprintf("anchor_undo_merge_%d", i), FaceID: items[i].ID, PersonID: id, ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	merged, err := store.MergePeople(context.Background(), face.MergePeopleCommand{EventID: "event_undo_merge", RequestHash: testFaceRequestHash, SourcePersonID: people[0], TargetPersonID: people[1], ExpectedSourceRevision: 2, ExpectedTargetRevision: 2, ConflictsAcknowledged: true, CreatedAt: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UndoFaceReview(context.Background(), face.UndoReviewCommand{EventID: "event_undo_merge_apply", RequestHash: testFaceRequestHash, ReviewID: merged.EventID, ExpectedRevision: merged.Revision, CreatedAt: now.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	for _, id := range people {
		person, err := store.GetPerson(context.Background(), id)
		if err != nil || person.Revision != 4 || person.ConfirmedFaceCount != 1 {
			t.Fatalf("person=%s value=%+v err=%v", id, person, err)
		}
	}
	var aliases int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM person_aliases WHERE source_person_id=?`, people[0]).Scan(&aliases); err != nil || aliases != 0 {
		t.Fatalf("aliases=%d err=%v", aliases, err)
	}
}
