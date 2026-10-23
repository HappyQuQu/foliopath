-- +goose Up
-- Legacy assignments, including explicitly accepted suggestions, remain manual.
ALTER TABLE asset_tags ADD COLUMN ai_model_identity TEXT;
ALTER TABLE asset_tags ADD COLUMN ai_source_fingerprint TEXT;
ALTER TABLE asset_tags ADD COLUMN ai_confidence REAL;
ALTER TABLE asset_tags ADD COLUMN source TEXT NOT NULL DEFAULT 'manual'
    CHECK (
        (source = 'manual' AND ai_model_identity IS NULL
            AND ai_source_fingerprint IS NULL AND ai_confidence IS NULL)
        OR
        (source = 'ai' AND ai_model_identity IS NOT NULL
            AND length(ai_model_identity) BETWEEN 1 AND 256
            AND ai_source_fingerprint IS NOT NULL
            AND length(ai_source_fingerprint) BETWEEN 1 AND 256
            AND ai_confidence IS NOT NULL AND ai_confidence BETWEEN 0.0 AND 1.0)
    );

CREATE TABLE asset_tag_suppressions (
    library_id       INTEGER NOT NULL,
    asset_id         INTEGER NOT NULL,
    normalized_name  TEXT NOT NULL CHECK (length(normalized_name) BETWEEN 1 AND 256),
    created_at_ms    INTEGER NOT NULL CHECK (created_at_ms > 0),
    updated_at_ms    INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    PRIMARY KEY (library_id, asset_id, normalized_name),
    FOREIGN KEY (library_id, asset_id)
        REFERENCES assets(library_id, id) ON DELETE CASCADE
);

-- Intentionally no tags/generation FK: a renamed/recreated tag or model must not
-- erase a person's deletion. Only actual catalog removal deletes this state.
-- +goose StatementBegin
CREATE TRIGGER curation_asset_tag_source_update
AFTER UPDATE OF source, ai_model_identity, ai_source_fingerprint, ai_confidence ON asset_tags
WHEN NEW.source IS NOT OLD.source
  OR NEW.ai_model_identity IS NOT OLD.ai_model_identity
  OR NEW.ai_source_fingerprint IS NOT OLD.ai_source_fingerprint
  OR NEW.ai_confidence IS NOT OLD.ai_confidence
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER curation_tag_suppression_insert
AFTER INSERT ON asset_tag_suppressions
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER curation_tag_suppression_delete
AFTER DELETE ON asset_tag_suppressions
BEGIN
    UPDATE curation_state SET revision = revision + 1 WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- Stale machine output is no longer a visible association. Manual curation and
-- suppression deliberately survive source/kind changes at an existing asset ID.
-- +goose StatementBegin
CREATE TRIGGER curation_automatic_tag_source_invalidation
AFTER UPDATE OF source_fingerprint, kind ON assets
WHEN NEW.source_fingerprint IS NOT OLD.source_fingerprint OR NEW.kind IS NOT OLD.kind
BEGIN
    DELETE FROM asset_tags WHERE asset_id = NEW.id AND source = 'ai';
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS curation_automatic_tag_source_invalidation;
DROP TRIGGER IF EXISTS curation_tag_suppression_delete;
DROP TRIGGER IF EXISTS curation_tag_suppression_insert;
DROP TRIGGER IF EXISTS curation_asset_tag_source_update;
DROP TABLE IF EXISTS asset_tag_suppressions;
-- Do not misrepresent generated assignments as manual when explicitly reverting.
DELETE FROM asset_tags WHERE source = 'ai';
ALTER TABLE asset_tags DROP COLUMN source;
ALTER TABLE asset_tags DROP COLUMN ai_confidence;
ALTER TABLE asset_tags DROP COLUMN ai_source_fingerprint;
ALTER TABLE asset_tags DROP COLUMN ai_model_identity;
