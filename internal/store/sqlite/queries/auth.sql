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

-- name: FindAdministratorCredential :one
SELECT
    id,
    username,
    username_key,
    display_name,
    password_hash,
    password_scheme,
    password_parameters,
    auth_version,
    created_at_ms,
    updated_at_ms,
    disabled_at_ms
FROM users
WHERE username_key = sqlc.arg(username_key)
  AND singleton_key = 1;

-- name: InsertSession :one
INSERT INTO sessions (
    user_id,
    token_hash,
    csrf_token_hash,
    auth_version,
    created_at_ms,
    last_seen_at_ms,
    expires_at_ms
) VALUES (
    sqlc.arg(user_id),
    sqlc.arg(token_hash),
    sqlc.arg(csrf_token_hash),
    sqlc.arg(auth_version),
    sqlc.arg(created_at_ms),
    sqlc.arg(created_at_ms),
    sqlc.arg(expires_at_ms)
)
RETURNING
    id,
    auth_version,
    created_at_ms,
    last_seen_at_ms,
    expires_at_ms;

-- name: FindSession :one
SELECT
    sessions.id,
    sessions.auth_version,
    sessions.created_at_ms,
    sessions.last_seen_at_ms,
    sessions.expires_at_ms,
    sessions.csrf_token_hash,
    sessions.revoked_at_ms,
    users.id AS user_id,
    users.username,
    users.display_name,
    users.auth_version AS user_auth_version,
    users.created_at_ms AS user_created_at_ms,
    users.updated_at_ms AS user_updated_at_ms,
    users.disabled_at_ms AS user_disabled_at_ms
FROM sessions
JOIN users ON users.id = sessions.user_id
WHERE sessions.token_hash = sqlc.arg(token_hash);

-- name: TouchSession :execrows
UPDATE sessions
SET last_seen_at_ms = sqlc.arg(used_at_ms)
WHERE sessions.id = sqlc.arg(id)
  AND sessions.token_hash = sqlc.arg(token_hash)
  AND sessions.auth_version = sqlc.arg(expected_version)
  AND sessions.revoked_at_ms IS NULL
  AND sessions.expires_at_ms > sqlc.arg(used_at_ms)
  AND EXISTS (
      SELECT 1
      FROM users
      WHERE users.id = sessions.user_id
        AND users.auth_version = sessions.auth_version
        AND users.disabled_at_ms IS NULL
  );

-- name: RevokeSession :execrows
UPDATE sessions
SET revoked_at_ms = sqlc.arg(revoked_at_ms)
WHERE id = sqlc.arg(id)
  AND token_hash = sqlc.arg(token_hash)
  AND revoked_at_ms IS NULL;

-- name: DeleteObsoleteSessions :execrows
DELETE FROM sessions
WHERE expires_at_ms <= sqlc.arg(cutoff_ms)
   OR (revoked_at_ms IS NOT NULL AND revoked_at_ms <= sqlc.arg(cutoff_ms));
