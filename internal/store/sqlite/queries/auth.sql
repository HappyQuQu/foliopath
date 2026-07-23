-- name: IsAdministratorInitialized :one
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE singleton_key = 1
);

-- name: InsertAdministrator :one
INSERT INTO users (
    singleton_key,
    username,
    username_key,
    display_name,
    password_hash,
    password_scheme,
    password_parameters,
    auth_version,
    created_at_ms,
    updated_at_ms,
    password_changed_at_ms
) VALUES (
    1,
    sqlc.arg(username),
    sqlc.arg(username_key),
    sqlc.arg(display_name),
    sqlc.arg(password_hash),
    sqlc.arg(password_scheme),
    sqlc.arg(password_parameters),
    1,
    sqlc.arg(created_at_ms),
    sqlc.arg(updated_at_ms),
    sqlc.arg(password_changed_at_ms)
)
RETURNING
    id,
    username,
    display_name,
    created_at_ms,
    updated_at_ms;
