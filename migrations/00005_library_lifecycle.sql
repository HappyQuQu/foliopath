-- +goose Up
-- Persist the locale-neutral numeric collation key used by bounded library
-- keyset pagination. It is derived from name and contains no path data.
ALTER TABLE libraries
    ADD COLUMN name_sort_key BLOB NOT NULL DEFAULT X'';

CREATE INDEX libraries_natural_name
    ON libraries(name_sort_key, name, id);

-- +goose Down
DROP INDEX IF EXISTS libraries_natural_name;
ALTER TABLE libraries DROP COLUMN name_sort_key;
