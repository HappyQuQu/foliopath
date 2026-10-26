package sqlite

import (
	"context"
	"errors"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestPersonMergeCheckSharesAuthoritativeConflictAndDoesNotWrite(t *testing.T) {
	s, first, _, now, command := seedCannotLinkScene(t)
	ctx := context.Background()
	before, err := s.GetPersonMergeCheck(ctx, command.SourcePersonID, command.TargetPersonID)
	if err != nil || !before.CanMerge() || len(before.Examples) != 0 || before.SourceRevision != 2 || before.TargetRevision != 2 {
		t.Fatalf("before=%+v %v", before, err)
	}
	linked, err := s.CannotLinkFaces(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	// The prior allowed preview does not authorize a later merge after a new link.
	_, err = s.MergePeople(ctx, face.MergePeopleCommand{EventID: "event_preview_race", RequestHash: testFaceRequestHash, SourcePersonID: before.SourcePersonID, TargetPersonID: before.TargetPersonID, ExpectedSourceRevision: before.SourceRevision, ExpectedTargetRevision: before.TargetRevision, ConflictsAcknowledged: true, CreatedAt: now})
	if !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("preview bypassed constraint %v", err)
	}
	for _, pair := range [][2]string{{command.SourcePersonID, command.TargetPersonID}, {command.TargetPersonID, command.SourcePersonID}} {
		check, err := s.GetPersonMergeCheck(ctx, pair[0], pair[1])
		if err != nil || check.CanMerge() || check.CannotLinkCount != 1 || check.DuplicateAssignmentCount != 0 || check.SourceLimitExceeded || len(check.Examples) != 1 || check.Examples[0].SourceAssetID != first || check.Examples[0].TargetAssetID != first {
			t.Fatalf("check=%+v %v", check, err)
		}
	}
	var events int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM face_audit_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 4 {
		t.Fatalf("preview wrote or failed merge mutated audits: %d", events)
	}
	if _, err := s.UndoFaceReview(ctx, face.UndoReviewCommand{EventID: "event_preview_undo", RequestHash: testFaceRequestHash, ReviewID: linked.EventID, ExpectedRevision: linked.Revision, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	after, err := s.GetPersonMergeCheck(ctx, command.SourcePersonID, command.TargetPersonID)
	if err != nil || !after.CanMerge() || after.SourceRevision != 2 || after.TargetRevision != 2 {
		t.Fatalf("after=%+v %v", after, err)
	}
}
func TestPersonMergeCheckHidesOfflineAndDetachedExamplesWithoutIgnoringConstraint(t *testing.T) {
	s, first, second, _, command := seedCannotLinkScene(t)
	ctx := context.Background()
	command.RightFaceID = "face_link_guard_2"
	command.TargetPersonID = "person_link_guard_c"
	if _, err := s.CannotLinkFaces(ctx, command); err != nil {
		t.Fatal(err)
	}
	value, err := s.GetPersonMergeCheck(ctx, command.SourcePersonID, command.TargetPersonID)
	if err != nil || len(value.Examples) != 1 || value.Examples[0].SourceAssetID != first || value.Examples[0].TargetAssetID != second {
		t.Fatalf("cross=%+v %v", value, err)
	}
	if _, err := s.db.Exec(`UPDATE libraries SET status='offline' WHERE id=?`, value.Examples[0].SourceLibraryID); err != nil {
		t.Fatal(err)
	}
	value, err = s.GetPersonMergeCheck(ctx, command.SourcePersonID, command.TargetPersonID)
	if err != nil || value.CannotLinkCount != 1 || value.CanMerge() || len(value.Examples) != 0 {
		t.Fatalf("offline=%+v %v", value, err)
	}
	if _, err := s.db.Exec(`UPDATE libraries SET status='ready'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE person_face_anchors SET state='needs_review',current_face_id=NULL WHERE person_id=?`, command.SourcePersonID); err != nil {
		t.Fatal(err)
	}
	value, err = s.GetPersonMergeCheck(ctx, command.SourcePersonID, command.TargetPersonID)
	if err != nil || value.CannotLinkCount != 1 || value.CanMerge() || len(value.Examples) != 0 {
		t.Fatalf("detached=%+v %v", value, err)
	}
}
func TestPersonMergeCheckSourceLimitAndInvalidPeople(t *testing.T) {
	s, _, _, _, command := seedCannotLinkScene(t)
	ctx := context.Background()
	s.maxBatchSize = 1
	if _, err := s.db.Exec(`UPDATE person_face_anchors SET person_id=? WHERE id='anchor_link_guard_2'`, command.SourcePersonID); err != nil {
		t.Fatal(err)
	}
	value, err := s.GetPersonMergeCheck(ctx, command.SourcePersonID, command.TargetPersonID)
	if err != nil || value.CanMerge() || !value.SourceLimitExceeded || value.CannotLinkCount != 0 {
		t.Fatalf("limit=%+v %v", value, err)
	}
	for _, pair := range [][2]string{{"", "person_link_guard_b"}, {command.SourcePersonID, command.SourcePersonID}, {"short", command.TargetPersonID}} {
		if _, err := s.GetPersonMergeCheck(ctx, pair[0], pair[1]); !errors.Is(err, face.ErrInvalidPerson) {
			t.Fatalf("invalid %v %v", pair, err)
		}
	}
	if _, err := s.GetPersonMergeCheck(ctx, command.SourcePersonID, "person_missing"); !errors.Is(err, face.ErrPersonNotFound) {
		t.Fatalf("missing=%v", err)
	}
}

func TestPersonMergeCheckBoundsDistinctExamplesAcrossLibraries(t *testing.T) {
	s, _ := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	service, _ := library.NewService(s)
	sourceLib, err := service.Create(ctx, "Example source", "source")
	if err != nil {
		t.Fatal(err)
	}
	targetLib, err := service.Create(ctx, "Example target", "target")
	if err != nil {
		t.Fatal(err)
	}
	for _, lib := range []int64{sourceLib.ID, targetLib.ID} {
		run, err := s.BeginFullScan(ctx, lib, scanner.TriggerManual)
		if err != nil {
			t.Fatal(err)
		}
		entries := []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}}
		for n := 0; n < 5; n++ {
			name := fmt.Sprintf("example-%d.jpg", n)
			entries = append(entries, scanner.CatalogEntry{Kind: scanner.CatalogEntryAsset, RelativePath: name, Name: name, AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg", SizeBytes: 10, MTimeNS: int64(n + 1)})
		}
		if err := s.UpsertCatalogBatch(ctx, run.ID, entries); err != nil {
			t.Fatal(err)
		}
		if _, err := s.CompleteFullScan(ctx, run.ID, scanner.SkipCounts{}); err != nil {
			t.Fatal(err)
		}
	}
	seedFaceGeneration(t, s, "generation_merge_examples", 2, "active", now)
	for _, id := range []string{"person_example_source", "person_example_target"} {
		if _, err := s.CreatePerson(ctx, face.CreatePersonCommand{ID: id, Name: id, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	for n := 0; n < 5; n++ {
		ids := []string{fmt.Sprintf("face_example_s_%d", n), fmt.Sprintf("face_example_t_%d", n)}
		for side, lib := range []int64{sourceLib.ID, targetLib.ID} {
			var asset int64
			if err := s.db.QueryRow(`SELECT id FROM assets WHERE library_id=? AND relative_path=?`, lib, fmt.Sprintf("example-%d.jpg", n)).Scan(&asset); err != nil {
				t.Fatal(err)
			}
			observation := face.ObservationItem{ID: ids[side], Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "synthetic-source", Vector: vector, CreatedAt: now}
			if err := s.ReplaceFaceObservations(ctx, face.ObservationBatch{GenerationID: "generation_merge_examples", LibraryID: lib, AssetID: asset, UpdatedAt: now, Items: []face.ObservationItem{observation}}); err != nil {
				t.Fatal(err)
			}
			personID := []string{"person_example_source", "person_example_target"}[side]
			owner, err := s.GetPerson(ctx, personID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.AssignFace(ctx, face.AssignFaceCommand{EventID: fmt.Sprintf("event_example_%d_%d", n, side), RequestHash: testFaceRequestHash, AnchorID: fmt.Sprintf("anchor_example_%d_%d", n, side), FaceID: ids[side], PersonID: personID, ExpectedFaceRevision: 1, ExpectedPersonRevision: owner.Revision, CreatedAt: now}); err != nil {
				t.Fatal(err)
			}
		}
		source, _ := s.GetPerson(ctx, "person_example_source")
		target, _ := s.GetPerson(ctx, "person_example_target")
		if _, err := s.CannotLinkFaces(ctx, face.CannotLinkCommand{EventID: fmt.Sprintf("event_example_link_%d", n), RequestHash: testFaceRequestHash, LeftFaceID: ids[0], RightFaceID: ids[1], SourcePersonID: source.ID, TargetPersonID: target.ID, ExpectedLeftRevision: 1, ExpectedRightRevision: 1, ExpectedSourceRevision: source.Revision, ExpectedTargetRevision: target.Revision, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	value, err := s.GetPersonMergeCheck(ctx, "person_example_source", "person_example_target")
	if err != nil || value.CannotLinkCount != 5 || len(value.Examples) != face.MaxMergeConflictExamples {
		t.Fatalf("examples=%+v %v", value, err)
	}
	for _, example := range value.Examples {
		if example.SourceLibraryID != sourceLib.ID || example.TargetLibraryID != targetLib.ID {
			t.Fatalf("wrong scope=%+v", example)
		}
	}
	reversed, err := s.GetPersonMergeCheck(ctx, "person_example_target", "person_example_source")
	if err != nil || reversed.CannotLinkCount != 5 || len(reversed.Examples) != 3 || reversed.Examples[0].SourceLibraryID != targetLib.ID {
		t.Fatalf("reversed=%+v %v", reversed, err)
	}
}
