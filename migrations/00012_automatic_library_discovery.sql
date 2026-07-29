-- +goose Up
ALTER TABLE settings
    ADD COLUMN automatic_discovery_enabled INTEGER NOT NULL DEFAULT 1
        CHECK (automatic_discovery_enabled IN (0, 1));

ALTER TABLE libraries
    ADD COLUMN automatic_discovery_status TEXT NOT NULL DEFAULT 'disabled'
        CHECK (
            automatic_discovery_status IN (
                'active', 'degraded', 'unsupported', 'disabled'
            )
        );
ALTER TABLE libraries
    ADD COLUMN automatic_discovery_error_code TEXT
        CHECK (
            automatic_discovery_error_code IS NULL
            OR automatic_discovery_error_code IN (
                'watch_unavailable',
                'watch_resource_limit',
                'watch_overflow',
                'source_unavailable',
                'internal_error'
            )
        );
ALTER TABLE libraries
    ADD COLUMN last_automatic_discovery_at_ms INTEGER
        CHECK (
            last_automatic_discovery_at_ms IS NULL
            OR last_automatic_discovery_at_ms >= 0
        );
ALTER TABLE libraries
    ADD COLUMN content_revision INTEGER NOT NULL DEFAULT 1
        CHECK (content_revision > 0);

ALTER TABLE catalog_search_state
    ADD COLUMN content_revision INTEGER NOT NULL DEFAULT 1
        CHECK (content_revision > 0);

CREATE TABLE catalog_reconcile_jobs (
    id                    INTEGER PRIMARY KEY,
    library_id            INTEGER NOT NULL
                          REFERENCES libraries(id) ON DELETE CASCADE,
    relative_dir_path     TEXT NOT NULL
                          CHECK (
                              length(relative_dir_path) <= 4096
                              AND instr(relative_dir_path, char(0)) = 0
                              AND (
                                  relative_dir_path = ''
                                  OR (
                                      substr(relative_dir_path, 1, 1) <> '/'
                                      AND substr(relative_dir_path, -1, 1) <> '/'
                                      AND instr(relative_dir_path, '//') = 0
                                      AND relative_dir_path <> '.'
                                      AND relative_dir_path <> '..'
                                      AND instr('/' || relative_dir_path || '/', '/./') = 0
                                      AND instr('/' || relative_dir_path || '/', '/../') = 0
                                  )
                              )
                          ),
    status                TEXT NOT NULL DEFAULT 'queued'
                          CHECK (status IN ('queued', 'running', 'failed')),
    requested_revision    INTEGER NOT NULL DEFAULT 1
                          CHECK (requested_revision > 0),
    claimed_revision      INTEGER
                          CHECK (
                              claimed_revision IS NULL
                              OR claimed_revision BETWEEN 1 AND requested_revision
                          ),
    available_at_ms       INTEGER NOT NULL CHECK (available_at_ms >= 0),
    lease_expires_at_ms   INTEGER,
    attempt_count         INTEGER NOT NULL DEFAULT 0
                          CHECK (attempt_count BETWEEN 0 AND 5),
    last_error_code       TEXT
                          CHECK (
                              last_error_code IS NULL
                              OR last_error_code IN (
                                  'source_unavailable',
                                  'source_unreadable',
                                  'source_changed',
                                  'watch_overflow',
                                  'watch_resource_limit',
                                  'internal_error'
                              )
                          ),
    created_at_ms         INTEGER NOT NULL CHECK (created_at_ms >= 0),
    updated_at_ms         INTEGER NOT NULL CHECK (updated_at_ms >= created_at_ms),
    UNIQUE (library_id, relative_dir_path),
    CHECK (
        (
            status = 'queued'
            AND claimed_revision IS NULL
            AND lease_expires_at_ms IS NULL
        )
        OR (
            status = 'running'
            AND claimed_revision IS NOT NULL
            AND lease_expires_at_ms IS NOT NULL
            AND attempt_count > 0
        )
        OR (
            status = 'failed'
            AND claimed_revision IS NULL
            AND lease_expires_at_ms IS NULL
            AND last_error_code IS NOT NULL
        )
    )
);

CREATE INDEX catalog_reconcile_jobs_ready
    ON catalog_reconcile_jobs(status, available_at_ms, library_id, id)
    WHERE status = 'queued';

CREATE INDEX catalog_reconcile_jobs_expired_lease
    ON catalog_reconcile_jobs(lease_expires_at_ms, id)
    WHERE status = 'running';

