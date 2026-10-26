package face

import (
	"context"
	"errors"
	"testing"
	"time"
)

type reviewServiceRepositoryStub struct {
	assignFace    AssignFaceCommand
	assignCluster AssignClusterCommand
	undo          UndoReviewCommand
	exclude       ExcludeFaceCommand
	link          CannotLinkCommand
}

func (stub *reviewServiceRepositoryStub) CreatePersonFromCluster(context.Context, CreatePersonFromClusterCommand) (ReviewResult, error) {
	return ReviewResult{}, errors.New("unexpected create")
}
func (stub *reviewServiceRepositoryStub) AssignFace(_ context.Context, command AssignFaceCommand) (ReviewResult, error) {
	stub.assignFace = command
	return ReviewResult{EventID: command.EventID, Action: "assign_face", Revision: 2, Undoable: true}, nil
}
func (stub *reviewServiceRepositoryStub) AssignCluster(_ context.Context, command AssignClusterCommand) (ReviewResult, error) {
	stub.assignCluster = command
	return ReviewResult{EventID: command.EventID, Action: "assign_cluster", Revision: 3, Undoable: true}, nil
}
func (stub *reviewServiceRepositoryStub) ExcludeFace(_ context.Context, c ExcludeFaceCommand) (ReviewResult, error) {
	stub.exclude = c
	return ReviewResult{EventID: c.EventID, Action: "exclude_face", Revision: 2}, nil
}
func (stub *reviewServiceRepositoryStub) CannotLinkFaces(_ context.Context, command CannotLinkCommand) (ReviewResult, error) {
	stub.link = command
	return ReviewResult{EventID: command.EventID, Action: "cannot_link", Revision: 1, Undoable: true}, nil
}
func (stub *reviewServiceRepositoryStub) MergePeople(context.Context, MergePeopleCommand) (ReviewResult, error) {
	return ReviewResult{}, errors.New("unexpected merge")
}
func (stub *reviewServiceRepositoryStub) SplitFace(context.Context, SplitFaceCommand) (ReviewResult, error) {
	return ReviewResult{}, errors.New("unexpected split")
}
func (stub *reviewServiceRepositoryStub) UndoFaceReview(_ context.Context, command UndoReviewCommand) (ReviewResult, error) {
	stub.undo = command
	return ReviewResult{EventID: command.EventID, Action: "undo", Revision: command.ExpectedRevision + 1}, nil
}

type reviewClusterStub struct{ ids []string }

func TestReviewServiceMoveSourceGuard(t *testing.T) {
	repository := &reviewServiceRepositoryStub{}
	service, _ := NewReviewService(repository, reviewClusterStub{}, time.Now)
	request := ReviewRequest{Action: "assign_face", FaceID: "face_move_source", PersonID: "person_move_target", ExpectedFaceRevision: 1, ExpectedPersonRevision: 2, SourcePersonID: "person_move_source", ExpectedSourceRevision: 3}
	if _, err := service.Review(context.Background(), "move-source-key-001", request); err != nil {
		t.Fatal(err)
	}
	if repository.assignFace.SourcePersonID != request.SourcePersonID || repository.assignFace.ExpectedSourceRevision != 3 {
		t.Fatal("source guard not dispatched")
	}
	request.ExpectedSourceRevision = 0
	if _, err := service.Review(context.Background(), "move-source-key-002", request); !errors.Is(err, ErrInvalidReview) {
		t.Fatalf("missing source revision: %v", err)
	}
	request.SourcePersonID, request.ExpectedSourceRevision = "", 3
	if _, err := service.Review(context.Background(), "move-source-key-003", request); !errors.Is(err, ErrInvalidReview) {
		t.Fatalf("missing source ID: %v", err)
	}
}

func (stub reviewClusterStub) ListCoreClusterFaceIDs(context.Context, string, int64, int) ([]string, error) {
	return append([]string(nil), stub.ids...), nil
}

