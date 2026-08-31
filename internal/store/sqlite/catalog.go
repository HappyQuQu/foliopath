package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/media"
)

func (s *Store) ResolveGlobalCatalogRevision(ctx context.Context) (int64, error) {
	var revision int64
	if err := s.db.QueryRowContext(ctx, `
        SELECT revision
        FROM catalog_search_state
        WHERE singleton_key = 1`).Scan(&revision); err != nil {
		return 0, fmt.Errorf("resolve global catalog revision: %w", err)
	}
	if revision < 1 {
		return 0, errors.New("global catalog revision is invalid")
	}
	return revision, nil
}

func (s *Store) ResolveCatalogContentRevision(ctx context.Context) (int64, error) {
	var revision int64
	if err := s.db.QueryRowContext(ctx, `
        SELECT content_revision
        FROM catalog_search_state
        WHERE singleton_key = 1`).Scan(&revision); err != nil {
		return 0, fmt.Errorf("resolve catalog content revision: %w", err)
	}
	if revision < 1 {
		return 0, errors.New("catalog content revision is invalid")
	}
	return revision, nil
}

func (s *Store) ResolveScope(
	ctx context.Context,
	libraryID, selectedDirectoryID int64,
) (catalog.Scope, error) {
	if libraryID <= 0 || selectedDirectoryID < 0 {
		return catalog.Scope{}, catalog.ErrInvalidQuery
	}
	var generation int64
	var status string
	if err := s.db.QueryRowContext(ctx, `
        SELECT current_generation, status
        FROM libraries
        WHERE id = ?`,
		libraryID,
	).Scan(&generation, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return catalog.Scope{}, catalog.ErrLibraryNotFound
		}
		return catalog.Scope{}, fmt.Errorf("resolve catalog library: %w", err)
	}

	var rootDirectoryID int64
	err := s.db.QueryRowContext(ctx, `
        SELECT id
        FROM directories
        WHERE library_id = ? AND relative_path = ''`,
		libraryID,
	).Scan(&rootDirectoryID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return catalog.Scope{}, fmt.Errorf("resolve indexed catalog root: %w", err)
	}

	directoryID := selectedDirectoryID
	if selectedDirectoryID == 0 {
		directoryID = rootDirectoryID
	} else {
		var exists int64
		if err := s.db.QueryRowContext(ctx, `
            SELECT id
            FROM directories
            WHERE library_id = ? AND id = ?`,
			libraryID, selectedDirectoryID,
		).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return catalog.Scope{}, catalog.ErrDirectoryNotFound
			}
			return catalog.Scope{}, fmt.Errorf("resolve catalog directory: %w", err)
		}
		directoryID = exists
	}

	availability := catalog.SourceAvailable
	if status == "offline" {
		availability = catalog.SourceOffline
	}
	return catalog.Scope{
		LibraryID:       libraryID,
		RootDirectoryID: rootDirectoryID,
		DirectoryID:     directoryID,
		Generation:      generation,
		Availability:    availability,
	}, nil
}

