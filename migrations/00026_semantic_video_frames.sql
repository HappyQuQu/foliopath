-- +goose Up
CREATE TABLE semantic_video_frames (
    generation_id         TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    library_id            INTEGER NOT NULL,
    asset_id              INTEGER NOT NULL,
    source_fingerprint    TEXT NOT NULL CHECK (length(source_fingerprint) BETWEEN 1 AND 256),
    storyboard_fingerprint TEXT NOT NULL CHECK (length(storyboard_fingerprint) BETWEEN 8 AND 256),
    storyboard_transform_version INTEGER NOT NULL CHECK (storyboard_transform_version > 0),
    plan_size             INTEGER NOT NULL CHECK (plan_size IN (4, 10)),
    ordinal               INTEGER NOT NULL CHECK (ordinal >= 0 AND ordinal < plan_size),
    timestamp_ms          INTEGER NOT NULL CHECK (timestamp_ms >= 0),
    vector                BLOB NOT NULL CHECK (length(vector) > 0 AND length(vector) % 2 = 0),
    created_at_ms         INTEGER NOT NULL CHECK (created_at_ms > 0),
    PRIMARY KEY (generation_id, library_id, asset_id, storyboard_fingerprint, plan_size, ordinal),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE
);

CREATE INDEX semantic_video_frames_search
    ON semantic_video_frames(generation_id, library_id, asset_id, plan_size, ordinal);

CREATE TABLE semantic_video_progress (
    generation_id   TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    library_id      INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    eligible_count  INTEGER NOT NULL DEFAULT 0 CHECK (eligible_count >= 0),
    ready_count     INTEGER NOT NULL DEFAULT 0 CHECK (ready_count >= 0),
    degraded_count  INTEGER NOT NULL DEFAULT 0 CHECK (degraded_count >= 0),
    failed_count    INTEGER NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
    stale_count     INTEGER NOT NULL DEFAULT 0 CHECK (stale_count >= 0),
    checkpoint_id   INTEGER NOT NULL DEFAULT 0 CHECK (checkpoint_id >= 0),
    revision        INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_at_ms   INTEGER NOT NULL CHECK (updated_at_ms > 0),
    PRIMARY KEY (generation_id, library_id),
    CHECK (ready_count + degraded_count + failed_count + stale_count <= eligible_count)
);

-- +goose Down
DROP TABLE IF EXISTS semantic_video_progress;
DROP INDEX IF EXISTS semantic_video_frames_search;
DROP TABLE IF EXISTS semantic_video_frames;
