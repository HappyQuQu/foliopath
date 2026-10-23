-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;
BEGIN IMMEDIATE;

ALTER TABLE ai_model_operations RENAME TO ai_model_operations_v27;

CREATE TABLE ai_model_operations (
    id               TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    kind             TEXT NOT NULL CHECK (kind IN (
                         'model_install', 'model_activate',
                         'semantic_missing', 'semantic_rebuild', 'semantic_clear',
                         'tag_suggestion_missing', 'tag_suggestion_rebuild', 'tag_review_clear',
                         'video_semantic_missing', 'video_semantic_rebuild'
                     )),
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

INSERT INTO ai_model_operations(
    id, kind, state, phase, model_id, library_id, completed_items, total_items,
    error_code, lease_expires_ms, revision, created_at_ms, updated_at_ms, finished_at_ms
)
SELECT
    id, kind, state, phase, model_id, library_id, completed_items, total_items,
    error_code, lease_expires_ms, revision, created_at_ms, updated_at_ms, finished_at_ms
FROM ai_model_operations_v27;

DROP TABLE ai_model_operations_v27;

CREATE INDEX ai_model_operations_state
    ON ai_model_operations(state, created_at_ms, id);
CREATE INDEX ai_model_operations_library
    ON ai_model_operations(library_id, updated_at_ms DESC, id);

CREATE TABLE semantic_video_jobs (
    id                 TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    library_id         INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    generation_id      TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    operation_id       TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    mode               TEXT NOT NULL CHECK (mode IN ('missing', 'all')),
    state              TEXT NOT NULL CHECK (state IN ('queued', 'running', 'cancelling', 'succeeded', 'failed', 'cancelled')),
    checkpoint_id      INTEGER NOT NULL DEFAULT 0 CHECK (checkpoint_id >= 0),
    requested_revision INTEGER NOT NULL DEFAULT 1 CHECK (requested_revision > 0),
    claimed_revision   INTEGER CHECK (claimed_revision IS NULL OR claimed_revision > 0),
    attempt_count      INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 3),
    lease_expires_ms   INTEGER,
    error_code         TEXT CHECK (error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    created_at_ms      INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms      INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms)
);

CREATE INDEX semantic_video_jobs_claim
    ON semantic_video_jobs(state, lease_expires_ms, created_at_ms, id);
CREATE UNIQUE INDEX semantic_video_jobs_one_active_mode
    ON semantic_video_jobs(library_id, generation_id, mode)
    WHERE state IN ('queued', 'running', 'cancelling');

CREATE TABLE semantic_video_job_requests (
    idempotency_key_hash TEXT PRIMARY KEY
        CHECK (length(idempotency_key_hash) = 64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*'),
    request_hash         TEXT NOT NULL
        CHECK (length(request_hash) = 64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    job_id               TEXT NOT NULL REFERENCES semantic_video_jobs(id) ON DELETE CASCADE,
    created_at_ms        INTEGER NOT NULL CHECK (created_at_ms > 0)
);

CREATE INDEX semantic_video_job_requests_job
    ON semantic_video_job_requests(job_id);

COMMIT;
PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;
BEGIN IMMEDIATE;

DROP INDEX IF EXISTS semantic_video_job_requests_job;
DROP TABLE IF EXISTS semantic_video_job_requests;
DROP INDEX IF EXISTS semantic_video_jobs_one_active_mode;
DROP INDEX IF EXISTS semantic_video_jobs_claim;
DROP TABLE IF EXISTS semantic_video_jobs;

ALTER TABLE ai_model_operations RENAME TO ai_model_operations_v27;

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

INSERT INTO ai_model_operations(
    id, kind, state, phase, model_id, library_id, completed_items, total_items,
    error_code, lease_expires_ms, revision, created_at_ms, updated_at_ms, finished_at_ms
)
SELECT
    id, kind, state, phase, model_id, library_id, completed_items, total_items,
    error_code, lease_expires_ms, revision, created_at_ms, updated_at_ms, finished_at_ms
FROM ai_model_operations_v27
WHERE kind IN ('model_install', 'model_activate', 'semantic_missing', 'semantic_rebuild', 'semantic_clear');

DROP TABLE ai_model_operations_v27;

CREATE INDEX ai_model_operations_state
    ON ai_model_operations(state, created_at_ms, id);
CREATE INDEX ai_model_operations_library
    ON ai_model_operations(library_id, updated_at_ms DESC, id);

COMMIT;
PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;
