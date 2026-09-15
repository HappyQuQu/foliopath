-- +goose Up
CREATE TABLE semantic_generation_spaces (
    generation_id TEXT PRIMARY KEY REFERENCES semantic_generations(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL CHECK (length(fingerprint) = 64 AND fingerprint NOT GLOB '*[^0-9a-f]*')
);
-- +goose StatementBegin
CREATE TRIGGER semantic_generation_spaces_immutable
BEFORE UPDATE ON semantic_generation_spaces
BEGIN
    SELECT RAISE(ABORT, 'semantic space binding is immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER semantic_generation_bound_identity_immutable
BEFORE UPDATE OF model_id, embedding_dimension, transform_version, output_schema_version, index_format_version
ON semantic_generations
WHEN EXISTS (SELECT 1 FROM semantic_generation_spaces WHERE generation_id=OLD.id)
AND (NEW.model_id != OLD.model_id OR NEW.embedding_dimension != OLD.embedding_dimension
     OR NEW.transform_version != OLD.transform_version OR NEW.output_schema_version != OLD.output_schema_version
     OR NEW.index_format_version != OLD.index_format_version)
BEGIN
    SELECT RAISE(ABORT, 'bound semantic generation identity is immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER semantic_generation_bound_identity_immutable;
DROP TRIGGER semantic_generation_spaces_immutable;
DROP TABLE semantic_generation_spaces;
