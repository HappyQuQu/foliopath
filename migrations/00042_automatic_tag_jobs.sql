-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys=OFF;
PRAGMA legacy_alter_table=ON;
BEGIN IMMEDIATE;
ALTER TABLE ai_model_operations RENAME TO ai_model_operations_v42;
CREATE TABLE ai_model_operations (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 8 AND 128),
 kind TEXT NOT NULL CHECK(kind IN ('model_install','model_activate','semantic_missing','semantic_rebuild','semantic_clear','tag_suggestion_missing','tag_suggestion_rebuild','tag_review_clear','video_semantic_missing','video_semantic_rebuild','face_missing','face_rebuild','face_derived_clear','face_manual_clear','automatic_tags_missing','automatic_tags_rebuild')),
 state TEXT NOT NULL CHECK(state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
 phase TEXT NOT NULL CHECK(phase IN ('queued','scanning','verifying','copying','loading','building','validating','clearing','finalizing','completed')),
 model_id TEXT REFERENCES ai_models(id) ON DELETE RESTRICT,
 library_id INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
 completed_items INTEGER NOT NULL DEFAULT 0 CHECK(completed_items>=0),
 total_items INTEGER CHECK(total_items IS NULL OR total_items>=0),
 error_code TEXT CHECK(error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
 lease_expires_ms INTEGER,
 revision INTEGER NOT NULL DEFAULT 1 CHECK(revision>0),
 created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0),
 updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>=created_at_ms),
 finished_at_ms INTEGER CHECK(finished_at_ms IS NULL OR finished_at_ms>=created_at_ms)
);
INSERT INTO ai_model_operations SELECT * FROM ai_model_operations_v42 ;
DROP TABLE ai_model_operations_v42;
CREATE INDEX ai_model_operations_state ON ai_model_operations(state,created_at_ms,id);
CREATE INDEX ai_model_operations_library ON ai_model_operations(library_id,updated_at_ms DESC,id);

CREATE TABLE autotag_library_settings (
 library_id INTEGER PRIMARY KEY REFERENCES libraries(id) ON DELETE CASCADE,
 enabled INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN (0,1)),
 revision INTEGER NOT NULL DEFAULT 1 CHECK(revision>0),
 coverage_revision INTEGER NOT NULL DEFAULT 1 CHECK(coverage_revision>0),
 blocked_model_identity TEXT CHECK(blocked_model_identity IS NULL OR length(blocked_model_identity)=64),
 updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>0)
);
CREATE INDEX autotag_settings_enabled ON autotag_library_settings(enabled,library_id);
CREATE TABLE autotag_jobs (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 8 AND 128),
 operation_id TEXT NOT NULL UNIQUE REFERENCES ai_model_operations(id) ON DELETE CASCADE,
 library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
 model_identity TEXT NOT NULL CHECK(length(model_identity)=64 AND model_identity NOT GLOB '*[^0-9a-f]*'),
 mode TEXT NOT NULL CHECK(mode IN ('missing','all')),
 state TEXT NOT NULL CHECK(state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
 checkpoint_id INTEGER NOT NULL DEFAULT 0 CHECK(checkpoint_id>=0),
 upper_asset_id INTEGER NOT NULL CHECK(upper_asset_id>=0),
 claim_revision INTEGER NOT NULL DEFAULT 0 CHECK(claim_revision>=0),
 attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count BETWEEN 0 AND 3),
 next_attempt_ms INTEGER NOT NULL DEFAULT 0 CHECK(next_attempt_ms>=0),
 lease_expires_ms INTEGER,
 created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0),
 updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>=created_at_ms)
);
CREATE UNIQUE INDEX autotag_jobs_one_active ON autotag_jobs(library_id) WHERE state IN ('queued','running','cancelling');
CREATE INDEX autotag_jobs_claim ON autotag_jobs(state,next_attempt_ms,created_at_ms,id);
CREATE TABLE autotag_job_requests (
 key_hash TEXT PRIMARY KEY CHECK(length(key_hash)=64),
 request_hash TEXT NOT NULL CHECK(length(request_hash)=64),
 job_id TEXT NOT NULL REFERENCES autotag_jobs(id) ON DELETE CASCADE
);
CREATE INDEX autotag_requests_job ON autotag_job_requests(job_id);
CREATE TABLE autotag_asset_runs (
 library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
 asset_id INTEGER NOT NULL,
 source_fingerprint TEXT NOT NULL CHECK(length(source_fingerprint) BETWEEN 6 AND 64),
 model_identity TEXT NOT NULL CHECK(length(model_identity)=64),
 outcome TEXT NOT NULL CHECK(outcome IN ('ready','failed','stale')),
 updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>0),
 PRIMARY KEY(library_id,asset_id),
 FOREIGN KEY(library_id,asset_id) REFERENCES assets(library_id,id) ON DELETE CASCADE
);
COMMIT;
PRAGMA legacy_alter_table=OFF;
PRAGMA foreign_keys=ON;
-- +goose Down
PRAGMA foreign_keys=OFF;
PRAGMA legacy_alter_table=ON;
BEGIN IMMEDIATE;
DROP TABLE autotag_asset_runs;
DROP TABLE autotag_job_requests;
DROP TABLE autotag_jobs;
DROP TABLE autotag_library_settings;
ALTER TABLE ai_model_operations RENAME TO ai_model_operations_v42;
CREATE TABLE ai_model_operations (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 8 AND 128),
 kind TEXT NOT NULL CHECK(kind IN ('model_install','model_activate','semantic_missing','semantic_rebuild','semantic_clear','tag_suggestion_missing','tag_suggestion_rebuild','tag_review_clear','video_semantic_missing','video_semantic_rebuild','face_missing','face_rebuild','face_derived_clear','face_manual_clear')),
 state TEXT NOT NULL CHECK(state IN ('queued','running','cancelling','succeeded','failed','cancelled')),
 phase TEXT NOT NULL CHECK(phase IN ('queued','scanning','verifying','copying','loading','building','validating','clearing','finalizing','completed')),
 model_id TEXT REFERENCES ai_models(id) ON DELETE RESTRICT,
 library_id INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
 completed_items INTEGER NOT NULL DEFAULT 0 CHECK(completed_items>=0),
 total_items INTEGER CHECK(total_items IS NULL OR total_items>=0),
 error_code TEXT CHECK(error_code IS NULL OR length(error_code) BETWEEN 1 AND 128),
 lease_expires_ms INTEGER,
 revision INTEGER NOT NULL DEFAULT 1 CHECK(revision>0),
 created_at_ms INTEGER NOT NULL CHECK(created_at_ms>0),
 updated_at_ms INTEGER NOT NULL CHECK(updated_at_ms>=created_at_ms),
 finished_at_ms INTEGER CHECK(finished_at_ms IS NULL OR finished_at_ms>=created_at_ms)
);
INSERT INTO ai_model_operations SELECT * FROM ai_model_operations_v42 WHERE kind NOT IN ('automatic_tags_missing','automatic_tags_rebuild');
DROP TABLE ai_model_operations_v42;
CREATE INDEX ai_model_operations_state ON ai_model_operations(state,created_at_ms,id);
CREATE INDEX ai_model_operations_library ON ai_model_operations(library_id,updated_at_ms DESC,id);

COMMIT;
PRAGMA legacy_alter_table=OFF;
PRAGMA foreign_keys=ON;

