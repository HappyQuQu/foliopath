package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/scanner"
	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
)

func TestSettingsUpdateAndDueLibrarySelection(t *testing.T) {
	now := time.UnixMilli(100_000_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	first := createWorkerLibrary(t, store, "Due", "due")
	second := createWorkerLibrary(t, store, "Recent", "recent")
	if _, err := store.AdmitFullScan(ctx, first.ID, scanner.TriggerCreation); err != nil {
		t.Fatal(err)
	}
	now = now.Add(25 * time.Hour)
	if _, err := store.AdmitFullScan(ctx, second.ID, scanner.TriggerCreation); err != nil {
		t.Fatal(err)
	}
	due, err := store.ListDueLibraryIDs(ctx, now.Add(-24*time.Hour).UnixMilli(), 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0] != first.ID {
		t.Fatalf("due libraries = %v", due)
	}

	values, err := store.GetSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	values.ScheduledScanIntervalHours = nil
	values.Language = "en"
	values.BackgroundConcurrency = 3
	values.ContentReadConcurrency = 12
	updated, err := store.UpdateSettings(ctx, values.Revision, values)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ScheduledScanIntervalHours != nil || updated.Language != "en" ||
		updated.BackgroundConcurrency != 3 || updated.ContentReadConcurrency != 12 ||
		updated.Revision != values.Revision+1 {
		t.Fatalf("updated settings = %#v", updated)
	}
	if _, err := store.UpdateSettings(ctx, values.Revision, values); !errors.Is(err, appsettings.ErrPreconditionFailed) {
		t.Fatalf("stale update error = %v", err)
	}
}
