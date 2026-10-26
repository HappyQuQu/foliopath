-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;
BEGIN IMMEDIATE;

ALTER TABLE ai_model_operations RENAME TO ai_model_operations_v34;

CREATE TABLE ai_model_operations (
    id               TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    kind             TEXT NOT NULL CHECK (kind IN (
                         'model_install', 'model_activate',
                         'semantic_missing', 'semantic_rebuild', 'semantic_clear',
                         'tag_suggestion_missing', 'tag_suggestion_rebuild', 'tag_review_clear',
                         'video_semantic_missing', 'video_semantic_rebuild',
                         'face_missing', 'face_rebuild', 'face_derived_clear', 'face_manual_clear'
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

INSERT INTO ai_model_operations SELECT * FROM ai_model_operations_v34;
DROP TABLE ai_model_operations_v34;
CREATE INDEX ai_model_operations_state ON ai_model_operations(state, created_at_ms, id);
CREATE INDEX ai_model_operations_library ON ai_model_operations(library_id, updated_at_ms DESC, id);

CREATE TABLE face_analysis_jobs (
    id                 TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    library_id         INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    generation_id      TEXT NOT NULL REFERENCES face_generations(id) ON DELETE CASCADE,
    operation_id       TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    mode               TEXT NOT NULL CHECK (mode IN ('missing', 'all')),
    state              TEXT NOT NULL CHECK (state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
    checkpoint_id      INTEGER NOT NULL DEFAULT 0 CHECK (checkpoint_id >= 0),
    requested_revision INTEGER NOT NULL DEFAULT 1 CHECK (requested_revision > 0),
    claimed_revision   INTEGER CHECK (claimed_revision IS NULL OR claimed_revision > 0),
    attempt_count      INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 3),
    lease_expires_ms   INTEGER,
    error_code         TEXT CHECK (error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    created_at_ms      INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms      INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms)
);
CREATE INDEX face_analysis_jobs_claim ON face_analysis_jobs(state,lease_expires_ms,created_at_ms,id);
CREATE UNIQUE INDEX face_analysis_jobs_one_active ON face_analysis_jobs(library_id)
    WHERE state IN ('queued','running','cancelling');

CREATE TABLE face_analysis_job_requests (
    idempotency_key_hash TEXT PRIMARY KEY CHECK(length(idempotency_key_hash)=64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*'),
    request_hash TEXT NOT NULL CHECK(length(request_hash)=64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    job_id TEXT NOT NULL REFERENCES face_analysis_jobs(id) ON DELETE CASCADE,
    created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0)
);
CREATE INDEX face_analysis_job_requests_job ON face_analysis_job_requests(job_id);

CREATE TABLE face_clear_jobs (
    id                         TEXT PRIMARY KEY CHECK(length(id) BETWEEN 8 AND 128),
    library_id                 INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    operation_id               TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    kind                       TEXT NOT NULL CHECK(kind IN ('derived','manual')),
    expected_settings_revision INTEGER NOT NULL CHECK(expected_settings_revision>0),
    expected_person_count      INTEGER CHECK(expected_person_count IS NULL OR expected_person_count>=0),
    expected_assignment_count  INTEGER CHECK(expected_assignment_count IS NULL OR expected_assignment_count>=0),
    expected_constraint_count  INTEGER CHECK(expected_constraint_count IS NULL OR expected_constraint_count>=0),
    state                      TEXT NOT NULL CHECK(state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
    deleted_count              INTEGER NOT NULL DEFAULT 0 CHECK(deleted_count>=0),
    requested_revision         INTEGER NOT NULL DEFAULT 1 CHECK(requested_revision>0),
    claimed_revision           INTEGER CHECK(claimed_revision IS NULL OR claimed_revision>0),
    attempt_count              INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count BETWEEN 0 AND 3),
    lease_expires_ms           INTEGER,
    error_code                 TEXT CHECK(error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    created_at_ms              INTEGER NOT NULL CHECK(created_at_ms>0),
    updated_at_ms              INTEGER NOT NULL CHECK(updated_at_ms>=created_at_ms),
    CHECK ((kind='derived' AND expected_person_count IS NULL AND expected_assignment_count IS NULL AND expected_constraint_count IS NULL)
        OR (kind='manual' AND expected_person_count IS NOT NULL AND expected_assignment_count IS NOT NULL AND expected_constraint_count IS NOT NULL))
);
CREATE INDEX face_clear_jobs_claim ON face_clear_jobs(state,lease_expires_ms,created_at_ms,id);
CREATE UNIQUE INDEX face_clear_jobs_one_active ON face_clear_jobs(library_id)
    WHERE state IN ('queued','running','cancelling');

CREATE TABLE face_clear_requests (
    idempotency_key_hash TEXT PRIMARY KEY CHECK(length(idempotency_key_hash)=64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*'),
    request_hash TEXT NOT NULL CHECK(length(request_hash)=64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    job_id TEXT NOT NULL REFERENCES face_clear_jobs(id) ON DELETE CASCADE,
    created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0)
);
CREATE INDEX face_clear_requests_job ON face_clear_requests(job_id);

COMMIT;
PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;
BEGIN IMMEDIATE;

DROP INDEX IF EXISTS face_clear_requests_job;
DROP TABLE IF EXISTS face_clear_requests;
DROP INDEX IF EXISTS face_clear_jobs_one_active;
DROP INDEX IF EXISTS face_clear_jobs_claim;
DROP TABLE IF EXISTS face_clear_jobs;
DROP INDEX IF EXISTS face_analysis_job_requests_job;
DROP TABLE IF EXISTS face_analysis_job_requests;
DROP INDEX IF EXISTS face_analysis_jobs_one_active;
DROP INDEX IF EXISTS face_analysis_jobs_claim;
DROP TABLE IF EXISTS face_analysis_jobs;

ALTER TABLE ai_model_operations RENAME TO ai_model_operations_v34;
CREATE TABLE ai_model_operations (
    id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 8 AND 128),
    kind TEXT NOT NULL CHECK(kind IN ('model_install','model_activate','semantic_missing','semantic_rebuild','semantic_clear','tag_suggestion_missing','tag_suggestion_rebuild','tag_review_clear','video_semantic_missing','video_semantic_rebuild')),
    state TEXT NOT NULL CHECK(state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
    phase TEXT NOT NULL CHECK(phase IN ('queued','scanning','verifying','copying','loading','building','validating','clearing','finalizing','completed')),
    model_id TEXT REFERENCES ai_models(id) ON DELETE RESTRICT,
    library_id INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
    completed_items INTEGER NOT NULL DEFAULT 0 CHECK(completed_items>=0),
    total_items INTEGER CHECK(total_items IS NULL OR total_items>=0),
    error_code TEXT CHECK(error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    lease_expires_ms INTEGER,
    revision INTEGER NOT NULL DEFAULT 1 CHECK(revision>0),
    created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0),
    updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>=created_at_ms),
    finished_at_ms INTEGER CHECK(finished_at_ms IS NULL OR finished_at_ms>=created_at_ms)
);
INSERT INTO ai_model_operations SELECT * FROM ai_model_operations_v34
WHERE kind NOT IN ('face_missing','face_rebuild','face_derived_clear','face_manual_clear');
DROP TABLE ai_model_operations_v34;
CREATE INDEX ai_model_operations_state ON ai_model_operations(state,created_at_ms,id);
CREATE INDEX ai_model_operations_library ON ai_model_operations(library_id,updated_at_ms DESC,id);

COMMIT;
PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;
