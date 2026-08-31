package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestControlledTagVocabularyPublishesImmutableCASBoundSnapshots(t *testing.T) {
	store, _ := openTestStore(t)
	initial, err := store.GetActiveTagVocabulary(context.Background())
	if err != nil || initial.Revision != 1 || initial.ID != "aivocab_initial" || len(initial.Entries) != 0 {
		t.Fatalf("initial=%#v err=%v", initial, err)
	}
	now := time.Date(2026, 8, 30, 9, 0, 0, 0, time.UTC)
	for id, name := range map[int64]string{3: "海边", 7: "Family"} {
		if _, err := store.db.Exec(`INSERT INTO tags(id,name,normalized_name,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?)`, id, name, name, now.UnixMilli(), now.UnixMilli()); err != nil {
			t.Fatal(err)
		}
	}
	service, err := semantic.NewTagVocabularyService(store, func() time.Time { return now }, func(string) (string, error) {
		return "aivocab_revision_two", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.Publish(context.Background(), 1, []int64{7, 3})
	if err != nil || published.Revision != 2 || len(published.Entries) != 2 || published.Entries[0].TagID != 3 {
		t.Fatalf("published=%#v err=%v", published, err)
	}
	if _, err := service.Publish(context.Background(), 1, []int64{3}); !errors.Is(err, semantic.ErrTagVocabularyConflict) {
		t.Fatalf("stale publish=%v", err)
	}
	if _, err := service.Publish(context.Background(), 2, []int64{99}); !errors.Is(err, curation.ErrTagNotFound) {
		t.Fatalf("missing tag=%v", err)
	}
	var active, retired int
	if err := store.db.QueryRow(`SELECT SUM(state='active'),SUM(state='retired') FROM ai_tag_vocabulary_snapshots`).Scan(&active, &retired); err != nil || active != 1 || retired != 1 {
		t.Fatalf("snapshots active=%d retired=%d err=%v", active, retired, err)
	}
}
