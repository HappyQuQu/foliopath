package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/media"
)

func (s *Store) Revision(ctx context.Context) (int64, error) {
	var revision int64
	if err := s.db.QueryRowContext(ctx, `
        SELECT revision FROM curation_state WHERE singleton_key = 1`,
	).Scan(&revision); err != nil {
		return 0, fmt.Errorf("read curation revision: %w", err)
	}
	if revision < 1 {
		return 0, errors.New("curation revision is invalid")
	}
	return revision, nil
}

func (s *Store) GetAssetState(ctx context.Context, assetID int64) (curation.AssetState, error) {
	if assetID <= 0 {
		return curation.AssetState{}, curation.ErrAssetNotFound
	}
	var libraryID, revision int64
	var favoriteAt sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `
        SELECT a.library_id, cs.revision, f.created_at_ms
        FROM assets a
        CROSS JOIN curation_state cs
        LEFT JOIN asset_favorites f ON f.asset_id = a.id
        WHERE a.id = ? AND cs.singleton_key = 1`, assetID,
	).Scan(&libraryID, &revision, &favoriteAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return curation.AssetState{}, curation.ErrAssetNotFound
		}
		return curation.AssetState{}, fmt.Errorf("read asset curation state: %w", err)
	}
	tags, err := s.listAssetTags(ctx, assetID)
	if err != nil {
		return curation.AssetState{}, err
	}
	state := curation.AssetState{AssetID: assetID, Revision: revision, Tags: tags}
	if favoriteAt.Valid {
		value := time.UnixMilli(favoriteAt.Int64).UTC()
		state.Favorite = true
		state.FavoritedAt = &value
	}
	_ = libraryID
	return state, nil
}

func (s *Store) SetFavorite(ctx context.Context, assetID int64, favorite bool, now time.Time) (bool, error) {
	if assetID <= 0 || now.IsZero() {
		return false, curation.ErrInvalidRequest
	}
	changed := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var libraryID int64
		if err := tx.QueryRowContext(ctx, `SELECT library_id FROM assets WHERE id = ?`, assetID).Scan(&libraryID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return curation.ErrAssetNotFound
			}
			return fmt.Errorf("resolve favorite asset: %w", err)
		}
		var result sql.Result
		var err error
		if favorite {
			result, err = tx.ExecContext(ctx, `
                INSERT INTO asset_favorites(asset_id, library_id, created_at_ms)
                VALUES(?, ?, ?)
                ON CONFLICT(asset_id) DO NOTHING`,
				assetID, libraryID, now.UTC().UnixMilli(),
			)
		} else {
			result, err = tx.ExecContext(ctx, `DELETE FROM asset_favorites WHERE asset_id = ?`, assetID)
		}
		if err != nil {
			return fmt.Errorf("set asset favorite: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read favorite mutation result: %w", err)
		}
		changed = rows > 0
		return nil
	})
	return changed, err
}

func (s *Store) CreateTag(ctx context.Context, name, normalizedName string, now time.Time) (curation.Tag, bool, error) {
	if name == "" || normalizedName == "" || now.IsZero() {
		return curation.Tag{}, false, curation.ErrInvalidRequest
	}
	var tag curation.Tag
	created := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
            SELECT id, name, normalized_name, created_at_ms, updated_at_ms,
                   (SELECT COUNT(*) FROM asset_tags at WHERE at.tag_id = tags.id)
            FROM tags WHERE normalized_name = ?`, normalizedName)
		if err := scanTag(row, &tag); err == nil {
			return nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("resolve equivalent tag: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
            INSERT INTO tags(name, normalized_name, created_at_ms, updated_at_ms)
            VALUES(?, ?, ?, ?)`, name, normalizedName, now.UTC().UnixMilli(), now.UTC().UnixMilli())
		if err != nil {
			return fmt.Errorf("create tag: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read created tag ID: %w", err)
		}
		tag = curation.Tag{ID: id, Name: name, NormalizedName: normalizedName, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
		created = true
		return nil
	})
	return tag, created, err
}

