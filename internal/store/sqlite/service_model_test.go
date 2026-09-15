package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestServiceModelMigrationPreservesGenerationAndReferences(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	store.db.SetMaxOpenConns(1)
	provider, err := goose.NewProvider(goose.DialectSQLite3, store.db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 38); err != nil {
		t.Fatal(err)
	}
	generation := seedEmbeddingGeneration(t, store, 512)
	if _, err := store.db.Exec(`UPDATE semantic_generations SET state='building' WHERE id=?`, generation); err != nil {
		t.Fatal(err)
	}
	if err := store.BindSemanticGenerationSpace(ctx, generation, spaceFixture()); err != nil {
		t.Fatal(err)
	}
	before, err := store.GetSemanticGenerationRuntime(ctx, generation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 39); err != nil {
		t.Fatal(err)
	}
	after, err := store.GetSemanticGenerationRuntime(ctx, generation)
	if err != nil || before != after {
		t.Fatalf("generation changed: %+v -> %+v %v", before, after, err)
	}
	rows, err := store.db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Fatal("foreign key violation after upgrade")
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DELETE FROM ai_models WHERE id=?`, after.ModelID); err == nil {
		t.Fatal("model deletion bypassed generation foreign key")
	}
	model := activationModelFixture(time.Now().UTC())
	model.ID = "aim_service_model"
	model.StorageMode = aimodel.StorageService
	model.SourceIdentity = model.Package.ContentHash
	model.Package.PackageID = "service-clip"
	registered, created, err := store.RegisterAIModel(ctx, model)
	if err != nil || !created || registered.StorageMode != aimodel.StorageService {
		t.Fatalf("register=%+v %v %v", registered, created, err)
	}
	_, created, err = store.RegisterAIModel(ctx, model)
	if err != nil || created {
		t.Fatalf("idempotency=%v %v", created, err)
	}
	if _, err := store.db.Exec(`UPDATE ai_models SET source_identity='/models/pretend' WHERE id=?`, model.ID); err == nil {
		t.Fatal("service accepted directory identity")
	}
	if _, err := store.db.Exec(`UPDATE ai_models SET state='installed' WHERE id=?`, model.ID); err == nil {
		t.Fatal("service accepted installed file state")
	}
	// Downgrade must refuse rather than delete service data. A failed NO TRANSACTION
	// migration leaves its explicit transaction open; roll it back before inspecting.
	if _, err := provider.DownTo(ctx, 38); err == nil {
		t.Fatal("lossy downgrade accepted")
	}
	if _, err := store.db.Exec(`ROLLBACK; PRAGMA legacy_alter_table=OFF; PRAGMA foreign_keys=ON;`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetAIModel(ctx, model.ID); err != nil {
		t.Fatalf("downgrade lost service model: %v", err)
	}
}
