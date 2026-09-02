package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func (s *Store) CreatePerson(ctx context.Context, command face.CreatePersonCommand) (face.Person, error) {
	name, err := face.NormalizePersonName(command.Name)
	if len(command.ID) < 8 || len(command.ID) > 128 || command.CreatedAt.IsZero() || err != nil {
		return face.Person{}, face.ErrInvalidPerson
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO people(id,name,state,revision,created_at_ms,updated_at_ms) VALUES(?,?,'active',1,?,?)`,
		command.ID, name, command.CreatedAt.UTC().UnixMilli(), command.CreatedAt.UTC().UnixMilli())
	if err != nil {
		return face.Person{}, fmt.Errorf("create person: %w", err)
	}
	return s.GetPerson(ctx, command.ID)
}

func (s *Store) RenamePerson(ctx context.Context, command face.RenamePersonCommand) (face.Person, error) {
	name, err := face.NormalizePersonName(command.Name)
	if len(command.ID) < 8 || len(command.ID) > 128 || command.ExpectedRevision < 1 || command.UpdatedAt.IsZero() || err != nil {
		return face.Person{}, face.ErrInvalidPerson
	}
	result, err := s.db.ExecContext(ctx, `UPDATE people SET name=?,revision=revision+1,updated_at_ms=MAX(created_at_ms,?) WHERE id=? AND state='active' AND revision=?`,
		name, command.UpdatedAt.UTC().UnixMilli(), command.ID, command.ExpectedRevision)
	if err != nil {
		return face.Person{}, fmt.Errorf("rename person: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		if err != nil {
			return face.Person{}, err
		}
		return face.Person{}, face.ErrPersonConflict
	}
	return s.GetPerson(ctx, command.ID)
}

func (s *Store) DeletePerson(ctx context.Context, command face.DeletePersonCommand) error {
	if len(command.ID) < 8 || len(command.ID) > 128 || command.ExpectedRevision < 1 || command.DeletedAt.IsZero() {
		return face.ErrInvalidPerson
	}
	result, err := s.db.ExecContext(ctx, `UPDATE people SET state='tombstoned',revision=revision+1,
		updated_at_ms=MAX(created_at_ms,?),tombstoned_at_ms=MAX(created_at_ms,?)
		WHERE id=? AND state='active' AND revision=?`, command.DeletedAt.UTC().UnixMilli(), command.DeletedAt.UTC().UnixMilli(), command.ID, command.ExpectedRevision)
	if err != nil {
		return fmt.Errorf("delete person: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		if err != nil {
			return err
		}
		return face.ErrPersonConflict
	}
	return nil
}

func (s *Store) GetPerson(ctx context.Context, id string) (face.Person, error) {
	if len(id) < 8 || len(id) > 128 {
		return face.Person{}, face.ErrInvalidPerson
	}
	value, err := scanPerson(s.db.QueryRowContext(ctx, personSelect+` WHERE person.id=? AND person.state='active'`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return face.Person{}, face.ErrPersonNotFound
	}
	if err != nil {
		return face.Person{}, fmt.Errorf("get person: %w", err)
	}
	return value, nil
}

func (s *Store) ListPeople(ctx context.Context, limit int) ([]face.Person, error) {
	if limit < 1 || limit > 200 {
		return nil, face.ErrInvalidPerson
	}
	rows, err := s.db.QueryContext(ctx, personSelect+` WHERE person.state='active' ORDER BY person.name,person.id LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list people: %w", err)
	}
	defer rows.Close()
	items := make([]face.Person, 0, limit)
	for rows.Next() {
		value, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const personSelect = `SELECT person.id,person.name,
	(SELECT COUNT(*) FROM person_face_anchors anchor WHERE anchor.person_id=person.id AND anchor.state='bound'),
	(SELECT COUNT(DISTINCT anchor.asset_id || ':' || anchor.library_id) FROM person_face_anchors anchor WHERE anchor.person_id=person.id AND anchor.state='bound'),
	person.revision,person.created_at_ms,person.updated_at_ms FROM people person`

type personScanner interface{ Scan(...any) error }

func scanPerson(row personScanner) (face.Person, error) {
	var value face.Person
	var createdAt, updatedAt int64
	err := row.Scan(&value.ID, &value.Name, &value.ConfirmedFaceCount, &value.AssetCount, &value.Revision, &createdAt, &updatedAt)
	value.CreatedAt, value.UpdatedAt = time.UnixMilli(createdAt).UTC(), time.UnixMilli(updatedAt).UTC()
	return value, err
}

var _ face.PeopleRepository = (*Store)(nil)
