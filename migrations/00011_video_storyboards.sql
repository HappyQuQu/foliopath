-- +goose Up
CREATE TABLE thumbnails_v11 (
    library_id          INTEGER NOT NULL,
    asset_id            INTEGER NOT NULL,
    variant             TEXT NOT NULL
                        CHECK (variant IN ('grid', 'storyboard')),
    source_fingerprint  TEXT NOT NULL
                        CHECK (
                            length(source_fingerprint) BETWEEN 6 AND 64
                            AND substr(source_fingerprint, 1, 3) = 'v1:'
                        ),
    transform_version   INTEGER NOT NULL CHECK (transform_version > 0),
    cache_rel_path      TEXT,
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'ready', 'failed')),
    error_code          TEXT
                        CHECK (
                            error_code IS NULL
                            OR error_code IN (
                                'unsupported_media',
                                'invalid_media',
                                'media_processing_failed',
                                'media_processing_timeout'
                            )
                        ),
    width               INTEGER CHECK (width IS NULL OR width > 0),
    height              INTEGER CHECK (height IS NULL OR height > 0),
    byte_size           INTEGER CHECK (byte_size IS NULL OR byte_size > 0),
    created_at_ms       INTEGER,
    last_accessed_at_ms INTEGER,
    frame_count         INTEGER,
    sprite_columns      INTEGER,
    sprite_rows         INTEGER,
    cell_width          INTEGER,
    cell_height         INTEGER,
    PRIMARY KEY (asset_id, variant),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE,
    CHECK (
        (
            status = 'ready'
            AND cache_rel_path IS NOT NULL
            AND error_code IS NULL
            AND width IS NOT NULL
            AND height IS NOT NULL
            AND byte_size IS NOT NULL
            AND created_at_ms IS NOT NULL
            AND last_accessed_at_ms IS NOT NULL
        )
        OR (
            status = 'pending'
            AND cache_rel_path IS NULL
            AND error_code IS NULL
            AND width IS NULL
            AND height IS NULL
            AND byte_size IS NULL
            AND created_at_ms IS NULL
            AND last_accessed_at_ms IS NULL
        )
        OR (
            status = 'failed'
            AND cache_rel_path IS NULL
            AND error_code IS NOT NULL
            AND width IS NULL
            AND height IS NULL
            AND byte_size IS NULL
            AND created_at_ms IS NULL
            AND last_accessed_at_ms IS NULL
        )
    ),
    CHECK (
        (
            variant = 'grid'
            AND frame_count IS NULL
            AND sprite_columns IS NULL
            AND sprite_rows IS NULL
            AND cell_width IS NULL
            AND cell_height IS NULL
        )
        OR (
            variant = 'storyboard'
            AND status <> 'ready'
            AND frame_count IS NULL
            AND sprite_columns IS NULL
            AND sprite_rows IS NULL
            AND cell_width IS NULL
            AND cell_height IS NULL
        )
        OR (
            variant = 'storyboard'
            AND status = 'ready'
            AND frame_count IN (4, 10)
            AND sprite_columns =
                CASE WHEN frame_count < 5 THEN frame_count ELSE 5 END
            AND sprite_rows =
                (frame_count + sprite_columns - 1) / sprite_columns
            AND sprite_rows BETWEEN 1 AND 2
            AND cell_width BETWEEN 1 AND 320
            AND cell_height BETWEEN 1 AND 320
            AND width = sprite_columns * cell_width
            AND height = sprite_rows * cell_height
        )
    )
);

INSERT INTO thumbnails_v11(
    library_id, asset_id, variant, source_fingerprint, transform_version,
    cache_rel_path, status, error_code, width, height, byte_size,
    created_at_ms, last_accessed_at_ms
)
SELECT
    library_id, asset_id, variant, source_fingerprint, transform_version,
    cache_rel_path, status, error_code, width, height, byte_size,
    created_at_ms, last_accessed_at_ms
