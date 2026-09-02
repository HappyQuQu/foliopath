package sqlite

import (
	"context"
	"testing"
	"time"
)

func TestFaceFoundationSeparatesDerivedAndManualState(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 15, 0, 0, 0, time.UTC).UnixMilli()
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO face_generations(
            id, detector_package_id, detector_content_hash, embedder_package_id,
            embedder_content_hash, embedding_dimension, transform_version,
            threshold_profile, state, created_at_ms, activated_at_ms, updated_at_ms
        ) VALUES('face_generation_1', 'yunet', ?, 'sface', ?, 2, 1, 'functional-v1', 'active', ?, ?, ?)`,
		hash, hash, now, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO face_library_settings(
            library_id, enabled, state, active_generation_id, created_at_ms, updated_at_ms
        ) VALUES(?, 1, 'ready', 'face_generation_1', ?, ?)`, libraryID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO face_observations(
            id, generation_id, library_id, asset_id, source_fingerprint,
            box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm,
            detection_ppm, quality_ppm, vector, created_at_ms, updated_at_ms
        ) VALUES('face_observation_1', 'face_generation_1', ?, ?, 'source-v1',
                 100000, 200000, 300000, 400000, 950000, 800000, X'003C0040', ?, ?)`,
		libraryID, assetID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO people(id, name, created_at_ms, updated_at_ms)
        VALUES('person_record_1', '同名允许', ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO person_face_anchors(
            id, person_id, library_id, asset_id, source_fingerprint,
            box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm,
            current_face_id, state, created_at_ms, updated_at_ms
        ) VALUES('face_anchor_1', 'person_record_1', ?, ?, 'source-v1',
                 100000, 200000, 300000, 400000, 'face_observation_1', 'bound', ?, ?)`,
		libraryID, assetID, now, now); err != nil {
		t.Fatal(err)
	}

	if _, err := store.db.ExecContext(context.Background(), `DELETE FROM face_observations WHERE library_id = ?`, libraryID); err != nil {
		t.Fatal(err)
	}
	var anchorCount, boundCount int
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT COUNT(*), COUNT(current_face_id) FROM person_face_anchors WHERE id='face_anchor_1'`).Scan(
		&anchorCount, &boundCount); err != nil {
		t.Fatal(err)
	}
	if anchorCount != 1 || boundCount != 0 {
		t.Fatalf("derived clear changed manual anchor: count=%d bound=%d", anchorCount, boundCount)
	}

	if _, err := store.db.ExecContext(context.Background(), `DELETE FROM libraries WHERE id = ?`, libraryID); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM person_face_anchors`).Scan(&anchorCount); err != nil {
		t.Fatal(err)
	}
	var personCount int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM people WHERE id='person_record_1'`).Scan(&personCount); err != nil {
		t.Fatal(err)
	}
	if anchorCount != 0 || personCount != 1 {
		t.Fatalf("library cascade = anchors %d people %d, want 0 and 1", anchorCount, personCount)
	}
}

func TestFaceFoundationRejectsInvalidGeometryAndCannotLinkOrdering(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 31, 16, 0, 0, 0, time.UTC).UnixMilli()
	hash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO face_generations(
            id, detector_package_id, detector_content_hash, embedder_package_id,
            embedder_content_hash, embedding_dimension, transform_version,
            threshold_profile, state, created_at_ms, updated_at_ms
        ) VALUES('face_generation_2', 'yunet', ?, 'sface', ?, 2, 1, 'functional-v1', 'ready', ?, ?)`,
		hash, hash, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO people(id, name, created_at_ms, updated_at_ms)
        VALUES('person_record_2', 'A', ?, ?), ('person_record_3', 'B', ?, ?)`, now, now, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO person_face_anchors(
            id, person_id, library_id, asset_id, source_fingerprint,
            box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm,
            state, created_at_ms, updated_at_ms
        ) VALUES
          ('face_anchor_2', 'person_record_2', ?, ?, 'source-v1', 0, 0, 500000, 500000, 'bound', ?, ?),
          ('face_anchor_3', 'person_record_3', ?, ?, 'source-v1', 500000, 0, 500000, 500000, 'bound', ?, ?)`,
		libraryID, assetID, now, now, libraryID, assetID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO face_cannot_links(left_anchor_id, right_anchor_id, created_at_ms)
        VALUES('face_anchor_3', 'face_anchor_2', ?)`, now); err == nil {
		t.Fatal("reversed cannot-link pair succeeded")
	}
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO face_observations(
            id, generation_id, library_id, asset_id, source_fingerprint,
            box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm,
            detection_ppm, quality_ppm, vector, created_at_ms, updated_at_ms
        ) VALUES('face_observation_bad', 'face_generation_2', ?, ?, 'source-v1',
                 900000, 0, 200000, 1000000, 1, 1, X'0000', ?, ?)`,
		libraryID, assetID, now, now); err == nil {
		t.Fatal("out-of-bounds observation succeeded")
	}
}
