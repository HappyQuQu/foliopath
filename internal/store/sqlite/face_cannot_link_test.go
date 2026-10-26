package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func seedCannotLinkScene(t *testing.T) (*Store, int64, int64, time.Time, face.CannotLinkCommand) {
	t.Helper()
	s, _ := openTestStore(t)
	library := seedBrowseCatalog(t, s)
	first := catalogAssetID(t, s, "photo-10.jpg")
	second := catalogAssetID(t, s, "photo-2.jpg")
	now := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, s, "face_generation_link_guard", 2, "active", now)
	seedFaceReadySettings(t, s, library, "face_generation_link_guard", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	for i, asset := range []int64{first, second} {
		items := []face.ObservationItem{{ID: fmt.Sprintf("face_link_guard_%d", i*2), Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "synthetic-source", Vector: vector, CreatedAt: now}}
		if i == 0 {
			items = append(items, face.ObservationItem{ID: "face_link_guard_1", Box: face.Box{X: .4, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "synthetic-source", Vector: vector, CreatedAt: now})
		}
		if err := s.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_link_guard", LibraryID: library, AssetID: asset, UpdatedAt: now, Items: items}); err != nil {
			t.Fatal(err)
		}
	}
	for i, id := range []string{"person_link_guard_a", "person_link_guard_b", "person_link_guard_c"} {
		if _, err := s.CreatePerson(context.Background(), face.CreatePersonCommand{ID: id, Name: id, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.AssignFace(context.Background(), face.AssignFaceCommand{EventID: fmt.Sprintf("event_link_guard_assign_%d", i), RequestHash: testFaceRequestHash, AnchorID: fmt.Sprintf("anchor_link_guard_%d", i), FaceID: fmt.Sprintf("face_link_guard_%d", i), PersonID: id, ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	return s, first, second, now, face.CannotLinkCommand{EventID: "event_link_guard", RequestHash: testFaceRequestHash, LeftFaceID: "face_link_guard_0", RightFaceID: "face_link_guard_1", ExpectedLeftRevision: 1, ExpectedRightRevision: 1, SourcePersonID: "person_link_guard_a", TargetPersonID: "person_link_guard_b", ExpectedSourceRevision: 2, ExpectedTargetRevision: 2, CreatedAt: now}
}

func TestCannotLinkGuardsRejectStaleCurationWithoutWrites(t *testing.T) {
	s, _, _, now, command := seedCannotLinkScene(t)
	for i, change := range []func(*face.CannotLinkCommand){func(c *face.CannotLinkCommand) {
		c.SourcePersonID = ""
		c.TargetPersonID = ""
		c.ExpectedSourceRevision = 0
		c.ExpectedTargetRevision = 0
	}, func(c *face.CannotLinkCommand) { c.SourcePersonID = "person_link_guard_c" }, func(c *face.CannotLinkCommand) { c.TargetPersonID = "person_link_guard_c" }, func(c *face.CannotLinkCommand) { c.ExpectedSourceRevision = 1 }, func(c *face.CannotLinkCommand) { c.ExpectedTargetRevision = 1 }, func(c *face.CannotLinkCommand) { c.ExpectedLeftRevision = 2 }} {
		c := command
		c.EventID = fmt.Sprintf("event_link_guard_stale_%d", i)
		change(&c)
		if _, err := s.CannotLinkFaces(context.Background(), c); !errors.Is(err, face.ErrReviewConflict) {
			t.Fatalf("case %d err=%v", i, err)
		}
	}
	var links, audits int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM face_cannot_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM face_audit_events WHERE action='cannot_link'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if links != 0 || audits != 0 {
		t.Fatalf("stale requests wrote %d/%d", links, audits)
	}
	linked, err := s.CannotLinkFaces(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := command
	duplicate.EventID = "event_link_guard_duplicate"
	if _, err := s.CannotLinkFaces(context.Background(), duplicate); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("duplicate=%v", err)
	}
	if _, err := s.RenamePerson(context.Background(), face.RenamePersonCommand{ID: command.SourcePersonID, Name: "new synthetic name", ExpectedRevision: 2, UpdatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	replay, err := s.CannotLinkFaces(context.Background(), command)
	if err != nil || !replay.Replayed || replay.Revision != linked.Revision {
		t.Fatalf("replay=%+v %v", replay, err)
	}
	if _, err := s.UndoFaceReview(context.Background(), face.UndoReviewCommand{EventID: "event_link_guard_undo_stale", RequestHash: testFaceRequestHash, ReviewID: linked.EventID, ExpectedRevision: linked.Revision, CreatedAt: now.Add(2 * time.Minute)}); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("stale undo=%v", err)
	}
	if _, err := s.MergePeople(context.Background(), face.MergePeopleCommand{EventID: "event_link_guard_merge_blocked", RequestHash: testFaceRequestHash, SourcePersonID: command.SourcePersonID, TargetPersonID: command.TargetPersonID, ExpectedSourceRevision: 3, ExpectedTargetRevision: 2, ConflictsAcknowledged: true, CreatedAt: now}); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("merge=%v", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM face_cannot_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != 1 {
		t.Fatalf("constraint lost: %d", links)
	}
}

func TestCannotLinkProjectionIsSymmetricAndNeverExposesCrossPhotoPeers(t *testing.T) {
	s, first, second, now, command := seedCannotLinkScene(t)
	linked, err := s.CannotLinkFaces(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	cross := command
	cross.EventID = "event_link_guard_cross_photo"
	cross.RightFaceID = "face_link_guard_2"
	cross.TargetPersonID = "person_link_guard_c"
	if _, err := s.CannotLinkFaces(context.Background(), cross); err != nil {
		t.Fatal(err)
	}
	values, err := s.ListAssetFaceViews(context.Background(), first)
	if err != nil || len(values) != 2 {
		t.Fatalf("views=%+v %v", values, err)
	}
	want := map[string][]string{"face_link_guard_0": {"face_link_guard_1"}, "face_link_guard_1": {"face_link_guard_0"}}
	for _, v := range values {
		if !reflect.DeepEqual(v.CannotLinkFaceIDs, want[v.FaceID]) {
			t.Fatalf("peers=%+v", v)
		}
	}
	other, err := s.ListAssetFaceViews(context.Background(), second)
	if err != nil || len(other) != 1 || len(other[0].CannotLinkFaceIDs) != 0 || other[0].CannotLinkFaceIDs == nil {
		t.Fatalf("cross-photo leaked: %+v %v", other, err)
	}
	if _, err := s.UndoFaceReview(context.Background(), face.UndoReviewCommand{EventID: "event_link_guard_undo", RequestHash: testFaceRequestHash, ReviewID: linked.EventID, ExpectedRevision: linked.Revision, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	values, err = s.ListAssetFaceViews(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range values {
		if len(v.CannotLinkFaceIDs) != 0 {
			t.Fatalf("removed pair remains %+v", v)
		}
	}
	replay, err := s.CannotLinkFaces(context.Background(), command)
	if err != nil || !replay.Replayed {
		t.Fatalf("replayed old receipt=%+v %v", replay, err)
	}
	var links int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM face_cannot_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != 1 {
		t.Fatalf("same-key replay recreated undone pair: %d", links)
	}
}

func TestCannotLinkLegacyReceiptAndSnapshotRemainReplayable(t *testing.T) {
	s, _, _, now, command := seedCannotLinkScene(t)
	legacy := face.ReviewRequest{Action: "cannot_link", LeftFaceID: command.LeftFaceID, RightFaceID: command.RightFaceID, ExpectedLeftRevision: 1, ExpectedRightRevision: 1}
	key := "legacy-link-identity-001"
	command.EventID, command.RequestHash, _ = face.ReviewIdentity(key, legacy)
	linked, err := s.CannotLinkFaces(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	// Detached synthetic historical snapshot: no owner/anchor guards were recorded.
	var raw string
	if err := s.db.QueryRow(`SELECT after_json FROM face_review_undo_snapshots WHERE event_id=?`, linked.EventID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var payload faceUndoPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	payload.SourcePersonID = ""
	payload.TargetPersonID = ""
	payload.SourcePersonRevision = 0
	payload.TargetPersonRevision = 0
	payload.LinkAnchors = nil
	encoded, _ := json.Marshal(payload)
	if _, err := s.db.Exec(`UPDATE face_review_undo_snapshots SET before_json=?,after_json=? WHERE event_id=?`, string(encoded), string(encoded), linked.EventID); err != nil {
		t.Fatal(err)
	}
	service, _ := face.NewReviewService(s, s, func() time.Time { return now })
	replay, err := service.Review(context.Background(), key, legacy)
	if err != nil || !replay.Replayed {
		t.Fatalf("legacy=%+v %v", replay, err)
	}
	if _, err := service.Undo(context.Background(), "legacy-link-undo-001", linked.EventID, linked.Revision); err != nil {
		t.Fatal(err)
	}
}

func TestCannotLinkReverseOrderAndAnchorChangesGuardUndo(t *testing.T) {
	s, _, _, now, command := seedCannotLinkScene(t)
	command.LeftFaceID, command.RightFaceID = command.RightFaceID, command.LeftFaceID
	command.SourcePersonID, command.TargetPersonID = command.TargetPersonID, command.SourcePersonID
	linked, err := s.CannotLinkFaces(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UndoFaceReview(context.Background(), face.UndoReviewCommand{EventID: "event_link_reverse_undo", RequestHash: testFaceRequestHash, ReviewID: linked.EventID, ExpectedRevision: linked.Revision, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	command.EventID = "event_link_anchor_changed"
	linked, err = s.CannotLinkFaces(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic detector-rebind revision; people revisions alone cannot prove anchor continuity.
	if _, err := s.db.Exec(`UPDATE person_face_anchors SET revision=revision+1 WHERE id='anchor_link_guard_0'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UndoFaceReview(context.Background(), face.UndoReviewCommand{EventID: "event_link_anchor_changed_undo", RequestHash: testFaceRequestHash, ReviewID: linked.EventID, ExpectedRevision: linked.Revision, CreatedAt: now}); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("anchor change undo=%v", err)
	}
	var links int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM face_cannot_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != 1 {
		t.Fatalf("changed anchor link lost %d", links)
	}
}
