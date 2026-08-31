-- +goose Up
CREATE TABLE semantic_clear_jobs (
    id                 TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    library_id         INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    operation_id       TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    state              TEXT NOT NULL CHECK (state IN ('queued', 'running', 'cancelling', 'succeeded', 'failed', 'cancelled')),
    requested_revision INTEGER NOT NULL DEFAULT 1 CHECK (requested_revision > 0),
    claimed_revision   INTEGER CHECK (claimed_revision IS NULL OR claimed_revision > 0),
    attempt_count      INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 3),
    lease_expires_ms   INTEGER,
    error_code         TEXT CHECK (error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    created_at_ms      INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms      INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms)
);

CREATE UNIQUE INDEX semantic_clear_jobs_one_active_library
    ON semantic_clear_jobs(library_id)
    WHERE state IN ('queued', 'running', 'cancelling');
CREATE INDEX semantic_clear_jobs_claim
    ON semantic_clear_jobs(state, lease_expires_ms, created_at_ms, id);

CREATE TABLE semantic_clear_requests (
    idempotency_key_hash TEXT PRIMARY KEY
        CHECK (length(idempotency_key_hash) = 64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*'),
    request_hash         TEXT NOT NULL
        CHECK (length(request_hash) = 64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    job_id               TEXT NOT NULL REFERENCES semantic_clear_jobs(id) ON DELETE CASCADE,
    expected_settings_revision INTEGER NOT NULL CHECK (expected_settings_revision > 0),
    created_at_ms        INTEGER NOT NULL CHECK (created_at_ms > 0)
);

CREATE INDEX semantic_clear_requests_job ON semantic_clear_requests(job_id);

-- +goose Down
DROP INDEX IF EXISTS semantic_clear_requests_job;
DROP TABLE IF EXISTS semantic_clear_requests;
DROP INDEX IF EXISTS semantic_clear_jobs_claim;
DROP INDEX IF EXISTS semantic_clear_jobs_one_active_library;
DROP TABLE IF EXISTS semantic_clear_jobs;
