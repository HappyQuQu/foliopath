-- +goose Up
-- UIF-S2 adds durable state required by the accepted frontend-fidelity
-- contracts. Original media remains read-only; all added state is application
-- data or reconstructible cache metadata.
ALTER TABLE users
    ADD COLUMN revision INTEGER NOT NULL DEFAULT 1
        CHECK (revision > 0);

ALTER TABLE directories
    ADD COLUMN search_name_key TEXT NOT NULL DEFAULT '';

CREATE INDEX directories_filter_children
    ON directories(library_id, parent_id, search_name_key, natural_name_key, name, id);

DROP INDEX IF EXISTS idempotency_records_expiry;
ALTER TABLE idempotency_records RENAME TO idempotency_records_before_uif;

CREATE TABLE idempotency_records (
    id                  INTEGER PRIMARY KEY,
    operation           TEXT NOT NULL
                            CHECK (operation IN (
                                'create_library',
                                'remove_library',
                                'cache_cleanup'
                            )),
    key_hash            BLOB NOT NULL CHECK (length(key_hash) = 32),
    request_hash        BLOB NOT NULL CHECK (length(request_hash) = 32),
    result_kind         TEXT NOT NULL
                            CHECK (result_kind IN (
                                'library',
                                'library_removal',
                                'cache_cleanup'
                            )),
    result_id           INTEGER NOT NULL CHECK (result_id > 0),
    created_at_ms       INTEGER NOT NULL,
    expires_at_ms       INTEGER NOT NULL,
    CHECK (expires_at_ms >= created_at_ms + 86400000),
    UNIQUE (operation, key_hash)
);

INSERT INTO idempotency_records (
    id, operation, key_hash, request_hash, result_kind, result_id,
    created_at_ms, expires_at_ms
)
SELECT
    id, operation, key_hash, request_hash, result_kind, result_id,
    created_at_ms, expires_at_ms
FROM idempotency_records_before_uif;

DROP TABLE idempotency_records_before_uif;

CREATE INDEX idempotency_records_expiry
    ON idempotency_records(expires_at_ms, id);

CREATE TABLE cache_cleanup_state (
    singleton_key          INTEGER PRIMARY KEY CHECK (singleton_key = 1),
    revision               INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    status                 TEXT NOT NULL DEFAULT 'idle'
                             CHECK (status IN ('idle', 'queued', 'running', 'succeeded', 'failed')),
    idempotency_key_hash   BLOB CHECK (
                             idempotency_key_hash IS NULL OR
                             length(idempotency_key_hash) = 32
                           ),
    requested_at_ms        INTEGER,
    started_at_ms          INTEGER,
    finished_at_ms         INTEGER,
    initial_usage_bytes    INTEGER NOT NULL DEFAULT 0 CHECK (initial_usage_bytes >= 0),
    remaining_usage_bytes  INTEGER NOT NULL DEFAULT 0 CHECK (remaining_usage_bytes >= 0),
    reclaimed_bytes        INTEGER NOT NULL DEFAULT 0 CHECK (reclaimed_bytes >= 0),
    deleted_entries        INTEGER NOT NULL DEFAULT 0 CHECK (deleted_entries >= 0),
    error_code             TEXT
                             CHECK (error_code IS NULL OR error_code IN (
                                 'cache_unavailable',
                                 'storage_unavailable',
                                 'internal_error'
                             )),
    CHECK (requested_at_ms IS NULL OR requested_at_ms >= 0),
    CHECK (started_at_ms IS NULL OR started_at_ms >= requested_at_ms),
    CHECK (finished_at_ms IS NULL OR finished_at_ms >= COALESCE(started_at_ms, requested_at_ms))
);

INSERT INTO cache_cleanup_state (singleton_key) VALUES (1);

-- +goose Down
DROP TABLE IF EXISTS cache_cleanup_state;
DROP INDEX IF EXISTS idempotency_records_expiry;
ALTER TABLE idempotency_records RENAME TO idempotency_records_with_cache;

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

INSERT INTO idempotency_records (
    id, operation, key_hash, request_hash, result_kind, result_id,
    created_at_ms, expires_at_ms
)
SELECT
    id, operation, key_hash, request_hash, result_kind, result_id,
    created_at_ms, expires_at_ms
FROM idempotency_records_with_cache
WHERE operation IN ('create_library', 'remove_library');

DROP TABLE idempotency_records_with_cache;

CREATE INDEX idempotency_records_expiry
    ON idempotency_records(expires_at_ms, id);

DROP INDEX IF EXISTS directories_filter_children;
ALTER TABLE directories DROP COLUMN search_name_key;
ALTER TABLE users DROP COLUMN revision;
