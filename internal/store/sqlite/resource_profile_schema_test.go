package sqlite

import (
	"context"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
)

func TestResourceProfileMigrationDefaultsAndConstrainsTheSetting(t *testing.T) {
	store, _ := openTestStore(t)
	values, err := store.GetSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if values.ResourceProfile != resourcecontrol.ProfileBalanced {
		t.Fatalf("resource profile = %q, want balanced", values.ResourceProfile)
	}
	if _, err := store.db.Exec(
		`UPDATE settings SET resource_profile = 'unbounded' WHERE singleton_key = 1`,
	); err == nil {
		t.Fatal("invalid resource profile bypassed the database constraint")
	}
}
