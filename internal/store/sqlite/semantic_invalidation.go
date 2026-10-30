package sqlite

import (
	"context"
	"database/sql"
)

func addSemanticCandidate(ctx context.Context, tx *sql.Tx, libraryID, now int64) error {
	if _, err := tx.ExecContext(ctx, `UPDATE semantic_library_progress
        SET eligible_count=eligible_count+1,revision=revision+1,updated_at_ms=MAX(updated_at_ms,?)
        WHERE library_id=?`, now, libraryID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE ai_library_settings SET state='building',revision=revision+1,
        updated_at_ms=MAX(updated_at_ms,?) WHERE library_id=? AND enabled=1 AND state='ready'`, now, libraryID)
	return err
}

// Successful reconciliation alone may retire missing assets. Count before FK
// cascades remove their embeddings; never run this for partial/failed scans.
func removeSemanticCandidates(ctx context.Context, tx *sql.Tx, libraryID, generation, now int64) error {
	_, err := tx.ExecContext(ctx, `WITH removed AS (
        SELECT id,source_fingerprint FROM visible_assets AS assets
        WHERE library_id=? AND last_seen_generation<? AND kind IN ('image','animated') AND archive_path = ''
    ), counts AS (
        SELECT p.generation_id, COUNT(r.id) AS eligible,
          COALESCE(SUM(e.source_fingerprint=r.source_fingerprint),0) AS completed,
          COALESCE(SUM(e.source_fingerprint<>r.source_fingerprint),0) AS stale
        FROM semantic_library_progress p CROSS JOIN removed r
        LEFT JOIN semantic_embeddings e ON e.generation_id=p.generation_id
          AND e.library_id=p.library_id AND e.asset_id=r.id
        WHERE p.library_id=? GROUP BY p.generation_id
    ) UPDATE semantic_library_progress AS p SET
        eligible_count=MAX(0,p.eligible_count-c.eligible),
        completed_count=MAX(0,p.completed_count-c.completed),
        stale_count=MAX(0,p.stale_count-c.stale),
        failed_count=MIN(p.failed_count,MAX(0,p.eligible_count-c.eligible)),
        revision=p.revision+1,updated_at_ms=MAX(p.updated_at_ms,?)
    FROM counts c WHERE p.library_id=? AND p.generation_id=c.generation_id`,
		libraryID, generation, libraryID, now, libraryID)
	return err
}

// Invalidate only embeddings that matched the pre-scan source. Retaining the
// old vector makes subsequent source changes idempotent until a new backfill.
// Run inside the catalog mutation transaction so search and coverage agree.
func invalidateSemanticSource(ctx context.Context, tx *sql.Tx, libraryID, assetID int64, previous string, now int64) error {
	result, err := tx.ExecContext(ctx, `UPDATE semantic_library_progress
        SET completed_count=MAX(0,completed_count-1), stale_count=stale_count+1,
            revision=revision+1, updated_at_ms=MAX(updated_at_ms,?)
        WHERE library_id=? AND generation_id IN (
            SELECT generation_id FROM semantic_embeddings
            WHERE library_id=? AND asset_id=? AND source_fingerprint=?)`,
		now, libraryID, libraryID, assetID, previous)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE ai_library_settings
        SET state='degraded', revision=revision+1, updated_at_ms=MAX(updated_at_ms,?)
        WHERE library_id=? AND enabled=1 AND state='ready'
          AND EXISTS (SELECT 1 FROM semantic_library_progress p
              JOIN ai_model_state s ON s.active_generation_id=p.generation_id
              WHERE p.library_id=ai_library_settings.library_id AND p.stale_count>0)`, now, libraryID)
	return err
}
