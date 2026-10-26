-- +goose Up
-- Internal dormant storage only; migration does not enable OCR or enqueue work.
CREATE TABLE ocr_generations (
 fingerprint TEXT PRIMARY KEY CHECK(length(fingerprint)=64 AND fingerprint NOT GLOB '*[^0-9a-f]*'),
 identity_json TEXT NOT NULL CHECK(length(identity_json) BETWEEN 2 AND 2048),
 created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0)
);
CREATE TABLE ocr_library_settings (
 library_id INTEGER PRIMARY KEY REFERENCES libraries(id) ON DELETE CASCADE,
 enabled INTEGER NOT NULL CHECK(enabled IN (0,1)),
 clearing INTEGER NOT NULL DEFAULT 0 CHECK(clearing IN (0,1)),
 revision INTEGER NOT NULL CHECK(revision>0),
 epoch INTEGER NOT NULL CHECK(epoch>0),
 fingerprint TEXT REFERENCES ocr_generations(fingerprint) ON DELETE RESTRICT,
 CHECK(enabled=0 OR (clearing=0 AND fingerprint IS NOT NULL))
);
CREATE TABLE ocr_jobs (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 8 AND 128),
 library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
 epoch INTEGER NOT NULL CHECK(epoch>0),
 fingerprint TEXT NOT NULL REFERENCES ocr_generations(fingerprint) ON DELETE RESTRICT,
 mode TEXT NOT NULL CHECK(mode IN ('missing','retry_failed')),
 state TEXT NOT NULL CHECK(state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
 checkpoint_id INTEGER NOT NULL DEFAULT 0 CHECK(checkpoint_id>=0),
 upper_asset_id INTEGER NOT NULL CHECK(upper_asset_id>=checkpoint_id),
 claim_revision INTEGER NOT NULL DEFAULT 0 CHECK(claim_revision>=0),
 attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count BETWEEN 0 AND 3),
 completed INTEGER NOT NULL DEFAULT 0 CHECK(completed>=0),
 failed INTEGER NOT NULL DEFAULT 0 CHECK(failed BETWEEN 0 AND completed),
 error_code TEXT NOT NULL DEFAULT '' CHECK(error_code IN ('','model_unavailable','identity_mismatch','inference_failed','output_limit','preprocess_failed','source_unavailable','interrupted')),
 lease_expires_ms INTEGER,
 created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0),
 updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>=created_at_ms)
);
CREATE UNIQUE INDEX ocr_jobs_one_active ON ocr_jobs(library_id) WHERE state IN ('queued','running','cancelling');
CREATE INDEX ocr_jobs_claim ON ocr_jobs(state,created_at_ms,id);
CREATE INDEX ocr_jobs_expired ON ocr_jobs(state,lease_expires_ms,id);
CREATE TABLE ocr_job_requests (
 key_hash TEXT PRIMARY KEY CHECK(length(key_hash)=64),
 request_hash TEXT NOT NULL CHECK(length(request_hash)=64),
 job_id TEXT NOT NULL UNIQUE REFERENCES ocr_jobs(id) ON DELETE CASCADE
);
CREATE TABLE ocr_asset_results (
 library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
 fingerprint TEXT NOT NULL REFERENCES ocr_generations(fingerprint) ON DELETE RESTRICT,
 asset_id INTEGER NOT NULL,
 source_fingerprint TEXT NOT NULL CHECK(length(source_fingerprint) BETWEEN 6 AND 64),
 outcome TEXT NOT NULL CHECK(outcome IN ('ready','failed')),
 error_code TEXT NOT NULL CHECK(error_code IN ('','invalid_media','input_limit')),
 width INTEGER NOT NULL,
 height INTEGER NOT NULL,
 lines_json TEXT NOT NULL CHECK(length(lines_json)<=131072),
 updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>0),
 PRIMARY KEY(library_id,fingerprint,asset_id),
 FOREIGN KEY(library_id,asset_id) REFERENCES assets(library_id,id) ON DELETE CASCADE,
 CHECK((outcome='ready' AND error_code='' AND width BETWEEN 32 AND 2048 AND height BETWEEN 32 AND 2048 AND width*height<=4194304) OR
       (outcome='failed' AND error_code<>'' AND width=0 AND height=0 AND lines_json='null'))
);
-- +goose Down
DROP TABLE ocr_asset_results;
DROP TABLE ocr_job_requests;
DROP TABLE ocr_jobs;
DROP TABLE ocr_library_settings;
DROP TABLE ocr_generations;
