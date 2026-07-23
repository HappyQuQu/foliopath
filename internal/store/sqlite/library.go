package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/library"
)

const libraryColumns = `
    id, name, root_rel_path, status, current_generation,
    created_at_ms, updated_at_ms`

func (s *Store) CreateLibrary(ctx context.Context, params library.CreateParams) (library.Library, error) {
	name, nameKey, err := library.NormalizeName(params.Name)
	if err != nil {
		return library.Library{}, err
	}
	root, err := library.NormalizeRoot(params.RootRelativePath)
	if err != nil {
		return library.Library{}, err
	}

	var created library.Library
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var existingID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM libraries WHERE name_key = ?`, nameKey).Scan(&existingID)
		switch {
		case err == nil:
			return library.ErrNameExists
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("check library name: %w", err)
		}

		err = tx.QueryRowContext(ctx, `
            SELECT id
            FROM libraries
            WHERE root_rel_path = ?
               OR root_rel_path = ''
               OR ? = ''
               OR instr(?, root_rel_path || '/') = 1
               OR instr(root_rel_path, ? || '/') = 1
            LIMIT 1`, root, root, root, root).Scan(&existingID)
		switch {
		case err == nil:
			return library.ErrRootOverlap
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("check library root overlap: %w", err)
		}

		now := s.nowMS()
		result, err := tx.ExecContext(ctx, `
            INSERT INTO libraries(name, name_key, root_rel_path, status, current_generation, created_at_ms, updated_at_ms)
            VALUES (?, ?, ?, 'pending', 0, ?, ?)`, name, nameKey, root, now, now)
		if err != nil {
			return fmt.Errorf("insert library: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read inserted library id: %w", err)
		}
		created = library.Library{
			ID: id, Name: name, RootRelativePath: root, Status: library.StatusPending,
			CreatedAtMS: now, UpdatedAtMS: now,
		}
		return nil
	})
	return created, err
}

func (s *Store) RenameLibrary(ctx context.Context, id int64, requestedName string) (library.Library, error) {
	name, nameKey, err := library.NormalizeName(requestedName)
	if err != nil {
		return library.Library{}, err
	}

	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var conflictingID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM libraries WHERE name_key = ? AND id <> ?`, nameKey, id).Scan(&conflictingID)
		switch {
		case err == nil:
			return library.ErrNameExists
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("check renamed library name: %w", err)
		}

		result, err := tx.ExecContext(ctx, `
            UPDATE libraries SET name = ?, name_key = ?, updated_at_ms = ? WHERE id = ?`,
			name, nameKey, s.nowMS(), id)
		if err != nil {
			return fmt.Errorf("rename library: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read renamed library row count: %w", err)
		}
		if rows == 0 {
			return library.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return library.Library{}, err
	}
	return s.GetLibrary(ctx, id)
}

func (s *Store) GetLibrary(ctx context.Context, id int64) (library.Library, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+libraryColumns+` FROM libraries WHERE id = ?`, id)
	result, err := scanLibrary(row)
	if errors.Is(err, sql.ErrNoRows) {
		return library.Library{}, library.ErrNotFound
	}
	if err != nil {
		return library.Library{}, fmt.Errorf("get library: %w", err)
	}
	return result, nil
}

func (s *Store) ListLibraries(ctx context.Context) ([]library.Library, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+libraryColumns+` FROM libraries ORDER BY name_key, id`)
	if err != nil {
		return nil, fmt.Errorf("list libraries: %w", err)
	}
	defer rows.Close()

	result := make([]library.Library, 0)
	for rows.Next() {
		item, err := scanLibrary(rows)
		if err != nil {
			return nil, fmt.Errorf("scan listed library: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate libraries: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanLibrary(row rowScanner) (library.Library, error) {
	var (
		result library.Library
		status string
	)
	if err := row.Scan(
		&result.ID,
		&result.Name,
		&result.RootRelativePath,
		&status,
		&result.CurrentGeneration,
		&result.CreatedAtMS,
		&result.UpdatedAtMS,
	); err != nil {
		return library.Library{}, err
	}
	parsedStatus, err := library.ValidateStatus(status)
	if err != nil {
		return library.Library{}, err
	}
	result.Status = parsedStatus
	return result, nil
}
