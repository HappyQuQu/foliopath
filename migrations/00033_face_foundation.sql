-- +goose Up
CREATE TABLE face_generations (
    id                      TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    detector_package_id     TEXT NOT NULL CHECK (length(detector_package_id) BETWEEN 1 AND 128),
    detector_content_hash   TEXT NOT NULL CHECK (length(detector_content_hash) = 64 AND detector_content_hash NOT GLOB '*[^0-9a-f]*'),
    embedder_package_id     TEXT NOT NULL CHECK (length(embedder_package_id) BETWEEN 1 AND 128),
    embedder_content_hash   TEXT NOT NULL CHECK (length(embedder_content_hash) = 64 AND embedder_content_hash NOT GLOB '*[^0-9a-f]*'),
    embedding_dimension     INTEGER NOT NULL CHECK (embedding_dimension BETWEEN 1 AND 4096),
    transform_version       INTEGER NOT NULL CHECK (transform_version > 0),
    threshold_profile       TEXT NOT NULL CHECK (length(threshold_profile) BETWEEN 1 AND 128),
    state                   TEXT NOT NULL CHECK (state IN ('building', 'ready', 'active', 'retired', 'failed')),
    created_at_ms           INTEGER NOT NULL CHECK (created_at_ms > 0),
    activated_at_ms         INTEGER CHECK (activated_at_ms IS NULL OR activated_at_ms >= created_at_ms),
    updated_at_ms           INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms)
);

CREATE UNIQUE INDEX face_generations_one_active
    ON face_generations(state) WHERE state = 'active';

CREATE TABLE face_cluster_builds (
    id              TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    generation_id   TEXT NOT NULL REFERENCES face_generations(id) ON DELETE CASCADE,
    library_id      INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    state           TEXT NOT NULL CHECK (state IN ('building', 'active', 'stale')),
    created_at_ms   INTEGER NOT NULL CHECK (created_at_ms > 0),
    activated_at_ms INTEGER CHECK (activated_at_ms IS NULL OR activated_at_ms >= created_at_ms)
);

CREATE UNIQUE INDEX face_cluster_builds_one_active
    ON face_cluster_builds(library_id) WHERE state='active';

CREATE TABLE face_library_settings (
    library_id            INTEGER PRIMARY KEY REFERENCES libraries(id) ON DELETE CASCADE,
    enabled               INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    state                 TEXT NOT NULL DEFAULT 'disabled' CHECK (state IN ('disabled', 'awaiting_model', 'building', 'ready', 'degraded', 'clearing')),
    active_generation_id  TEXT REFERENCES face_generations(id) ON DELETE RESTRICT,
    active_cluster_build_id TEXT REFERENCES face_cluster_builds(id) ON DELETE SET NULL,
    revision              INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    coverage_revision     INTEGER NOT NULL DEFAULT 1 CHECK (coverage_revision > 0),
    created_at_ms         INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms         INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    CHECK (enabled = 1 OR state IN ('disabled', 'clearing'))
);

CREATE TABLE face_library_progress (
    generation_id    TEXT NOT NULL REFERENCES face_generations(id) ON DELETE CASCADE,
    library_id       INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    eligible_count   INTEGER NOT NULL DEFAULT 0 CHECK (eligible_count >= 0),
    completed_count  INTEGER NOT NULL DEFAULT 0 CHECK (completed_count >= 0),
    failed_count     INTEGER NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
    stale_count      INTEGER NOT NULL DEFAULT 0 CHECK (stale_count >= 0),
    checkpoint_id    INTEGER NOT NULL DEFAULT 0 CHECK (checkpoint_id >= 0),
    revision         INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_at_ms    INTEGER NOT NULL CHECK (updated_at_ms > 0),
    PRIMARY KEY (generation_id, library_id),
    CHECK (completed_count + failed_count + stale_count <= eligible_count)
);

