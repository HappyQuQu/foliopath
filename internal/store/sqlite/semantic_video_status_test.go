package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestVideoStatusPreservesUnknownAndOfflineCoverage(t *testing.T) {
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	ctx := context.Background()
	status, err := store.GetVideoStatus(ctx, 1)
	if err != nil || status.Coverage != nil {
		t.Fatalf("unrecorded: %+v %v", status, err)
	}
	generation := seedEmbeddingGeneration(t, store, 2)
	if _, err := store.db.Exec(`UPDATE ai_model_state SET active_generation_id=?, active_model_id=(SELECT model_id FROM semantic_generations WHERE id=?) WHERE singleton_key=1`, generation, generation); err != nil {
		t.Fatal(err)
	}
	status, err = store.GetVideoStatus(ctx, 1)
	if err != nil || status.Coverage != nil || status.GenerationID != generation {
		t.Fatalf("new generation: %+v %v", status, err)
	}
	if _, err := store.db.Exec(`INSERT INTO semantic_video_progress(generation_id,library_id,eligible_count,ready_count,degraded_count,revision,updated_at_ms) VALUES(?,1,3,1,1,2,1)`, generation); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE libraries SET status='offline' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	status, err = store.GetVideoStatus(ctx, 1)
	if err != nil || !status.Offline || status.Coverage == nil || status.Coverage.Ready != 1 || status.Coverage.Degraded != 1 || status.Coverage.Complete() {
		t.Fatalf("offline: %+v %v", status, err)
	}
	if _, err := store.GetVideoStatus(ctx, 999); !errors.Is(err, semantic.ErrSemanticLibraryNotFound) {
		t.Fatal(err)
	}
}
