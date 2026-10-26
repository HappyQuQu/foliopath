package face

import (
	"context"
	"errors"
	"testing"
	"time"
)

type peopleListStub struct {
	snapshot PeopleSnapshot
	items    []Person
}

func (s *peopleListStub) GetPeopleSnapshot(context.Context) (PeopleSnapshot, error) {
	return s.snapshot, nil
}
func (s *peopleListStub) ListPeoplePage(_ context.Context, q PeopleQuery) ([]Person, error) {
	start := 0
	if q.After != nil {
		for index, item := range s.items {
			if item.Name > q.After.Name || item.Name == q.After.Name && item.ID > q.After.ID {
				start = index
				break
			}
			start = len(s.items)
		}
	}
	end := min(len(s.items), start+q.Limit)
	return append([]Person(nil), s.items[start:end]...), nil
}

func TestPeopleListCursorBindsSearchAndSnapshot(t *testing.T) {
	repository := &peopleListStub{snapshot: PeopleSnapshot{Revision: 1}, items: []Person{{ID: "person_list_01", Name: "同名", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}, {ID: "person_list_02", Name: "同名", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}, {ID: "person_list_03", Name: "其他", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}}}
	service, err := NewPeopleListService(repository, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.List(context.Background(), PeopleListRequest{Limit: 1})
	if err != nil || len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := service.List(context.Background(), PeopleListRequest{Limit: 1, Cursor: first.NextCursor})
	if err != nil || len(second.Items) != 1 || second.Items[0].ID != "person_list_02" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if _, err := service.List(context.Background(), PeopleListRequest{Search: "different", Limit: 1, Cursor: first.NextCursor}); !errors.Is(err, ErrInvalidPeopleCursor) {
		t.Fatalf("query err=%v", err)
	}
	repository.snapshot.Revision++
	if _, err := service.List(context.Background(), PeopleListRequest{Limit: 1, Cursor: first.NextCursor}); !errors.Is(err, ErrPeopleCursorStale) {
		t.Fatalf("stale err=%v", err)
	}
}
