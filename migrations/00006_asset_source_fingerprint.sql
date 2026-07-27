-- +goose Up
-- Source fingerprints are versioned tuples of size and nanosecond mtime. The
-- temporary default lets SQLite add the non-null column to existing tables;
-- every existing row is backfilled before the migration commits and all
-- production writers supply the canonical value explicitly.
ALTER TABLE assets
    ADD COLUMN source_fingerprint TEXT NOT NULL DEFAULT 'v1:0:0'
        CHECK (
            length(source_fingerprint) BETWEEN 6 AND 64
            AND substr(source_fingerprint, 1, 3) = 'v1:'
        );

UPDATE assets
SET source_fingerprint = 'v1:' || size_bytes || ':' || mtime_ns;

-- +goose Down
ALTER TABLE assets DROP COLUMN source_fingerprint;
