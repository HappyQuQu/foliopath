-- +goose Up
CREATE TABLE ai_tag_review_state (
    library_id INTEGER PRIMARY KEY REFERENCES libraries(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision>0),
    updated_at_ms INTEGER NOT NULL CHECK (updated_at_ms>0)
);

INSERT INTO ai_tag_review_state(library_id,revision,updated_at_ms)
SELECT id,1,MAX(updated_at_ms,1) FROM libraries;

-- +goose StatementBegin
CREATE TRIGGER ai_tag_review_state_library_insert
AFTER INSERT ON libraries
BEGIN
    INSERT INTO ai_tag_review_state(library_id,revision,updated_at_ms)
    VALUES(NEW.id,1,MAX(NEW.updated_at_ms,1));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER ai_tag_review_state_insert
AFTER INSERT ON ai_tag_reviews
BEGIN
    UPDATE ai_tag_review_state
    SET revision=revision+1,updated_at_ms=MAX(updated_at_ms,NEW.reviewed_at_ms)
    WHERE library_id=NEW.library_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER ai_tag_review_state_delete
AFTER DELETE ON ai_tag_reviews
BEGIN
    UPDATE ai_tag_review_state
    SET revision=revision+1
    WHERE library_id=OLD.library_id;
END;
-- +goose StatementEnd

CREATE TABLE ai_tag_review_clear_jobs (
    id TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    operation_id TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    expected_review_revision INTEGER NOT NULL CHECK (expected_review_revision>0),
    state TEXT NOT NULL CHECK (state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
    deleted_count INTEGER NOT NULL DEFAULT 0 CHECK (deleted_count>=0),
    requested_revision INTEGER NOT NULL DEFAULT 1 CHECK (requested_revision>0),
    claimed_revision INTEGER CHECK (claimed_revision IS NULL OR claimed_revision>0),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 3),
    lease_expires_ms INTEGER,
    error_code TEXT CHECK (error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
    created_at_ms INTEGER NOT NULL CHECK (created_at_ms>0),
    updated_at_ms INTEGER NOT NULL CHECK (updated_at_ms>=created_at_ms)
);

CREATE INDEX ai_tag_review_clear_jobs_claim
    ON ai_tag_review_clear_jobs(state,lease_expires_ms,created_at_ms,id);
CREATE UNIQUE INDEX ai_tag_review_clear_jobs_one_active
    ON ai_tag_review_clear_jobs(library_id)
    WHERE state IN ('queued','running','cancelling');

CREATE TABLE ai_tag_review_clear_requests (
    idempotency_key_hash TEXT PRIMARY KEY
        CHECK (length(idempotency_key_hash)=64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*'),
    request_hash TEXT NOT NULL
        CHECK (length(request_hash)=64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    job_id TEXT NOT NULL REFERENCES ai_tag_review_clear_jobs(id) ON DELETE CASCADE,
    created_at_ms INTEGER NOT NULL CHECK (created_at_ms>0)
);

CREATE INDEX ai_tag_review_clear_requests_job ON ai_tag_review_clear_requests(job_id);

-- +goose Down
DROP INDEX IF EXISTS ai_tag_review_clear_requests_job;
DROP TABLE IF EXISTS ai_tag_review_clear_requests;
DROP INDEX IF EXISTS ai_tag_review_clear_jobs_one_active;
DROP INDEX IF EXISTS ai_tag_review_clear_jobs_claim;
DROP TABLE IF EXISTS ai_tag_review_clear_jobs;
DROP TRIGGER IF EXISTS ai_tag_review_state_delete;
DROP TRIGGER IF EXISTS ai_tag_review_state_insert;
DROP TRIGGER IF EXISTS ai_tag_review_state_library_insert;
DROP TABLE IF EXISTS ai_tag_review_state;
