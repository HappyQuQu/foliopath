-- +goose Up
CREATE TABLE model_proxy_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    mode TEXT NOT NULL CHECK (mode IN ('inherit','direct','custom')),
    proxy_url TEXT NOT NULL CHECK (length(proxy_url) <= 2048),
    revision INTEGER NOT NULL CHECK (revision > 0),
    CHECK ((mode = 'custom' AND proxy_url <> '') OR (mode <> 'custom' AND proxy_url = ''))
);
INSERT INTO model_proxy_settings VALUES (1, 'inherit', '', 1);
-- +goose Down
DROP TABLE model_proxy_settings;
