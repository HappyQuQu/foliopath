package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestFaceServiceMigrationRefusesLossyDowngrade(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	store.db.SetMaxOpenConns(1)
	provider, err := goose.NewProvider(goose.DialectSQLite3, store.db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 39); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 40); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	model := faceActivationModelFixture(now)
	model.StorageMode = aimodel.StorageService
	model.SourceIdentity = model.Package.ContentHash
	if _, _, err := store.RegisterAIModel(ctx, model); err != nil {
		t.Fatal(err)
	}
	seedActiveFaceGeneration(t, store, model, "face_service_migration", now)
	if _, err := provider.DownTo(ctx, 39); err == nil {
		t.Fatal("downgrade accepted service face state unsupported by v39")
	}
	// Migration 40 owns an explicit transaction; rollback failed DDL before reads.
	if _, err := store.db.Exec(`ROLLBACK; PRAGMA legacy_alter_table=OFF; PRAGMA foreign_keys=ON;`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetAIModel(ctx, model.ID); err != nil {
		t.Fatalf("rejected downgrade lost face model: %v", err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT count(*) FROM face_generations WHERE id='face_service_migration' AND model_id=?`, model.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("rejected downgrade lost generation: %d %v", count, err)
	}
	rows, err := store.db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign key violation after rejected downgrade")
	}
}
