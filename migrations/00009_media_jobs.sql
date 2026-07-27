-- +goose Up
CREATE TABLE media_jobs (
    id                   INTEGER PRIMARY KEY,
    library_id           INTEGER NOT NULL,
    asset_id             INTEGER NOT NULL,
    variant              TEXT NOT NULL CHECK (variant = 'grid'),
    transform_version    INTEGER NOT NULL CHECK (transform_version > 0),
    source_fingerprint   TEXT NOT NULL
                         CHECK (
                             length(source_fingerprint) BETWEEN 6 AND 64
                             AND substr(source_fingerprint, 1, 3) = 'v1:'
                         ),
    status               TEXT NOT NULL DEFAULT 'queued'
                         CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    last_error_code      TEXT
                         CHECK (
                             last_error_code IS NULL
                             OR last_error_code IN (
                                 'invalid_media',
                                 'unsupported_media',
                                 'media_processing_failed',
                                 'media_processing_timeout',
                                 'source_unavailable',
                                 'cache_unavailable'
                             )
                         ),
    available_at_ms      INTEGER NOT NULL CHECK (available_at_ms >= 0),
    started_at_ms        INTEGER,
    heartbeat_at_ms      INTEGER,
    lease_expires_at_ms  INTEGER,
    attempt_count        INTEGER NOT NULL DEFAULT 0
                         CHECK (attempt_count BETWEEN 0 AND 3),
    created_at_ms        INTEGER NOT NULL CHECK (created_at_ms >= 0),
    finished_at_ms       INTEGER,
    UNIQUE (asset_id, variant),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE,
    CHECK (
        (
            status = 'queued'
            AND heartbeat_at_ms IS NULL
            AND lease_expires_at_ms IS NULL
            AND finished_at_ms IS NULL
        )
        OR (
            status = 'running'
            AND started_at_ms IS NOT NULL
            AND heartbeat_at_ms IS NOT NULL
            AND lease_expires_at_ms IS NOT NULL
            AND finished_at_ms IS NULL
            AND attempt_count > 0
        )
        OR (
            status IN ('succeeded', 'failed')
            AND heartbeat_at_ms IS NULL
            AND lease_expires_at_ms IS NULL
            AND finished_at_ms IS NOT NULL
        )
    )
);

CREATE INDEX media_jobs_ready_queue
    ON media_jobs(status, available_at_ms, id);
CREATE INDEX media_jobs_expired_lease
    ON media_jobs(status, lease_expires_at_ms, id);
CREATE INDEX media_jobs_library
    ON media_jobs(library_id, status, id);

CREATE TABLE media_job_library_state (
    library_id          INTEGER PRIMARY KEY REFERENCES libraries(id) ON DELETE CASCADE,
    last_claim_sequence INTEGER NOT NULL DEFAULT 0 CHECK (last_claim_sequence >= 0)
);

CREATE TABLE media_job_queue_state (
    singleton_key       INTEGER PRIMARY KEY CHECK (singleton_key = 1),
    next_claim_sequence INTEGER NOT NULL CHECK (next_claim_sequence > 0)
);

INSERT INTO media_job_queue_state(singleton_key, next_claim_sequence)
VALUES (1, 1);

CREATE TABLE cache_deletions (
    id             INTEGER PRIMARY KEY,
    library_id     INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    cache_rel_path TEXT NOT NULL UNIQUE,
    byte_size      INTEGER NOT NULL CHECK (byte_size > 0),
    created_at_ms  INTEGER NOT NULL CHECK (created_at_ms >= 0)
);

CREATE INDEX cache_deletions_created
    ON cache_deletions(created_at_ms, id);

INSERT INTO media_jobs(
    library_id, asset_id, variant, transform_version, source_fingerprint,
    status, available_at_ms, attempt_count, created_at_ms
)
SELECT library_id, id, 'grid', 1, source_fingerprint, 'queued', 0, 0, 0
FROM assets;

-- +goose Down
DROP TABLE IF EXISTS cache_deletions;
DROP TABLE IF EXISTS media_job_queue_state;
DROP TABLE IF EXISTS media_job_library_state;
DROP TABLE IF EXISTS media_jobs;
