// Package auth owns FolioPath's single-administrator identity and
// authentication state transitions.
package auth

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrSetupClosed        = errors.New("administrator setup is closed")
	ErrSetupInProgress    = errors.New("administrator setup is in progress")
	ErrRepositoryNotReady = errors.New("authentication repository is not ready")
	ErrInvalidUsername    = errors.New("invalid administrator username")
	ErrInvalidDisplayName = errors.New("invalid administrator display name")
	ErrInvalidPassword    = errors.New("invalid administrator password")
)

type SetupState string

const (
	SetupRequired SetupState = "setup_required"
	SetupComplete SetupState = "setup_complete"
)

type Administrator struct {
	ID          int64
	Username    string
	DisplayName string
	CreatedAtMS int64
	UpdatedAtMS int64
}

type PasswordVerifier struct {
	EncodedHash string
	Scheme      string
	Parameters  string
}

type InitializeParams struct {
	Username    string
	DisplayName string
	Password    string
}

type CreateAdministratorParams struct {
	Username         string
	UsernameKey      string
	DisplayName      string
	PasswordVerifier PasswordVerifier
}

type Repository interface {
	AdministratorInitialized(context.Context) (bool, error)
	CreateAdministrator(context.Context, CreateAdministratorParams) (Administrator, error)
}

type PasswordManager interface {
	Hash(context.Context, string) (PasswordVerifier, error)
	Verify(context.Context, string, PasswordVerifier) (bool, error)
}

type Service struct {
	repository Repository
	passwords  PasswordManager
	setupGate  chan struct{}
}

func NewService(repository Repository, passwords PasswordManager) (*Service, error) {
	if repository == nil {
		return nil, errors.New("authentication repository is required")
	}
	if passwords == nil {
		return nil, errors.New("password manager is required")
	}
	return &Service{
		repository: repository,
		passwords:  passwords,
		setupGate:  make(chan struct{}, 1),
	}, nil
}

func (service *Service) SetupState(ctx context.Context) (SetupState, error) {
	if ctx == nil {
		return "", errors.New("authentication context is nil")
	}
	initialized, err := service.repository.AdministratorInitialized(ctx)
	if err != nil {
		return "", fmt.Errorf("read administrator setup state: %w", err)
	}
	if initialized {
		return SetupComplete, nil
	}
	return SetupRequired, nil
}

func (service *Service) Initialize(
	ctx context.Context,
	params InitializeParams,
) (Administrator, error) {
	if ctx == nil {
		return Administrator{}, errors.New("authentication context is nil")
	}

	username, usernameKey, err := NormalizeUsername(params.Username)
	if err != nil {
		return Administrator{}, err
	}
	displayName, err := NormalizeDisplayName(params.DisplayName)
	if err != nil {
		return Administrator{}, err
	}
	if err := ValidatePassword(params.Password); err != nil {
		return Administrator{}, err
	}

	select {
	case service.setupGate <- struct{}{}:
		defer func() { <-service.setupGate }()
	default:
		return Administrator{}, ErrSetupInProgress
	}

	state, err := service.SetupState(ctx)
	if err != nil {
		return Administrator{}, err
	}
	if state == SetupComplete {
		return Administrator{}, ErrSetupClosed
	}

	verifier, err := service.passwords.Hash(ctx, params.Password)
	if err != nil {
		return Administrator{}, fmt.Errorf("create password verifier: %w", err)
	}
	if ctx.Err() != nil {
		return Administrator{}, ctx.Err()
	}

	administrator, err := service.repository.CreateAdministrator(
		ctx,
		CreateAdministratorParams{
			Username:         username,
			UsernameKey:      usernameKey,
			DisplayName:      displayName,
			PasswordVerifier: verifier,
		},
	)
	if err != nil {
		return Administrator{}, fmt.Errorf("persist administrator: %w", err)
	}
	return administrator, nil
}