FROM thumbnails;

DROP TABLE thumbnails;
ALTER TABLE thumbnails_v11 RENAME TO thumbnails;

CREATE INDEX thumbnails_library_status
    ON thumbnails(library_id, status, asset_id);
CREATE INDEX thumbnails_lru
    ON thumbnails(status, last_accessed_at_ms, asset_id)
    WHERE status = 'ready';

CREATE TABLE media_jobs_v11 (
    id                   INTEGER PRIMARY KEY,
    library_id           INTEGER NOT NULL,
    asset_id             INTEGER NOT NULL,
    variant              TEXT NOT NULL
                         CHECK (variant IN ('grid', 'storyboard')),
    priority             INTEGER NOT NULL DEFAULT 0
                         CHECK (
                             (variant = 'grid' AND priority = 0)
                             OR (variant = 'storyboard' AND priority = 100)
                         ),
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

INSERT INTO media_jobs_v11(
    id, library_id, asset_id, variant, priority, transform_version,
    source_fingerprint, status, last_error_code, available_at_ms,
    started_at_ms, heartbeat_at_ms, lease_expires_at_ms, attempt_count,
    created_at_ms, finished_at_ms
)
SELECT
    id, library_id, asset_id, variant, 0, transform_version,
    source_fingerprint, status, last_error_code, available_at_ms,
    started_at_ms, heartbeat_at_ms, lease_expires_at_ms, attempt_count,
    created_at_ms, finished_at_ms
FROM media_jobs;

DROP TABLE media_jobs;
ALTER TABLE media_jobs_v11 RENAME TO media_jobs;

CREATE INDEX media_jobs_ready_queue
    ON media_jobs(status, priority, available_at_ms, library_id, id);
CREATE INDEX media_jobs_expired_lease
    ON media_jobs(status, lease_expires_at_ms, id);
CREATE INDEX media_jobs_library
    ON media_jobs(library_id, status, priority, id);

CREATE TABLE media_job_library_state_v11 (
    library_id          INTEGER NOT NULL
                        REFERENCES libraries(id) ON DELETE CASCADE,
    priority            INTEGER NOT NULL CHECK (priority IN (0, 100)),
    last_claim_sequence INTEGER NOT NULL DEFAULT 0
                        CHECK (last_claim_sequence >= 0),
    PRIMARY KEY (library_id, priority)
);

INSERT INTO media_job_library_state_v11(
    library_id, priority, last_claim_sequence
)
SELECT library_id, 0, last_claim_sequence
FROM media_job_library_state;

DROP TABLE media_job_library_state;
ALTER TABLE media_job_library_state_v11 RENAME TO media_job_library_state;

-- +goose Down
-- A downgrade is safe only before storyboard state exists. The guard CHECK
-- makes Goose roll the whole migration back instead of discarding derived rows.
CREATE TABLE video_storyboard_downgrade_guard (
    storyboard_rows INTEGER NOT NULL CHECK (storyboard_rows = 0)
);
INSERT INTO video_storyboard_downgrade_guard(storyboard_rows)
SELECT
    (SELECT count(*) FROM thumbnails WHERE variant = 'storyboard')
    + (SELECT count(*) FROM media_jobs WHERE variant = 'storyboard')
    + (SELECT count(*) FROM media_job_library_state WHERE priority = 100);
DROP TABLE video_storyboard_downgrade_guard;

CREATE TABLE media_job_library_state_v10 (
    library_id          INTEGER PRIMARY KEY REFERENCES libraries(id) ON DELETE CASCADE,
    last_claim_sequence INTEGER NOT NULL DEFAULT 0 CHECK (last_claim_sequence >= 0)
);
INSERT INTO media_job_library_state_v10(library_id, last_claim_sequence)
SELECT library_id, last_claim_sequence
FROM media_job_library_state
WHERE priority = 0;
DROP TABLE media_job_library_state;
ALTER TABLE media_job_library_state_v10 RENAME TO media_job_library_state;

CREATE TABLE thumbnails_v10 (
    library_id          INTEGER NOT NULL,
    asset_id            INTEGER NOT NULL,
    variant             TEXT NOT NULL CHECK (variant = 'grid'),
    source_fingerprint  TEXT NOT NULL
                        CHECK (
                            length(source_fingerprint) BETWEEN 6 AND 64
                            AND substr(source_fingerprint, 1, 3) = 'v1:'
                        ),
    transform_version   INTEGER NOT NULL CHECK (transform_version > 0),
    cache_rel_path      TEXT,
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'ready', 'failed')),
    error_code          TEXT
                        CHECK (
                            error_code IS NULL
                            OR error_code IN (
                                'unsupported_media',
                                'invalid_media',
                                'media_processing_failed',
                                'media_processing_timeout'
                            )
                        ),
    width               INTEGER CHECK (width IS NULL OR width > 0),
    height              INTEGER CHECK (height IS NULL OR height > 0),
    byte_size           INTEGER CHECK (byte_size IS NULL OR byte_size > 0),
    created_at_ms       INTEGER,
    last_accessed_at_ms INTEGER,
    PRIMARY KEY (asset_id, variant),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE,
    CHECK (
        (
            status = 'ready'
            AND cache_rel_path IS NOT NULL
            AND error_code IS NULL
            AND width IS NOT NULL
            AND height IS NOT NULL
            AND byte_size IS NOT NULL
            AND created_at_ms IS NOT NULL
            AND last_accessed_at_ms IS NOT NULL
        )
        OR (
            status = 'pending'
            AND cache_rel_path IS NULL
            AND error_code IS NULL
            AND width IS NULL
            AND height IS NULL
            AND byte_size IS NULL
            AND created_at_ms IS NULL
            AND last_accessed_at_ms IS NULL
        )
        OR (
            status = 'failed'
            AND cache_rel_path IS NULL
            AND error_code IS NOT NULL
            AND width IS NULL
            AND height IS NULL
            AND byte_size IS NULL
            AND created_at_ms IS NULL
            AND last_accessed_at_ms IS NULL
        )
    )
);

