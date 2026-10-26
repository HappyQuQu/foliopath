package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceDetachedReassignment(t *testing.T) {
	for _, scenario := range []string{"undo", "source_changed", "target_changed", "linked", "bound", "rollback", "target_revision", "direct", "direct_stale", "direct_wrong_source"} {
		t.Run(scenario, func(t *testing.T) {
			s, _ := openTestStore(t)
			ctx := context.Background()
			library := seedBrowseCatalog(t, s)
			asset := catalogAssetID(t, s, "photo-10.jpg")
			now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
			seedFaceGeneration(t, s, "generation_reassign", 2, "active", now)
			vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
			if err := s.ReplaceFaceObservations(ctx, face.ObservationBatch{GenerationID: "generation_reassign", LibraryID: library, AssetID: asset, UpdatedAt: now, Items: []face.ObservationItem{
				{ID: "face_reassign_one", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
				{ID: "face_reassign_two", Box: face.Box{X: .4, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now},
			}}); err != nil {
				t.Fatal(err)
			}
			for _, id := range []string{"person_source", "person_target"} {
				if _, err := s.CreatePerson(ctx, face.CreatePersonCommand{ID: id, Name: id, CreatedAt: now}); err != nil {
					t.Fatal(err)
				}
			}
			cmd := face.AssignFaceCommand{EventID: "event_initial", RequestHash: testFaceRequestHash, AnchorID: "anchor_initial", FaceID: "face_reassign_one", PersonID: "person_source", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now}
			if _, err := s.AssignFace(ctx, cmd); err != nil {
				t.Fatal(err)
			}
			if scenario == "linked" {
				other := cmd
				other.EventID, other.AnchorID, other.FaceID, other.PersonID = "event_otherface", "anchor_otherface", "face_reassign_two", "person_target"
				if _, err := s.AssignFace(ctx, other); err != nil {
					t.Fatal(err)
				}
				if _, err := s.CannotLinkFaces(ctx, face.CannotLinkCommand{EventID: "event_cannotlink", RequestHash: testFaceRequestHash, LeftFaceID: cmd.FaceID, RightFaceID: other.FaceID, ExpectedLeftRevision: 1, ExpectedRightRevision: 1, CreatedAt: now}); err != nil {
					t.Fatal(err)
				}
			}
			direct := scenario == "direct" || scenario == "direct_stale" || scenario == "direct_wrong_source"
			if scenario != "bound" && !direct {
				if _, err := s.SplitFace(ctx, face.SplitFaceCommand{EventID: "event_splitface", RequestHash: testFaceRequestHash, FaceID: cmd.FaceID, SourcePersonID: cmd.PersonID, ExpectedFaceRevision: 1, ExpectedSourceRevision: 2, CreatedAt: now}); err != nil {
					t.Fatal(err)
				}
			}
			beforeSource, _ := s.GetPerson(ctx, "person_source")
			beforeTarget, _ := s.GetPerson(ctx, "person_target")
			cmd.EventID, cmd.AnchorID, cmd.PersonID, cmd.ExpectedPersonRevision = "event_reassign", "anchor_unused", "person_target", beforeTarget.Revision
			if direct {
				cmd.SourcePersonID, cmd.ExpectedSourceRevision = "person_source", beforeSource.Revision
				if scenario == "direct_stale" {
					cmd.ExpectedSourceRevision++
				}
				if scenario == "direct_wrong_source" {
					cmd.SourcePersonID = "person_wrong"
				}
			}
			if scenario == "target_revision" {
				cmd.ExpectedPersonRevision++
			}
			if scenario == "rollback" {
				if _, err := s.db.Exec(`CREATE TRIGGER reject_reassign_audit BEFORE INSERT ON face_audit_events WHEN NEW.id='event_reassign' BEGIN SELECT RAISE(ABORT, 'test rollback'); END`); err != nil {
					t.Fatal(err)
				}
			}
			result, err := s.AssignFace(ctx, cmd)
			if scenario == "linked" || scenario == "bound" || scenario == "rollback" || scenario == "target_revision" || scenario == "direct_stale" || scenario == "direct_wrong_source" {
				if err == nil {
					t.Fatal("unsafe reassignment succeeded")
				}
				if scenario != "rollback" && !errors.Is(err, face.ErrReviewConflict) {
					t.Fatal(err)
				}
				afterSource, _ := s.GetPerson(ctx, "person_source")
				afterTarget, _ := s.GetPerson(ctx, "person_target")
				if afterSource.Revision != beforeSource.Revision || afterTarget.Revision != beforeTarget.Revision || afterSource.ConfirmedFaceCount != beforeSource.ConfirmedFaceCount || afterTarget.ConfirmedFaceCount != beforeTarget.ConfirmedFaceCount {
					t.Fatal("failed transaction mutated people")
				}
				var owner string
				if err := s.db.QueryRow(`SELECT person_id FROM person_face_anchors WHERE id='anchor_initial'`).Scan(&owner); err != nil || owner != "person_source" {
					t.Fatalf("owner=%s err=%v", owner, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			replayed, err := s.AssignFace(ctx, cmd)
			if err != nil || !replayed.Replayed || replayed.Revision != result.Revision {
				t.Fatalf("replay=%+v %v", replayed, err)
			}
			if len(replayed.AffectedPersonIDs) != 2 || replayed.AffectedPersonIDs[1] != "person_source" {
				t.Fatalf("replay affected=%v", replayed.AffectedPersonIDs)
			}
			source, _ := s.GetPerson(ctx, "person_source")
			target, _ := s.GetPerson(ctx, "person_target")
			if source.ConfirmedFaceCount != 0 || target.ConfirmedFaceCount != 1 || source.Revision != beforeSource.Revision+1 || target.Revision != beforeTarget.Revision+1 {
				t.Fatalf("source=%+v target=%+v", source, target)
			}
			if scenario == "source_changed" || scenario == "target_changed" {
				changed := source
				if scenario == "target_changed" {
					changed = target
				}
				if _, err := s.RenamePerson(ctx, face.RenamePersonCommand{ID: changed.ID, Name: "new", ExpectedRevision: changed.Revision, UpdatedAt: now}); err != nil {
					t.Fatal(err)
				}
			}
			_, err = s.UndoFaceReview(ctx, face.UndoReviewCommand{EventID: "event_undo_reassign", RequestHash: testFaceRequestHash, ReviewID: result.EventID, ExpectedRevision: result.Revision, CreatedAt: now})
			if scenario != "undo" && scenario != "direct" {
				if !errors.Is(err, face.ErrReviewConflict) {
					t.Fatalf("unguarded undo: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var owner, state string
			var bound bool
			wantState := "needs_review"
			if direct {
				wantState = "bound"
			}
			if err := s.db.QueryRow(`SELECT person_id,state,current_face_id IS NOT NULL FROM person_face_anchors WHERE id='anchor_initial'`).Scan(&owner, &state, &bound); err != nil || owner != "person_source" || state != wantState || bound != direct {
				t.Fatalf("undo owner=%s state=%s bound=%v err=%v", owner, state, bound, err)
			}
		})
	}
}
