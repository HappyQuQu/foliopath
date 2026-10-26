-- +goose Up
CREATE TABLE face_asset_failures (
    generation_id TEXT NOT NULL REFERENCES face_generations(id) ON DELETE CASCADE,
    library_id INTEGER NOT NULL,
    asset_id INTEGER NOT NULL,
    job_id TEXT NOT NULL REFERENCES face_analysis_jobs(id) ON DELETE CASCADE,
    source_fingerprint TEXT NOT NULL CHECK(length(source_fingerprint) BETWEEN 1 AND 256),
    error_code TEXT NOT NULL CHECK(error_code IN ('media_unreadable','invalid_media','inference_invalid_output','inference_timeout','analysis_failed')),
    updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>0),
    PRIMARY KEY(generation_id,library_id,asset_id),
    FOREIGN KEY(library_id,asset_id) REFERENCES assets(library_id,id) ON DELETE CASCADE
);
CREATE INDEX face_failures_job ON face_asset_failures(job_id);
CREATE INDEX face_jobs_diagnostics ON face_analysis_jobs(library_id,generation_id,created_at_ms DESC,id DESC);

-- +goose Down
DROP TABLE face_asset_failures;
DROP INDEX face_jobs_diagnostics;
