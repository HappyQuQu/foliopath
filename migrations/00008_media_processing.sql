-- +goose Up
ALTER TABLE assets
    ADD COLUMN width INTEGER CHECK (width IS NULL OR width > 0);
ALTER TABLE assets
    ADD COLUMN height INTEGER CHECK (height IS NULL OR height > 0);
ALTER TABLE assets
    ADD COLUMN duration_ms INTEGER CHECK (duration_ms IS NULL OR duration_ms >= 0);
ALTER TABLE assets
    ADD COLUMN probe_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (probe_status IN ('pending', 'ready', 'failed', 'unsupported'));
ALTER TABLE assets
    ADD COLUMN probe_error_code TEXT
        CHECK (
            probe_error_code IS NULL
            OR probe_error_code IN (
                'unsupported_media',
                'invalid_media',
                'media_processing_failed',
                'media_processing_timeout'
            )
        );
ALTER TABLE assets
    ADD COLUMN playback_status TEXT NOT NULL DEFAULT 'unknown'
        CHECK (
            playback_status IN (
                'playable',
                'unsupported_codec',
                'not_applicable',
                'unknown'
            )
        );

CREATE UNIQUE INDEX assets_library_identity
    ON assets(library_id, id);

CREATE TABLE thumbnails (
    library_id          INTEGER NOT NULL,
    asset_id            INTEGER NOT NULL,
    variant             TEXT NOT NULL CHECK (variant = 'grid'),
    source_fingerprint  TEXT NOT NULL
                        CHECK (
                            length(source_fingerprint) BETWEEN 6 AND 64
                            AND substr(source_fingerprint, 1, 3) = 'v1:'
                        ),
    transform_version   INTEGER NOT NULL CHECK (transform_version > 0),
    cache_rel_path      TEXT,
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'ready', 'failed')),
    error_code          TEXT
                        CHECK (
                            error_code IS NULL
                            OR error_code IN (
                                'unsupported_media',
                                'invalid_media',
                                'media_processing_failed',
                                'media_processing_timeout'
                            )
                        ),
    width               INTEGER CHECK (width IS NULL OR width > 0),
    height              INTEGER CHECK (height IS NULL OR height > 0),
    byte_size           INTEGER CHECK (byte_size IS NULL OR byte_size > 0),
    created_at_ms       INTEGER,
    last_accessed_at_ms INTEGER,
    PRIMARY KEY (asset_id, variant),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE,
    CHECK (
        (
            status = 'ready'
            AND cache_rel_path IS NOT NULL
            AND error_code IS NULL
            AND width IS NOT NULL
            AND height IS NOT NULL
            AND byte_size IS NOT NULL
            AND created_at_ms IS NOT NULL
            AND last_accessed_at_ms IS NOT NULL
        )
        OR (
            status = 'pending'
            AND cache_rel_path IS NULL
            AND error_code IS NULL
            AND width IS NULL
            AND height IS NULL
            AND byte_size IS NULL
            AND created_at_ms IS NULL
            AND last_accessed_at_ms IS NULL
        )
        OR (
            status = 'failed'
            AND cache_rel_path IS NULL
            AND error_code IS NOT NULL
            AND width IS NULL
            AND height IS NULL
            AND byte_size IS NULL
            AND created_at_ms IS NULL
            AND last_accessed_at_ms IS NULL
        )
    )
);

CREATE INDEX thumbnails_library_status
    ON thumbnails(library_id, status, asset_id);
CREATE INDEX thumbnails_lru
    ON thumbnails(status, last_accessed_at_ms, asset_id)
    WHERE status = 'ready';

-- +goose Down
DROP TABLE IF EXISTS thumbnails;
DROP INDEX IF EXISTS assets_library_identity;
ALTER TABLE assets DROP COLUMN playback_status;
ALTER TABLE assets DROP COLUMN probe_error_code;
ALTER TABLE assets DROP COLUMN probe_status;
ALTER TABLE assets DROP COLUMN duration_ms;
ALTER TABLE assets DROP COLUMN height;
ALTER TABLE assets DROP COLUMN width;
