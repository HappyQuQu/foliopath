-- +goose Up
CREATE TABLE semantic_tag_asset_progress (
    generation_id TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    library_id INTEGER NOT NULL,
    asset_id INTEGER NOT NULL,
    vocabulary_snapshot_id TEXT NOT NULL REFERENCES ai_tag_vocabulary_snapshots(id) ON DELETE CASCADE,
    source_fingerprint TEXT NOT NULL CHECK(length(source_fingerprint) BETWEEN 1 AND 256),
    outcome TEXT NOT NULL CHECK(outcome IN ('ready','degraded','failed','stale')),
    updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>0),
    PRIMARY KEY(generation_id,library_id,asset_id,vocabulary_snapshot_id),
    FOREIGN KEY(library_id,asset_id) REFERENCES assets(library_id,id) ON DELETE CASCADE
);

CREATE INDEX semantic_tag_asset_progress_library
    ON semantic_tag_asset_progress(generation_id,library_id,vocabulary_snapshot_id,outcome,asset_id);

CREATE TABLE semantic_tag_library_progress (
    generation_id TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    vocabulary_snapshot_id TEXT NOT NULL REFERENCES ai_tag_vocabulary_snapshots(id) ON DELETE CASCADE,
    eligible_count INTEGER NOT NULL CHECK(eligible_count>=0),
    ready_count INTEGER NOT NULL DEFAULT 0 CHECK(ready_count>=0),
    degraded_count INTEGER NOT NULL DEFAULT 0 CHECK(degraded_count>=0),
    failed_count INTEGER NOT NULL DEFAULT 0 CHECK(failed_count>=0),
    stale_count INTEGER NOT NULL DEFAULT 0 CHECK(stale_count>=0),
    checkpoint_id INTEGER NOT NULL DEFAULT 0 CHECK(checkpoint_id>=0),
    revision INTEGER NOT NULL DEFAULT 1 CHECK(revision>0),
    updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>0),
    PRIMARY KEY(generation_id,library_id,vocabulary_snapshot_id),
    CHECK(ready_count+degraded_count+failed_count+stale_count<=eligible_count)
);

CREATE TABLE semantic_tag_jobs (
    id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 8 AND 128),
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    generation_id TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    vocabulary_snapshot_id TEXT NOT NULL REFERENCES ai_tag_vocabulary_snapshots(id) ON DELETE CASCADE,
    operation_id TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    mode TEXT NOT NULL CHECK(mode IN ('missing','all')),
    state TEXT NOT NULL CHECK(state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
    checkpoint_id INTEGER NOT NULL DEFAULT 0 CHECK(checkpoint_id>=0),
    requested_revision INTEGER NOT NULL DEFAULT 1 CHECK(requested_revision>0),
    claimed_revision INTEGER CHECK(claimed_revision IS NULL OR claimed_revision>0),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count BETWEEN 0 AND 3),
    lease_expires_ms INTEGER,
    error_code TEXT CHECK(error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0),
    updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>=created_at_ms)
);

CREATE INDEX semantic_tag_jobs_claim ON semantic_tag_jobs(state,lease_expires_ms,created_at_ms,id);
CREATE UNIQUE INDEX semantic_tag_jobs_one_active ON semantic_tag_jobs(library_id)
    WHERE state IN ('queued','running','cancelling');

CREATE TABLE semantic_tag_job_requests (
    idempotency_key_hash TEXT PRIMARY KEY CHECK(length(idempotency_key_hash)=64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*'),
    request_hash TEXT NOT NULL CHECK(length(request_hash)=64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    job_id TEXT NOT NULL REFERENCES semantic_tag_jobs(id) ON DELETE CASCADE,
    eligible_count INTEGER NOT NULL CHECK(eligible_count>=0),
    created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0)
);

CREATE INDEX semantic_tag_job_requests_job ON semantic_tag_job_requests(job_id);

-- +goose Down
DROP INDEX IF EXISTS semantic_tag_job_requests_job;
DROP TABLE IF EXISTS semantic_tag_job_requests;
DROP INDEX IF EXISTS semantic_tag_jobs_one_active;
DROP INDEX IF EXISTS semantic_tag_jobs_claim;
DROP TABLE IF EXISTS semantic_tag_jobs;
DROP TABLE IF EXISTS semantic_tag_library_progress;
DROP INDEX IF EXISTS semantic_tag_asset_progress_library;
DROP TABLE IF EXISTS semantic_tag_asset_progress;
