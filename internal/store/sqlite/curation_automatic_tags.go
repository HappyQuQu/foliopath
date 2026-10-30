package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
)

func suppressTagAssignmentsTx(ctx context.Context, tx *sql.Tx, tagID int64, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO asset_tag_suppressions
        (library_id,asset_id,normalized_name,created_at_ms,updated_at_ms)
        SELECT assignment.library_id,assignment.asset_id,tag.normalized_name,?,?
        FROM asset_tags assignment JOIN tags tag ON tag.id=assignment.tag_id
        WHERE assignment.tag_id=?
        ON CONFLICT(library_id,asset_id,normalized_name) DO UPDATE SET
            updated_at_ms=MAX(asset_tag_suppressions.updated_at_ms,excluded.updated_at_ms)`, now.UnixMilli(), now.UnixMilli(), tagID)
	if err != nil {
		return fmt.Errorf("protect global tag edit: %w", err)
	}
	return nil
}

func suppressAssetTagTx(ctx context.Context, tx *sql.Tx, libraryID, assetID, tagID int64, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO asset_tag_suppressions
        (library_id,asset_id,normalized_name,created_at_ms,updated_at_ms)
        SELECT ?,?,normalized_name,?,? FROM tags WHERE id=?
        ON CONFLICT(library_id,asset_id,normalized_name) DO UPDATE SET
            updated_at_ms=MAX(asset_tag_suppressions.updated_at_ms,excluded.updated_at_ms)`, libraryID, assetID, now.UnixMilli(), now.UnixMilli(), tagID)
	if err != nil {
		return fmt.Errorf("protect removed asset tag: %w", err)
	}
	return nil
}

