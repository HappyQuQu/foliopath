package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func TestFaceModelActivationMigrationPreservesSemanticStateAndRoundTrips(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "face-model-activation.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 34); err != nil {
		t.Fatalf("apply version 34: %v", err)
	}
	hash := strings.Repeat("a", 64)
	if _, err := db.ExecContext(ctx, `
        INSERT INTO ai_models(
            id,purpose,package_id,version,architecture,content_hash,license_id,package_size_bytes,
            storage_mode,state,source_identity,availability_revision,created_at_ms,updated_at_ms
        ) VALUES('aim_semantic_upgrade','semantic_image_text','semantic-v1','1.0.0','arm64',?,'Apache-2.0',1,
                 'managed','available','managed:upgrade',1,1,1);
        INSERT INTO semantic_generations(
            id,model_id,transform_version,output_schema_version,index_format_version,embedding_dimension,
            state,created_at_ms,activated_at_ms,updated_at_ms
        ) VALUES('aig_semantic_upgrade','aim_semantic_upgrade',1,1,1,768,'active',1,1,1);
        UPDATE ai_model_state SET active_model_id='aim_semantic_upgrade',active_generation_id='aig_semantic_upgrade'
        WHERE singleton_key=1;`, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 35); err != nil {
		t.Fatalf("upgrade to version 35: %v", err)
	}
	assertFaceActivationMigrationState(t, db, true)
	if _, err := provider.DownTo(ctx, 34); err != nil {
		t.Fatalf("downgrade to version 34: %v", err)
	}
	assertFaceActivationMigrationState(t, db, false)
}

func assertFaceActivationMigrationState(t *testing.T, db *sql.DB, facePurposeAllowed bool) {
	t.Helper()
	var activeModel, activeGeneration string
	if err := db.QueryRow(`SELECT active_model_id,active_generation_id FROM ai_model_state WHERE singleton_key=1`).
		Scan(&activeModel, &activeGeneration); err != nil {
		t.Fatal(err)
	}
	if activeModel != "aim_semantic_upgrade" || activeGeneration != "aig_semantic_upgrade" {
		t.Fatalf("semantic state=%q/%q", activeModel, activeGeneration)
	}
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		_ = rows.Close()
		t.Fatal("foreign key violation after face activation migration")
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	_, insertErr := db.Exec(`
        INSERT INTO ai_models(
            id,purpose,package_id,version,architecture,content_hash,license_id,package_size_bytes,
            storage_mode,state,source_identity,availability_revision,created_at_ms,updated_at_ms
        ) VALUES('aim_face_schema','face_detection_embedding','face-v3','1.0.0','arm64',?,'MIT',1,
                 'managed','available','managed:face',1,1,1)`, strings.Repeat("b", 64))
	if facePurposeAllowed && insertErr != nil {
		t.Fatalf("face purpose rejected after upgrade: %v", insertErr)
	}
	if !facePurposeAllowed && insertErr == nil {
		t.Fatal("face purpose accepted after downgrade")
	}
	if facePurposeAllowed {
		if _, err := db.Exec(`DELETE FROM ai_models WHERE id='aim_face_schema'`); err != nil {
			t.Fatalf("remove face downgrade fixture: %v", err)
		}
	}
}
