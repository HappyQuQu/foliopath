package sqlite

import (
	"context"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func (s *Store) GetPeopleSnapshot(ctx context.Context) (face.PeopleSnapshot, error) {
	var value face.PeopleSnapshot
	err := s.db.QueryRowContext(ctx, `SELECT MAX(1,COALESCE(SUM(revision+updated_at_ms),0)) FROM people`).Scan(&value.Revision)
	if err != nil {
		return face.PeopleSnapshot{}, fmt.Errorf("get people snapshot: %w", err)
	}
	return value, nil
}
func (s *Store) ListPeoplePage(ctx context.Context, query face.PeopleQuery) ([]face.Person, error) {
	if query.Limit < 1 || query.Limit > face.MaxPeoplePageSize+1 {
		return nil, face.ErrInvalidPerson
	}
	args := []any{}
	filter := ` WHERE person.state='active'`
	if query.Search != "" {
		filter += ` AND instr(person.name,?)>0`
		args = append(args, query.Search)
	}
	if query.After != nil {
		filter += ` AND (person.name>? OR (person.name=? AND person.id>?))`
		args = append(args, query.After.Name, query.After.Name, query.After.ID)
	}
	args = append(args, query.Limit)
	rows, err := s.db.QueryContext(ctx, personSelect+filter+` ORDER BY person.name,person.id LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list people page: %w", err)
	}
	defer rows.Close()
	items := make([]face.Person, 0, query.Limit)
	for rows.Next() {
		item, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

var _ face.PeopleListRepository = (*Store)(nil)
