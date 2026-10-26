-- +goose Up
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

CREATE INDEX ai_models_purpose_state
    ON ai_models(purpose, state, package_id, version, id);

CREATE TABLE semantic_generations (
    id                       TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    model_id                 TEXT NOT NULL REFERENCES ai_models(id) ON DELETE RESTRICT,
    transform_version        INTEGER NOT NULL CHECK (transform_version > 0),
    output_schema_version    INTEGER NOT NULL CHECK (output_schema_version > 0),
    index_format_version     INTEGER NOT NULL CHECK (index_format_version > 0),
    embedding_dimension      INTEGER NOT NULL CHECK (embedding_dimension BETWEEN 1 AND 65536),
    state                    TEXT NOT NULL CHECK (state IN ('building', 'ready', 'active', 'retired', 'failed')),
    created_at_ms            INTEGER NOT NULL CHECK (created_at_ms > 0),
    activated_at_ms          INTEGER CHECK (activated_at_ms IS NULL OR activated_at_ms >= created_at_ms),
    updated_at_ms            INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms)
);

CREATE UNIQUE INDEX semantic_generations_one_active
    ON semantic_generations(state) WHERE state = 'active';
CREATE INDEX semantic_generations_model_state
    ON semantic_generations(model_id, state, created_at_ms DESC, id);

CREATE TABLE ai_model_state (
    singleton_key        INTEGER PRIMARY KEY CHECK (singleton_key = 1),
    revision             INTEGER NOT NULL CHECK (revision > 0),
    active_model_id      TEXT REFERENCES ai_models(id) ON DELETE RESTRICT,
    active_generation_id TEXT REFERENCES semantic_generations(id) ON DELETE RESTRICT,
    CHECK ((active_model_id IS NULL) = (active_generation_id IS NULL))
);

INSERT INTO ai_model_state(singleton_key, revision, active_model_id, active_generation_id)
VALUES(1, 1, NULL, NULL);