func (s *Store) ListDirectoryPage(
	ctx context.Context,
	params catalog.DirectoryListParams,
) ([]catalog.Directory, error) {
	if params.Scope.LibraryID <= 0 || params.Scope.DirectoryID < 0 ||
		params.Limit < 1 || params.Limit > catalog.MaxPageSize+1 {
		return nil, catalog.ErrInvalidQuery
	}
	if params.Scope.DirectoryID == 0 {
		return []catalog.Directory{}, nil
	}
	query := `
        SELECT id, library_id, parent_id, relative_path, name, natural_name_key,
               direct_asset_count, recursive_asset_count,
               EXISTS(
                   SELECT 1 FROM directories child
                   WHERE child.library_id = directories.library_id
                     AND child.parent_id = directories.id
               )
        FROM directories
        WHERE library_id = ? AND parent_id = ?`
	args := []any{params.Scope.LibraryID, params.Scope.DirectoryID}
	for _, term := range params.SearchTerms {
		if term == "" {
			return nil, catalog.ErrInvalidQuery
		}
		query += `
          AND instr(search_name_key, ?) > 0`
		args = append(args, term)
	}
	if params.After != nil {
		if params.After.ID <= 0 || params.After.Name == "" || len(params.After.NaturalNameKey) == 0 {
			return nil, catalog.ErrInvalidCursor
		}
		query += `
          AND (
            natural_name_key > ?
            OR (natural_name_key = ? AND name > ?)
            OR (natural_name_key = ? AND name = ? AND id > ?)
          )`
		args = append(args,
			params.After.NaturalNameKey,
			params.After.NaturalNameKey, params.After.Name,
			params.After.NaturalNameKey, params.After.Name, params.After.ID,
		)
	}
	query += ` ORDER BY natural_name_key, name, id LIMIT ?`
	args = append(args, params.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog directories: %w", err)
	}
	defer rows.Close()
	items := make([]catalog.Directory, 0, params.Limit)
	for rows.Next() {
		var item catalog.Directory
		var parentID sql.NullInt64
		var hasChildren int64
		if err := rows.Scan(
			&item.ID, &item.LibraryID, &parentID, &item.RelativePath,
			&item.Name, &item.NaturalNameKey, &item.DirectAssetCount,
			&item.RecursiveAssetCount, &hasChildren,
		); err != nil {
			return nil, fmt.Errorf("read catalog directory: %w", err)
		}
		if parentID.Valid {
			item.ParentID = &parentID.Int64
		}
		item.HasChildren = hasChildren != 0
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog directories: %w", err)
	}
	return items, nil
}

func (s *Store) GetDirectoryLineage(
	ctx context.Context,
	directoryID int64,
	maximum int,
) (catalog.DirectoryLineage, error) {
	if directoryID <= 0 {
		return catalog.DirectoryLineage{}, catalog.ErrDirectoryNotFound
	}
	if maximum < 1 || maximum > catalog.MaxBreadcrumbs {
		return catalog.DirectoryLineage{}, catalog.ErrInvalidQuery
	}
	var libraryID int64
	var libraryName string
	if err := s.db.QueryRowContext(ctx, `
        SELECT d.library_id, l.name
        FROM directories d
        JOIN libraries l ON l.id = d.library_id
        WHERE d.id = ?`,
		directoryID,
	).Scan(&libraryID, &libraryName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return catalog.DirectoryLineage{}, catalog.ErrDirectoryNotFound
		}
		return catalog.DirectoryLineage{}, fmt.Errorf("resolve catalog directory lineage: %w", err)
	}

	reversed := make([]catalog.Directory, 0, 16)
	seen := make(map[int64]struct{}, 16)
	currentID := directoryID
	for {
		if err := ctx.Err(); err != nil {
			return catalog.DirectoryLineage{}, err
		}
		if len(reversed) >= maximum {
			return catalog.DirectoryLineage{}, catalog.ErrInvalidTopology
		}
		if _, duplicate := seen[currentID]; duplicate {
			return catalog.DirectoryLineage{}, catalog.ErrInvalidTopology
		}
		seen[currentID] = struct{}{}
		var item catalog.Directory
		var parentID sql.NullInt64
		var hasChildren int64
		err := s.db.QueryRowContext(ctx, `
            SELECT d.id, d.library_id, d.parent_id, d.relative_path, d.name,
                   d.natural_name_key, d.direct_asset_count,
                   d.recursive_asset_count,
                   EXISTS(
                       SELECT 1 FROM directories child
                       WHERE child.library_id = d.library_id
                         AND child.parent_id = d.id
                   )
            FROM directories d
            WHERE d.id = ? AND d.library_id = ?`,
			currentID, libraryID,
		).Scan(
			&item.ID, &item.LibraryID, &parentID, &item.RelativePath, &item.Name,
			&item.NaturalNameKey, &item.DirectAssetCount, &item.RecursiveAssetCount,
			&hasChildren,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return catalog.DirectoryLineage{}, catalog.ErrInvalidTopology
			}
			return catalog.DirectoryLineage{}, fmt.Errorf("read catalog directory ancestor: %w", err)
		}
		item.HasChildren = hasChildren != 0
		if parentID.Valid {
			parent := parentID.Int64
			item.ParentID = &parent
		}
		reversed = append(reversed, item)
		if item.ParentID == nil {
			break
		}
		currentID = *item.ParentID
	}

	items := make([]catalog.Directory, len(reversed))
	for index := range reversed {
		items[len(reversed)-1-index] = reversed[index]
	}
	return catalog.DirectoryLineage{LibraryName: libraryName, Items: items}, nil
}

