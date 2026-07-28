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

func (store *Store) CreateAdministratorWithSession(
	ctx context.Context,
	params auth.CreateAdministratorParams,
	sessionParams auth.CreateSessionParams,
) (auth.Administrator, auth.StoredSession, error) {
	var created auth.Administrator
	var session auth.StoredSession
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
		created = administratorFromInsert(record)
		sessionRecord, err := queries.InsertSession(
			ctx,
			dbgen.InsertSessionParams{
				UserID:        record.ID,
				TokenHash:     sessionParams.TokenHash[:],
				CsrfTokenHash: sessionParams.CSRFTokenHash[:],
				AuthVersion:   sessionParams.AuthVersion,
				CreatedAtMs:   sessionParams.CreatedAtMS,
				ExpiresAtMs:   sessionParams.ExpiresAtMS,
			},
		)
		if err != nil {
			return fmt.Errorf("insert initial administrator session: %w", err)
		}
		session = storedSessionFromInsert(sessionRecord, created)
		return nil
	})
	if errors.Is(err, auth.ErrSetupClosed) {
		return auth.Administrator{}, auth.StoredSession{}, auth.ErrSetupClosed
	}
	return created, session, err
}

func (store *Store) FindAdministratorCredential(
	ctx context.Context,
	usernameKey string,
) (auth.AdministratorCredential, error) {
	record, err := store.queries.FindAdministratorCredential(ctx, usernameKey)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.AdministratorCredential{}, auth.ErrAdministratorNotFound
	}
	if err != nil {
		return auth.AdministratorCredential{}, fmt.Errorf("find administrator credential: %w", err)
	}
	return auth.AdministratorCredential{
		Administrator: auth.Administrator{
			ID:          record.ID,
			Username:    record.Username,
			DisplayName: record.DisplayName,
			CreatedAtMS: record.CreatedAtMs,
			UpdatedAtMS: record.UpdatedAtMs,
		},
		UsernameKey: record.UsernameKey,
		PasswordVerifier: auth.PasswordVerifier{
			EncodedHash: record.PasswordHash,
			Scheme:      record.PasswordScheme,
			Parameters:  record.PasswordParameters,
		},
		AuthVersion: record.AuthVersion,
		Disabled:    record.DisabledAtMs.Valid,
	}, nil
}

func (store *Store) CreateSession(
	ctx context.Context,
	params auth.CreateSessionParams,
	obsoleteCutoffMS int64,
) (auth.StoredSession, error) {
	var created auth.StoredSession
	err := store.withWriteTx(ctx, func(tx *sql.Tx) error {
		queries := dbgen.New(tx)
		if _, err := queries.DeleteObsoleteSessions(ctx, obsoleteCutoffMS); err != nil {
			return fmt.Errorf("delete obsolete sessions: %w", err)
		}
		record, err := queries.InsertSession(
			ctx,
			dbgen.InsertSessionParams{
				UserID:        params.UserID,
				TokenHash:     params.TokenHash[:],
				CsrfTokenHash: params.CSRFTokenHash[:],
				AuthVersion:   params.AuthVersion,
				CreatedAtMs:   params.CreatedAtMS,
				ExpiresAtMs:   params.ExpiresAtMS,
			},
		)
		if err != nil {
			return fmt.Errorf("insert session: %w", err)
		}
		created = storedSessionFromInsert(record, auth.Administrator{ID: params.UserID})
		return nil
	})
	return created, err
}

func (store *Store) FindSession(
	ctx context.Context,
	tokenHash [32]byte,
) (auth.StoredSession, error) {
	record, err := store.queries.FindSession(ctx, tokenHash[:])
	if errors.Is(err, sql.ErrNoRows) {
		return auth.StoredSession{}, auth.ErrAuthenticationRequired
	}
	if err != nil {
		return auth.StoredSession{}, fmt.Errorf("find session: %w", err)
	}
	var revokedAtMS *int64
	if record.RevokedAtMs.Valid {
		revokedAtMS = &record.RevokedAtMs.Int64
	}
	var csrfTokenHash [32]byte
	copy(csrfTokenHash[:], record.CsrfTokenHash)
	return auth.StoredSession{
		ID: record.ID,
		Administrator: auth.Administrator{
			ID:          record.UserID,
			Username:    record.Username,
			DisplayName: record.DisplayName,
			CreatedAtMS: record.UserCreatedAtMs,
			UpdatedAtMS: record.UserUpdatedAtMs,
		},
		AuthVersion:     record.AuthVersion,
		UserAuthVersion: record.UserAuthVersion,
		CreatedAtMS:     record.CreatedAtMs,
		LastSeenAtMS:    record.LastSeenAtMs,
		ExpiresAtMS:     record.ExpiresAtMs,
		CSRFTokenHash:   csrfTokenHash,
		RevokedAtMS:     revokedAtMS,
		UserDisabled:    record.UserDisabledAtMs.Valid,
	}, nil
}

func (store *Store) TouchSession(
	ctx context.Context,
	params auth.TouchSessionParams,
) (bool, error) {
	var touched bool
	err := store.withWriteTx(ctx, func(tx *sql.Tx) error {
		affected, err := dbgen.New(tx).TouchSession(
			ctx,
			dbgen.TouchSessionParams{
				UsedAtMs:        params.UsedAtMS,
				ID:              params.SessionID,
				TokenHash:       params.TokenHash[:],
				ExpectedVersion: params.ExpectedVersion,
			},
		)
		if err != nil {
			return fmt.Errorf("touch session: %w", err)
		}
		touched = affected == 1
		return nil
	})
	return touched, err
}

func (store *Store) RevokeSession(
	ctx context.Context,
	params auth.RevokeSessionParams,
) (bool, error) {
	affected, err := store.queries.RevokeSession(
		ctx,
		dbgen.RevokeSessionParams{
			RevokedAtMs: sql.NullInt64{Int64: params.RevokedAtMS, Valid: true},
			ID:          params.SessionID,
			TokenHash:   params.TokenHash[:],
		},
	)
	if err != nil {
		return false, fmt.Errorf("revoke session: %w", err)
	}
	return affected == 1, nil
}

func administratorFromInsert(record dbgen.InsertAdministratorRow) auth.Administrator {
	return auth.Administrator{
		ID:          record.ID,
		Username:    record.Username,
		DisplayName: record.DisplayName,
		CreatedAtMS: record.CreatedAtMs,
		UpdatedAtMS: record.UpdatedAtMs,
	}
}

func storedSessionFromInsert(
	record dbgen.InsertSessionRow,
	administrator auth.Administrator,
) auth.StoredSession {
	return auth.StoredSession{
		ID:              record.ID,
		Administrator:   administrator,
		AuthVersion:     record.AuthVersion,
		UserAuthVersion: record.AuthVersion,
		CreatedAtMS:     record.CreatedAtMs,
		LastSeenAtMS:    record.LastSeenAtMs,
		ExpiresAtMS:     record.ExpiresAtMs,
	}
}
