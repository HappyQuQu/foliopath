-- +goose Up
CREATE TABLE media_job_attempts (
    id INTEGER PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES media_jobs(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
    outcome TEXT NOT NULL CHECK (outcome IN ('succeeded', 'retry', 'permanent_failure')),
    stage TEXT,
    reason_code TEXT,
    tool TEXT CHECK (tool IS NULL OR tool IN ('ffmpeg', 'ffprobe', 'libvips', 'filesystem', 'cache')),
    exit_code INTEGER,
    duration_ms INTEGER NOT NULL CHECK (duration_ms >= 0),
    finished_at_ms INTEGER NOT NULL,
    UNIQUE(job_id, attempt_number, finished_at_ms)
);

CREATE INDEX media_job_attempts_job_recent_idx
    ON media_job_attempts(job_id, id DESC);

-- +goose Down
DROP INDEX media_job_attempts_job_recent_idx;
DROP TABLE media_job_attempts;
