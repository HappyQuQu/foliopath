-- +goose Up
ALTER TABLE semantic_jobs
    ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 3);

CREATE TABLE semantic_job_requests (
    idempotency_key_hash TEXT PRIMARY KEY
        CHECK (length(idempotency_key_hash) = 64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*'),
    request_hash         TEXT NOT NULL
        CHECK (length(request_hash) = 64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    job_id               TEXT NOT NULL REFERENCES semantic_jobs(id) ON DELETE CASCADE,
    created_at_ms        INTEGER NOT NULL CHECK (created_at_ms > 0)
);

CREATE INDEX semantic_job_requests_job ON semantic_job_requests(job_id);

-- +goose Down
DROP INDEX IF EXISTS semantic_job_requests_job;
DROP TABLE IF EXISTS semantic_job_requests;
ALTER TABLE semantic_jobs DROP COLUMN attempt_count;
