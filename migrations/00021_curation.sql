-- +goose Up
CREATE TABLE curation_state (
    singleton_key INTEGER PRIMARY KEY CHECK (singleton_key = 1),
    revision      INTEGER NOT NULL CHECK (revision > 0)
);

INSERT INTO curation_state(singleton_key, revision) VALUES (1, 1);

CREATE TABLE asset_favorites (
    asset_id       INTEGER PRIMARY KEY,
    library_id     INTEGER NOT NULL,
    created_at_ms  INTEGER NOT NULL CHECK (created_at_ms > 0),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE
);

CREATE INDEX asset_favorites_recent
    ON asset_favorites(created_at_ms DESC, asset_id DESC);
CREATE INDEX asset_favorites_library_recent
    ON asset_favorites(library_id, created_at_ms DESC, asset_id DESC);

CREATE TABLE tags (
    id               INTEGER PRIMARY KEY,
    name             TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 128),
    normalized_name  TEXT NOT NULL UNIQUE
                     CHECK (length(normalized_name) BETWEEN 1 AND 256),
    created_at_ms    INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms    INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms)
);

CREATE INDEX tags_name_page ON tags(normalized_name, id);

CREATE TABLE asset_tags (
    asset_id       INTEGER NOT NULL,
    library_id     INTEGER NOT NULL,
    tag_id         INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at_ms  INTEGER NOT NULL CHECK (created_at_ms > 0),
    PRIMARY KEY (asset_id, tag_id),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE
);

CREATE INDEX asset_tags_tag_recent
    ON asset_tags(tag_id, created_at_ms DESC, asset_id DESC);
CREATE INDEX asset_tags_library_tag
    ON asset_tags(library_id, tag_id, asset_id);

-- Revision triggers also cover scanner- or library-owned asset deletion through
-- foreign-key cascades. More than one increment for a compound change is valid:
-- the revision is a monotonic invalidation token, not an event count.
-- +goose StatementBegin
CREATE TRIGGER curation_favorite_insert
AFTER INSERT ON asset_favorites
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER curation_favorite_delete
AFTER DELETE ON asset_favorites
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER curation_tag_insert
AFTER INSERT ON tags
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER curation_tag_update
AFTER UPDATE OF name, normalized_name ON tags
WHEN NEW.name <> OLD.name OR NEW.normalized_name <> OLD.normalized_name
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER curation_tag_delete
AFTER DELETE ON tags
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER curation_asset_tag_insert
AFTER INSERT ON asset_tags
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER curation_asset_tag_delete
AFTER DELETE ON asset_tags
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS curation_asset_tag_delete;
DROP TRIGGER IF EXISTS curation_asset_tag_insert;
DROP TRIGGER IF EXISTS curation_tag_delete;
DROP TRIGGER IF EXISTS curation_tag_update;
DROP TRIGGER IF EXISTS curation_tag_insert;
DROP TRIGGER IF EXISTS curation_favorite_delete;
DROP TRIGGER IF EXISTS curation_favorite_insert;
DROP TABLE IF EXISTS asset_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS asset_favorites;
DROP TABLE IF EXISTS curation_state;
