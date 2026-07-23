package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

func applyMigrations(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("create goose migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply sqlite migrations: %w", err)
	}
	return nil
}
