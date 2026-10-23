-- +goose Up
CREATE TABLE analysis_job_handoffs (
    kind TEXT NOT NULL CHECK (kind IN ('automatic_tags', 'semantic', 'face')),
    job_id TEXT NOT NULL,
    claim_revision INTEGER NOT NULL CHECK (claim_revision > 0),
    operation_id TEXT NOT NULL REFERENCES ai_model_operations(id) ON DELETE CASCADE,
    attempt_before INTEGER NOT NULL CHECK (attempt_before BETWEEN 1 AND 3),
    checkpoint_id INTEGER NOT NULL CHECK (checkpoint_id >= 0),
    completed_items INTEGER NOT NULL CHECK (completed_items >= 0),
    created_at_ms INTEGER NOT NULL,
    PRIMARY KEY (kind, job_id, claim_revision)
);

-- +goose Down
DROP TABLE analysis_job_handoffs;
