-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;
BEGIN IMMEDIATE;

DROP TRIGGER assets_search_insert;
DROP TRIGGER assets_search_delete;
DROP TRIGGER assets_search_update;

ALTER TABLE assets RENAME TO assets_v14;

CREATE TABLE assets (
    id                   INTEGER PRIMARY KEY,
    library_id           INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    directory_id         INTEGER NOT NULL,
    relative_path        TEXT NOT NULL,
    name                 TEXT NOT NULL,
    kind                 TEXT NOT NULL CHECK (kind IN ('image', 'animated', 'video')),
    media_format         TEXT NOT NULL
                         CHECK (media_format IN (
                             'jpeg', 'png', 'webp', 'gif',
                             'mp4', 'mov', 'mkv', 'avi'
                         )),
    mime_type            TEXT NOT NULL,
    size_bytes           INTEGER NOT NULL CHECK (size_bytes >= 0),
    mtime_ns             INTEGER NOT NULL,
    last_seen_generation INTEGER NOT NULL CHECK (last_seen_generation > 0),
    source_fingerprint   TEXT NOT NULL DEFAULT 'v1:0:0'
                         CHECK (
                             length(source_fingerprint) BETWEEN 6 AND 64
                             AND substr(source_fingerprint, 1, 3) = 'v1:'
                         ),
    natural_name_key     BLOB NOT NULL DEFAULT X'',
    width                INTEGER CHECK (width IS NULL OR width > 0),
    height               INTEGER CHECK (height IS NULL OR height > 0),
    duration_ms          INTEGER CHECK (duration_ms IS NULL OR duration_ms >= 0),
    probe_status         TEXT NOT NULL DEFAULT 'pending'
                         CHECK (probe_status IN (
                             'pending', 'ready', 'failed', 'unsupported'
                         )),
    probe_error_code     TEXT
                         CHECK (
                             probe_error_code IS NULL
                             OR probe_error_code IN (
                                 'unsupported_media',
                                 'invalid_media',
                                 'media_processing_failed',
                                 'media_processing_timeout'
                             )
                         ),
    playback_status      TEXT NOT NULL DEFAULT 'unknown'
                         CHECK (playback_status IN (
                             'playable',
                             'unsupported_codec',
                             'not_applicable',
                             'unknown'
                         )),
    search_name_key      TEXT NOT NULL DEFAULT '',
    search_path_key      TEXT NOT NULL DEFAULT '',
    UNIQUE (library_id, relative_path),
    FOREIGN KEY (library_id, directory_id)
        REFERENCES directories(library_id, id) ON DELETE CASCADE
);

INSERT INTO assets(
    id, library_id, directory_id, relative_path, name, kind, media_format,
    mime_type, size_bytes, mtime_ns, last_seen_generation,
    source_fingerprint, natural_name_key, width, height, duration_ms,
    probe_status, probe_error_code, playback_status,
    search_name_key, search_path_key
)
SELECT
    id, library_id, directory_id, relative_path, name, kind, media_format,
    mime_type, size_bytes, mtime_ns, last_seen_generation,
    source_fingerprint, natural_name_key, width, height, duration_ms,
    probe_status, probe_error_code, playback_status,
    search_name_key, search_path_key
FROM assets_v14;

DROP TABLE assets_v14;

CREATE INDEX assets_directory_name
    ON assets(library_id, directory_id, name, id);
CREATE INDEX assets_modified
    ON assets(library_id, mtime_ns DESC, id DESC);
CREATE INDEX assets_generation
    ON assets(library_id, last_seen_generation);
CREATE INDEX assets_browse_directory_name
    ON assets(library_id, directory_id, natural_name_key, name, relative_path, id);
CREATE INDEX assets_browse_library_name
    ON assets(library_id, natural_name_key, name, relative_path, id);
CREATE INDEX assets_browse_directory_modified
    ON assets(library_id, directory_id, mtime_ns DESC, id DESC);
CREATE UNIQUE INDEX assets_library_identity
    ON assets(library_id, id);
CREATE INDEX assets_search_global_name
    ON assets(natural_name_key, name, library_id, relative_path, id);
CREATE INDEX assets_search_global_modified
    ON assets(mtime_ns DESC, id DESC);
CREATE INDEX assets_browse_directory_size
    ON assets(library_id, directory_id, size_bytes, id);
CREATE INDEX assets_browse_library_size
    ON assets(library_id, size_bytes, id);
CREATE INDEX assets_search_global_size
    ON assets(size_bytes, id);

-- +goose StatementBegin
CREATE TRIGGER assets_search_insert
AFTER INSERT ON assets
BEGIN
    INSERT INTO asset_search(rowid, search_name_key, search_path_key)
    VALUES (NEW.id, NEW.search_name_key, NEW.search_path_key);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER assets_search_delete
AFTER DELETE ON assets
BEGIN
    INSERT INTO asset_search(asset_search, rowid, search_name_key, search_path_key)
    VALUES ('delete', OLD.id, OLD.search_name_key, OLD.search_path_key);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER assets_search_update
AFTER UPDATE OF search_name_key, search_path_key ON assets
WHEN NEW.search_name_key <> OLD.search_name_key
  OR NEW.search_path_key <> OLD.search_path_key
BEGIN
    INSERT INTO asset_search(asset_search, rowid, search_name_key, search_path_key)
    VALUES ('delete', OLD.id, OLD.search_name_key, OLD.search_path_key);
    INSERT INTO asset_search(rowid, search_name_key, search_path_key)
    VALUES (NEW.id, NEW.search_name_key, NEW.search_path_key);
END;
-- +goose StatementEnd

INSERT INTO asset_search(asset_search) VALUES('rebuild');

COMMIT;
PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;
BEGIN IMMEDIATE;

