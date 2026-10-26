package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"strings"
)

func nextOCRCandidate(ctx context.Context, q queryRower, j ocr.Job) (ocr.Candidate, bool, error) {
	var c ocr.Candidate
	// A current failed source is excluded from missing to avoid poison loops.
	// Explicit retry_failed revisits failures only, never already-ready images.
	err := q.QueryRowContext(ctx, `SELECT a.id,a.source_fingerprint FROM assets a
 LEFT JOIN ocr_asset_results r ON r.library_id=a.library_id AND r.asset_id=a.id AND r.fingerprint=? AND r.source_fingerprint=a.source_fingerprint
 WHERE a.library_id=? AND a.kind IN ('image','animated') AND a.id>? AND a.id<=?
 AND ((?='missing' AND r.asset_id IS NULL) OR (?='retry_failed' AND r.outcome='failed'))
 ORDER BY a.id LIMIT 1`, j.Fingerprint, j.LibraryID, j.CheckpointID, j.UpperAssetID, j.Mode, j.Mode).Scan(&c.AssetID, &c.SourceFingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return c, false, nil
	}
	return c, err == nil, err
}
func (s *Store) NextOCRCandidate(ctx context.Context, j ocr.Job, now time.Time) (ocr.Candidate, bool, error) {
	var c ocr.Candidate
	var found bool
	if now.UnixMilli() < 1 {
		return c, false, ocr.ErrInvalidPersistence
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := requireOCRClaim(ctx, tx, j, now, ocrClaimProgress); err != nil {
			return err
		}
		var err error
		c, found, err = nextOCRCandidate(ctx, tx, j)
		return err
	})
	return c, found, err
}
func (s *Store) CommitOCRResult(ctx context.Context, commit ocr.Commit, now time.Time) error {
	if now.UnixMilli() < 1 || commit.Value.Fingerprint != commit.Job.Fingerprint {
		return ocr.ErrInvalidPersistence
	}
	v, err := ocr.ValidateStoredResult(commit.Value)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(v.Result.Lines)
	if err != nil {
		return ocr.ErrInvalidResult
	}
	if len(raw) > 128<<10 {
		return ocr.ErrInvalidResult
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		j := commit.Job
		if err := requireOCRClaim(ctx, tx, j, now, ocrClaimProgress); err != nil {
			return err
		}
		c, found, err := nextOCRCandidate(ctx, tx, j)
		if err != nil {
			return err
		}
		if !found || c != v.Candidate {
			return ocr.ErrConflict
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ocr_asset_results(library_id,fingerprint,asset_id,source_fingerprint,outcome,error_code,width,height,lines_json,updated_at_ms) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(library_id,fingerprint,asset_id) DO UPDATE SET source_fingerprint=excluded.source_fingerprint,outcome=excluded.outcome,error_code=excluded.error_code,width=excluded.width,height=excluded.height,lines_json=excluded.lines_json,updated_at_ms=excluded.updated_at_ms`, j.LibraryID, j.Fingerprint, v.AssetID, v.SourceFingerprint, v.Outcome, v.ErrorCode, v.Width, v.Height, string(raw), now.UnixMilli())
		if err != nil {
			return err
		}
		if v.Outcome == "ready" {
			texts := make([]string, len(v.Result.Lines))
			for i, line := range v.Result.Lines {
				texts[i] = line.Text
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO ocr_search_documents(library_id,fingerprint,asset_id,text_key) VALUES(?,?,?,?)`, j.LibraryID, j.Fingerprint, v.AssetID, catalog.SearchTextKey(strings.Join(texts, "\n"))); err != nil {
				return err
			}
		}
		failure := 0
		if v.Outcome == "failed" {
			failure = 1
		}
		_, err = tx.ExecContext(ctx, `UPDATE ocr_jobs SET checkpoint_id=?,completed=completed+1,failed=failed+?,updated_at_ms=MAX(created_at_ms,?) WHERE id=?`, v.AssetID, failure, now.UnixMilli(), j.ID)
		return err
	})
}
func (s *Store) GetCurrentOCRResult(ctx context.Context, libraryID, assetID int64) (ocr.StoredResult, bool, error) {
	var v ocr.StoredResult
	var raw string
	if libraryID < 1 || assetID < 1 {
		return v, false, ocr.ErrInvalidPersistence
	}
	err := s.db.QueryRowContext(ctx, `SELECT r.asset_id,r.source_fingerprint,r.fingerprint,r.outcome,r.error_code,r.width,r.height,r.lines_json
 FROM ocr_asset_results r JOIN ocr_library_settings s ON s.library_id=r.library_id AND s.fingerprint=r.fingerprint
 JOIN libraries l ON l.id=r.library_id JOIN assets a ON a.library_id=r.library_id AND a.id=r.asset_id AND a.source_fingerprint=r.source_fingerprint
 WHERE r.library_id=? AND r.asset_id=? AND s.enabled=1 AND s.clearing=0 AND l.status='ready' AND a.kind IN ('image','animated')`, libraryID, assetID).Scan(&v.AssetID, &v.SourceFingerprint, &v.Fingerprint, &v.Outcome, &v.ErrorCode, &v.Width, &v.Height, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return v, false, nil
	}
	if err != nil {
		return v, false, err
	}
	if len(raw) > 128<<10 || json.Unmarshal([]byte(raw), &v.Result.Lines) != nil {
		return ocr.StoredResult{}, false, ocr.ErrInvalidResult
	}
	v, err = ocr.ValidateStoredResult(v)
	if err != nil {
		return ocr.StoredResult{}, false, err
	}
	return v, true, nil
}