-- A successful analysis may legitimately detect zero faces. Keep an explicit
-- source-bound completion marker so missing-only jobs remain restart-safe.
CREATE TABLE face_asset_results (
    generation_id       TEXT NOT NULL REFERENCES face_generations(id) ON DELETE CASCADE,
    library_id          INTEGER NOT NULL,
    asset_id            INTEGER NOT NULL,
    source_fingerprint  TEXT NOT NULL CHECK (length(source_fingerprint) BETWEEN 1 AND 256),
    face_count          INTEGER NOT NULL CHECK (face_count BETWEEN 0 AND 64),
    revision            INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_at_ms       INTEGER NOT NULL CHECK (updated_at_ms > 0),
    PRIMARY KEY (generation_id, library_id, asset_id),
    FOREIGN KEY (library_id, asset_id) REFERENCES assets(library_id, id) ON DELETE CASCADE
);

CREATE INDEX face_asset_results_source
    ON face_asset_results(library_id, generation_id, source_fingerprint, asset_id);

CREATE TABLE face_observations (
    id                    TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    generation_id         TEXT NOT NULL REFERENCES face_generations(id) ON DELETE CASCADE,
    library_id            INTEGER NOT NULL,
    asset_id              INTEGER NOT NULL,
    source_fingerprint    TEXT NOT NULL CHECK (length(source_fingerprint) BETWEEN 1 AND 256),
    box_x_ppm             INTEGER NOT NULL CHECK (box_x_ppm BETWEEN 0 AND 999999),
    box_y_ppm             INTEGER NOT NULL CHECK (box_y_ppm BETWEEN 0 AND 999999),
    box_width_ppm         INTEGER NOT NULL CHECK (box_width_ppm BETWEEN 1 AND 1000000),
    box_height_ppm        INTEGER NOT NULL CHECK (box_height_ppm BETWEEN 1 AND 1000000),
    detection_ppm         INTEGER NOT NULL CHECK (detection_ppm BETWEEN 0 AND 1000000),
    quality_ppm           INTEGER NOT NULL CHECK (quality_ppm BETWEEN 0 AND 1000000),
    vector                BLOB NOT NULL CHECK (length(vector) > 0 AND length(vector) % 2 = 0),
    revision              INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at_ms         INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms         INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    UNIQUE (generation_id, library_id, asset_id, source_fingerprint, box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm),
    CHECK (box_x_ppm + box_width_ppm <= 1000000),
    CHECK (box_y_ppm + box_height_ppm <= 1000000),
    FOREIGN KEY (library_id, asset_id) REFERENCES assets(library_id, id) ON DELETE CASCADE
);

CREATE INDEX face_observations_asset
    ON face_observations(library_id, asset_id, generation_id, id);

CREATE TABLE face_clusters (
	build_id        TEXT NOT NULL REFERENCES face_cluster_builds(id) ON DELETE CASCADE,
    id              TEXT NOT NULL CHECK (length(id) BETWEEN 8 AND 128),
    generation_id   TEXT NOT NULL REFERENCES face_generations(id) ON DELETE CASCADE,
    library_id      INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('core', 'edge')),
    revision        INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at_ms   INTEGER NOT NULL CHECK (created_at_ms > 0),
	updated_at_ms   INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
	PRIMARY KEY (build_id, id)
);

CREATE INDEX face_clusters_page
    ON face_clusters(library_id, generation_id, role, id);

CREATE TABLE face_cluster_members (
	build_id       TEXT NOT NULL,
    cluster_id      TEXT NOT NULL,
    face_id         TEXT NOT NULL REFERENCES face_observations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('core', 'edge')),
    confidence_ppm  INTEGER NOT NULL CHECK (confidence_ppm BETWEEN 0 AND 1000000),
	PRIMARY KEY (build_id, cluster_id, face_id),
	UNIQUE (build_id, face_id),
	FOREIGN KEY (build_id, cluster_id) REFERENCES face_clusters(build_id, id) ON DELETE CASCADE
);

CREATE TABLE people (
    id              TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    name            TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 400),
    state           TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'tombstoned')),
    revision        INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at_ms   INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms   INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    tombstoned_at_ms INTEGER CHECK (tombstoned_at_ms IS NULL OR tombstoned_at_ms >= created_at_ms)
);

