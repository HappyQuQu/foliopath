-- +goose Up
UPDATE media_jobs
SET status = 'queued',
    last_error_code = NULL,
    available_at_ms = 0,
    started_at_ms = NULL,
    heartbeat_at_ms = NULL,
    lease_expires_at_ms = NULL,
    attempt_count = 0,
    finished_at_ms = NULL
WHERE variant = 'grid'
  AND asset_id IN (
      SELECT id
      FROM assets
      WHERE media_format = 'avi'
  );

UPDATE assets
SET width = NULL,
    height = NULL,
    duration_ms = NULL,
    probe_status = 'pending',
    probe_error_code = NULL,
    playback_status = 'unknown'
WHERE media_format = 'avi';

-- +goose Down
SELECT 1;
