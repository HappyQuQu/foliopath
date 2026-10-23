-- +goose Up
CREATE INDEX ai_model_install_requests_candidate_latest
    ON ai_model_install_requests(candidate_id, created_at_ms DESC, operation_id DESC);

-- +goose Down
DROP INDEX ai_model_install_requests_candidate_latest;
