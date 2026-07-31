-- +goose Up
ALTER TABLE settings
    ADD COLUMN resource_profile TEXT NOT NULL DEFAULT 'balanced'
        CHECK (resource_profile IN ('eco', 'balanced', 'performance'));

-- +goose Down
ALTER TABLE settings DROP COLUMN resource_profile;
