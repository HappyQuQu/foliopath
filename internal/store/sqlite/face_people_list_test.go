package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestPeopleListUsesStableNameAndIDKeyset(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 31, 23, 0, 0, 0, time.UTC)
	for _, id := range []string{"person_page_02", "person_page_01", "person_page_03"} {
		if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: id, Name: "同名", CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	items, err := store.ListPeoplePage(context.Background(), face.PeopleQuery{Limit: 2})
	if err != nil || len(items) != 2 || items[0].ID != "person_page_01" || items[1].ID != "person_page_02" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	next, err := store.ListPeoplePage(context.Background(), face.PeopleQuery{After: &face.PeoplePosition{Name: items[1].Name, ID: items[1].ID}, Limit: 2})
	if err != nil || len(next) != 1 || next[0].ID != "person_page_03" {
		t.Fatalf("next=%+v err=%v", next, err)
	}
}
