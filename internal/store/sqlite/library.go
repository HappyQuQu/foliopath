package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite/dbgen"
)

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
		queries := dbgen.New(tx)
		_, err := queries.FindLibraryIDByNameKey(ctx, nameKey)
		switch {
		case err == nil:
			return library.ErrNameExists
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("check library name: %w", err)
		}

		_, err = queries.FindOverlappingLibraryID(ctx, root)
		switch {
		case err == nil:
			return library.ErrRootOverlap
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("check library root overlap: %w", err)
		}

		now := s.nowMS()
		record, err := queries.InsertLibrary(ctx, dbgen.InsertLibraryParams{
			Name:        name,
			NameKey:     nameKey,
			RootRelPath: root,
			CreatedAtMs: now,
			UpdatedAtMs: now,
		})
		if err != nil {
			return fmt.Errorf("insert library: %w", err)
		}
		created, err = libraryFromDatabase(record)
		if err != nil {
			return fmt.Errorf("map inserted library: %w", err)
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
		queries := dbgen.New(tx)
		_, err := queries.FindOtherLibraryIDByNameKey(ctx, dbgen.FindOtherLibraryIDByNameKeyParams{
			NameKey: nameKey,
			ID:      id,
		})
		switch {
		case err == nil:
			return library.ErrNameExists
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("check renamed library name: %w", err)
		}

		rows, err := queries.RenameLibrary(ctx, dbgen.RenameLibraryParams{
			Name:        name,
			NameKey:     nameKey,
			UpdatedAtMs: s.nowMS(),
			ID:          id,
		})
		if err != nil {
			return fmt.Errorf("rename library: %w", err)
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
	record, err := s.queries.GetLibrary(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return library.Library{}, library.ErrNotFound
	}
	if err != nil {
		return library.Library{}, fmt.Errorf("get library: %w", err)
	}
	result, err := libraryFromDatabase(record)
	if err != nil {
		return library.Library{}, fmt.Errorf("map library: %w", err)
	}
	return result, nil
}

func (s *Store) ListLibraries(ctx context.Context) ([]library.Library, error) {
	records, err := s.queries.ListLibraries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list libraries: %w", err)
	}

	result := make([]library.Library, 0, len(records))
	for _, record := range records {
		item, err := libraryFromDatabase(record)
		if err != nil {
			return nil, fmt.Errorf("map listed library: %w", err)
		}
		result = append(result, item)
	}
	return result, nil
}

func libraryFromDatabase(record dbgen.Library) (library.Library, error) {
	status, err := library.ValidateStatus(record.Status)
	if err != nil {
		return library.Library{}, err
	}
	return library.Library{
		ID:                record.ID,
		Name:              record.Name,
		RootRelativePath:  record.RootRelPath,
		Status:            status,
		CurrentGeneration: record.CurrentGeneration,
		CreatedAtMS:       record.CreatedAtMs,
		UpdatedAtMS:       record.UpdatedAtMs,
	}, nil
}
