-- name: FindLibraryIDByNameKey :one
SELECT id
FROM libraries
WHERE name_key = sqlc.arg(name_key);

-- name: FindOtherLibraryIDByNameKey :one
SELECT id
FROM libraries
WHERE name_key = sqlc.arg(name_key)
  AND id <> sqlc.arg(id);

-- name: FindOverlappingLibraryID :one
SELECT id
FROM libraries
WHERE root_rel_path = sqlc.arg(root_rel_path)
   OR root_rel_path = ''
   OR sqlc.arg(root_rel_path) = ''
   OR instr(sqlc.arg(root_rel_path), root_rel_path || '/') = 1
   OR instr(root_rel_path, sqlc.arg(root_rel_path) || '/') = 1
LIMIT 1;

-- name: InsertLibrary :one
INSERT INTO libraries(
    name,
    name_key,
    name_sort_key,
    root_rel_path,
    status,
    current_generation,
    revision,
    created_at_ms,
    updated_at_ms
) VALUES (
    sqlc.arg(name),
    sqlc.arg(name_key),
    sqlc.arg(name_sort_key),
    sqlc.arg(root_rel_path),
    'pending',
    0,
    1,
    sqlc.arg(created_at_ms),
    sqlc.arg(updated_at_ms)
)
RETURNING
    id,
    name,
    name_key,
    name_sort_key,
    root_rel_path,
    status,
    current_generation,
    revision,
    created_at_ms,
    updated_at_ms;

-- name: RenameLibrary :one
UPDATE libraries
SET name = sqlc.arg(name),
    name_key = sqlc.arg(name_key),
    name_sort_key = sqlc.arg(name_sort_key),
    revision = revision + 1,
    updated_at_ms = sqlc.arg(updated_at_ms)
WHERE id = sqlc.arg(id)
RETURNING
    id,
    name,
    name_key,
    name_sort_key,
    root_rel_path,
    status,
    current_generation,
    revision,
    created_at_ms,
    updated_at_ms;

-- name: GetLibrary :one
SELECT
    id,
    name,
    name_key,
    root_rel_path,
    status,
    current_generation,
    revision,
    created_at_ms,
    updated_at_ms
FROM libraries
WHERE id = sqlc.arg(id);

-- name: ListLibraries :many
SELECT
    id,
    name,
    name_key,
    root_rel_path,
    status,
    current_generation,
    revision,
    created_at_ms,
    updated_at_ms
FROM libraries
ORDER BY name_key, id;
