-- +goose Up
-- Natural-name keys are derived by the catalog capability and backfilled by
-- application startup immediately after this append-only schema migration.
ALTER TABLE directories
    ADD COLUMN natural_name_key BLOB NOT NULL DEFAULT X'';

ALTER TABLE assets
    ADD COLUMN natural_name_key BLOB NOT NULL DEFAULT X'';

CREATE INDEX directories_browse_children
    ON directories(library_id, parent_id, natural_name_key, name, id);

CREATE INDEX assets_browse_directory_name
    ON assets(library_id, directory_id, natural_name_key, name, relative_path, id);

CREATE INDEX assets_browse_library_name
    ON assets(library_id, natural_name_key, name, relative_path, id);

CREATE INDEX assets_browse_directory_modified
    ON assets(library_id, directory_id, mtime_ns DESC, id DESC);

-- The existing assets_modified index has this library-wide modified tuple.

-- +goose Down
DROP INDEX IF EXISTS assets_browse_directory_modified;
DROP INDEX IF EXISTS assets_browse_library_name;
DROP INDEX IF EXISTS assets_browse_directory_name;
DROP INDEX IF EXISTS directories_browse_children;
ALTER TABLE assets DROP COLUMN natural_name_key;
ALTER TABLE directories DROP COLUMN natural_name_key;
