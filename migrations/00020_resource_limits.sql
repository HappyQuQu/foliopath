-- +goose Up
ALTER TABLE settings
    ADD COLUMN background_concurrency INTEGER NOT NULL DEFAULT 2
        CHECK (background_concurrency BETWEEN 1 AND 4);
ALTER TABLE settings
    ADD COLUMN content_read_concurrency INTEGER NOT NULL DEFAULT 8
        CHECK (content_read_concurrency BETWEEN 1 AND 16);

UPDATE settings
SET background_concurrency = CASE resource_profile
        WHEN 'eco' THEN 1
        WHEN 'performance' THEN 4
        ELSE 2
    END,
    content_read_concurrency = CASE resource_profile
        WHEN 'eco' THEN 4
        WHEN 'performance' THEN 16
        ELSE 8
    END;

-- +goose Down
ALTER TABLE settings DROP COLUMN content_read_concurrency;
ALTER TABLE settings DROP COLUMN background_concurrency;
