package face

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	ErrInvalidPerson  = errors.New("invalid person")
	ErrPersonConflict = errors.New("person revision conflict")
	ErrPersonNotFound = errors.New("person not found")
)

type Person struct {
	ID                 string
	Name               string
	ConfirmedFaceCount int64
	AssetCount         int64
	Revision           int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreatePersonCommand struct {
	ID, Name  string
	CreatedAt time.Time
}
type RenamePersonCommand struct {
	ID, Name         string
	ExpectedRevision int64
	UpdatedAt        time.Time
}
type DeletePersonCommand struct {
	ID               string
	ExpectedRevision int64
	DeletedAt        time.Time
}

type PeopleRepository interface {
	CreatePerson(context.Context, CreatePersonCommand) (Person, error)
	RenamePerson(context.Context, RenamePersonCommand) (Person, error)
	DeletePerson(context.Context, DeletePersonCommand) error
	GetPerson(context.Context, string) (Person, error)
	ListPeople(context.Context, int) ([]Person, error)
}

func NormalizePersonName(value string) (string, error) {
	value = norm.NFC.String(strings.TrimSpace(value))
	if value == "" || utf8.RuneCountInString(value) > 100 || strings.ContainsRune(value, '\x00') {
		return "", ErrInvalidPerson
	}
	return value, nil
}
