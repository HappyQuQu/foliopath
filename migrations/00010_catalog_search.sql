-- +goose Up
-- Search keys are derived by the catalog capability and backfilled by
-- application startup. FTS5 provides a bounded candidate set for terms of
-- three or more Unicode code points; exact instr() predicates remain the
-- semantic authority and cover one- and two-character terms.
ALTER TABLE assets
    ADD COLUMN search_name_key TEXT NOT NULL DEFAULT '';

ALTER TABLE assets
    ADD COLUMN search_path_key TEXT NOT NULL DEFAULT '';

CREATE VIRTUAL TABLE asset_search USING fts5(
    search_name_key,
    search_path_key,
    content='assets',
    content_rowid='id',
    tokenize='trigram case_sensitive 1'
);

INSERT INTO asset_search(asset_search) VALUES('rebuild');

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

CREATE TABLE catalog_search_state (
    singleton_key INTEGER PRIMARY KEY CHECK (singleton_key = 1),
    revision      INTEGER NOT NULL CHECK (revision > 0)
);

INSERT INTO catalog_search_state(singleton_key, revision) VALUES (1, 1);

-- +goose StatementBegin
CREATE TRIGGER catalog_revision_library_insert
AFTER INSERT ON libraries
BEGIN
    UPDATE catalog_search_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER catalog_revision_library_delete
AFTER DELETE ON libraries
BEGIN
    UPDATE catalog_search_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER catalog_revision_generation_publish
AFTER UPDATE OF current_generation ON libraries
WHEN NEW.current_generation <> OLD.current_generation
BEGIN
    UPDATE catalog_search_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

CREATE INDEX assets_search_global_name
    ON assets(natural_name_key, name, library_id, relative_path, id);

CREATE INDEX assets_search_global_modified
    ON assets(mtime_ns DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS assets_search_global_modified;
DROP INDEX IF EXISTS assets_search_global_name;
DROP TRIGGER IF EXISTS catalog_revision_generation_publish;
DROP TRIGGER IF EXISTS catalog_revision_library_delete;
DROP TRIGGER IF EXISTS catalog_revision_library_insert;
DROP TABLE IF EXISTS catalog_search_state;
DROP TRIGGER IF EXISTS assets_search_update;
DROP TRIGGER IF EXISTS assets_search_delete;
DROP TRIGGER IF EXISTS assets_search_insert;
DROP TABLE IF EXISTS asset_search;
ALTER TABLE assets DROP COLUMN search_path_key;
ALTER TABLE assets DROP COLUMN search_name_key;
