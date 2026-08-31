-- +goose Up
CREATE TABLE ai_tag_vocabulary_snapshots (
    id              TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    revision        INTEGER NOT NULL UNIQUE CHECK (revision > 0),
    state           TEXT NOT NULL CHECK (state IN ('active', 'retired')),
    created_at_ms   INTEGER NOT NULL CHECK (created_at_ms > 0)
);

CREATE UNIQUE INDEX ai_tag_vocabulary_one_active
    ON ai_tag_vocabulary_snapshots(state) WHERE state = 'active';

CREATE TABLE ai_tag_vocabulary_entries (
    snapshot_id TEXT NOT NULL REFERENCES ai_tag_vocabulary_snapshots(id) ON DELETE CASCADE,
    tag_id      INTEGER NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
    PRIMARY KEY (snapshot_id, tag_id)
);

CREATE TABLE semantic_tag_embeddings (
    generation_id TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    snapshot_id   TEXT NOT NULL REFERENCES ai_tag_vocabulary_snapshots(id) ON DELETE CASCADE,
    tag_id        INTEGER NOT NULL,
    vector        BLOB NOT NULL CHECK (length(vector) > 0 AND length(vector) % 2 = 0),
    created_at_ms INTEGER NOT NULL CHECK (created_at_ms > 0),
    PRIMARY KEY (generation_id, snapshot_id, tag_id),
    FOREIGN KEY (snapshot_id, tag_id)
        REFERENCES ai_tag_vocabulary_entries(snapshot_id, tag_id) ON DELETE CASCADE
);

CREATE TABLE ai_tag_suggestions (
    id                   TEXT PRIMARY KEY CHECK (length(id) BETWEEN 8 AND 128),
    generation_id        TEXT NOT NULL REFERENCES semantic_generations(id) ON DELETE CASCADE,
    library_id           INTEGER NOT NULL,
    asset_id             INTEGER NOT NULL,
    vocabulary_snapshot_id TEXT NOT NULL REFERENCES ai_tag_vocabulary_snapshots(id) ON DELETE CASCADE,
    tag_id               INTEGER NOT NULL,
    source_fingerprint   TEXT NOT NULL CHECK (length(source_fingerprint) BETWEEN 1 AND 256),
    confidence           REAL NOT NULL CHECK (confidence >= 0.0 AND confidence <= 1.0),
    state                TEXT NOT NULL CHECK (state IN ('pending', 'invalidated')),
    revision             INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at_ms        INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms        INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    UNIQUE (generation_id, library_id, asset_id, vocabulary_snapshot_id, tag_id),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE,
    FOREIGN KEY (vocabulary_snapshot_id, tag_id)
        REFERENCES ai_tag_vocabulary_entries(snapshot_id, tag_id) ON DELETE CASCADE
);

CREATE INDEX ai_tag_suggestions_pending_page
    ON ai_tag_suggestions(library_id, state, confidence DESC, id);
CREATE INDEX ai_tag_suggestions_asset
    ON ai_tag_suggestions(library_id, asset_id, state, tag_id);

CREATE TABLE ai_tag_reviews (
    library_id                  INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    asset_id                    INTEGER NOT NULL,
    tag_id                      INTEGER NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
    decision                    TEXT NOT NULL CHECK (decision IN ('accepted', 'dismissed')),
    source_suggestion_id        TEXT,
    accepted_curation_revision  INTEGER CHECK (accepted_curation_revision IS NULL OR accepted_curation_revision > 0),
    revision                    INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    reviewed_at_ms              INTEGER NOT NULL CHECK (reviewed_at_ms > 0),
    PRIMARY KEY (library_id, asset_id, tag_id),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE,
    CHECK ((decision = 'accepted') = (accepted_curation_revision IS NOT NULL))
);

CREATE INDEX ai_tag_reviews_page
    ON ai_tag_reviews(library_id, decision, reviewed_at_ms DESC, asset_id, tag_id);

-- +goose Down
DROP INDEX IF EXISTS ai_tag_reviews_page;
DROP TABLE IF EXISTS ai_tag_reviews;
DROP INDEX IF EXISTS ai_tag_suggestions_asset;
DROP INDEX IF EXISTS ai_tag_suggestions_pending_page;
DROP TABLE IF EXISTS ai_tag_suggestions;
DROP TABLE IF EXISTS semantic_tag_embeddings;
DROP TABLE IF EXISTS ai_tag_vocabulary_entries;
DROP INDEX IF EXISTS ai_tag_vocabulary_one_active;
DROP TABLE IF EXISTS ai_tag_vocabulary_snapshots;
