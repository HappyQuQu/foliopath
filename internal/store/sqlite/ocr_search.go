package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"github.com/HappyQuQu/foliopath/internal/search"
	"strings"
)

// Backfill is restart-safe, bounded and inference-free. Empty documents are valid.
func (s *Store) backfillOCRSearch(ctx context.Context) error {
	for {
		count := 0
		err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
			rows, err := tx.QueryContext(ctx, `SELECT r.library_id,r.fingerprint,r.asset_id,r.lines_json FROM ocr_asset_results r LEFT JOIN ocr_search_documents d USING(library_id,fingerprint,asset_id) WHERE r.outcome='ready' AND d.id IS NULL LIMIT 100`)
			if err != nil {
				return err
			}
			type doc struct {
				library, asset    int64
				fingerprint, text string
			}
			docs := []doc{}
			for rows.Next() {
				var d doc
				var raw string
				if err = rows.Scan(&d.library, &d.fingerprint, &d.asset, &raw); err != nil {
					rows.Close()
					return err
				}
				texts, err := ocrSearchLines(raw)
				if err != nil {
					rows.Close()
					return err
				}
				d.text = catalog.SearchTextKey(strings.Join(texts, "\n"))
				docs = append(docs, d)
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
			for _, d := range docs {
				if _, err = tx.ExecContext(ctx, `INSERT INTO ocr_search_documents(library_id,fingerprint,asset_id,text_key) VALUES(?,?,?,?)`, d.library, d.fingerprint, d.asset, d.text); err != nil {
					return err
				}
			}
			count = len(docs)
			return nil
		})
		if err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
	}
}
func ocrSearchLines(raw string) ([]string, error) {
	var lines []ocr.Line
	if len(raw) > 128<<10 || json.Unmarshal([]byte(raw), &lines) != nil || len(lines) > 128 {
		return nil, ocr.ErrInvalidResult
	}
	texts := make([]string, len(lines))
	for i, line := range lines {
		texts[i] = line.Text
	}
	return texts, nil
}

func (s *Store) GetOCRSearchCoverage(ctx context.Context, scan search.Scan) (search.OCRCoverage, error) {
	var c search.OCRCoverage
	query := `SELECT COUNT(*),COALESCE(SUM(r.outcome='ready' AND r.source_fingerprint=a.source_fingerprint AND d.id IS NOT NULL),0),COALESCE(SUM(r.outcome='failed' AND r.source_fingerprint=a.source_fingerprint),0) FROM assets a JOIN libraries l ON l.id=a.library_id JOIN ocr_library_settings os ON os.library_id=a.library_id AND os.enabled=1 AND os.clearing=0 AND os.fingerprint=? LEFT JOIN ocr_asset_results r ON r.library_id=a.library_id AND r.asset_id=a.id AND r.fingerprint=os.fingerprint LEFT JOIN ocr_search_documents d ON d.library_id=r.library_id AND d.fingerprint=r.fingerprint AND d.asset_id=r.asset_id WHERE l.status='ready' AND a.kind IN ('image','animated')`
	args := []any{scan.OCRFingerprint}
	query, args = searchScopeSQL(query, args, scan.Request)
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&c.Eligible, &c.Ready, &c.Failed)
	return c, err
}