func (s *Store) RenameTag(ctx context.Context, tagID int64, name, normalizedName string, now time.Time) (curation.Tag, error) {
	if tagID <= 0 || name == "" || normalizedName == "" || now.IsZero() {
		return curation.Tag{}, curation.ErrInvalidRequest
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var exists int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM tags WHERE id = ?`, tagID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return curation.ErrTagNotFound
			}
			return fmt.Errorf("resolve tag for rename: %w", err)
		}
		var conflict int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM tags WHERE normalized_name = ? AND id <> ?`, normalizedName, tagID).Scan(&conflict)
		if err == nil {
			return curation.ErrTagNameConflict
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check tag rename conflict: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tags SET name = ?, normalized_name = ?, updated_at_ms = ? WHERE id = ?`, name, normalizedName, now.UTC().UnixMilli(), tagID); err != nil {
			return fmt.Errorf("rename tag: %w", err)
		}
		return nil
	})
	if err != nil {
		return curation.Tag{}, err
	}
	return s.getTag(ctx, tagID)
}

func (s *Store) DeleteTag(ctx context.Context, tagID int64) error {
	if tagID <= 0 {
		return curation.ErrTagNotFound
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, tagID)
		if err != nil {
			return fmt.Errorf("delete tag: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read deleted tag count: %w", err)
		}
		if rows == 0 {
			return curation.ErrTagNotFound
		}
		return nil
	})
}

func (s *Store) ReplaceAssetTags(ctx context.Context, assetID, expectedRevision int64, tagIDs []int64, now time.Time) error {
	if assetID <= 0 || expectedRevision <= 0 || len(tagIDs) > curation.MaxTagsPerAsset || now.IsZero() {
		return curation.ErrInvalidRequest
	}
	return s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM curation_state WHERE singleton_key = 1`).Scan(&revision); err != nil {
			return fmt.Errorf("read curation precondition: %w", err)
		}
		if revision != expectedRevision {
			return curation.ErrPreconditionFailed
		}
		var libraryID int64
		if err := tx.QueryRowContext(ctx, `SELECT library_id FROM assets WHERE id = ?`, assetID).Scan(&libraryID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return curation.ErrAssetNotFound
			}
			return fmt.Errorf("resolve tagged asset: %w", err)
		}
		if len(tagIDs) > 0 {
			placeholders := strings.TrimSuffix(strings.Repeat("?,", len(tagIDs)), ",")
			args := make([]any, len(tagIDs))
			for index, id := range tagIDs {
				args[index] = id
			}
			var count int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE id IN (`+placeholders+`)`, args...).Scan(&count); err != nil {
				return fmt.Errorf("validate replacement tags: %w", err)
			}
			if count != len(tagIDs) {
				return curation.ErrTagNotFound
			}
		}
		current, err := listAssetTagIDs(ctx, tx, assetID)
		if err != nil {
			return err
		}
		if slices.Equal(current, tagIDs) {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM asset_tags WHERE asset_id = ?`, assetID); err != nil {
			return fmt.Errorf("clear asset tags: %w", err)
		}
		for _, tagID := range tagIDs {
			if _, err := tx.ExecContext(ctx, `
                INSERT INTO asset_tags(asset_id, library_id, tag_id, created_at_ms)
                VALUES(?, ?, ?, ?)`, assetID, libraryID, tagID, now.UTC().UnixMilli()); err != nil {
				return fmt.Errorf("attach asset tag: %w", err)
			}
		}
		return nil
	})
}

func listAssetTagIDs(ctx context.Context, tx *sql.Tx, assetID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT tag_id FROM asset_tags WHERE asset_id = ? ORDER BY tag_id`, assetID)
	if err != nil {
		return nil, fmt.Errorf("list current asset tags: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read current asset tag: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate current asset tags: %w", err)
	}
	return ids, nil
}

func (s *Store) ListTagPage(ctx context.Context, params curation.TagListParams) ([]curation.Tag, error) {
	if params.Limit < 1 || params.Limit > curation.MaxPageSize+1 {
		return nil, curation.ErrInvalidRequest
	}
	query := `
        SELECT t.id, t.name, t.normalized_name, t.created_at_ms, t.updated_at_ms,
               COUNT(at.asset_id)
        FROM tags t
        LEFT JOIN asset_tags at ON at.tag_id = t.id
        WHERE 1 = 1`
	args := make([]any, 0, 5)
	if params.SearchKey != "" {
		query += ` AND instr(t.normalized_name, ?) > 0`
		args = append(args, params.SearchKey)
	}
	if params.After != nil {
		query += ` AND (t.normalized_name > ? OR (t.normalized_name = ? AND t.id > ?))`
		args = append(args, params.After.NormalizedName, params.After.NormalizedName, params.After.ID)
	}
	query += ` GROUP BY t.id ORDER BY t.normalized_name, t.id LIMIT ?`
	args = append(args, params.Limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	items := make([]curation.Tag, 0, params.Limit)
	for rows.Next() {
		var tag curation.Tag
		if err := scanTag(rows, &tag); err != nil {
			return nil, fmt.Errorf("read tag page: %w", err)
		}
		items = append(items, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag page: %w", err)
	}
	return items, nil
}

func (s *Store) ResolveLibrary(ctx context.Context, libraryID int64) error {
	var id int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM libraries WHERE id = ?`, libraryID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return curation.ErrLibraryNotFound
		}
		return fmt.Errorf("resolve curation library: %w", err)
	}
	return nil
}

