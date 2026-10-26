package app

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceResultsCompositionWithDatabaseLifecycle(t *testing.T) {
	ctx := context.Background()
	component, database := newDatabaseComponent(t.TempDir(), newReadinessState())
	var routes api.RouteDependencies
	if err := composeFaceResults(&routes, database); err != nil {
		t.Fatal(err)
	}
	if routes.FaceClusters == nil || routes.FacePeople == nil || routes.FacePerson == nil ||
		routes.FaceAssetFaces == nil || routes.FacePersonAssets == nil || routes.FaceClusterDetails == nil ||
		routes.FaceReviews == nil || routes.FacePersonMutations == nil {
		t.Fatal("incomplete face routes")
	}
	if _, err := routes.FacePeople.List(ctx, face.PeopleListRequest{Limit: 10}); err == nil {
		t.Fatal("database should not be ready")
	}
	if err := component.start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = component.stop(ctx) })
	revision := int64(1)
	if _, _, err := routes.FacePersonMutations.Create(ctx, "face-unapproved-group", "Unapproved", "cluster-unapproved", &revision); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("uncalibrated group creation: %v", err)
	}
	if _, err := routes.FaceReviews.Review(ctx, "face-unapproved-assign", face.ReviewRequest{Action: "assign_cluster", ClusterID: "cluster-unapproved", ExpectedClusterRevision: 1}); !errors.Is(err, face.ErrReviewConflict) {
		t.Fatalf("uncalibrated group assignment: %v", err)
	}
	person, replayed, err := routes.FacePersonMutations.Create(ctx, "face-composition-create", "测试人物", "", nil)
	if err != nil || replayed {
		t.Fatalf("create: %v %v", replayed, err)
	}
	again, replayed, err := routes.FacePersonMutations.Create(ctx, "face-composition-create", "测试人物", "", nil)
	if err != nil || !replayed || again.ID != person.ID {
		t.Fatalf("replay: %+v %v %v", again, replayed, err)
	}
	got, err := routes.FacePerson.GetPerson(ctx, person.ID)
	if err != nil || got.Name != person.Name {
		t.Fatalf("get: %+v %v", got, err)
	}
	page, err := routes.FacePeople.List(ctx, face.PeopleListRequest{Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("list: %+v %v", page, err)
	}
	renamed, err := routes.FacePersonMutations.Rename(ctx, person.ID, "新名字", person.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := routes.FacePersonMutations.Rename(ctx, person.ID, "旧版本", person.Revision); err == nil {
		t.Fatal("stale revision accepted")
	}
	if err := routes.FacePersonMutations.Delete(ctx, person.ID, renamed.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := routes.FacePerson.GetPerson(ctx, person.ID); !errors.Is(err, face.ErrPersonNotFound) {
		t.Fatalf("deleted person: %v", err)
	}
	if err := component.stop(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := routes.FacePerson.GetPerson(ctx, person.ID); err == nil {
		t.Fatal("closed database accepted access")
	}
}
