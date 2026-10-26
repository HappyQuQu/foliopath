-- +goose Up
CREATE TABLE ai_tag_review_requests (
    idempotency_key_hash TEXT PRIMARY KEY
        CHECK (length(idempotency_key_hash)=64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*'),
    request_hash TEXT NOT NULL
        CHECK (length(request_hash)=64 AND request_hash NOT GLOB '*[^0-9a-f]*'),
    state TEXT NOT NULL CHECK (state IN ('running','completed')),
    created_at_ms INTEGER NOT NULL CHECK (created_at_ms>0),
    updated_at_ms INTEGER NOT NULL CHECK (updated_at_ms>=created_at_ms)
);

CREATE TABLE ai_tag_review_request_items (
    idempotency_key_hash TEXT NOT NULL REFERENCES ai_tag_review_requests(idempotency_key_hash) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL CHECK (ordinal BETWEEN 0 AND 99),
    suggestion_id TEXT NOT NULL CHECK (length(suggestion_id) BETWEEN 8 AND 128),
    action TEXT NOT NULL CHECK (action IN ('accepted','dismissed')),
    expected_suggestion_revision INTEGER NOT NULL CHECK (expected_suggestion_revision>0),
    expected_curation_revision INTEGER NOT NULL CHECK (expected_curation_revision>=0),
    outcome TEXT CHECK (outcome IS NULL OR outcome IN ('accepted','dismissed','conflict')),
    outcome_revision INTEGER CHECK (outcome_revision IS NULL OR outcome_revision>=0),
    PRIMARY KEY(idempotency_key_hash, ordinal),
    UNIQUE(idempotency_key_hash, suggestion_id),
    CHECK ((outcome IS NULL) = (outcome_revision IS NULL))
);

-- +goose Down
DROP TABLE IF EXISTS ai_tag_review_request_items;
DROP TABLE IF EXISTS ai_tag_review_requests;