func clearAssetTagSuppressionTx(ctx context.Context, tx *sql.Tx, libraryID, assetID, tagID int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM asset_tag_suppressions
        WHERE library_id=? AND asset_id=? AND normalized_name=(SELECT normalized_name FROM tags WHERE id=?)`, libraryID, assetID, tagID)
	if err != nil {
		return fmt.Errorf("restore manually added tag: %w", err)
	}
	return nil
}

// mergeAutomaticAssetTagsTx is the one adapter-level composition point for the
// curation policy. The job owner MUST fence its live claim, settings, model and
// cancellation in tx, and commit the checkpoint in that same tx. It must roll back
// on every error. No inference, filesystem access, or separate commit occurs here.
func mergeAutomaticAssetTagsTx(ctx context.Context, tx *sql.Tx, input curation.AutomaticTagPlan, now time.Time) (curation.AutomaticTagMergeResult, error) {
	var result curation.AutomaticTagMergeResult
	plan, err := curation.NormalizeAutomaticTagPlan(input)
	if err != nil || tx == nil || now.IsZero() || now.UnixMilli() <= 0 {
		return result, curation.ErrInvalidRequest
	}
	var fingerprint, kind string
	if err := tx.QueryRowContext(ctx, `SELECT source_fingerprint,kind FROM visible_assets AS assets WHERE library_id=? AND id=?`, plan.LibraryID, plan.AssetID).Scan(&fingerprint, &kind); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, curation.ErrAutomaticTagSourceChanged
		}
		return result, fmt.Errorf("resolve automatic tag source: %w", err)
	}
	if fingerprint != plan.SourceFingerprint || kind != "image" && kind != "animated" {
		return result, curation.ErrAutomaticTagSourceChanged
	}

	// One image has at most 20 visible associations. Historical suppression rows
	// are not loaded; each candidate performs an indexed existence lookup instead.
	rows, err := tx.QueryContext(ctx, `SELECT assignment.tag_id,tag.normalized_name,assignment.source
        FROM asset_tags assignment JOIN tags tag ON tag.id=assignment.tag_id
        WHERE assignment.library_id=? AND assignment.asset_id=?`, plan.LibraryID, plan.AssetID)
	if err != nil {
		return result, fmt.Errorf("read protected automatic tag assignments: %w", err)
	}
	manual := make(map[string]bool)
	currentAI := make(map[int64]bool)
	for rows.Next() {
		var id int64
		var key, source string
		if err := rows.Scan(&id, &key, &source); err != nil {
			rows.Close()
			return result, err
		}
		if source == string(curation.TagSourceManual) {
			manual[key] = true
			if category := curation.AutomaticCategoryKey(key); category != "" {
				manual[category] = true
			}
		} else if source == string(curation.TagSourceAI) {
			currentAI[id] = true
		} else {
			rows.Close()
			return result, curation.ErrInvalidRequest
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	// Alias keys are additional protection, not additional visible assignments.
	var manualCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM asset_tags WHERE library_id=? AND asset_id=? AND source='manual'`, plan.LibraryID, plan.AssetID).Scan(&manualCount); err != nil {
		return result, err
	}
	if manualCount+len(currentAI) > curation.MaxTagsPerAsset {
		return result, curation.ErrInvalidRequest
	}
	type selectedLabel struct {
		id         int64
		confidence float64
	}
	wanted := make([]selectedLabel, 0, len(plan.Labels))
	wantedIDs := make(map[int64]bool)
	for _, label := range plan.Labels {
		name, key, _ := curation.NormalizeTagName(label.Name) // already validated above
		if manual[key] {
			continue
		}
		var suppressed bool
		keys := curation.AutomaticTagProtectionKeys(key)
		args := []any{plan.LibraryID, plan.AssetID}
		for _, protectionKey := range keys {
			args = append(args, protectionKey)
		}
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM asset_tag_suppressions
            WHERE library_id=? AND asset_id=? AND normalized_name IN (`+strings.TrimSuffix(strings.Repeat("?,", len(keys)), ",")+`))`, args...).Scan(&suppressed); err != nil {
			return result, fmt.Errorf("read manual tag suppression: %w", err)
		}
		if suppressed {
			result.SkippedSuppressed++
			continue
		}
		if manualCount+len(wanted) >= curation.MaxTagsPerAsset {
			result.SkippedLimit++
			continue
		}
		var tagID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM tags WHERE normalized_name=?`, key).Scan(&tagID)
		if errors.Is(err, sql.ErrNoRows) {
			inserted, insertErr := tx.ExecContext(ctx, `INSERT INTO tags(name,normalized_name,created_at_ms,updated_at_ms)
                VALUES(?,?,?,?)`, name, key, now.UnixMilli(), now.UnixMilli())
			if insertErr != nil {
				return result, fmt.Errorf("create automatic result tag: %w", insertErr)
			}
			tagID, err = inserted.LastInsertId()
		}
		if err != nil {
			return result, fmt.Errorf("resolve automatic result tag: %w", err)
		}
		wanted = append(wanted, selectedLabel{id: tagID, confidence: label.Confidence})
		wantedIDs[tagID] = true
	}
	for id := range currentAI {
		if wantedIDs[id] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM asset_tags WHERE library_id=? AND asset_id=? AND tag_id=? AND source='ai'`, plan.LibraryID, plan.AssetID, id); err != nil {
			return result, fmt.Errorf("retire replaced automatic tag: %w", err)
		}
		result.Removed++
	}
	for _, label := range wanted {
		if _, err := tx.ExecContext(ctx, `INSERT INTO asset_tags
            (library_id,asset_id,tag_id,created_at_ms,source,ai_model_identity,ai_source_fingerprint,ai_confidence)
            VALUES(?,?,?,?,'ai',?,?,?)
            ON CONFLICT(asset_id,tag_id) DO UPDATE SET
                ai_model_identity=excluded.ai_model_identity,
                ai_source_fingerprint=excluded.ai_source_fingerprint,
                ai_confidence=excluded.ai_confidence
            WHERE asset_tags.source='ai' AND
              (asset_tags.ai_model_identity IS NOT excluded.ai_model_identity
                OR asset_tags.ai_source_fingerprint IS NOT excluded.ai_source_fingerprint
                OR asset_tags.ai_confidence IS NOT excluded.ai_confidence)`, plan.LibraryID, plan.AssetID, label.id, now.UnixMilli(), plan.ModelIdentity, plan.SourceFingerprint, label.confidence); err != nil {
			return result, fmt.Errorf("merge automatic tag association: %w", err)
		}
		if !currentAI[label.id] {
			result.Added++
		}
	}
	return result, nil
}
