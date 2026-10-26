-- +goose Up
ALTER TABLE ocr_jobs ADD COLUMN next_attempt_ms INTEGER NOT NULL DEFAULT 0 CHECK(next_attempt_ms>=0);
CREATE INDEX ocr_jobs_library_created ON ocr_jobs(library_id,created_at_ms DESC,id DESC);
CREATE TABLE ocr_job_handoffs (
 job_id TEXT NOT NULL REFERENCES ocr_jobs(id) ON DELETE CASCADE,
 claim_revision INTEGER NOT NULL CHECK(claim_revision>0),
 attempt_before INTEGER NOT NULL CHECK(attempt_before BETWEEN 1 AND 3),
 checkpoint_id INTEGER NOT NULL CHECK(checkpoint_id>=0),
 completed INTEGER NOT NULL CHECK(completed>=0),
 created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0),
 PRIMARY KEY(job_id,claim_revision)
);
-- +goose Down
DROP TABLE ocr_job_handoffs;
DROP INDEX ocr_jobs_library_created;
ALTER TABLE ocr_jobs DROP COLUMN next_attempt_ms;