func (s *Store) ResolveTag(ctx context.Context, tagID int64) error {
	_, err := s.getTag(ctx, tagID)
	return err
}

func (s *Store) getTag(ctx context.Context, tagID int64) (curation.Tag, error) {
	var tag curation.Tag
	err := scanTag(s.db.QueryRowContext(ctx, `
        SELECT t.id, t.name, t.normalized_name, t.created_at_ms, t.updated_at_ms,
               (SELECT COUNT(*) FROM asset_tags at WHERE at.tag_id = t.id)
        FROM tags t WHERE t.id = ?`, tagID), &tag)
	if errors.Is(err, sql.ErrNoRows) {
		return curation.Tag{}, curation.ErrTagNotFound
	}
	if err != nil {
		return curation.Tag{}, fmt.Errorf("get tag: %w", err)
	}
	return tag, nil
}

func scanTag(scanner rowScanner, tag *curation.Tag) error {
	var createdAtMS, updatedAtMS int64
	if err := scanner.Scan(&tag.ID, &tag.Name, &tag.NormalizedName, &createdAtMS, &updatedAtMS, &tag.AssetCount); err != nil {
		return err
	}
	tag.CreatedAt = time.UnixMilli(createdAtMS).UTC()
	tag.UpdatedAt = time.UnixMilli(updatedAtMS).UTC()
	return nil
}