CREATE TABLE ai_model_operations (
    id               TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    kind             TEXT NOT NULL CHECK (kind IN ('model_install', 'model_activate', 'semantic_missing', 'semantic_rebuild', 'semantic_clear')),
    state            TEXT NOT NULL CHECK (state IN ('queued', 'running', 'cancelling', 'succeeded', 'failed', 'cancelled')),
    phase            TEXT NOT NULL CHECK (phase IN ('queued', 'scanning', 'verifying', 'copying', 'loading', 'building', 'validating', 'clearing', 'finalizing', 'completed')),
    model_id         TEXT REFERENCES ai_models(id) ON DELETE RESTRICT,
    library_id       INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
    completed_items  INTEGER NOT NULL DEFAULT 0 CHECK (completed_items >= 0),
    total_items      INTEGER CHECK (total_items IS NULL OR total_items >= 0),
    error_code       TEXT CHECK (error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    lease_expires_ms INTEGER,
    revision         INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at_ms    INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms    INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    finished_at_ms   INTEGER CHECK (finished_at_ms IS NULL OR finished_at_ms >= created_at_ms)
);

CREATE INDEX ai_model_operations_state
    ON ai_model_operations(state, created_at_ms, id);
CREATE INDEX ai_model_operations_library
    ON ai_model_operations(library_id, updated_at_ms DESC, id);

CREATE TABLE ai_model_install_requests (
    idempotency_key TEXT PRIMARY KEY CHECK (length(idempotency_key) BETWEEN 8 AND 128),
    request_hash    TEXT NOT NULL CHECK (length(request_hash) = 64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    operation_id   TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    candidate_id   TEXT NOT NULL CHECK (length(candidate_id) BETWEEN 8 AND 128),
    storage_mode   TEXT NOT NULL CHECK (storage_mode IN ('managed', 'direct')),
    package_json   BLOB NOT NULL CHECK (length(package_json) BETWEEN 2 AND 65536),
    manifest_json  BLOB NOT NULL CHECK (length(manifest_json) BETWEEN 2 AND 65536),
    source_identity TEXT NOT NULL CHECK (length(source_identity) BETWEEN 1 AND 256),
    created_at_ms  INTEGER NOT NULL CHECK (created_at_ms > 0)
);

CREATE INDEX ai_model_install_requests_operation
    ON ai_model_install_requests(operation_id);

CREATE TABLE ai_model_activation_requests (
    idempotency_key TEXT PRIMARY KEY CHECK (length(idempotency_key) BETWEEN 8 AND 128),
    request_hash    TEXT NOT NULL CHECK (length(request_hash) = 64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    operation_id   TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    model_id       TEXT NOT NULL REFERENCES ai_models(id) ON DELETE RESTRICT,
    expected_availability_revision INTEGER NOT NULL CHECK (expected_availability_revision > 0),
    created_at_ms  INTEGER NOT NULL CHECK (created_at_ms > 0)
);

CREATE INDEX ai_model_activation_requests_operation
    ON ai_model_activation_requests(operation_id);

CREATE TABLE ai_library_settings (
    library_id          INTEGER PRIMARY KEY REFERENCES libraries(id) ON DELETE CASCADE,
    enabled             INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    state               TEXT NOT NULL DEFAULT 'disabled' CHECK (state IN ('disabled', 'awaiting_model', 'building', 'ready', 'degraded', 'clearing')),
    revision            INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    coverage_revision   INTEGER NOT NULL DEFAULT 1 CHECK (coverage_revision > 0),
    created_at_ms       INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms       INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms)
);

CREATE TABLE semantic_library_progress (
    generation_id  TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    library_id     INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    eligible_count INTEGER NOT NULL DEFAULT 0 CHECK (eligible_count >= 0),
    completed_count INTEGER NOT NULL DEFAULT 0 CHECK (completed_count >= 0),
    failed_count   INTEGER NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
    stale_count    INTEGER NOT NULL DEFAULT 0 CHECK (stale_count >= 0),
    checkpoint_id  INTEGER NOT NULL DEFAULT 0 CHECK (checkpoint_id >= 0),
    revision       INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_at_ms  INTEGER NOT NULL CHECK (updated_at_ms > 0),
    PRIMARY KEY (generation_id, library_id),
    CHECK (completed_count + failed_count + stale_count <= eligible_count)
);

CREATE TABLE semantic_embeddings (
    generation_id     TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    library_id        INTEGER NOT NULL,
    asset_id          INTEGER NOT NULL,
    source_fingerprint TEXT NOT NULL CHECK (length(source_fingerprint) BETWEEN 1 AND 256),
    vector            BLOB NOT NULL CHECK (length(vector) > 0),
    created_at_ms     INTEGER NOT NULL CHECK (created_at_ms > 0),
    PRIMARY KEY (generation_id, library_id, asset_id),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE
);

CREATE INDEX semantic_embeddings_library_generation
    ON semantic_embeddings(library_id, generation_id, asset_id);

CREATE TABLE semantic_jobs (
    id               TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    library_id       INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    generation_id    TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    operation_id     TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    mode             TEXT NOT NULL CHECK (mode IN ('missing', 'all', 'clear')),
    state            TEXT NOT NULL CHECK (state IN ('queued', 'running', 'cancelling', 'succeeded', 'failed', 'cancelled')),
    checkpoint_id    INTEGER NOT NULL DEFAULT 0 CHECK (checkpoint_id >= 0),
    requested_revision INTEGER NOT NULL DEFAULT 1 CHECK (requested_revision > 0),
    claimed_revision INTEGER CHECK (claimed_revision IS NULL OR claimed_revision > 0),
    lease_expires_ms INTEGER,
    error_code       TEXT CHECK (error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    created_at_ms    INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms    INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms)
);

CREATE INDEX semantic_jobs_claim
    ON semantic_jobs(state, lease_expires_ms, created_at_ms, id);
CREATE UNIQUE INDEX semantic_jobs_one_active_mode
    ON semantic_jobs(library_id, generation_id, mode)
    WHERE state IN ('queued', 'running', 'cancelling');

-- +goose Down
DROP INDEX IF EXISTS ai_model_activation_requests_operation;
DROP TABLE IF EXISTS ai_model_activation_requests;
DROP INDEX IF EXISTS ai_model_install_requests_operation;
DROP TABLE IF EXISTS ai_model_install_requests;
DROP INDEX IF EXISTS semantic_jobs_one_active_mode;
DROP INDEX IF EXISTS semantic_jobs_claim;
DROP TABLE IF EXISTS semantic_jobs;
DROP INDEX IF EXISTS semantic_embeddings_library_generation;
DROP TABLE IF EXISTS semantic_embeddings;
DROP TABLE IF EXISTS semantic_library_progress;
DROP TABLE IF EXISTS ai_library_settings;
DROP INDEX IF EXISTS ai_model_operations_library;
DROP INDEX IF EXISTS ai_model_operations_state;
DROP TABLE IF EXISTS ai_model_operations;
DROP TABLE IF EXISTS ai_model_state;
DROP INDEX IF EXISTS semantic_generations_model_state;
DROP INDEX IF EXISTS semantic_generations_one_active;
DROP TABLE IF EXISTS semantic_generations;
DROP INDEX IF EXISTS ai_models_purpose_state;
DROP TABLE IF EXISTS ai_models;