DELETE FROM thumbnails
WHERE asset_id IN (SELECT id FROM assets WHERE media_format = 'avi');
DELETE FROM media_jobs
WHERE asset_id IN (SELECT id FROM assets WHERE media_format = 'avi');

DROP TRIGGER assets_search_insert;
DROP TRIGGER assets_search_delete;
DROP TRIGGER assets_search_update;

ALTER TABLE assets RENAME TO assets_v14;

CREATE TABLE assets (
    id                   INTEGER PRIMARY KEY,
    library_id           INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    directory_id         INTEGER NOT NULL,
    relative_path        TEXT NOT NULL,
    name                 TEXT NOT NULL,
    kind                 TEXT NOT NULL CHECK (kind IN ('image', 'animated', 'video')),
    media_format         TEXT NOT NULL
                         CHECK (media_format IN (
                             'jpeg', 'png', 'webp', 'gif',
                             'mp4', 'mov', 'mkv'
                         )),
    mime_type            TEXT NOT NULL,
    size_bytes           INTEGER NOT NULL CHECK (size_bytes >= 0),
    mtime_ns             INTEGER NOT NULL,
    last_seen_generation INTEGER NOT NULL CHECK (last_seen_generation > 0),
    source_fingerprint   TEXT NOT NULL DEFAULT 'v1:0:0'
                         CHECK (
                             length(source_fingerprint) BETWEEN 6 AND 64
                             AND substr(source_fingerprint, 1, 3) = 'v1:'
                         ),
    natural_name_key     BLOB NOT NULL DEFAULT X'',
    width                INTEGER CHECK (width IS NULL OR width > 0),
    height               INTEGER CHECK (height IS NULL OR height > 0),
    duration_ms          INTEGER CHECK (duration_ms IS NULL OR duration_ms >= 0),
    probe_status         TEXT NOT NULL DEFAULT 'pending'
                         CHECK (probe_status IN (
                             'pending', 'ready', 'failed', 'unsupported'
                         )),
    probe_error_code     TEXT
                         CHECK (
                             probe_error_code IS NULL
                             OR probe_error_code IN (
                                 'unsupported_media',
                                 'invalid_media',
                                 'media_processing_failed',
                                 'media_processing_timeout'
                             )
                         ),
    playback_status      TEXT NOT NULL DEFAULT 'unknown'
                         CHECK (playback_status IN (
                             'playable',
                             'unsupported_codec',
                             'not_applicable',
                             'unknown'
                         )),
    search_name_key      TEXT NOT NULL DEFAULT '',
    search_path_key      TEXT NOT NULL DEFAULT '',
    UNIQUE (library_id, relative_path),
    FOREIGN KEY (library_id, directory_id)
        REFERENCES directories(library_id, id) ON DELETE CASCADE
);

INSERT INTO assets(
    id, library_id, directory_id, relative_path, name, kind, media_format,
    mime_type, size_bytes, mtime_ns, last_seen_generation,
    source_fingerprint, natural_name_key, width, height, duration_ms,
    probe_status, probe_error_code, playback_status,
    search_name_key, search_path_key
)
SELECT
    id, library_id, directory_id, relative_path, name, kind, media_format,
    mime_type, size_bytes, mtime_ns, last_seen_generation,
    source_fingerprint, natural_name_key, width, height, duration_ms,
    probe_status, probe_error_code, playback_status,
    search_name_key, search_path_key
FROM assets_v14
WHERE media_format <> 'avi';

DROP TABLE assets_v14;

CREATE INDEX assets_directory_name
    ON assets(library_id, directory_id, name, id);
CREATE INDEX assets_modified
    ON assets(library_id, mtime_ns DESC, id DESC);
CREATE INDEX assets_generation
    ON assets(library_id, last_seen_generation);
CREATE INDEX assets_browse_directory_name
    ON assets(library_id, directory_id, natural_name_key, name, relative_path, id);
CREATE INDEX assets_browse_library_name
    ON assets(library_id, natural_name_key, name, relative_path, id);
CREATE INDEX assets_browse_directory_modified
    ON assets(library_id, directory_id, mtime_ns DESC, id DESC);
CREATE UNIQUE INDEX assets_library_identity
    ON assets(library_id, id);
CREATE INDEX assets_search_global_name
    ON assets(natural_name_key, name, library_id, relative_path, id);
CREATE INDEX assets_search_global_modified
    ON assets(mtime_ns DESC, id DESC);

-- +goose StatementBegin
CREATE TRIGGER assets_search_insert
AFTER INSERT ON assets
BEGIN
    INSERT INTO asset_search(rowid, search_name_key, search_path_key)
    VALUES (NEW.id, NEW.search_name_key, NEW.search_path_key);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER assets_search_delete
AFTER DELETE ON assets
BEGIN
    INSERT INTO asset_search(asset_search, rowid, search_name_key, search_path_key)
    VALUES ('delete', OLD.id, OLD.search_name_key, OLD.search_path_key);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER assets_search_update
AFTER UPDATE OF search_name_key, search_path_key ON assets
WHEN NEW.search_name_key <> OLD.search_name_key
  OR NEW.search_path_key <> OLD.search_path_key
BEGIN
    INSERT INTO asset_search(asset_search, rowid, search_name_key, search_path_key)
    VALUES ('delete', OLD.id, OLD.search_name_key, OLD.search_path_key);
    INSERT INTO asset_search(rowid, search_name_key, search_path_key)
    VALUES (NEW.id, NEW.search_name_key, NEW.search_path_key);
END;
-- +goose StatementEnd

INSERT INTO asset_search(asset_search) VALUES('rebuild');

COMMIT;
PRAGMA legacy_alter_table = OFF;
PRAGMA foreign_keys = ON;
