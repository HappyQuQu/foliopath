package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/auth"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite/dbgen"
)

var _ auth.Repository = (*Store)(nil)

func (store *Store) AdministratorInitialized(ctx context.Context) (bool, error) {
	initialized, err := store.queries.IsAdministratorInitialized(ctx)
	if err != nil {
		return false, fmt.Errorf("read administrator setup state: %w", err)
	}
	return initialized, nil
}

func (store *Store) CreateAdministrator(
	ctx context.Context,
	params auth.CreateAdministratorParams,
) (auth.Administrator, error) {
	var created auth.Administrator
	err := store.withWriteTx(ctx, func(tx *sql.Tx) error {
		queries := dbgen.New(tx)
		initialized, err := queries.IsAdministratorInitialized(ctx)
		if err != nil {
			return fmt.Errorf("check administrator setup state: %w", err)
		}
		if initialized {
			return auth.ErrSetupClosed
		}

		now := store.nowMS()
		record, err := queries.InsertAdministrator(
			ctx,
			dbgen.InsertAdministratorParams{
				Username:            params.Username,
				UsernameKey:         params.UsernameKey,
				DisplayName:         params.DisplayName,
				PasswordHash:        params.PasswordVerifier.EncodedHash,
				PasswordScheme:      params.PasswordVerifier.Scheme,
				PasswordParameters:  params.PasswordVerifier.Parameters,
				CreatedAtMs:         now,
				UpdatedAtMs:         now,
				PasswordChangedAtMs: now,
			},
		)
		if err != nil {
			initialized, stateErr := queries.IsAdministratorInitialized(ctx)
			if stateErr == nil && initialized {
				return auth.ErrSetupClosed
			}
			return fmt.Errorf("insert administrator: %w", err)
		}
		created = auth.Administrator{
			ID:          record.ID,
			Username:    record.Username,
			DisplayName: record.DisplayName,
			CreatedAtMS: record.CreatedAtMs,
			UpdatedAtMS: record.UpdatedAtMs,
		}
		return nil
	})
	if errors.Is(err, auth.ErrSetupClosed) {
		return auth.Administrator{}, auth.ErrSetupClosed
	}
	return created, err
}
