package auth

import (
	"context"
	"errors"
	"fmt"
)

type ProfileUpdate struct {
	DisplayName      string
	ExpectedRevision int64
}

type PasswordChange struct {
	CurrentPassword  string
	NewPassword      string
	ExpectedRevision int64
}

func (service *Service) Account(
	ctx context.Context,
	cookieToken string,
) (Account, error) {
	if service.accounts == nil {
		return Account{}, ErrRepositoryNotReady
	}
	current, err := service.Session(ctx, cookieToken)
	if err != nil {
		return Account{}, err
	}
	account, err := service.accounts.GetAccount(ctx, current.Administrator.ID)
	if err != nil {
		return Account{}, fmt.Errorf("read administrator account: %w", err)
	}
	return account, nil
}

func (service *Service) UpdateProfile(
	ctx context.Context,
	cookieToken string,
	update ProfileUpdate,
) (Account, error) {
	if service.accounts == nil {
		return Account{}, ErrRepositoryNotReady
	}
	if update.ExpectedRevision < 1 {
		return Account{}, ErrPreconditionFailed
	}
	displayName, err := NormalizeDisplayName(update.DisplayName)
	if err != nil {
		return Account{}, err
	}
	current, err := service.Session(ctx, cookieToken)
	if err != nil {
		return Account{}, err
	}
	account, err := service.accounts.UpdateAccount(ctx, UpdateAccountParams{
		UserID:           current.Administrator.ID,
		DisplayName:      displayName,
		ExpectedRevision: update.ExpectedRevision,
		UpdatedAtMS:      service.now().UTC().UnixMilli(),
	})
	if err != nil {
		return Account{}, fmt.Errorf("update administrator account: %w", err)
	}
	return account, nil
}

func (service *Service) ChangeAccountPassword(
	ctx context.Context,
	cookieToken string,
	change PasswordChange,
) (Account, error) {
	if service.accounts == nil {
		return Account{}, ErrRepositoryNotReady
	}
	if change.ExpectedRevision < 1 {
		return Account{}, ErrPreconditionFailed
	}
	if err := ValidatePassword(change.NewPassword); err != nil {
		return Account{}, err
	}

	select {
	case service.accountGate <- struct{}{}:
		defer func() { <-service.accountGate }()
	case <-ctx.Done():
		return Account{}, ctx.Err()
	}

	current, err := service.Session(ctx, cookieToken)
	if err != nil {
		return Account{}, err
	}
	tokenHash, _, _, err := parseSessionCookie(cookieToken)
	if err != nil {
		return Account{}, ErrAuthenticationRequired
	}
	session, err := service.repository.FindSession(ctx, tokenHash)
	if err != nil {
		return Account{}, err
	}
	_, usernameKey, err := NormalizeUsername(current.Administrator.Username)
	if err != nil {
		return Account{}, fmt.Errorf("normalize stored administrator username: %w", err)
	}
	credential, err := service.repository.FindAdministratorCredential(ctx, usernameKey)
	if err != nil {
		return Account{}, err
	}
	matches, err := service.passwords.Verify(ctx, change.CurrentPassword, credential.PasswordVerifier)
	if err != nil {
		return Account{}, fmt.Errorf("verify current administrator password: %w", err)
	}
	if !matches {
		return Account{}, ErrInvalidCredentials
	}
	verifier, err := service.passwords.Hash(ctx, change.NewPassword)
	if err != nil {
		return Account{}, fmt.Errorf("create replacement password verifier: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	account, err := service.accounts.ChangePassword(ctx, ChangePasswordParams{
		UserID:              current.Administrator.ID,
		CurrentSessionID:    session.ID,
		CurrentTokenHash:    tokenHash,
		ExpectedRevision:    change.ExpectedRevision,
		ExpectedAuthVersion: credential.AuthVersion,
		PasswordVerifier:    verifier,
		ChangedAtMS:         service.now().UTC().UnixMilli(),
	})
	if err != nil {
		if errors.Is(err, ErrPreconditionFailed) || errors.Is(err, ErrSessionExpired) {
			return Account{}, err
		}
		return Account{}, fmt.Errorf("change administrator password: %w", err)
	}
	return account, nil
}
