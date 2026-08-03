-- +goose Up
CREATE TABLE system_events (
    id             INTEGER PRIMARY KEY,
    occurred_at_ms INTEGER NOT NULL,
    level          TEXT NOT NULL CHECK (level IN ('info', 'warning', 'error')),
    module         TEXT NOT NULL CHECK (length(module) BETWEEN 1 AND 64),
    event_code     TEXT NOT NULL CHECK (length(event_code) BETWEEN 1 AND 128),
    request_id     TEXT,
    method         TEXT,
    route_pattern  TEXT,
    status_code    INTEGER CHECK (status_code IS NULL OR status_code BETWEEN 100 AND 599),
    duration_ms    INTEGER CHECK (duration_ms IS NULL OR duration_ms >= 0)
);

CREATE INDEX system_events_time
    ON system_events(id DESC);

CREATE INDEX system_events_level_time
    ON system_events(level, id DESC);

-- +goose Down
DROP TABLE system_events;
