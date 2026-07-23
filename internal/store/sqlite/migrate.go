package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

// ErrMigration classifies failures while constructing or applying the embedded
// schema. Callers may use it to report a safe operational state without
// exposing SQL or filesystem details.
var ErrMigration = errors.New("sqlite migration failed")

func applyMigrations(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("%w: create provider: %w", ErrMigration, err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("%w: apply embedded schema: %w", ErrMigration, err)
	}
	return nil
}
