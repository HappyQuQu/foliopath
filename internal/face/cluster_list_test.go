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

func (s *clusterListStub) GetFaceClusterSnapshot(context.Context, int64) (FaceClusterSnapshot, error) {
	return s.snapshot, nil
}
func (s *clusterListStub) ListFaceClusterViews(_ context.Context, q FaceClusterListQuery) ([]FaceClusterView, error) {
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
	return append([]FaceClusterView(nil), s.items[start:end]...), nil
}
func TestFaceClusterCursorBindsLibraryRoleGenerationAndSnapshot(t *testing.T) {
	repository := &clusterListStub{snapshot: FaceClusterSnapshot{LibraryID: 1, GenerationID: "face_generation_1", Revision: 1, Coverage: FaceCoverage{Revision: 1}}, items: []FaceClusterView{{ID: "face_group_0001", LibraryID: 1, Role: "core"}, {ID: "face_group_0002", LibraryID: 1, Role: "edge"}}}
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
	repository.snapshot.Revision++
	if _, err := service.List(context.Background(), FaceClusterListRequest{LibraryID: 1, Limit: 1, Cursor: first.NextCursor}); !errors.Is(err, ErrFaceClusterCursorStale) {
		t.Fatalf("stale err=%v", err)
	}
}
