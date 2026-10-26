-- +goose Up
ALTER TABLE ai_tag_reviews ADD COLUMN source_generation_id TEXT
    CHECK (source_generation_id IS NULL OR length(source_generation_id) BETWEEN 8 AND 128);
ALTER TABLE ai_tag_reviews ADD COLUMN source_vocabulary_snapshot_id TEXT
    CHECK (source_vocabulary_snapshot_id IS NULL OR length(source_vocabulary_snapshot_id) BETWEEN 8 AND 128);
ALTER TABLE ai_tag_reviews ADD COLUMN source_vocabulary_revision INTEGER
    CHECK (source_vocabulary_revision IS NULL OR source_vocabulary_revision > 0);
ALTER TABLE ai_tag_reviews ADD COLUMN source_confidence REAL
    CHECK (source_confidence IS NULL OR (source_confidence >= 0.0 AND source_confidence <= 1.0));
ALTER TABLE ai_tag_reviews ADD COLUMN source_suggestion_revision INTEGER
    CHECK (source_suggestion_revision IS NULL OR source_suggestion_revision > 0);

-- +goose Down
ALTER TABLE ai_tag_reviews DROP COLUMN source_suggestion_revision;
ALTER TABLE ai_tag_reviews DROP COLUMN source_confidence;
ALTER TABLE ai_tag_reviews DROP COLUMN source_vocabulary_revision;
ALTER TABLE ai_tag_reviews DROP COLUMN source_vocabulary_snapshot_id;
ALTER TABLE ai_tag_reviews DROP COLUMN source_generation_id;
