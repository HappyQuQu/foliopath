-- +goose Up
-- Durable full-scan admission and observation contract. scan_runs is the
-- authoritative queue record; process-local channels are wake-up signals only.
ALTER TABLE scan_runs
    ADD COLUMN revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0);
ALTER TABLE scan_runs
    ADD COLUMN phase TEXT NOT NULL DEFAULT 'queued'
        CHECK (phase IN ('queued', 'checking_root', 'walking', 'indexing', 'finalizing', 'completed'));
ALTER TABLE scan_runs
    ADD COLUMN processed_assets INTEGER NOT NULL DEFAULT 0 CHECK (processed_assets >= 0);
ALTER TABLE scan_runs
    ADD COLUMN skipped_directories INTEGER NOT NULL DEFAULT 0 CHECK (skipped_directories >= 0);
ALTER TABLE scan_runs
    ADD COLUMN skipped_files INTEGER NOT NULL DEFAULT 0 CHECK (skipped_files >= 0);
ALTER TABLE scan_runs
    ADD COLUMN error_count INTEGER NOT NULL DEFAULT 0 CHECK (error_count >= 0);
ALTER TABLE scan_runs
    ADD COLUMN issues_truncated INTEGER NOT NULL DEFAULT 0 CHECK (issues_truncated IN (0, 1));
ALTER TABLE scan_runs
    ADD COLUMN cancel_requested_at_ms INTEGER;
ALTER TABLE scan_runs
    ADD COLUMN heartbeat_at_ms INTEGER;
ALTER TABLE scan_runs
    ADD COLUMN available_at_ms INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scan_runs
    ADD COLUMN lease_expires_at_ms INTEGER;
ALTER TABLE scan_runs
    ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 3);

UPDATE scan_runs
SET phase = CASE
        WHEN status = 'queued' THEN 'queued'
        WHEN status = 'running' THEN 'walking'
        ELSE 'completed'
    END,
    processed_assets = discovered_assets,
    skipped_files = skipped_count,
    error_count = CASE WHEN error_code IS NULL THEN 0 ELSE 1 END,
    available_at_ms = created_at_ms,
    heartbeat_at_ms = CASE WHEN status = 'running' THEN started_at_ms ELSE NULL END,
    lease_expires_at_ms = CASE
        WHEN status = 'running' THEN started_at_ms + 120000
        ELSE NULL
    END,
    attempt_count = CASE WHEN status = 'running' THEN 1 ELSE 0 END;

CREATE INDEX scan_runs_ready_queue
    ON scan_runs(available_at_ms, created_at_ms, id)
    WHERE status = 'queued';

CREATE INDEX scan_runs_expired_lease
    ON scan_runs(lease_expires_at_ms, id)
    WHERE status = 'running';

CREATE TABLE scan_issues (
    id                   INTEGER PRIMARY KEY,
    scan_run_id          INTEGER NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    code                 TEXT NOT NULL
                             CHECK (code IN (
                                 'unreadable_directory',
                                 'unsupported_file',
                                 'invalid_media',
                                 'media_probe_failed',
                                 'symlink_skipped',
                                 'maintained_directory_skipped',
                                 'source_changed',
                                 'io_error'
                             )),
    issue_count          INTEGER NOT NULL CHECK (issue_count > 0),
    sample_rel_path      TEXT
                             CHECK (
                                 sample_rel_path IS NULL
                                 OR (
                                     length(sample_rel_path) BETWEEN 1 AND 4096
                                     AND instr(sample_rel_path, char(0)) = 0
                                 )
                             ),
    created_at_ms        INTEGER NOT NULL,
    UNIQUE (scan_run_id, code, sample_rel_path)
);

CREATE INDEX scan_issues_run
    ON scan_issues(scan_run_id, id);

-- +goose StatementBegin
CREATE TRIGGER scan_issues_bounded
BEFORE INSERT ON scan_issues
WHEN (SELECT COUNT(*) FROM scan_issues WHERE scan_run_id = NEW.scan_run_id) >= 50
BEGIN
    SELECT RAISE(ABORT, 'scan issue limit reached');
END;
-- +goose StatementEnd

-- A typed singleton setting keeps the scheduler restart-safe. Other accepted
-- settings share this row so the settings API has one validator.
CREATE TABLE settings (
    singleton_key                 INTEGER PRIMARY KEY CHECK (singleton_key = 1),
    scheduled_scan_interval_hours INTEGER
                                      CHECK (
                                          scheduled_scan_interval_hours IS NULL
                                          OR scheduled_scan_interval_hours BETWEEN 1 AND 8760
                                      ),
    thumbnail_cache_quota_bytes   INTEGER NOT NULL DEFAULT 10737418240
                                      CHECK (thumbnail_cache_quota_bytes > 0),
    language                      TEXT NOT NULL DEFAULT 'browser'
                                      CHECK (language IN ('browser', 'zh-CN', 'en')),
    revision                      INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_at_ms                 INTEGER NOT NULL
);

INSERT INTO settings(
    singleton_key,
    scheduled_scan_interval_hours,
    thumbnail_cache_quota_bytes,
    language,
    revision,
    updated_at_ms
) VALUES (1, 24, 10737418240, 'browser', 1, 0);

-- +goose Down
DROP TABLE IF EXISTS settings;
DROP TRIGGER IF EXISTS scan_issues_bounded;
DROP INDEX IF EXISTS scan_issues_run;
DROP TABLE IF EXISTS scan_issues;
DROP INDEX IF EXISTS scan_runs_expired_lease;
DROP INDEX IF EXISTS scan_runs_ready_queue;
ALTER TABLE scan_runs DROP COLUMN attempt_count;
ALTER TABLE scan_runs DROP COLUMN lease_expires_at_ms;
ALTER TABLE scan_runs DROP COLUMN available_at_ms;
ALTER TABLE scan_runs DROP COLUMN heartbeat_at_ms;
ALTER TABLE scan_runs DROP COLUMN cancel_requested_at_ms;
ALTER TABLE scan_runs DROP COLUMN issues_truncated;
ALTER TABLE scan_runs DROP COLUMN error_count;
ALTER TABLE scan_runs DROP COLUMN skipped_files;
ALTER TABLE scan_runs DROP COLUMN skipped_directories;
ALTER TABLE scan_runs DROP COLUMN processed_assets;
ALTER TABLE scan_runs DROP COLUMN phase;
ALTER TABLE scan_runs DROP COLUMN revision;
