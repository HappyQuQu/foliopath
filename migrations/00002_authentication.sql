-- +goose Up
-- Authentication state is non-reconstructible application data. The schema
-- stores password verifiers and digests of random browser tokens, never
-- plaintext passwords, session cookies, or CSRF tokens.
CREATE TABLE users (
    id                      INTEGER PRIMARY KEY,
    singleton_key           INTEGER NOT NULL DEFAULT 1
                              CHECK (singleton_key = 1)
                              UNIQUE,
    username                TEXT NOT NULL
                              CHECK (length(username) BETWEEN 1 AND 64),
    username_key            TEXT NOT NULL UNIQUE
                              CHECK (length(username_key) > 0),
    display_name            TEXT NOT NULL
                              CHECK (length(display_name) BETWEEN 1 AND 128),
    password_hash           TEXT NOT NULL
                              CHECK (length(password_hash) > 0),
    password_scheme         TEXT NOT NULL
                              CHECK (length(password_scheme) > 0),
    password_parameters     TEXT NOT NULL
                              CHECK (length(password_parameters) > 0),
    auth_version            INTEGER NOT NULL DEFAULT 1
                              CHECK (auth_version > 0),
    created_at_ms           INTEGER NOT NULL,
    updated_at_ms           INTEGER NOT NULL,
    password_changed_at_ms  INTEGER NOT NULL,
    disabled_at_ms          INTEGER,
    CHECK (updated_at_ms >= created_at_ms),
    CHECK (password_changed_at_ms >= created_at_ms),
    CHECK (disabled_at_ms IS NULL OR disabled_at_ms >= created_at_ms)
);

CREATE TABLE sessions (
    id                 INTEGER PRIMARY KEY,
    user_id            INTEGER NOT NULL
                         REFERENCES users(id) ON DELETE CASCADE,
    token_hash         BLOB NOT NULL UNIQUE
                         CHECK (length(token_hash) = 32),
    csrf_token_hash    BLOB NOT NULL
                         CHECK (length(csrf_token_hash) = 32),
    auth_version       INTEGER NOT NULL CHECK (auth_version > 0),
    created_at_ms      INTEGER NOT NULL,
    last_seen_at_ms    INTEGER NOT NULL,
    expires_at_ms      INTEGER NOT NULL,
    revoked_at_ms      INTEGER,
    CHECK (last_seen_at_ms >= created_at_ms),
    CHECK (last_seen_at_ms <= expires_at_ms),
    CHECK (expires_at_ms > created_at_ms),
    CHECK (revoked_at_ms IS NULL OR revoked_at_ms >= created_at_ms)
);

CREATE INDEX sessions_user_expiry
    ON sessions(user_id, expires_at_ms, id);

CREATE INDEX sessions_expiry_cleanup
    ON sessions(expires_at_ms, id)
    WHERE revoked_at_ms IS NULL;

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
