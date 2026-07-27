-- name: CountActiveScans :one
SELECT COUNT(*)
FROM scan_runs
WHERE status IN ('queued', 'running');

-- name: FindActiveScanForLibrary :one
SELECT *
FROM scan_runs
WHERE library_id = sqlc.arg(library_id)
  AND status IN ('queued', 'running')
ORDER BY id
LIMIT 1;

-- name: NextScanGeneration :one
SELECT COALESCE(MAX(generation), 0) + 1
FROM scan_runs
WHERE library_id = sqlc.arg(library_id);

-- name: InsertQueuedScan :one
INSERT INTO scan_runs(
    library_id,
    generation,
    trigger_kind,
    status,
    phase,
    created_at_ms,
    available_at_ms
) VALUES (
    sqlc.arg(library_id),
    sqlc.arg(generation),
    sqlc.arg(trigger_kind),
    'queued',
    'queued',
    sqlc.arg(created_at_ms),
    sqlc.arg(available_at_ms)
)
RETURNING *;

-- name: ClaimNextQueuedScan :one
UPDATE scan_runs
SET status = 'running',
    phase = 'checking_root',
    started_at_ms = sqlc.arg(now_ms),
    heartbeat_at_ms = sqlc.arg(now_ms),
    lease_expires_at_ms = sqlc.arg(lease_expires_at_ms),
    attempt_count = attempt_count + 1,
    revision = revision + 1
WHERE id = (
    SELECT id
    FROM scan_runs
    WHERE status = 'queued'
      AND cancel_requested_at_ms IS NULL
      AND available_at_ms <= sqlc.arg(now_ms)
      AND attempt_count < 3
    ORDER BY available_at_ms, created_at_ms, id
    LIMIT 1
)
  AND status = 'queued'
RETURNING *;

-- name: TouchScanLease :one
UPDATE scan_runs
SET heartbeat_at_ms = sqlc.arg(now_ms),
    lease_expires_at_ms = sqlc.arg(lease_expires_at_ms)
WHERE id = sqlc.arg(id)
  AND status = 'running'
RETURNING *;

-- name: UpdateRunningScanPhase :one
UPDATE scan_runs
SET phase = sqlc.arg(phase),
    revision = revision + 1
WHERE id = sqlc.arg(id)
  AND status = 'running'
  AND phase <> sqlc.arg(phase)
RETURNING *;

-- name: RequestRunningScanCancellation :one
UPDATE scan_runs
SET cancel_requested_at_ms = COALESCE(cancel_requested_at_ms, sqlc.arg(now_ms)),
    revision = CASE
        WHEN cancel_requested_at_ms IS NULL THEN revision + 1
        ELSE revision
    END
WHERE id = sqlc.arg(id)
  AND status = 'running'
RETURNING *;

-- name: CancelQueuedScan :one
UPDATE scan_runs
SET status = 'cancelled',
    phase = 'completed',
    cancel_requested_at_ms = COALESCE(cancel_requested_at_ms, sqlc.arg(now_ms)),
    finished_at_ms = sqlc.arg(now_ms),
    revision = revision + 1
WHERE id = sqlc.arg(id)
  AND status = 'queued'
RETURNING *;

-- name: RecoverNextExpiredScan :one
UPDATE scan_runs
SET status = CASE WHEN attempt_count >= 3 THEN 'interrupted' ELSE 'queued' END,
    phase = CASE WHEN attempt_count >= 3 THEN 'completed' ELSE 'queued' END,
    started_at_ms = CASE WHEN attempt_count >= 3 THEN started_at_ms ELSE NULL END,
    finished_at_ms = CASE WHEN attempt_count >= 3 THEN sqlc.arg(now_ms) ELSE NULL END,
    heartbeat_at_ms = NULL,
    lease_expires_at_ms = NULL,
    available_at_ms = sqlc.arg(now_ms),
    error_code = CASE WHEN attempt_count >= 3 THEN 'scan_interrupted' ELSE NULL END,
    revision = revision + 1
WHERE id = (
    SELECT id
    FROM scan_runs
    WHERE status = 'running'
      AND lease_expires_at_ms <= sqlc.arg(now_ms)
    ORDER BY lease_expires_at_ms, id
    LIMIT 1
)
  AND status = 'running'
RETURNING *;

-- name: GetScanContractRun :one
SELECT *
FROM scan_runs
WHERE id = sqlc.arg(id);

-- name: ListLibraryScanContractRuns :many
SELECT *
FROM scan_runs
WHERE library_id = sqlc.arg(library_id)
  AND (
      created_at_ms < sqlc.arg(before_created_at_ms)
      OR (
          created_at_ms = sqlc.arg(before_created_at_ms)
          AND id < sqlc.arg(before_id)
      )
  )
ORDER BY created_at_ms DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: ListScanIssues :many
SELECT *
FROM scan_issues
WHERE scan_run_id = sqlc.arg(scan_run_id)
ORDER BY id
LIMIT 50;

-- name: GetSettings :one
SELECT *
FROM settings
WHERE singleton_key = 1;

-- name: UpdateScheduledScanInterval :one
UPDATE settings
SET scheduled_scan_interval_hours = sqlc.narg(scheduled_scan_interval_hours),
    revision = revision + 1,
    updated_at_ms = sqlc.arg(updated_at_ms)
WHERE singleton_key = 1
  AND revision = sqlc.arg(expected_revision)
RETURNING *;