func (s *Store) ListAssetPage(
	ctx context.Context,
	params catalog.AssetListParams,
) ([]catalog.Asset, error) {
	if params.Limit < 1 || params.Limit > catalog.MaxPageSize+1 ||
		(params.Query.Sort != catalog.SortName &&
			params.Query.Sort != catalog.SortModifiedAt &&
			params.Query.Sort != catalog.SortSize) ||
		(params.Query.Order != catalog.OrderAsc && params.Query.Order != catalog.OrderDesc) {
		return nil, catalog.ErrInvalidQuery
	}
	switch params.Query.ScopeKind {
	case "", catalog.ScopeDirectory:
		if params.Query.Scope.LibraryID <= 0 || params.Query.Scope.DirectoryID < 0 {
			return nil, catalog.ErrInvalidQuery
		}
	case catalog.ScopeLibrary:
		if params.Query.Scope.LibraryID <= 0 || len(params.Query.SearchTerms) == 0 {
			return nil, catalog.ErrInvalidQuery
		}
	case catalog.ScopeGlobal:
		if params.Query.Scope.LibraryID != 0 || params.Query.CatalogRevision < 1 ||
			len(params.Query.SearchTerms) == 0 || params.Query.Recursive {
			return nil, catalog.ErrInvalidQuery
		}
	default:
		return nil, catalog.ErrInvalidQuery
	}
	if params.Query.ModifiedFromNS != nil && params.Query.ModifiedBeforeNS != nil &&
		*params.Query.ModifiedFromNS >= *params.Query.ModifiedBeforeNS {
		return nil, catalog.ErrInvalidQuery
	}
	if (params.Query.ScopeKind == "" || params.Query.ScopeKind == catalog.ScopeDirectory) &&
		params.Query.Scope.DirectoryID == 0 {
		return []catalog.Asset{}, nil
	}

	var builder strings.Builder
	args := make([]any, 0, 40)
	if (params.Query.ScopeKind == "" || params.Query.ScopeKind == catalog.ScopeDirectory) &&
		params.Query.Recursive && params.Query.Scope.CanonicalDirectoryID != 0 {
		builder.WriteString(`
        WITH RECURSIVE subtree(id) AS (
            SELECT ?
            UNION
            SELECT d.id
            FROM directories d
            JOIN subtree parent ON d.parent_id = parent.id
            WHERE d.library_id = ?
        )`)
		args = append(args, params.Query.Scope.DirectoryID, params.Query.Scope.LibraryID)
	}
	builder.WriteString(`
        SELECT a.id, a.library_id, l.name, a.directory_id, a.relative_path, a.name,
               a.natural_name_key, a.kind, a.media_format, a.mime_type,
               a.size_bytes, a.mtime_ns, a.source_fingerprint,
               a.width, a.height, a.duration_ms, a.probe_status,
               a.probe_error_code, a.playback_status,
               t.status, t.error_code,
               storyboard.status, storyboard.error_code,
               storyboard.frame_count, storyboard.sprite_columns,
               storyboard.sprite_rows, storyboard.cell_width,
               storyboard.cell_height,
               l.status,
               EXISTS(SELECT 1 FROM asset_favorites favorite WHERE favorite.asset_id = a.id)
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        LEFT JOIN thumbnails t ON t.asset_id = a.id AND t.variant = 'grid'
        LEFT JOIN thumbnails storyboard
          ON storyboard.asset_id = a.id
         AND storyboard.variant = 'storyboard'`)
	if anchor := ftsAnchor(params.Query.SearchTerms); anchor != "" {
		builder.WriteString(`
        JOIN asset_search ON asset_search.rowid = a.id
        WHERE asset_search MATCH ?`)
		args = append(args, anchor)
	} else {
		builder.WriteString(`
        WHERE 1 = 1`)
	}
	switch params.Query.ScopeKind {
	case catalog.ScopeLibrary:
		builder.WriteString(` AND a.library_id = ?`)
		args = append(args, params.Query.Scope.LibraryID)
	case "", catalog.ScopeDirectory:
		builder.WriteString(` AND a.library_id = ?`)
		args = append(args, params.Query.Scope.LibraryID)
		if !params.Query.Recursive {
			builder.WriteString(` AND a.directory_id = ?`)
			args = append(args, params.Query.Scope.DirectoryID)
		} else if params.Query.Scope.CanonicalDirectoryID != 0 {
			builder.WriteString(` AND a.directory_id IN (SELECT id FROM subtree)`)
		}
	}
	for _, term := range params.Query.SearchTerms {
		builder.WriteString(`
          AND (
            instr(a.search_name_key, ?) > 0
            OR instr(a.search_path_key, ?) > 0
          )`)
		args = append(args, term, term)
	}
	if len(params.Query.Kinds) > 0 {
		builder.WriteString(` AND a.kind IN (`)
		for index, kind := range params.Query.Kinds {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteByte('?')
			args = append(args, string(kind))
		}
		builder.WriteByte(')')
	}
	if params.Query.ModifiedFromNS != nil {
		builder.WriteString(` AND a.mtime_ns >= ?`)
		args = append(args, *params.Query.ModifiedFromNS)
	}
	if params.Query.ModifiedBeforeNS != nil {
		builder.WriteString(` AND a.mtime_ns < ?`)
		args = append(args, *params.Query.ModifiedBeforeNS)
	}
	if params.After != nil {
		if params.After.ID <= 0 {
			return nil, catalog.ErrInvalidCursor
		}
		appendAssetKeyset(&builder, &args, params.Query, *params.After)
	}
	appendAssetOrder(&builder, params.Query)
	builder.WriteString(` LIMIT ?`)
	args = append(args, params.Limit)

	rows, err := s.db.QueryContext(ctx, builder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog assets: %w", err)
	}
	defer rows.Close()
	items := make([]catalog.Asset, 0, params.Limit)
	for rows.Next() {
		var item catalog.Asset
		var kind, probeStatus, playbackStatus, libraryStatus string
		var width, height, durationMS sql.NullInt64
		var storyboardFrameCount, storyboardColumns, storyboardRows sql.NullInt64
		var storyboardCellWidth, storyboardCellHeight sql.NullInt64
		var probeError, thumbnailStatus, thumbnailError sql.NullString
		var storyboardStatus, storyboardError sql.NullString
		var favorite int64
		if err := rows.Scan(
			&item.ID, &item.LibraryID, &item.LibraryName,
			&item.DirectoryID, &item.RelativePath,
			&item.Name, &item.NaturalNameKey, &kind, &item.MediaFormat,
			&item.MIMEType, &item.SizeBytes, &item.ModifiedAtNS,
			&item.SourceFingerprint, &width, &height, &durationMS,
			&probeStatus, &probeError, &playbackStatus,
			&thumbnailStatus, &thumbnailError,
			&storyboardStatus, &storyboardError,
			&storyboardFrameCount, &storyboardColumns, &storyboardRows,
			&storyboardCellWidth, &storyboardCellHeight,
			&libraryStatus, &favorite,
		); err != nil {
			return nil, fmt.Errorf("read catalog asset: %w", err)
		}
		item.Kind = catalog.AssetKind(kind)
		item.Availability = catalog.SourceAvailable
		if libraryStatus == "offline" {
			item.Availability = catalog.SourceOffline
		}
		item.ProbeStatus = media.ProbeStatus(probeStatus)
		item.PlaybackStatus = media.PlaybackStatus(playbackStatus)
		item.Favorite = favorite != 0
		if width.Valid {
			item.Width = &width.Int64
		}
		if height.Valid {
			item.Height = &height.Int64
		}
		if durationMS.Valid {
			item.DurationMS = &durationMS.Int64
		}
		if probeError.Valid {
			value := media.ProcessingErrorCode(probeError.String)
			item.ProbeErrorCode = &value
		}
		if thumbnailStatus.Valid {
			item.ThumbnailStatus = thumbnailStatus.String
		} else {
			item.ThumbnailStatus = "pending"
		}
		if thumbnailError.Valid {
			value := media.ProcessingErrorCode(thumbnailError.String)
			item.ThumbnailErrorCode = &value
		}
		applyStoryboardCatalogState(
			&item,
			storyboardStatus,
			storyboardError,
			storyboardFrameCount,
			storyboardColumns,
			storyboardRows,
			storyboardCellWidth,
			storyboardCellHeight,
		)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog assets: %w", err)
	}
	return items, nil
}

