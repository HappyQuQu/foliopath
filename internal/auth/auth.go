// Package auth owns FolioPath's single-administrator identity and
// authentication state transitions.
package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"
)

var (
	ErrSetupClosed            = errors.New("administrator setup is closed")
	ErrSetupInProgress        = errors.New("administrator setup is in progress")
	ErrRepositoryNotReady     = errors.New("authentication repository is not ready")
	ErrInvalidUsername        = errors.New("invalid administrator username")
	ErrInvalidDisplayName     = errors.New("invalid administrator display name")
	ErrInvalidPassword        = errors.New("invalid administrator password")
	ErrInvalidCredentials     = errors.New("invalid administrator credentials")
	ErrAdministratorNotFound  = errors.New("administrator does not exist")
	ErrAuthenticationRequired = errors.New("administrator authentication is required")
	ErrSessionExpired         = errors.New("administrator session has expired")
	ErrPreconditionFailed     = errors.New("account revision precondition failed")
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

type Account struct {
	ID          int64
	Username    string
	DisplayName string
	Revision    int64
	UpdatedAtMS int64
}

type UpdateAccountParams struct {
	UserID           int64
	DisplayName      string
	ExpectedRevision int64
	UpdatedAtMS      int64
}

type ChangePasswordParams struct {
	UserID              int64
	CurrentSessionID    int64
	CurrentTokenHash    [sessionDigestBytes]byte
	ExpectedRevision    int64
	ExpectedAuthVersion int64
	PasswordVerifier    PasswordVerifier
	ChangedAtMS         int64
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
	CreateAdministratorWithSession(
		context.Context,
		CreateAdministratorParams,
		CreateSessionParams,
	) (Administrator, StoredSession, error)
	FindAdministratorCredential(context.Context, string) (AdministratorCredential, error)
	CreateSession(context.Context, CreateSessionParams, int64) (StoredSession, error)
	FindSession(context.Context, [sessionDigestBytes]byte) (StoredSession, error)
	TouchSession(context.Context, TouchSessionParams) (bool, error)
	RevokeSession(context.Context, RevokeSessionParams) (bool, error)
}

type AccountRepository interface {
	GetAccount(context.Context, int64) (Account, error)
	UpdateAccount(context.Context, UpdateAccountParams) (Account, error)
	ChangePassword(context.Context, ChangePasswordParams) (Account, error)
}

type PasswordManager interface {
	Hash(context.Context, string) (PasswordVerifier, error)
	Verify(context.Context, string, PasswordVerifier) (bool, error)
}

type Service struct {
	repository  Repository
	accounts    AccountRepository
	passwords   PasswordManager
	random      io.Reader
	now         func() time.Time
	setupGate   chan struct{}
	accountGate chan struct{}
}

type ServiceOptions struct {
	Random io.Reader
	Now    func() time.Time
}

func NewService(
	repository Repository,
	passwords PasswordManager,
	options ServiceOptions,
) (*Service, error) {
	if repository == nil {
		return nil, errors.New("authentication repository is required")
	}
	if passwords == nil {
		return nil, errors.New("password manager is required")
	}
	if options.Random == nil {
		options.Random = rand.Reader
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	service := &Service{
		repository:  repository,
		passwords:   passwords,
		random:      options.Random,
		now:         options.Now,
		setupGate:   make(chan struct{}, 1),
		accountGate: make(chan struct{}, 1),
	}
	service.accounts, _ = repository.(AccountRepository)
	return service, nil
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
) (EstablishedSession, error) {
	if ctx == nil {
		return EstablishedSession{}, errors.New("authentication context is nil")
	}

	username, usernameKey, err := NormalizeUsername(params.Username)
	if err != nil {
		return EstablishedSession{}, err
	}
	displayName, err := NormalizeDisplayName(params.DisplayName)
	if err != nil {
		return EstablishedSession{}, err
	}
	if err := ValidatePassword(params.Password); err != nil {
		return EstablishedSession{}, err
	}

	select {
	case service.setupGate <- struct{}{}:
		defer func() { <-service.setupGate }()
	default:
		return EstablishedSession{}, ErrSetupInProgress
	}

	state, err := service.SetupState(ctx)
	if err != nil {
		return EstablishedSession{}, err
	}
	if state == SetupComplete {
		return EstablishedSession{}, ErrSetupClosed
	}

	verifier, err := service.passwords.Hash(ctx, params.Password)
	if err != nil {
		return EstablishedSession{}, fmt.Errorf("create password verifier: %w", err)
	}
	if ctx.Err() != nil {
		return EstablishedSession{}, ctx.Err()
	}

	secrets, err := issueSessionSecrets(service.random)
	if err != nil {
		return EstablishedSession{}, err
	}
	now := service.now().UTC()
	sessionParams := newCreateSessionParams(0, 1, secrets, now)
	administrator, session, err := service.repository.CreateAdministratorWithSession(
		ctx,
		CreateAdministratorParams{
			Username:         username,
			UsernameKey:      usernameKey,
			DisplayName:      displayName,
			PasswordVerifier: verifier,
		},
		sessionParams,
	)
	if err != nil {
		return EstablishedSession{}, fmt.Errorf("persist administrator and session: %w", err)
	}
	return establishedSession(administrator, session, secrets), nil
}