INSERT INTO thumbnails_v10
SELECT
    library_id, asset_id, variant, source_fingerprint, transform_version,
    cache_rel_path, status, error_code, width, height, byte_size,
    created_at_ms, last_accessed_at_ms
FROM thumbnails;
DROP TABLE thumbnails;
ALTER TABLE thumbnails_v10 RENAME TO thumbnails;
CREATE INDEX thumbnails_library_status
    ON thumbnails(library_id, status, asset_id);
CREATE INDEX thumbnails_lru
    ON thumbnails(status, last_accessed_at_ms, asset_id)
    WHERE status = 'ready';

CREATE TABLE media_jobs_v10 (
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

INSERT INTO media_jobs_v10(
    id, library_id, asset_id, variant, transform_version, source_fingerprint,
    status, last_error_code, available_at_ms, started_at_ms, heartbeat_at_ms,
    lease_expires_at_ms, attempt_count, created_at_ms, finished_at_ms
)
SELECT
    id, library_id, asset_id, variant, transform_version, source_fingerprint,
    status, last_error_code, available_at_ms, started_at_ms, heartbeat_at_ms,
    lease_expires_at_ms, attempt_count, created_at_ms, finished_at_ms
FROM media_jobs;
DROP TABLE media_jobs;
ALTER TABLE media_jobs_v10 RENAME TO media_jobs;
CREATE INDEX media_jobs_ready_queue
    ON media_jobs(status, available_at_ms, id);
CREATE INDEX media_jobs_expired_lease
    ON media_jobs(status, lease_expires_at_ms, id);
CREATE INDEX media_jobs_library
    ON media_jobs(library_id, status, id);