func (s *Store) CountAssets(
	ctx context.Context,
	query catalog.AssetQuery,
) (catalog.AssetCounts, error) {
	var builder strings.Builder
	args := make([]any, 0, 16)
	if query.ScopeKind == catalog.ScopeDirectory &&
		query.Recursive && query.Scope.CanonicalDirectoryID != 0 {
		builder.WriteString(`
        WITH RECURSIVE subtree(id) AS (
            SELECT ?
            UNION
            SELECT d.id FROM directories d
            JOIN subtree parent ON d.parent_id = parent.id
            WHERE d.library_id = ?
        )`)
		args = append(args, query.Scope.DirectoryID, query.Scope.LibraryID)
	}
	builder.WriteString(`
        SELECT COUNT(*),
               COALESCE(SUM(CASE WHEN a.kind IN ('image', 'animated') THEN 1 ELSE 0 END), 0),
               COALESCE(SUM(CASE WHEN a.kind = 'video' THEN 1 ELSE 0 END), 0)
        FROM assets a`)
	if anchor := ftsAnchor(query.SearchTerms); anchor != "" {
		builder.WriteString(`
        JOIN asset_search ON asset_search.rowid = a.id
        WHERE asset_search MATCH ?`)
		args = append(args, anchor)
	} else {
		builder.WriteString(` WHERE 1 = 1`)
	}
	switch query.ScopeKind {
	case catalog.ScopeGlobal:
	case catalog.ScopeLibrary:
		builder.WriteString(` AND a.library_id = ?`)
		args = append(args, query.Scope.LibraryID)
	case catalog.ScopeDirectory:
		builder.WriteString(` AND a.library_id = ?`)
		args = append(args, query.Scope.LibraryID)
		if !query.Recursive {
			builder.WriteString(` AND a.directory_id = ?`)
			args = append(args, query.Scope.DirectoryID)
		} else if query.Scope.CanonicalDirectoryID != 0 {
			builder.WriteString(` AND a.directory_id IN (SELECT id FROM subtree)`)
		}
	default:
		return catalog.AssetCounts{}, catalog.ErrInvalidQuery
	}
	for _, term := range query.SearchTerms {
		builder.WriteString(`
          AND (
            instr(a.search_name_key, ?) > 0
            OR instr(a.search_path_key, ?) > 0
          )`)
		args = append(args, term, term)
	}
	if query.ModifiedFromNS != nil {
		builder.WriteString(` AND a.mtime_ns >= ?`)
		args = append(args, *query.ModifiedFromNS)
	}
	if query.ModifiedBeforeNS != nil {
		builder.WriteString(` AND a.mtime_ns < ?`)
		args = append(args, *query.ModifiedBeforeNS)
	}
	var counts catalog.AssetCounts
	if err := s.db.QueryRowContext(ctx, builder.String(), args...).Scan(
		&counts.All, &counts.Images, &counts.Videos,
	); err != nil {
		return catalog.AssetCounts{}, fmt.Errorf("count catalog assets: %w", err)
	}
	return counts, nil
}

