package sqlite

import (
	"context"
	"testing"
)

func TestResourceLimitMigrationDefaultsAndConstrainsTheSettings(t *testing.T) {
	store, _ := openTestStore(t)
	values, err := store.GetSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if values.BackgroundConcurrency != 2 || values.ContentReadConcurrency != 8 {
		t.Fatalf("resource limits = %d/%d, want 2/8", values.BackgroundConcurrency, values.ContentReadConcurrency)
	}
	if _, err := store.db.Exec(
		`UPDATE settings SET background_concurrency = 5 WHERE singleton_key = 1`,
	); err == nil {
		t.Fatal("invalid background concurrency bypassed the database constraint")
	}
	if _, err := store.db.Exec(
		`UPDATE settings SET content_read_concurrency = 17 WHERE singleton_key = 1`,
	); err == nil {
		t.Fatal("invalid content concurrency bypassed the database constraint")
	}
}
