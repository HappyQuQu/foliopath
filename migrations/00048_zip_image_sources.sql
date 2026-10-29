-- +goose Up
ALTER TABLE directories ADD COLUMN archive_path TEXT NOT NULL DEFAULT '';
ALTER TABLE directories ADD COLUMN archive_status TEXT NOT NULL DEFAULT ''
    CHECK ((archive_path = '' AND archive_status = '') OR
        (archive_path <> '' AND archive_status IN ('ready', 'unreadable', 'encrypted', 'unsupported', 'limit_exceeded')));

ALTER TABLE assets ADD COLUMN archive_path TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN archive_member_path TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN archive_size_bytes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN archive_mtime_ns INTEGER NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN archive_crc32 INTEGER NOT NULL DEFAULT 0
    CHECK ((archive_path = '' AND archive_member_path = '' AND archive_size_bytes = 0 AND archive_mtime_ns = 0 AND archive_crc32 = 0) OR
        (archive_path <> '' AND archive_member_path <> '' AND archive_size_bytes >= 0 AND archive_crc32 BETWEEN 0 AND 4294967295));

CREATE INDEX directories_archive ON directories(library_id, archive_path);
CREATE INDEX assets_archive ON assets(library_id, archive_path, natural_name_key, name, id);

-- +goose StatementBegin
CREATE TRIGGER catalog_zip_directory_update AFTER UPDATE OF archive_path, archive_status ON directories
WHEN OLD.archive_path <> NEW.archive_path OR OLD.archive_status <> NEW.archive_status
BEGIN
    UPDATE catalog_search_state SET revision = revision + 1, content_revision = content_revision + 1
    WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER catalog_zip_directory_update;
DROP INDEX assets_archive;
DROP INDEX directories_archive;
ALTER TABLE assets DROP COLUMN archive_crc32;
ALTER TABLE assets DROP COLUMN archive_mtime_ns;
ALTER TABLE assets DROP COLUMN archive_size_bytes;
ALTER TABLE assets DROP COLUMN archive_member_path;
ALTER TABLE assets DROP COLUMN archive_path;
ALTER TABLE directories DROP COLUMN archive_status;
ALTER TABLE directories DROP COLUMN archive_path;
