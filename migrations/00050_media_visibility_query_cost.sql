-- +goose Up
-- Skip per-row subtree checks when a library has no active media mutation.
DROP VIEW visible_assets;
DROP VIEW visible_directories;
CREATE VIEW visible_assets AS
SELECT assets.* FROM assets
WHERE trash_entry_id IS NULL AND (library_id NOT IN (
    SELECT library_id FROM media_mutation_scopes WHERE active = 1
) OR NOT EXISTS (
    SELECT 1 FROM media_mutation_scopes scope
    WHERE scope.library_id = assets.library_id AND scope.active = 1
      AND (assets.relative_path = scope.relative_path OR instr(assets.relative_path, scope.relative_path || '/') = 1)
));
CREATE VIEW visible_directories AS
SELECT directories.* FROM directories
WHERE trash_entry_id IS NULL AND (library_id NOT IN (
    SELECT library_id FROM media_mutation_scopes WHERE active = 1
) OR NOT EXISTS (
    SELECT 1 FROM media_mutation_scopes scope
    WHERE scope.library_id = directories.library_id AND scope.active = 1
      AND (directories.relative_path = scope.relative_path OR instr(directories.relative_path, scope.relative_path || '/') = 1)
));

DROP INDEX assets_trash_entry;
DROP INDEX directories_trash_entry;
CREATE INDEX assets_trash_entry ON assets(trash_entry_id, id) WHERE trash_entry_id IS NOT NULL;
CREATE INDEX directories_trash_entry ON directories(trash_entry_id, id) WHERE trash_entry_id IS NOT NULL;

-- +goose Down
DROP VIEW visible_assets;
DROP VIEW visible_directories;
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

DROP INDEX assets_trash_entry;
DROP INDEX directories_trash_entry;
CREATE INDEX assets_trash_entry ON assets(trash_entry_id, id);
CREATE INDEX directories_trash_entry ON directories(trash_entry_id, id);
