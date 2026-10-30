-- +goose Up
CREATE TABLE media_deletion_settings (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    recycle_enabled INTEGER NOT NULL DEFAULT 1 CHECK (recycle_enabled IN (0, 1)),
    retention_days INTEGER CHECK (retention_days IN (7, 30, 90)),
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    instance_id TEXT NOT NULL UNIQUE
);
INSERT INTO media_deletion_settings(singleton, instance_id) VALUES (1, lower(hex(randomblob(16))));

CREATE TABLE media_deletion_preflights (
    id TEXT PRIMARY KEY,
    expires_at_ms INTEGER NOT NULL,
    settings_revision INTEGER NOT NULL,
    payload TEXT NOT NULL
);
CREATE INDEX media_deletion_preflights_expiry ON media_deletion_preflights(expires_at_ms);

CREATE TABLE media_deletion_operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL CHECK (kind IN ('trash', 'permanent', 'restore', 'purge')),
    state TEXT NOT NULL DEFAULT 'queued'
        CHECK (state IN ('queued', 'running', 'reconciling', 'stopped', 'partial', 'failed', 'succeeded', 'needs_attention', 'cancelled')),
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    stop_requested INTEGER NOT NULL DEFAULT 0 CHECK (stop_requested IN (0, 1)),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 3),
    fence INTEGER NOT NULL DEFAULT 0,
    lease_expires_at_ms INTEGER,
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL,
    empty_upper_id INTEGER,
    empty_sequence INTEGER,
    empty_after_id INTEGER NOT NULL DEFAULT 0,
    empty_library_id INTEGER,
    total_items INTEGER NOT NULL DEFAULT 0,
    after_repair_state TEXT NOT NULL DEFAULT 'partial',
    policy_revision INTEGER,
    last_handoff_fence INTEGER,
    last_handoff_at_ms INTEGER
);
CREATE INDEX media_deletion_operations_queue ON media_deletion_operations(state, id);
CREATE UNIQUE INDEX media_deletion_one_worker ON media_deletion_operations((1)) WHERE state = 'running';

CREATE TABLE media_trash_clock(singleton INTEGER PRIMARY KEY CHECK(singleton=1),sequence INTEGER NOT NULL);
INSERT INTO media_trash_clock VALUES(1,0);

CREATE TABLE media_trash_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    target_type TEXT NOT NULL CHECK (target_type IN ('asset', 'directory')),
    target_id INTEGER NOT NULL,
    relative_path TEXT NOT NULL CHECK (relative_path <> ''),
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('file', 'directory', 'archive')),
    identity TEXT NOT NULL,
    library_identity TEXT NOT NULL,
    deleted_at_ms INTEGER,
    revision INTEGER NOT NULL DEFAULT 1,
    available_sequence INTEGER NOT NULL DEFAULT 0,
    state TEXT NOT NULL CHECK (state IN ('moving', 'available', 'restoring', 'purging', 'missing', 'needs_attention'))
);
CREATE INDEX media_trash_library_page ON media_trash_entries(library_id, id DESC);
CREATE INDEX media_trash_expiration ON media_trash_entries(state, deleted_at_ms, id);

-- +goose StatementBegin
CREATE TRIGGER media_trash_available_sequence AFTER UPDATE OF state ON media_trash_entries
WHEN NEW.state='available' AND OLD.state<>'available'
BEGIN
    UPDATE media_trash_clock SET sequence=sequence+1 WHERE singleton=1;
    UPDATE media_trash_entries SET available_sequence=(SELECT sequence FROM media_trash_clock WHERE singleton=1) WHERE id=NEW.id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER media_trash_available_insert AFTER INSERT ON media_trash_entries
WHEN NEW.state='available'
BEGIN
    UPDATE media_trash_clock SET sequence=sequence+1 WHERE singleton=1;
    UPDATE media_trash_entries SET available_sequence=(SELECT sequence FROM media_trash_clock WHERE singleton=1) WHERE id=NEW.id;
END;
-- +goose StatementEnd

CREATE TABLE media_deletion_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation_id INTEGER NOT NULL REFERENCES media_deletion_operations(id) ON DELETE CASCADE,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK (target_type IN ('asset', 'directory')),
    target_id INTEGER NOT NULL,
    relative_path TEXT NOT NULL CHECK (relative_path <> ''),
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('file', 'directory', 'archive')),
    identity TEXT NOT NULL,
    library_identity TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'queued'
        CHECK (state IN ('queued', 'running', 'trashed', 'restored', 'deleted', 'failed', 'partial', 'cancelled', 'needs_attention')),
    stage TEXT NOT NULL DEFAULT 'queued'
        CHECK (stage IN ('queued', 'intent', 'filesystem_applied', 'catalog_finalizing', 'finished')),
    removed_entries INTEGER NOT NULL DEFAULT 0,
    error_code TEXT,
    trash_entry_id INTEGER,
    trash_token TEXT NOT NULL DEFAULT '',
    repair_scan_id INTEGER REFERENCES scan_runs(id) ON DELETE SET NULL,
    repair_state TEXT NOT NULL DEFAULT 'queued' CHECK (repair_state IN ('queued','running','succeeded','failed'))
);
CREATE INDEX media_deletion_items_operation ON media_deletion_items(operation_id, id);
CREATE INDEX media_deletion_items_library ON media_deletion_items(library_id, operation_id);
CREATE INDEX media_deletion_items_trash ON media_deletion_items(trash_entry_id,operation_id) WHERE trash_entry_id IS NOT NULL;

