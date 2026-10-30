package sqlite

import (
	"context"
	"database/sql"
	"strings"
)

func addVideoSemanticCandidate(ctx context.Context, tx *sql.Tx, libraryID, now int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE semantic_video_progress SET eligible_count=eligible_count+1,
 revision=revision+1,updated_at_ms=MAX(updated_at_ms,?) WHERE library_id=?`, now, libraryID)
	return err
}

// Aggregate failures have no per-asset attribution. After membership shrinks,
// retain only provably current ready coverage and return other survivors to
// pending. Historical operations remain untouched; a new backfill classifies
// pending videos. Do not re-credit old frames during an active full rebuild.
func removeVideoSemanticCandidates(ctx context.Context, tx *sql.Tx, libraryID, generation, now int64) error {
	currentPlan := strings.ReplaceAll(completeCurrentVideoPlan, "?", "p.generation_id")
	_, err := tx.ExecContext(ctx, `WITH surviving AS MATERIALIZED (
 SELECT p.generation_id, COUNT(asset.id) AS eligible,
   COALESCE(SUM(CASE WHEN `+currentPlan+` THEN 1 ELSE 0 END),0) AS ready
 FROM semantic_video_progress p
 LEFT JOIN visible_assets asset ON asset.library_id=p.library_id AND asset.kind='video'
   AND asset.last_seen_generation>=?
 LEFT JOIN thumbnails thumbnail ON thumbnail.library_id=asset.library_id
   AND thumbnail.asset_id=asset.id AND thumbnail.variant='storyboard'
 WHERE p.library_id=? AND EXISTS (
   SELECT 1 FROM visible_assets removed WHERE removed.library_id=p.library_id
     AND removed.kind='video' AND removed.last_seen_generation<?)
 GROUP BY p.generation_id
 ) UPDATE semantic_video_progress AS p SET
 eligible_count=surviving.eligible,ready_count=MIN(p.ready_count,surviving.ready),
 degraded_count=0,failed_count=0,stale_count=0,
 revision=p.revision+1,updated_at_ms=MAX(p.updated_at_ms,?)
 FROM surviving WHERE p.library_id=? AND p.generation_id=surviving.generation_id`,
		generation, libraryID, generation, now, libraryID)
	return err
}

// Count videos, never frames. Only move a previously completed result to stale;
// old frame bytes remain derived state and cannot match the new source.
func invalidateVideoSemanticSource(ctx context.Context, tx *sql.Tx, libraryID, assetID int64, previous string, now int64) error {
	// Reuse the candidate owner's complete-plan predicate. A retained old plan
	// alone does not prove this asset is still counted ready (e.g. after retry).
	currentPlan := strings.ReplaceAll(completeCurrentVideoPlan, "?", "p.generation_id")
	_, err := tx.ExecContext(ctx, `WITH live AS MATERIALIZED (
 SELECT p.generation_id, (
   SELECT COUNT(*) FROM visible_assets asset
   LEFT JOIN thumbnails thumbnail ON thumbnail.library_id=asset.library_id
     AND thumbnail.asset_id=asset.id AND thumbnail.variant='storyboard'
   WHERE asset.library_id=p.library_id AND asset.kind='video' AND (`+currentPlan+`)
 ) AS ready FROM semantic_video_progress p
 WHERE p.library_id=? AND p.generation_id IN (
   SELECT DISTINCT generation_id FROM semantic_video_frames
   WHERE library_id=? AND asset_id=? AND source_fingerprint=?)
 ) UPDATE semantic_video_progress AS p
 SET ready_count=live.ready,stale_count=p.stale_count+p.ready_count-live.ready,
 revision=p.revision+1,updated_at_ms=MAX(p.updated_at_ms,?)
 FROM live WHERE p.library_id=? AND p.generation_id=live.generation_id AND p.ready_count>live.ready`,
		libraryID, libraryID, assetID, previous, now, libraryID)
	return err
}
