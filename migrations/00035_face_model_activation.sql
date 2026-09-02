-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;
BEGIN IMMEDIATE;

ALTER TABLE ai_models RENAME TO ai_models_v35;

CREATE TABLE ai_models (
    id                    TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    purpose               TEXT NOT NULL CHECK (purpose IN ('semantic_image_text', 'face_detection_embedding')),
    package_id            TEXT NOT NULL CHECK (length(package_id) BETWEEN 1 AND 128),
    version               TEXT NOT NULL CHECK (length(version) BETWEEN 1 AND 64),
    architecture          TEXT NOT NULL CHECK (architecture IN ('amd64', 'arm64')),
    content_hash          TEXT NOT NULL CHECK (length(content_hash) = 64 AND content_hash NOT GLOB '*[^0-9a-f]*'),
    license_id            TEXT NOT NULL CHECK (length(license_id) BETWEEN 1 AND 128),
    package_size_bytes    INTEGER NOT NULL CHECK (package_size_bytes > 0),
    storage_mode          TEXT NOT NULL CHECK (storage_mode IN ('managed', 'direct')),
    state                 TEXT NOT NULL CHECK (state IN ('installed', 'available', 'unavailable')),
    source_identity       TEXT NOT NULL CHECK (length(source_identity) BETWEEN 1 AND 256),
    availability_revision INTEGER NOT NULL DEFAULT 1 CHECK (availability_revision > 0),
    created_at_ms         INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms         INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    UNIQUE(purpose, package_id, version, architecture, content_hash)
);

INSERT INTO ai_models SELECT * FROM ai_models_v35;
DROP TABLE ai_models_v35;
CREATE INDEX ai_models_purpose_state
    ON ai_models(purpose, state, package_id, version, id);

ALTER TABLE face_generations ADD COLUMN model_id TEXT REFERENCES ai_models(id) ON DELETE RESTRICT;
ALTER TABLE face_generations ADD COLUMN package_id TEXT
    CHECK (package_id IS NULL OR length(package_id) BETWEEN 1 AND 128);
ALTER TABLE face_generations ADD COLUMN threshold_profile_hash TEXT
    CHECK (threshold_profile_hash IS NULL OR (
        length(threshold_profile_hash) = 64 AND threshold_profile_hash NOT GLOB '*[^0-9a-f]*'
    ));
CREATE INDEX face_generations_model_state
    ON face_generations(model_id, state, created_at_ms DESC, id);

COMMIT;
PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;
BEGIN IMMEDIATE;

DROP INDEX IF EXISTS face_generations_model_state;
ALTER TABLE face_generations DROP COLUMN threshold_profile_hash;
ALTER TABLE face_generations DROP COLUMN package_id;
ALTER TABLE face_generations DROP COLUMN model_id;

ALTER TABLE ai_models RENAME TO ai_models_v35;

CREATE TABLE ai_models (
    id                    TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    purpose               TEXT NOT NULL CHECK (purpose = 'semantic_image_text'),
    package_id            TEXT NOT NULL CHECK (length(package_id) BETWEEN 1 AND 128),
    version               TEXT NOT NULL CHECK (length(version) BETWEEN 1 AND 64),
    architecture          TEXT NOT NULL CHECK (architecture IN ('amd64', 'arm64')),
    content_hash          TEXT NOT NULL CHECK (length(content_hash) = 64 AND content_hash NOT GLOB '*[^0-9a-f]*'),
    license_id            TEXT NOT NULL CHECK (length(license_id) BETWEEN 1 AND 128),
    package_size_bytes    INTEGER NOT NULL CHECK (package_size_bytes > 0),
    storage_mode          TEXT NOT NULL CHECK (storage_mode IN ('managed', 'direct')),
    state                 TEXT NOT NULL CHECK (state IN ('installed', 'available', 'unavailable')),
    source_identity       TEXT NOT NULL CHECK (length(source_identity) BETWEEN 1 AND 256),
    availability_revision INTEGER NOT NULL DEFAULT 1 CHECK (availability_revision > 0),
    created_at_ms         INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms         INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    UNIQUE(purpose, package_id, version, architecture, content_hash)
);

INSERT INTO ai_models SELECT * FROM ai_models_v35;
DROP TABLE ai_models_v35;
CREATE INDEX ai_models_purpose_state
    ON ai_models(purpose, state, package_id, version, id);

COMMIT;
PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;