CREATE TABLE media_deletion_requests (
    key_hash BLOB PRIMARY KEY CHECK (length(key_hash) = 32),
    request_hash BLOB NOT NULL CHECK (length(request_hash) = 32),
    operation_id INTEGER NOT NULL REFERENCES media_deletion_operations(id) ON DELETE CASCADE
);

CREATE TABLE media_mutation_scopes (
    item_id INTEGER PRIMARY KEY REFERENCES media_deletion_items(id) ON DELETE CASCADE,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    relative_path TEXT NOT NULL CHECK (relative_path <> ''),
    active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1))
);
CREATE INDEX media_mutation_scopes_library ON media_mutation_scopes(library_id, active);

CREATE VIEW media_mutation_blocked_libraries AS
SELECT DISTINCT item.library_id FROM media_deletion_items item
JOIN media_deletion_operations op ON op.id=item.operation_id
LEFT JOIN media_mutation_scopes scope ON scope.item_id=item.id
WHERE op.state IN ('queued','running') OR (scope.active=1 AND op.state<>'reconciling');

ALTER TABLE assets ADD COLUMN trash_entry_id INTEGER REFERENCES media_trash_entries(id) ON DELETE RESTRICT;
ALTER TABLE assets ADD COLUMN restoration_path TEXT;
ALTER TABLE directories ADD COLUMN trash_entry_id INTEGER REFERENCES media_trash_entries(id) ON DELETE RESTRICT;
ALTER TABLE directories ADD COLUMN restoration_path TEXT;
CREATE INDEX assets_trash_entry ON assets(trash_entry_id, id);
CREATE INDEX directories_trash_entry ON directories(trash_entry_id, id);

CREATE VIEW visible_assets AS
SELECT assets.* FROM assets
WHERE trash_entry_id IS NULL AND NOT EXISTS (
    SELECT 1 FROM media_mutation_scopes scope
    WHERE scope.library_id = assets.library_id AND scope.active = 1
      AND (assets.relative_path = scope.relative_path OR instr(assets.relative_path, scope.relative_path || '/') = 1)
);
CREATE VIEW visible_directories AS
SELECT directories.* FROM directories
WHERE trash_entry_id IS NULL AND NOT EXISTS (
    SELECT 1 FROM media_mutation_scopes scope
    WHERE scope.library_id = directories.library_id AND scope.active = 1
      AND (directories.relative_path = scope.relative_path OR instr(directories.relative_path, scope.relative_path || '/') = 1)
);

-- +goose StatementBegin
CREATE TRIGGER media_mutation_visibility_insert AFTER INSERT ON media_mutation_scopes
WHEN NEW.active=1
BEGIN
    UPDATE catalog_search_state SET revision = revision + 1, content_revision = content_revision + 1 WHERE singleton_key = 1;
    UPDATE libraries SET revision=revision+1,content_revision=content_revision+1 WHERE id=NEW.library_id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER media_mutation_visibility_update AFTER UPDATE OF active ON media_mutation_scopes
WHEN NEW.active<>OLD.active
BEGIN
    UPDATE catalog_search_state SET revision = revision + 1, content_revision = content_revision + 1 WHERE singleton_key = 1;
    UPDATE libraries SET revision=revision+1,content_revision=content_revision+1 WHERE id=NEW.library_id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER media_mutation_visibility_delete AFTER DELETE ON media_mutation_scopes
WHEN OLD.active=1
BEGIN
    UPDATE catalog_search_state SET revision = revision + 1, content_revision = content_revision + 1 WHERE singleton_key = 1;
    UPDATE libraries SET revision=revision+1,content_revision=content_revision+1 WHERE id=OLD.library_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER media_trash_assets_no_stale_delete BEFORE DELETE ON assets
WHEN OLD.trash_entry_id IS NOT NULL AND OLD.restoration_path IS NOT NULL AND EXISTS (
    SELECT 1 FROM media_trash_entries WHERE id = OLD.trash_entry_id AND state NOT IN ('purging', 'missing')
)
BEGIN
    SELECT RAISE(IGNORE);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER media_trash_directories_no_stale_delete BEFORE DELETE ON directories
WHEN OLD.trash_entry_id IS NOT NULL AND OLD.restoration_path IS NOT NULL AND EXISTS (
    SELECT 1 FROM media_trash_entries WHERE id = OLD.trash_entry_id AND state NOT IN ('purging', 'missing')
)
BEGIN
    SELECT RAISE(IGNORE);
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER media_trash_directories_no_stale_delete;
DROP TRIGGER media_trash_assets_no_stale_delete;
DROP TRIGGER media_mutation_visibility_delete;
DROP TRIGGER media_mutation_visibility_update;
DROP TRIGGER media_mutation_visibility_insert;
DROP VIEW visible_directories;
DROP VIEW visible_assets;
DROP VIEW media_mutation_blocked_libraries;
DROP INDEX directories_trash_entry;
DROP INDEX assets_trash_entry;
ALTER TABLE directories DROP COLUMN restoration_path;
ALTER TABLE directories DROP COLUMN trash_entry_id;
ALTER TABLE assets DROP COLUMN restoration_path;
ALTER TABLE assets DROP COLUMN trash_entry_id;
DROP TABLE media_mutation_scopes;
DROP TABLE media_deletion_requests;
DROP TABLE media_deletion_items;
DROP TRIGGER media_trash_available_sequence;
DROP TRIGGER media_trash_available_insert;
DROP TABLE media_trash_entries;
DROP TABLE media_trash_clock;
DROP TABLE media_deletion_operations;
DROP TABLE media_deletion_preflights;
DROP TABLE media_deletion_settings;
