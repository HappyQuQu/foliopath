-- +goose Up
-- Name order v2 groups assets by their source directory before applying the
-- existing locale-neutral natural filename key. The directory path is derived
-- from canonical asset fields so it cannot drift from scanner-owned identity.
CREATE INDEX assets_browse_folder_name_v2
    ON assets(
        library_id,
        (CASE
            WHEN length(relative_path) = length(name) THEN ''
            ELSE substr(relative_path, 1, length(relative_path) - length(name) - 1)
        END),
        natural_name_key,
        name,
        relative_path,
        id
    );

-- +goose Down
DROP INDEX IF EXISTS assets_browse_folder_name_v2;
