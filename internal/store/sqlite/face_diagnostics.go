package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

// The settings read and diagnostics share one read transaction, so counts,
// generation and task samples cannot describe different attempts.
func readFaceDiagnostics(ctx context.Context, tx *sql.Tx, value *face.LibrarySettings) error {
	d := face.FailureDiagnostics{Reasons: []face.FailureReason{}, Samples: []face.FailureSample{}}
	if value.ActiveGenerationID == "" || value.State == "clearing" || value.State == "disabled" && value.Coverage.Eligible == 0 {
		value.Diagnostics = d
		return nil
	}
	var task face.DiagnosticTask
	var code sql.NullString
	var updated int64
	var jobID string
	err := tx.QueryRowContext(ctx, `SELECT id,operation_id,state,error_code,updated_at_ms FROM face_analysis_jobs WHERE library_id=? AND generation_id=? ORDER BY created_at_ms DESC,id DESC LIMIT 1`, value.LibraryID, value.ActiveGenerationID).Scan(&jobID, &task.OperationID, &task.State, &code, &updated)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		task.ErrorCode, task.UpdatedAt = code.String, time.UnixMilli(updated).UTC()
		d.LatestTask = &task
	}
	rows, err := tx.QueryContext(ctx, `SELECT failure.error_code,COUNT(*) FROM face_asset_failures failure JOIN visible_assets asset ON asset.id=failure.asset_id AND asset.library_id=failure.library_id AND asset.source_fingerprint=failure.source_fingerprint WHERE failure.library_id=? AND failure.generation_id=? AND failure.job_id=? GROUP BY failure.error_code ORDER BY failure.error_code`, value.LibraryID, value.ActiveGenerationID, jobID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var reason face.FailureReason
		if err := rows.Scan(&reason.Code, &reason.Count); err != nil {
			rows.Close()
			return err
		}
		d.Reasons = append(d.Reasons, reason)
		d.RecordedFailures += reason.Count
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	d.UnrecordedFailures = max(0, value.Coverage.Failed-d.RecordedFailures)
	rows, err = tx.QueryContext(ctx, `SELECT failure.asset_id,asset.name,asset.size_bytes,failure.error_code,failure.updated_at_ms FROM face_asset_failures failure JOIN visible_assets asset ON asset.id=failure.asset_id AND asset.library_id=failure.library_id AND asset.source_fingerprint=failure.source_fingerprint WHERE failure.library_id=? AND failure.generation_id=? AND failure.job_id=? ORDER BY failure.asset_id LIMIT ?`, value.LibraryID, value.ActiveGenerationID, jobID, face.FailureSampleLimit)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var sample face.FailureSample
		if err := rows.Scan(&sample.AssetID, &sample.Name, &sample.SizeBytes, &sample.Code, &updated); err != nil {
			return err
		}
		sample.UpdatedAt = time.UnixMilli(updated).UTC()
		d.Samples = append(d.Samples, sample)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	value.Diagnostics = d
	return nil
}
