package sqlite

import (
	"context"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/search"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) StreamSearchCandidates(ctx context.Context, scan search.Scan, offer func(search.Candidate) error) error {
	r := scan.Request
	if len(scan.Terms) == 0 || offer == nil {
		return search.ErrInvalid
	}
	if scan.GenerationID != "" {
		_, state, e := s.semanticGenerationContract(ctx, s.db, scan.GenerationID)
		if e != nil {
			return e
		}
		if state != "active" {
			return semantic.ErrSemanticGenerationUnavailable
		}
	}
	sql := `SELECT a.source_fingerprint,a.library_id,a.id,a.name,a.relative_path,e.vector,CASE WHEN d.id IS NOT NULL THEN r.lines_json ELSE '[]' END FROM assets a
 JOIN libraries l ON l.id=a.library_id
 LEFT JOIN ai_library_settings setting ON setting.library_id=a.library_id
 LEFT JOIN semantic_embeddings e ON e.library_id=a.library_id AND e.asset_id=a.id
 AND e.generation_id=? AND e.source_fingerprint=a.source_fingerprint
 AND setting.enabled=1 AND l.status<>'offline'
 LEFT JOIN ocr_library_settings os ON os.library_id=a.library_id AND os.enabled=1 AND os.clearing=0 AND os.fingerprint=? AND l.status='ready' AND a.kind IN ('image','animated')
 LEFT JOIN ocr_asset_results r ON r.library_id=a.library_id AND r.asset_id=a.id AND r.fingerprint=os.fingerprint AND r.source_fingerprint=a.source_fingerprint AND r.outcome='ready'
 LEFT JOIN ocr_search_documents d ON d.library_id=r.library_id AND d.fingerprint=r.fingerprint AND d.asset_id=r.asset_id WHERE 1=1`
	args := []any{scan.GenerationID, scan.OCRFingerprint}
	sql, args = searchScopeSQL(sql, args, r)
	sql += ` AND ((`
	for i, t := range scan.Terms {
		if i > 0 {
			sql += ` AND `
		}
		sql += `(instr(a.search_name_key,?)>0 OR instr(a.search_path_key,?)>0)`
		args = append(args, t, t)
	}
	sql += `) OR e.asset_id IS NOT NULL OR (d.id IS NOT NULL`
	if scan.OCRFingerprint != "" {
		if anchor := ftsAnchor(scan.Terms); anchor != "" {
			sql += ` AND d.id IN (SELECT rowid FROM ocr_search WHERE ocr_search MATCH ?)`
			args = append(args, anchor)
		}
		for _, term := range scan.Terms {
			sql += ` AND instr(d.text_key,?)>0`
			args = append(args, term)
		}
	} else {
		sql += ` AND 0`
	}
	sql += `)) ORDER BY a.id`
	rows, e := s.db.QueryContext(ctx, sql, args...)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		if e = ctx.Err(); e != nil {
			return e
		}
		var c search.Candidate
		var raw string
		if e = rows.Scan(&c.SourceFingerprint, &c.LibraryID, &c.AssetID, &c.Name, &c.RelativePath, &c.Vector, &raw); e != nil {
			return e
		}
		c.OCRLines, e = ocrSearchLines(raw)
		if e != nil {
			return e
		}
		if e = offer(c); e != nil {
			return e
		}
	}
	return rows.Err()
}

func searchScopeSQL(sql string, args []any, r search.Request) (string, []any) {
	if r.LibraryID > 0 {
		sql += ` AND a.library_id=?`
		args = append(args, r.LibraryID)
	}
	if r.DirectoryID > 0 {
		if r.Recursive {
			sql += ` AND a.directory_id IN (WITH RECURSIVE subtree(id) AS (SELECT id FROM directories WHERE library_id=? AND id=? UNION ALL SELECT d.id FROM directories d JOIN subtree p ON d.parent_id=p.id WHERE d.library_id=?) SELECT id FROM subtree)`
			args = append(args, r.LibraryID, r.DirectoryID, r.LibraryID)
		} else {
			sql += ` AND a.directory_id=?`
			args = append(args, r.DirectoryID)
		}
	}
	if len(r.Kinds) > 0 {
		sql += ` AND a.kind IN (` + strings.TrimSuffix(strings.Repeat("?,", len(r.Kinds)), ",") + `)`
		for _, k := range r.Kinds {
			args = append(args, k)
		}
	}
	if r.ModifiedFromNS != nil {
		sql += ` AND a.mtime_ns>=?`
		args = append(args, *r.ModifiedFromNS)
	}
	if r.ModifiedBeforeNS != nil {
		sql += ` AND a.mtime_ns<?`
		args = append(args, *r.ModifiedBeforeNS)
	}

	return sql, args
}