CREATE INDEX people_page ON people(state, name, id);

CREATE TABLE person_face_anchors (
    id                    TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    person_id             TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    library_id            INTEGER NOT NULL,
    asset_id              INTEGER NOT NULL,
    source_fingerprint    TEXT NOT NULL CHECK (length(source_fingerprint) BETWEEN 1 AND 256),
    box_x_ppm             INTEGER NOT NULL CHECK (box_x_ppm BETWEEN 0 AND 999999),
    box_y_ppm             INTEGER NOT NULL CHECK (box_y_ppm BETWEEN 0 AND 999999),
    box_width_ppm         INTEGER NOT NULL CHECK (box_width_ppm BETWEEN 1 AND 1000000),
    box_height_ppm        INTEGER NOT NULL CHECK (box_height_ppm BETWEEN 1 AND 1000000),
    current_face_id       TEXT REFERENCES face_observations(id) ON DELETE SET NULL,
    state                 TEXT NOT NULL CHECK (state IN ('bound', 'needs_review', 'excluded')),
    revision              INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at_ms         INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms         INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    UNIQUE (person_id, library_id, asset_id, source_fingerprint, box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm),
    CHECK (box_x_ppm + box_width_ppm <= 1000000),
    CHECK (box_y_ppm + box_height_ppm <= 1000000),
    FOREIGN KEY (library_id, asset_id) REFERENCES assets(library_id, id) ON DELETE CASCADE
);

CREATE INDEX person_face_anchors_person ON person_face_anchors(person_id, state, id);
CREATE INDEX person_face_anchors_asset ON person_face_anchors(library_id, asset_id, id);
CREATE UNIQUE INDEX person_face_anchors_current_face
    ON person_face_anchors(current_face_id) WHERE current_face_id IS NOT NULL;
CREATE UNIQUE INDEX person_face_anchors_lineage
    ON person_face_anchors(library_id,asset_id,source_fingerprint,box_x_ppm,box_y_ppm,box_width_ppm,box_height_ppm);

CREATE TABLE face_exclusions (
    id                    TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    library_id            INTEGER NOT NULL,
    asset_id              INTEGER NOT NULL,
    source_fingerprint    TEXT NOT NULL CHECK (length(source_fingerprint) BETWEEN 1 AND 256),
    box_x_ppm             INTEGER NOT NULL CHECK (box_x_ppm BETWEEN 0 AND 999999),
    box_y_ppm             INTEGER NOT NULL CHECK (box_y_ppm BETWEEN 0 AND 999999),
    box_width_ppm         INTEGER NOT NULL CHECK (box_width_ppm BETWEEN 1 AND 1000000),
    box_height_ppm        INTEGER NOT NULL CHECK (box_height_ppm BETWEEN 1 AND 1000000),
    current_face_id       TEXT REFERENCES face_observations(id) ON DELETE SET NULL,
    revision              INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at_ms         INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms         INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    UNIQUE (library_id, asset_id, source_fingerprint, box_x_ppm, box_y_ppm, box_width_ppm, box_height_ppm),
    CHECK (box_x_ppm + box_width_ppm <= 1000000),
    CHECK (box_y_ppm + box_height_ppm <= 1000000),
    FOREIGN KEY (library_id, asset_id) REFERENCES assets(library_id, id) ON DELETE CASCADE
);

CREATE INDEX face_exclusions_asset ON face_exclusions(library_id, asset_id, id);

CREATE TABLE face_cannot_links (
    left_anchor_id  TEXT NOT NULL REFERENCES person_face_anchors(id) ON DELETE CASCADE,
    right_anchor_id TEXT NOT NULL REFERENCES person_face_anchors(id) ON DELETE CASCADE,
    revision        INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at_ms   INTEGER NOT NULL CHECK (created_at_ms > 0),
    PRIMARY KEY (left_anchor_id, right_anchor_id),
    CHECK (left_anchor_id < right_anchor_id)
);