CREATE UNIQUE INDEX catalog_reconcile_jobs_one_running_per_library
    ON catalog_reconcile_jobs(library_id)
    WHERE status = 'running';

-- +goose StatementBegin
CREATE TRIGGER libraries_automatic_discovery_state_insert
BEFORE INSERT ON libraries
WHEN NOT (
    (
        NEW.automatic_discovery_status IN ('active', 'disabled')
        AND NEW.automatic_discovery_error_code IS NULL
    )
    OR (
        NEW.automatic_discovery_status IN ('degraded', 'unsupported')
        AND NEW.automatic_discovery_error_code IS NOT NULL
    )
)
BEGIN
    SELECT RAISE(ABORT, 'invalid automatic discovery state');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER libraries_automatic_discovery_state_update
BEFORE UPDATE OF automatic_discovery_status, automatic_discovery_error_code
ON libraries
WHEN NOT (
    (
        NEW.automatic_discovery_status IN ('active', 'disabled')
        AND NEW.automatic_discovery_error_code IS NULL
    )
    OR (
        NEW.automatic_discovery_status IN ('degraded', 'unsupported')
        AND NEW.automatic_discovery_error_code IS NOT NULL
    )
)
BEGIN
    SELECT RAISE(ABORT, 'invalid automatic discovery state');
END;
-- +goose StatementEnd

-- Full scans and targeted reconciliation are serialized per library. Claim
-- queries still skip conflicting work; these triggers are the final invariant.
-- +goose StatementBegin
CREATE TRIGGER catalog_reconcile_no_active_full_scan
BEFORE UPDATE OF status ON catalog_reconcile_jobs
WHEN NEW.status = 'running'
  AND OLD.status <> 'running'
  AND EXISTS (
      SELECT 1
      FROM scan_runs
      WHERE library_id = NEW.library_id
        AND status IN ('queued', 'running')
  )
BEGIN
    SELECT RAISE(ABORT, 'library full scan is active');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER scan_run_no_running_reconcile
BEFORE UPDATE OF status ON scan_runs
WHEN NEW.status = 'running'
  AND OLD.status <> 'running'
  AND EXISTS (
      SELECT 1
      FROM catalog_reconcile_jobs
      WHERE library_id = NEW.library_id
        AND status = 'running'
  )
BEGIN
    SELECT RAISE(ABORT, 'library reconciliation is active');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER catalog_content_revision_library_insert
AFTER INSERT ON libraries
BEGIN
    UPDATE catalog_search_state
    SET content_revision = content_revision + 1
    WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER catalog_content_revision_library_delete
AFTER DELETE ON libraries
BEGIN
    UPDATE catalog_search_state
    SET content_revision = content_revision + 1
    WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER catalog_content_revision_generation_publish
AFTER UPDATE OF current_generation ON libraries
WHEN NEW.current_generation <> OLD.current_generation
BEGIN
    UPDATE libraries
    SET content_revision = content_revision + 1
    WHERE id = NEW.id;
    UPDATE catalog_search_state
    SET content_revision = content_revision + 1
    WHERE singleton_key = 1;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS catalog_content_revision_generation_publish;
DROP TRIGGER IF EXISTS catalog_content_revision_library_delete;
DROP TRIGGER IF EXISTS catalog_content_revision_library_insert;
DROP TRIGGER IF EXISTS scan_run_no_running_reconcile;
DROP TRIGGER IF EXISTS catalog_reconcile_no_active_full_scan;
DROP TRIGGER IF EXISTS libraries_automatic_discovery_state_update;
DROP TRIGGER IF EXISTS libraries_automatic_discovery_state_insert;
DROP INDEX IF EXISTS catalog_reconcile_jobs_one_running_per_library;
DROP INDEX IF EXISTS catalog_reconcile_jobs_expired_lease;
DROP INDEX IF EXISTS catalog_reconcile_jobs_ready;
DROP TABLE IF EXISTS catalog_reconcile_jobs;
ALTER TABLE catalog_search_state DROP COLUMN content_revision;
ALTER TABLE libraries DROP COLUMN content_revision;
ALTER TABLE libraries DROP COLUMN last_automatic_discovery_at_ms;
ALTER TABLE libraries DROP COLUMN automatic_discovery_error_code;
ALTER TABLE libraries DROP COLUMN automatic_discovery_status;
ALTER TABLE settings DROP COLUMN automatic_discovery_enabled;
