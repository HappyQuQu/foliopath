package face

import (
	"context"
	"errors"
	"testing"
)

type clusterListStub struct {
	snapshot FaceClusterSnapshot
	items    []FaceClusterView
}

func (s *clusterListStub) ReadFaceClusterPage(_ context.Context, q FaceClusterListQuery) (FaceClusterReadPage, error) {
	start := 0
	if q.After != nil {
		for index, item := range s.items {
			if item.Role > q.After.Role || item.Role == q.After.Role && item.ID > q.After.ID {
				start = index
				break
			}
			start = len(s.items)
		}
	}
	end := min(len(s.items), start+q.Limit)
	return FaceClusterReadPage{Snapshot: s.snapshot, Items: append([]FaceClusterView(nil), s.items[start:end]...)}, nil
}
func TestFaceClusterCursorBindsLibraryRoleGenerationAndSnapshot(t *testing.T) {
	repository := &clusterListStub{snapshot: FaceClusterSnapshot{LibraryID: 1, GenerationID: "face_generation_1", Revision: 1, Digest: "snapshot1", Coverage: FaceCoverage{Revision: 1}}, items: []FaceClusterView{{ID: "face_group_0001", LibraryID: 1, Role: "core"}, {ID: "face_group_0002", LibraryID: 1, Role: "edge"}}}
	service, err := NewFaceClusterListService(repository, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.List(context.Background(), FaceClusterListRequest{LibraryID: 1, Limit: 1})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := service.List(context.Background(), FaceClusterListRequest{LibraryID: 1, Limit: 1, Cursor: first.NextCursor})
	if err != nil || len(second.Items) != 1 || second.Items[0].ID != "face_group_0002" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if _, err := service.List(context.Background(), FaceClusterListRequest{LibraryID: 1, Role: "core", Limit: 1, Cursor: first.NextCursor}); !errors.Is(err, ErrInvalidFaceClusterCursor) {
		t.Fatalf("role err=%v", err)
	}
	repository.snapshot.Digest = "snapshot2"
	if _, err := service.List(context.Background(), FaceClusterListRequest{LibraryID: 1, Limit: 1, Cursor: first.NextCursor}); !errors.Is(err, ErrFaceClusterCursorStale) {
		t.Fatalf("stale err=%v", err)
	}
}

func TestFaceClusterCursorBindsReviewFilter(t *testing.T) {
	repository := &clusterListStub{snapshot: FaceClusterSnapshot{GenerationID: "generation_01", Revision: 1, Digest: "snapshot"}, items: []FaceClusterView{{ID: "cluster_first", Role: "core"}, {ID: "cluster_second", Role: "core"}}}
	service, _ := NewFaceClusterListService(repository, []byte("0123456789abcdef0123456789abcdef"))
	page, err := service.List(context.Background(), FaceClusterListRequest{LibraryID: 1, Review: "pending", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), FaceClusterListRequest{LibraryID: 1, Review: "all", Cursor: page.NextCursor, Limit: 1}); !errors.Is(err, ErrInvalidFaceClusterCursor) {
		t.Fatalf("filter reuse=%v", err)
	}
	if _, err := service.List(context.Background(), FaceClusterListRequest{LibraryID: 1, Review: "other", Limit: 1}); !errors.Is(err, ErrInvalidClusterRecord) {
		t.Fatalf("filter validation=%v", err)
	}
}