func ftsAnchor(terms []string) string {
	longest := ""
	for _, term := range terms {
		if strings.ContainsRune(term, '"') || utf8.RuneCountInString(term) < 3 {
			continue
		}
		if utf8.RuneCountInString(term) > utf8.RuneCountInString(longest) {
			longest = term
		}
	}
	if longest == "" {
		return ""
	}
	return `"` + longest + `"`
}

func (s *Store) GetAsset(ctx context.Context, assetID int64) (catalog.Asset, error) {
	if assetID <= 0 {
		return catalog.Asset{}, catalog.ErrAssetNotFound
	}
	items, err := s.GetAssetsByIDs(ctx, []int64{assetID})
	if err != nil {
		return catalog.Asset{}, err
	}
	return items[0], nil
}

func (s *Store) GetAssetsByIDs(ctx context.Context, assetIDs []int64) ([]catalog.Asset, error) {
	if len(assetIDs) < 1 || len(assetIDs) > catalog.MaxPageSize {
		return nil, catalog.ErrInvalidQuery
	}
	arguments := make([]any, len(assetIDs))
	placeholders := make([]string, len(assetIDs))
	seen := make(map[int64]struct{}, len(assetIDs))
	for index, assetID := range assetIDs {
		if assetID < 1 {
			return nil, catalog.ErrInvalidQuery
		}
		if _, exists := seen[assetID]; exists {
			return nil, catalog.ErrInvalidQuery
		}
		seen[assetID] = struct{}{}
		arguments[index], placeholders[index] = assetID, "?"
	}
	rows, err := s.db.QueryContext(ctx, catalogAssetProjectionSQL+` WHERE a.id IN (`+strings.Join(placeholders, ",")+`)`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("get catalog asset batch: %w", err)
	}
	defer rows.Close()
	byID := make(map[int64]catalog.Asset, len(assetIDs))
	for rows.Next() {
		item, err := scanCatalogAsset(rows)
		if err != nil {
			return nil, err
		}
		byID[item.ID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog asset batch: %w", err)
	}
	items := make([]catalog.Asset, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		item, found := byID[assetID]
		if !found {
			return nil, catalog.ErrAssetNotFound
		}
		items = append(items, item)
	}
	return items, nil
}

const catalogAssetProjectionSQL = `
        SELECT a.id, a.library_id, l.name, a.directory_id, a.relative_path, a.name,
               a.natural_name_key, a.kind, a.media_format, a.mime_type,
               a.size_bytes, a.mtime_ns, a.source_fingerprint,
               a.width, a.height, a.duration_ms, a.probe_status,
               a.probe_error_code, a.playback_status,
               t.status, t.error_code,
               storyboard.status, storyboard.error_code,
               storyboard.frame_count, storyboard.sprite_columns,
               storyboard.sprite_rows, storyboard.cell_width,
               storyboard.cell_height,
               l.status,
               EXISTS(SELECT 1 FROM asset_favorites favorite WHERE favorite.asset_id = a.id)
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        LEFT JOIN thumbnails t ON t.asset_id = a.id AND t.variant = 'grid'
        LEFT JOIN thumbnails storyboard
          ON storyboard.asset_id = a.id
         AND storyboard.variant = 'storyboard'`

func scanCatalogAsset(row interface{ Scan(...any) error }) (catalog.Asset, error) {
	var item catalog.Asset
	var kind, probeStatus, playbackStatus, libraryStatus string
	var width, height, durationMS sql.NullInt64
	var storyboardFrameCount, storyboardColumns, storyboardRows sql.NullInt64
	var storyboardCellWidth, storyboardCellHeight sql.NullInt64
	var probeError, thumbnailStatus, thumbnailError sql.NullString
	var storyboardStatus, storyboardError sql.NullString
	var favorite int64
	err := row.Scan(
		&item.ID, &item.LibraryID, &item.LibraryName,
		&item.DirectoryID, &item.RelativePath, &item.Name,
		&item.NaturalNameKey, &kind, &item.MediaFormat, &item.MIMEType,
		&item.SizeBytes, &item.ModifiedAtNS, &item.SourceFingerprint,
		&width, &height, &durationMS, &probeStatus, &probeError,
		&playbackStatus, &thumbnailStatus, &thumbnailError,
		&storyboardStatus, &storyboardError,
		&storyboardFrameCount, &storyboardColumns, &storyboardRows,
		&storyboardCellWidth, &storyboardCellHeight,
		&libraryStatus, &favorite,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Asset{}, catalog.ErrAssetNotFound
	}
	if err != nil {
		return catalog.Asset{}, fmt.Errorf("get catalog asset: %w", err)
	}
	item.Kind = catalog.AssetKind(kind)
	item.Availability = catalog.SourceAvailable
	if libraryStatus == "offline" {
		item.Availability = catalog.SourceOffline
	}
	item.ProbeStatus = media.ProbeStatus(probeStatus)
	item.PlaybackStatus = media.PlaybackStatus(playbackStatus)
	item.Favorite = favorite != 0
	if width.Valid {
		item.Width = &width.Int64
	}
	if height.Valid {
		item.Height = &height.Int64
	}
	if durationMS.Valid {
		item.DurationMS = &durationMS.Int64
	}
	if probeError.Valid {
		value := media.ProcessingErrorCode(probeError.String)
		item.ProbeErrorCode = &value
	}
	item.ThumbnailStatus = "pending"
	if thumbnailStatus.Valid {
		item.ThumbnailStatus = thumbnailStatus.String
	}
	if thumbnailError.Valid {
		value := media.ProcessingErrorCode(thumbnailError.String)
		item.ThumbnailErrorCode = &value
	}
	applyStoryboardCatalogState(
		&item,
		storyboardStatus,
		storyboardError,
		storyboardFrameCount,
		storyboardColumns,
		storyboardRows,
		storyboardCellWidth,
		storyboardCellHeight,
	)
	return item, nil
}

func applyStoryboardCatalogState(
	item *catalog.Asset,
	status, errorCode sql.NullString,
	frameCount, columns, rows, cellWidth, cellHeight sql.NullInt64,
) {
	item.StoryboardStatus = "pending"
	if status.Valid {
		item.StoryboardStatus = status.String
	}
	if errorCode.Valid {
		value := media.ProcessingErrorCode(errorCode.String)
		item.StoryboardErrorCode = &value
	}
	if frameCount.Valid {
		item.StoryboardFrameCount = &frameCount.Int64
	}
	if columns.Valid {
		item.StoryboardColumns = &columns.Int64
	}
	if rows.Valid {
		item.StoryboardRows = &rows.Int64
	}
	if cellWidth.Valid {
		item.StoryboardCellWidth = &cellWidth.Int64
	}
	if cellHeight.Valid {
		item.StoryboardCellHeight = &cellHeight.Int64
	}
}

func appendAssetKeyset(
	builder *strings.Builder,
	args *[]any,
	query catalog.AssetQuery,
	after catalog.AssetPosition,
) {
	operator := ">"
	if query.Order == catalog.OrderDesc {
		operator = "<"
	}
	if query.Sort == catalog.SortModifiedAt {
		builder.WriteString(`
          AND (
            a.mtime_ns ` + operator + ` ?
            OR (a.mtime_ns = ? AND a.id ` + operator + ` ?)
          )`)
		*args = append(*args, after.ModifiedAtNS, after.ModifiedAtNS, after.ID)
		return
	}
	if query.Sort == catalog.SortSize {
		builder.WriteString(`
          AND (
            a.size_bytes ` + operator + ` ?
            OR (a.size_bytes = ? AND a.id ` + operator + ` ?)
          )`)
		*args = append(*args, after.SizeBytes, after.SizeBytes, after.ID)
		return
	}
	if query.ScopeKind == catalog.ScopeGlobal {
		builder.WriteString(`
          AND (
            a.library_id ` + operator + ` ?
            OR (a.library_id = ? AND ` + assetDirectoryPathSQL + ` ` + operator + ` ?)
            OR (
                a.library_id = ? AND ` + assetDirectoryPathSQL + ` = ?
                AND a.natural_name_key ` + operator + ` ?
            )
            OR (
                a.library_id = ? AND ` + assetDirectoryPathSQL + ` = ?
                AND a.natural_name_key = ? AND a.name ` + operator + ` ?
            )
            OR (
                a.library_id = ? AND ` + assetDirectoryPathSQL + ` = ?
                AND a.natural_name_key = ? AND a.name = ?
                AND a.relative_path ` + operator + ` ?
            )
            OR (
                a.library_id = ? AND ` + assetDirectoryPathSQL + ` = ?
                AND a.natural_name_key = ? AND a.name = ?
                AND a.relative_path = ? AND a.id ` + operator + ` ?
            )
          )`)
		*args = append(*args,
			after.LibraryID,
			after.LibraryID, after.DirectoryPath,
			after.LibraryID, after.DirectoryPath, after.NaturalNameKey,
			after.LibraryID, after.DirectoryPath, after.NaturalNameKey, after.Name,
			after.LibraryID, after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath,
			after.LibraryID, after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath, after.ID,
		)
		return
	}
	builder.WriteString(`
          AND (
            ` + assetDirectoryPathSQL + ` ` + operator + ` ?
            OR (` + assetDirectoryPathSQL + ` = ? AND a.natural_name_key ` + operator + ` ?)
            OR (
                ` + assetDirectoryPathSQL + ` = ? AND a.natural_name_key = ?
                AND a.name ` + operator + ` ?
            )
            OR (
                ` + assetDirectoryPathSQL + ` = ? AND a.natural_name_key = ?
                AND a.name = ? AND a.relative_path ` + operator + ` ?
            )
            OR (
                ` + assetDirectoryPathSQL + ` = ? AND a.natural_name_key = ?
                AND a.name = ? AND a.relative_path = ?
                AND a.id ` + operator + ` ?
            )
          )`)
	*args = append(*args,
		after.DirectoryPath,
		after.DirectoryPath, after.NaturalNameKey,
		after.DirectoryPath, after.NaturalNameKey, after.Name,
		after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath,
		after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath, after.ID,
	)
}

const assetDirectoryPathSQL = `CASE
    WHEN length(a.relative_path) = length(a.name) THEN ''
    ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
END`

func appendAssetOrder(builder *strings.Builder, query catalog.AssetQuery) {
	direction := " ASC"
	if query.Order == catalog.OrderDesc {
		direction = " DESC"
	}
	if query.Sort == catalog.SortModifiedAt {
		builder.WriteString(` ORDER BY a.mtime_ns` + direction + `, a.id` + direction)
		return
	}
	if query.Sort == catalog.SortSize {
		builder.WriteString(` ORDER BY a.size_bytes` + direction + `, a.id` + direction)
		return
	}
	if query.ScopeKind == catalog.ScopeGlobal {
		builder.WriteString(
			` ORDER BY a.library_id` + direction +
				`, ` + assetDirectoryPathSQL + direction +
				`, a.natural_name_key` + direction +
				`, a.name` + direction +
				`, a.relative_path` + direction +
				`, a.id` + direction,
		)
		return
	}
	builder.WriteString(
		` ORDER BY ` + assetDirectoryPathSQL + direction +
			`, a.natural_name_key` + direction +
			`, a.name` + direction +
			`, a.relative_path` + direction +
			`, a.id` + direction,
	)
}