func TestReviewServiceDerivesStableOpaqueResourcesAndDispatchesTypedCommands(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	repository := &reviewServiceRepositoryStub{}
	service, err := NewReviewService(repository, reviewClusterStub{[]string{"face_service_01", "face_service_02"}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	request := ReviewRequest{Action: "assign_face", FaceID: "face_service_01", PersonID: "person_service_1", ExpectedFaceRevision: 1, ExpectedPersonRevision: 4}
	first, err := service.Review(context.Background(), "review-service-key-001", request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Review(context.Background(), "review-service-key-001", request)
	if err != nil {
		t.Fatal(err)
	}
	if first.EventID != second.EventID || repository.assignFace.EventID != first.EventID || repository.assignFace.AnchorID == "" || repository.assignFace.AnchorID == request.FaceID || repository.assignFace.RequestHash == "" || !repository.assignFace.CreatedAt.Equal(now) {
		t.Fatalf("result=%+v command=%+v", first, repository.assignFace)
	}

	cluster, err := service.Review(context.Background(), "review-service-key-002", ReviewRequest{Action: "assign_cluster", ClusterID: "cluster_service_1", PersonID: "person_service_1", ExpectedClusterRevision: 2, ExpectedPersonRevision: 5})
	if err != nil || cluster.Action != "assign_cluster" || len(repository.assignCluster.AnchorIDs) != 2 || repository.assignCluster.AnchorIDs[0] == repository.assignCluster.AnchorIDs[1] {
		t.Fatalf("cluster=%+v command=%+v err=%v", cluster, repository.assignCluster, err)
	}

	undo, err := service.Undo(context.Background(), "review-service-key-003", first.EventID, first.Revision)
	if err != nil || undo.Action != "undo" || repository.undo.ReviewID != first.EventID || repository.undo.ExpectedRevision != first.Revision {
		t.Fatalf("undo=%+v command=%+v err=%v", undo, repository.undo, err)
	}
}

func TestReviewServiceRejectsUnknownActionAndOversizedGroup(t *testing.T) {
	repository := &reviewServiceRepositoryStub{}
	service, _ := NewReviewService(repository, reviewClusterStub{}, time.Now)
	if _, err := service.Review(context.Background(), "review-service-key-004", ReviewRequest{Action: "identify_person"}); !errors.Is(err, ErrInvalidReview) {
		t.Fatalf("unknown action err=%v", err)
	}
	ids := make([]string, MaxGroupReviewFaces+1)
	for index := range ids {
		ids[index] = "face_service_overflow_" + string(rune('a'+index%26)) + string(rune('A'+index/26))
	}
	service, _ = NewReviewService(repository, reviewClusterStub{ids}, time.Now)
	if _, err := service.Review(context.Background(), "review-service-key-005", ReviewRequest{Action: "assign_cluster", ClusterID: "cluster_service_2", PersonID: "person_service_1", ExpectedClusterRevision: 1, ExpectedPersonRevision: 1}); !errors.Is(err, ErrReviewConflict) {
		t.Fatalf("oversized group err=%v", err)
	}
}

func TestReviewServiceExclusionSourceGuard(t *testing.T) {
	repo := &reviewServiceRepositoryStub{}
	service, _ := NewReviewService(repo, reviewClusterStub{}, time.Now)
	r := ReviewRequest{Action: "exclude_face", FaceID: "face_exclude_source", ExpectedFaceRevision: 1, SourcePersonID: "person_exclude_source", ExpectedSourceRevision: 7}
	if _, err := service.Review(context.Background(), "exclude-source-key-001", r); err != nil || repo.exclude.SourcePersonID != r.SourcePersonID || repo.exclude.ExpectedSourceRevision != 7 {
		t.Fatalf("dispatch=%+v %v", repo.exclude, err)
	}
	for _, g := range []struct {
		id  string
		rev int64
	}{{"person_exclude_source", 0}, {"", 1}, {"short", 1}, {"person_exclude_source", -1}} {
		r.SourcePersonID = g.id
		r.ExpectedSourceRevision = g.rev
		if _, err := service.Review(context.Background(), "exclude-source-key-002", r); !errors.Is(err, ErrInvalidReview) {
			t.Fatalf("invalid guard=%+v err=%v", g, err)
		}
	}
}

func TestReviewServiceCannotLinkRequiresCompleteDistinctPersonGuards(t *testing.T) {
	repository := &reviewServiceRepositoryStub{}
	service, _ := NewReviewService(repository, reviewClusterStub{}, time.Now)
	request := ReviewRequest{Action: "cannot_link", LeftFaceID: "face_left_01", RightFaceID: "face_right_01", ExpectedLeftRevision: 1, ExpectedRightRevision: 2, SourcePersonID: "person_left_01", TargetPersonID: "person_right_01", ExpectedSourceRevision: 7, ExpectedTargetRevision: 4}
	if _, err := service.Review(context.Background(), "link-pair-key-001", request); err != nil {
		t.Fatal(err)
	}
	if repository.link.SourcePersonID != request.SourcePersonID || repository.link.TargetPersonID != request.TargetPersonID || repository.link.ExpectedSourceRevision != 7 || repository.link.ExpectedTargetRevision != 4 {
		t.Fatalf("guard not dispatched: %+v", repository.link)
	}
	for _, change := range []func(*ReviewRequest){func(r *ReviewRequest) { r.TargetPersonID = "" }, func(r *ReviewRequest) { r.ExpectedSourceRevision = 0 }, func(r *ReviewRequest) { r.ExpectedTargetRevision = -1 }, func(r *ReviewRequest) { r.TargetPersonID = r.SourcePersonID }} {
		copy := request
		change(&copy)
		if _, err := service.Review(context.Background(), "link-pair-bad-001", copy); !errors.Is(err, ErrInvalidReview) {
			t.Fatalf("invalid pair=%v", err)
		}
	}
}
