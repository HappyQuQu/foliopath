-- +goose Up
PRAGMA foreign_keys = ON;

CREATE TABLE libraries (
    id                 INTEGER PRIMARY KEY,
    name               TEXT NOT NULL CHECK (length(trim(name)) > 0),
    name_key           TEXT NOT NULL UNIQUE,
    root_rel_path      TEXT NOT NULL UNIQUE,
    status             TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'scanning', 'ready', 'offline', 'error')),
    current_generation INTEGER NOT NULL DEFAULT 0 CHECK (current_generation >= 0),
    created_at_ms      INTEGER NOT NULL,
    updated_at_ms      INTEGER NOT NULL
);

CREATE TABLE scan_runs (
    id                     INTEGER PRIMARY KEY,
    library_id             INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    generation             INTEGER NOT NULL CHECK (generation > 0),
    trigger_kind           TEXT NOT NULL
                           CHECK (trigger_kind IN ('library_created', 'startup', 'manual', 'scheduled')),
    status                 TEXT NOT NULL
                           CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled', 'offline', 'interrupted')),
    discovered_directories INTEGER NOT NULL DEFAULT 0 CHECK (discovered_directories >= 0),
    discovered_assets      INTEGER NOT NULL DEFAULT 0 CHECK (discovered_assets >= 0),
    skipped_count          INTEGER NOT NULL DEFAULT 0 CHECK (skipped_count >= 0),
    error_code             TEXT,
    created_at_ms          INTEGER NOT NULL,
    started_at_ms          INTEGER,
    finished_at_ms         INTEGER,
    UNIQUE (library_id, generation)
);

CREATE UNIQUE INDEX scan_runs_one_active_per_library
    ON scan_runs(library_id)
    WHERE status IN ('queued', 'running');

CREATE INDEX scan_runs_library_created
    ON scan_runs(library_id, created_at_ms DESC, id DESC);

CREATE TABLE directories (
    id                    INTEGER PRIMARY KEY,
    library_id            INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    parent_id             INTEGER,
    relative_path         TEXT NOT NULL,
    name                  TEXT NOT NULL,
    mtime_ns              INTEGER NOT NULL DEFAULT 0,
    direct_asset_count    INTEGER NOT NULL DEFAULT 0 CHECK (direct_asset_count >= 0),
    recursive_asset_count INTEGER NOT NULL DEFAULT 0 CHECK (recursive_asset_count >= 0),
    last_seen_generation  INTEGER NOT NULL CHECK (last_seen_generation > 0),
    UNIQUE (library_id, relative_path),
    UNIQUE (library_id, id),
    FOREIGN KEY (library_id, parent_id)
        REFERENCES directories(library_id, id) ON DELETE CASCADE
);

CREATE INDEX directories_tree
    ON directories(library_id, parent_id, name, id);

CREATE INDEX directories_generation
    ON directories(library_id, last_seen_generation);

CREATE TABLE assets (
    id                   INTEGER PRIMARY KEY,
    library_id           INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    directory_id         INTEGER NOT NULL,
    relative_path        TEXT NOT NULL,
    name                 TEXT NOT NULL,
    kind                 TEXT NOT NULL CHECK (kind IN ('image', 'animated', 'video')),
    media_format         TEXT NOT NULL
                         CHECK (media_format IN ('jpeg', 'png', 'webp', 'gif', 'mp4', 'mov', 'mkv')),
    mime_type            TEXT NOT NULL,
    size_bytes           INTEGER NOT NULL CHECK (size_bytes >= 0),
    mtime_ns             INTEGER NOT NULL,
    last_seen_generation INTEGER NOT NULL CHECK (last_seen_generation > 0),
    UNIQUE (library_id, relative_path),
    FOREIGN KEY (library_id, directory_id)
        REFERENCES directories(library_id, id) ON DELETE CASCADE
);

CREATE INDEX assets_directory_name
    ON assets(library_id, directory_id, name, id);

CREATE INDEX assets_modified
    ON assets(library_id, mtime_ns DESC, id DESC);

CREATE INDEX assets_generation
    ON assets(library_id, last_seen_generation);

-- The root is immutable in the MVP. Keep this invariant below the service
-- layer so maintenance SQL cannot accidentally retarget a library.
-- +goose StatementBegin
CREATE TRIGGER libraries_root_is_immutable
BEFORE UPDATE OF root_rel_path ON libraries
WHEN NEW.root_rel_path <> OLD.root_rel_path
BEGIN
    SELECT RAISE(ABORT, 'library root is immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS libraries_root_is_immutable;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS directories;
DROP TABLE IF EXISTS scan_runs;
DROP TABLE IF EXISTS libraries;
