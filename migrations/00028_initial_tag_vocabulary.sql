-- +goose Up
INSERT INTO ai_tag_vocabulary_snapshots(id, revision, state, created_at_ms)
VALUES('aivocab_initial', 1, 'active', 1);

-- +goose Down
DELETE FROM ai_tag_vocabulary_snapshots
WHERE id='aivocab_initial' AND revision=1;