func (s *Store) ListCuratedAssetPage(ctx context.Context, params curation.AssetListParams) ([]curation.CuratedAsset, error) {
	if params.Limit < 1 || params.Limit > curation.MaxPageSize+1 {
		return nil, curation.ErrInvalidRequest
	}
	query, args, err := curatedAssetQuery(params.Query, params.After, false)
	if err != nil {
		return nil, err
	}
	query += ` LIMIT ?`
	args = append(args, params.Limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list curated assets: %w", err)
	}
	defer rows.Close()
	items := make([]curation.CuratedAsset, 0, params.Limit)
	for rows.Next() {
		item, err := scanCuratedAsset(rows, params.Query.Revision)
		if err != nil {
			return nil, fmt.Errorf("read curated asset: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate curated assets: %w", err)
	}
	for index := range items {
		tags, err := s.listAssetTags(ctx, items[index].Asset.ID)
		if err != nil {
			return nil, err
		}
		items[index].State.Tags = tags
	}
	return items, nil
}

func (s *Store) CountCuratedAssets(ctx context.Context, query curation.AssetQuery) (catalog.AssetCounts, error) {
	statement, args, err := curatedAssetQuery(query, nil, true)
	if err != nil {
		return catalog.AssetCounts{}, err
	}
	var counts catalog.AssetCounts
	if err := s.db.QueryRowContext(ctx, statement, args...).Scan(&counts.All, &counts.Images, &counts.Videos); err != nil {
		return catalog.AssetCounts{}, fmt.Errorf("count curated assets: %w", err)
	}
	return counts, nil
}

func curatedAssetQuery(query curation.AssetQuery, after *curation.AssetPosition, count bool) (string, []any, error) {
	var builder strings.Builder
	args := make([]any, 0, 24)
	if count {
		builder.WriteString(`
        SELECT COUNT(*),
               COALESCE(SUM(CASE WHEN a.kind IN ('image', 'animated') THEN 1 ELSE 0 END), 0),
               COALESCE(SUM(CASE WHEN a.kind = 'video' THEN 1 ELSE 0 END), 0)`)
	} else {
		builder.WriteString(`
        SELECT a.id, a.library_id, l.name, a.directory_id, a.relative_path, a.name,
               a.natural_name_key, a.kind, a.media_format, a.mime_type,
               a.size_bytes, a.mtime_ns, a.source_fingerprint,
               a.width, a.height, a.duration_ms, a.probe_status,
               a.probe_error_code, a.playback_status,
               grid.status, grid.error_code,
               storyboard.status, storyboard.error_code,
               storyboard.frame_count, storyboard.sprite_columns,
               storyboard.sprite_rows, storyboard.cell_width,
               storyboard.cell_height, l.status, favorite.created_at_ms`)
	}
	builder.WriteString(`
        FROM assets a
        JOIN libraries l ON l.id = a.library_id`)
	if query.FavoriteOnly {
		builder.WriteString(` JOIN asset_favorites selected_favorite ON selected_favorite.asset_id = a.id`)
	} else if query.TagID > 0 {
		builder.WriteString(` JOIN asset_tags selected_tag ON selected_tag.asset_id = a.id AND selected_tag.tag_id = ?`)
		args = append(args, query.TagID)
	} else {
		return "", nil, curation.ErrInvalidRequest
	}
	if !count {
		builder.WriteString(`
        LEFT JOIN asset_favorites favorite ON favorite.asset_id = a.id
        LEFT JOIN thumbnails grid ON grid.asset_id = a.id AND grid.variant = 'grid'
        LEFT JOIN thumbnails storyboard ON storyboard.asset_id = a.id AND storyboard.variant = 'storyboard'`)
	}
	builder.WriteString(` WHERE 1 = 1`)
	if query.LibraryID > 0 {
		builder.WriteString(` AND a.library_id = ?`)
		args = append(args, query.LibraryID)
	}
	if len(query.Kinds) > 0 {
		builder.WriteString(` AND a.kind IN (`)
		for index, kind := range query.Kinds {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteByte('?')
			args = append(args, string(kind))
		}
		builder.WriteByte(')')
	}
	if count {
		return builder.String(), args, nil
	}
	if after != nil {
		appendCurationKeyset(&builder, &args, query, *after)
	}
	appendCurationOrder(&builder, query)
	return builder.String(), args, nil
}

func appendCurationKeyset(builder *strings.Builder, args *[]any, query curation.AssetQuery, after curation.AssetPosition) {
	operator := ">"
	if query.Order == curation.OrderDesc {
		operator = "<"
	}
	switch query.Sort {
	case curation.SortFavoriteAt:
		builder.WriteString(` AND (selected_favorite.created_at_ms ` + operator + ` ? OR (selected_favorite.created_at_ms = ? AND a.id ` + operator + ` ?))`)
		*args = append(*args, after.FavoriteAtMS, after.FavoriteAtMS, after.ID)
	case curation.SortModifiedAt:
		builder.WriteString(` AND (a.mtime_ns ` + operator + ` ? OR (a.mtime_ns = ? AND a.id ` + operator + ` ?))`)
		*args = append(*args, after.ModifiedAtNS, after.ModifiedAtNS, after.ID)
	case curation.SortSize:
		builder.WriteString(` AND (a.size_bytes ` + operator + ` ? OR (a.size_bytes = ? AND a.id ` + operator + ` ?))`)
		*args = append(*args, after.SizeBytes, after.SizeBytes, after.ID)
	case curation.SortName:
		directory := `(CASE WHEN length(a.relative_path) = length(a.name) THEN '' ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1) END)`
		builder.WriteString(` AND ((a.library_id, ` + directory + `, a.natural_name_key, a.name, a.relative_path, a.id) ` + operator + ` (?, ?, ?, ?, ?, ?))`)
		*args = append(*args, after.LibraryID, after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath, after.ID)
	}
}

func appendCurationOrder(builder *strings.Builder, query curation.AssetQuery) {
	direction := " ASC"
	if query.Order == curation.OrderDesc {
		direction = " DESC"
	}
	switch query.Sort {
	case curation.SortFavoriteAt:
		builder.WriteString(` ORDER BY selected_favorite.created_at_ms` + direction + `, a.id` + direction)
	case curation.SortModifiedAt:
		builder.WriteString(` ORDER BY a.mtime_ns` + direction + `, a.id` + direction)
	case curation.SortSize:
		builder.WriteString(` ORDER BY a.size_bytes` + direction + `, a.id` + direction)
	case curation.SortName:
		builder.WriteString(` ORDER BY a.library_id` + direction + `, (CASE WHEN length(a.relative_path) = length(a.name) THEN '' ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1) END)` + direction + `, a.natural_name_key` + direction + `, a.name` + direction + `, a.relative_path` + direction + `, a.id` + direction)
	}
}

func scanCuratedAsset(scanner rowScanner, revision int64) (curation.CuratedAsset, error) {
	var item curation.CuratedAsset
	var kind, probeStatus, playbackStatus, libraryStatus string
	var width, height, durationMS, favoriteAt sql.NullInt64
	var storyboardFrameCount, storyboardColumns, storyboardRows sql.NullInt64
	var storyboardCellWidth, storyboardCellHeight sql.NullInt64
	var probeError, thumbnailStatus, thumbnailError sql.NullString
	var storyboardStatus, storyboardError sql.NullString
	err := scanner.Scan(
		&item.Asset.ID, &item.Asset.LibraryID, &item.Asset.LibraryName,
		&item.Asset.DirectoryID, &item.Asset.RelativePath, &item.Asset.Name,
		&item.Asset.NaturalNameKey, &kind, &item.Asset.MediaFormat, &item.Asset.MIMEType,
		&item.Asset.SizeBytes, &item.Asset.ModifiedAtNS, &item.Asset.SourceFingerprint,
		&width, &height, &durationMS, &probeStatus, &probeError, &playbackStatus,
		&thumbnailStatus, &thumbnailError, &storyboardStatus, &storyboardError,
		&storyboardFrameCount, &storyboardColumns, &storyboardRows,
		&storyboardCellWidth, &storyboardCellHeight, &libraryStatus, &favoriteAt,
	)
	if err != nil {
		return curation.CuratedAsset{}, err
	}
	item.Asset.Kind = catalog.AssetKind(kind)
	item.Asset.Availability = catalog.SourceAvailable
	if libraryStatus == "offline" {
		item.Asset.Availability = catalog.SourceOffline
	}
	item.Asset.ProbeStatus = media.ProbeStatus(probeStatus)
	item.Asset.PlaybackStatus = media.PlaybackStatus(playbackStatus)
	if width.Valid {
		item.Asset.Width = &width.Int64
	}
	if height.Valid {
		item.Asset.Height = &height.Int64
	}
	if durationMS.Valid {
		item.Asset.DurationMS = &durationMS.Int64
	}
	if probeError.Valid {
		value := media.ProcessingErrorCode(probeError.String)
		item.Asset.ProbeErrorCode = &value
	}
	item.Asset.ThumbnailStatus = "pending"
	if thumbnailStatus.Valid {
		item.Asset.ThumbnailStatus = thumbnailStatus.String
	}
	if thumbnailError.Valid {
		value := media.ProcessingErrorCode(thumbnailError.String)
		item.Asset.ThumbnailErrorCode = &value
	}
	applyStoryboardCatalogState(&item.Asset, storyboardStatus, storyboardError, storyboardFrameCount, storyboardColumns, storyboardRows, storyboardCellWidth, storyboardCellHeight)
	item.State = curation.AssetState{AssetID: item.Asset.ID, Revision: revision}
	if favoriteAt.Valid {
		value := time.UnixMilli(favoriteAt.Int64).UTC()
		item.Asset.Favorite = true
		item.State.Favorite = true
		item.State.FavoritedAt = &value
	}
	return item, nil
}

func (s *Store) listAssetTags(ctx context.Context, assetID int64) ([]curation.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT t.id, t.name, t.normalized_name, t.created_at_ms, t.updated_at_ms,
               (SELECT COUNT(*) FROM asset_tags counted WHERE counted.tag_id = t.id)
        FROM asset_tags at
        JOIN tags t ON t.id = at.tag_id
        WHERE at.asset_id = ?
        ORDER BY t.normalized_name, t.id`, assetID)
	if err != nil {
		return nil, fmt.Errorf("list asset tags: %w", err)
	}
	defer rows.Close()
	tags := make([]curation.Tag, 0, curation.MaxTagsPerAsset)
	for rows.Next() {
		var tag curation.Tag
		if err := scanTag(rows, &tag); err != nil {
			return nil, fmt.Errorf("read asset tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate asset tags: %w", err)
	}
	return tags, nil
}
