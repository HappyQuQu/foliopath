-- +goose Up
-- Stage 2 media-library contract state. Client idempotency keys are secrets
-- only in the sense that they must not be logged or stored as replayable
-- plaintext; persist fixed-size SHA-256 digests instead.
ALTER TABLE libraries
    ADD COLUMN revision INTEGER NOT NULL DEFAULT 1
        CHECK (revision > 0);

-- A creation-triggered full scan is durable and unique for a library. The
-- library capability inserts the library, this scan row, and its idempotency
-- record in one short transaction before waking any worker.
CREATE UNIQUE INDEX scan_runs_one_creation_per_library
    ON scan_runs(library_id)
    WHERE trigger_kind = 'library_created';

-- Removal is a restart-safe application-data cleanup. It deliberately has no
-- foreign key to libraries: the terminal audit/result row must remain
-- pollable after the library configuration and derived rows are gone.
CREATE TABLE library_removals (
    id               INTEGER PRIMARY KEY,
    library_id       INTEGER NOT NULL CHECK (library_id > 0),
    library_name     TEXT NOT NULL
                         CHECK (length(trim(library_name)) BETWEEN 1 AND 128),
    status           TEXT NOT NULL
                         CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    revision         INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    error_code       TEXT,
    created_at_ms    INTEGER NOT NULL,
    started_at_ms    INTEGER,
    finished_at_ms   INTEGER,
    CHECK (started_at_ms IS NULL OR started_at_ms >= created_at_ms),
    CHECK (finished_at_ms IS NULL OR finished_at_ms >= created_at_ms),
    CHECK (
        (status = 'queued' AND started_at_ms IS NULL AND finished_at_ms IS NULL AND error_code IS NULL)
        OR
        (status = 'running' AND started_at_ms IS NOT NULL AND finished_at_ms IS NULL AND error_code IS NULL)
        OR
        (status = 'succeeded' AND started_at_ms IS NOT NULL AND finished_at_ms IS NOT NULL AND error_code IS NULL)
        OR
        (status = 'failed' AND started_at_ms IS NOT NULL AND finished_at_ms IS NOT NULL AND error_code IS NOT NULL)
    )
);

CREATE UNIQUE INDEX library_removals_one_active_per_library
    ON library_removals(library_id)
    WHERE status IN ('queued', 'running');

CREATE INDEX library_removals_created
    ON library_removals(created_at_ms DESC, id DESC);

CREATE TABLE idempotency_records (
    id                  INTEGER PRIMARY KEY,
    operation           TEXT NOT NULL
                            CHECK (operation IN ('create_library', 'remove_library')),
    key_hash            BLOB NOT NULL CHECK (length(key_hash) = 32),
    request_hash        BLOB NOT NULL CHECK (length(request_hash) = 32),
    result_kind         TEXT NOT NULL
                            CHECK (result_kind IN ('library', 'library_removal')),
    result_id           INTEGER NOT NULL CHECK (result_id > 0),
    created_at_ms       INTEGER NOT NULL,
    expires_at_ms       INTEGER NOT NULL,
    CHECK (expires_at_ms >= created_at_ms + 86400000),
    UNIQUE (operation, key_hash)
);

CREATE INDEX idempotency_records_expiry
    ON idempotency_records(expires_at_ms, id);

-- +goose Down
DROP INDEX IF EXISTS idempotency_records_expiry;
DROP TABLE IF EXISTS idempotency_records;
DROP INDEX IF EXISTS library_removals_created;
DROP INDEX IF EXISTS library_removals_one_active_per_library;
DROP TABLE IF EXISTS library_removals;
DROP INDEX IF EXISTS scan_runs_one_creation_per_library;
ALTER TABLE libraries DROP COLUMN revision;
