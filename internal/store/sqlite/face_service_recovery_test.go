package sqlite

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceServiceRetryRecoversAwaitingSameModel(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	library := seedBrowseCatalog(t, store)
	now := time.Now().UTC()
	model := faceActivationModelFixture(now)
	model.StorageMode = aimodel.StorageService
	model.SourceIdentity = model.Package.ContentHash
	if _, _, err := store.RegisterAIModel(ctx, model); err != nil {
		t.Fatal(err)
	}
	const generation = "face_recovery_same_model"
	seedActiveFaceGeneration(t, store, model, generation, now)
	seedFaceReadySettings(t, store, library, generation, now)
	if _, err := store.db.Exec(`UPDATE face_library_settings SET state='awaiting_model' WHERE library_id=?`, library); err != nil {
		t.Fatal(err)
	}
	settings, _ := face.NewSettingsService(store, func() time.Time { return now })
	unavailable, err := store.SetAIModelAvailability(ctx, model.ID, model.AvailabilityRevision, aimodel.StateUnavailable, now)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := store.GetFaceLibrarySettings(ctx, library)
	if _, err := settings.JobSettings(ctx, library); !errors.Is(err, face.ErrFaceModelUnavailable) {
		t.Fatalf("offline retry bypassed availability: %v", err)
	}
	after, _ := store.GetFaceLibrarySettings(ctx, library)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected retry changed settings")
	}
	if _, err := store.SetAIModelAvailability(ctx, model.ID, unavailable.AvailabilityRevision, aimodel.StateAvailable, now); err != nil {
		t.Fatal(err)
	}
	recovered, err := settings.JobSettings(ctx, library)
	if err != nil || recovered.State != "building" || recovered.ActiveGenerationID != generation || recovered.Revision != before.Revision+1 {
		t.Fatalf("same-generation recovery: %+v %v", recovered, err)
	}
	jobs, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := jobs.Request(ctx, library, generation, face.JobMissing, "recovered-same-model-request"); err != nil {
		t.Fatalf("recovered job admission: %v", err)
	}
	if _, err := settings.Update(ctx, library, false, recovered.Revision); err != nil {
		t.Fatal(err)
	}
	disabled, err := settings.JobSettings(ctx, library)
	if err != nil || disabled.Enabled || disabled.State != "disabled" {
		t.Fatalf("retry re-enabled disabled library: %+v %v", disabled, err)
	}
}