CREATE TABLE face_audit_events (
    id                 TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    request_hash       TEXT NOT NULL CHECK (length(request_hash) = 64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    library_id         INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
    action             TEXT NOT NULL CHECK (action IN ('create_person', 'rename_person', 'assign_face', 'assign_cluster', 'exclude_face', 'cannot_link', 'merge_people', 'split_face', 'delete_person', 'undo')),
    primary_target_id  TEXT NOT NULL CHECK (length(primary_target_id) BETWEEN 8 AND 128),
    secondary_target_id TEXT CHECK (secondary_target_id IS NULL OR length(secondary_target_id) BETWEEN 8 AND 128),
    before_revision    INTEGER CHECK (before_revision IS NULL OR before_revision > 0),
    after_revision     INTEGER NOT NULL CHECK (after_revision > 0),
    undo_of_event_id   TEXT REFERENCES face_audit_events(id) ON DELETE RESTRICT,
    created_at_ms      INTEGER NOT NULL CHECK (created_at_ms > 0)
);

CREATE INDEX face_audit_events_target
    ON face_audit_events(primary_target_id, created_at_ms DESC, id);
CREATE INDEX face_audit_events_library ON face_audit_events(library_id, created_at_ms DESC, id);

CREATE TABLE face_review_undo_snapshots (
    event_id       TEXT PRIMARY KEY REFERENCES face_audit_events(id) ON DELETE CASCADE,
    before_json    TEXT NOT NULL CHECK (length(before_json) BETWEEN 2 AND 1048576),
    after_json     TEXT NOT NULL CHECK (length(after_json) BETWEEN 2 AND 1048576),
    state          TEXT NOT NULL DEFAULT 'available' CHECK (state IN ('available','consumed')),
    revision       INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    consumed_by_event_id TEXT REFERENCES face_audit_events(id) ON DELETE RESTRICT,
    created_at_ms  INTEGER NOT NULL CHECK (created_at_ms > 0),
    CHECK ((state='available' AND consumed_by_event_id IS NULL) OR (state='consumed' AND consumed_by_event_id IS NOT NULL))
);

CREATE TABLE person_aliases (
    source_person_id TEXT PRIMARY KEY REFERENCES people(id) ON DELETE CASCADE,
    target_person_id TEXT NOT NULL REFERENCES people(id) ON DELETE RESTRICT,
    created_at_ms    INTEGER NOT NULL CHECK (created_at_ms > 0),
    CHECK (source_person_id <> target_person_id)
);

CREATE INDEX person_aliases_target ON person_aliases(target_person_id, source_person_id);

-- +goose Down
DROP INDEX IF EXISTS person_aliases_target;
DROP TABLE IF EXISTS person_aliases;
DROP INDEX IF EXISTS face_audit_events_target;
DROP INDEX IF EXISTS face_audit_events_library;
DROP TABLE IF EXISTS face_review_undo_snapshots;
DROP TABLE IF EXISTS face_audit_events;
DROP TABLE IF EXISTS face_cannot_links;
DROP INDEX IF EXISTS person_face_anchors_asset;
DROP INDEX IF EXISTS person_face_anchors_person;
DROP INDEX IF EXISTS person_face_anchors_current_face;
DROP INDEX IF EXISTS person_face_anchors_lineage;
DROP INDEX IF EXISTS face_exclusions_asset;
DROP TABLE IF EXISTS face_exclusions;
DROP TABLE IF EXISTS person_face_anchors;
DROP INDEX IF EXISTS people_page;
DROP TABLE IF EXISTS people;
DROP TABLE IF EXISTS face_cluster_members;
DROP INDEX IF EXISTS face_clusters_page;
DROP TABLE IF EXISTS face_clusters;
DROP INDEX IF EXISTS face_cluster_builds_one_active;
DROP TABLE IF EXISTS face_cluster_builds;
DROP INDEX IF EXISTS face_observations_asset;
DROP TABLE IF EXISTS face_observations;
DROP INDEX IF EXISTS face_asset_results_source;
DROP TABLE IF EXISTS face_asset_results;
DROP TABLE IF EXISTS face_library_progress;
DROP TABLE IF EXISTS face_library_settings;
DROP INDEX IF EXISTS face_generations_one_active;
DROP TABLE IF EXISTS face_generations;
