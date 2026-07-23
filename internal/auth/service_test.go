package auth

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryRepository struct {
	mutex          sync.Mutex
	initialized    bool
	created        CreateAdministratorParams
	createdSession CreateSessionParams
}

func (repository *memoryRepository) AdministratorInitialized(context.Context) (bool, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	return repository.initialized, nil
}

func (repository *memoryRepository) CreateAdministratorWithSession(
	_ context.Context,
	params CreateAdministratorParams,
	sessionParams CreateSessionParams,
) (Administrator, StoredSession, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	if repository.initialized {
		return Administrator{}, StoredSession{}, ErrSetupClosed
	}
	repository.initialized = true
	repository.created = params
	repository.createdSession = sessionParams
	administrator := Administrator{
		ID:          1,
		Username:    params.Username,
		DisplayName: params.DisplayName,
		CreatedAtMS: 1000,
		UpdatedAtMS: 1000,
	}
	return administrator, StoredSession{
		ID:              1,
		Administrator:   administrator,
		AuthVersion:     sessionParams.AuthVersion,
		UserAuthVersion: sessionParams.AuthVersion,
		CreatedAtMS:     sessionParams.CreatedAtMS,
		LastSeenAtMS:    sessionParams.CreatedAtMS,
		ExpiresAtMS:     sessionParams.ExpiresAtMS,
	}, nil
}

func (*memoryRepository) FindAdministratorCredential(
	context.Context,
	string,
) (AdministratorCredential, error) {
	return AdministratorCredential{}, ErrAdministratorNotFound
}

func (*memoryRepository) CreateSession(
	context.Context,
	CreateSessionParams,
	int64,
) (StoredSession, error) {
	return StoredSession{}, errors.New("not implemented by setup test")
}

func (*memoryRepository) FindSession(
	context.Context,
	[sessionDigestBytes]byte,
) (StoredSession, error) {
	return StoredSession{}, ErrAuthenticationRequired
}

func (*memoryRepository) TouchSession(
	context.Context,
	TouchSessionParams,
) (bool, error) {
	return false, errors.New("not implemented by setup test")
}

func (*memoryRepository) RevokeSession(
	context.Context,
	RevokeSessionParams,
) (bool, error) {
	return false, errors.New("not implemented by setup test")
}

type recordingPasswordManager struct {
	mutex    sync.Mutex
	password string
	calls    int
	started  chan struct{}
	release  chan struct{}
	err      error
}

func (manager *recordingPasswordManager) Hash(
	ctx context.Context,
	password string,
) (PasswordVerifier, error) {
	manager.mutex.Lock()
	manager.password = password
	manager.calls++
	started := manager.started
	release := manager.release
	err := manager.err
	manager.mutex.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			return PasswordVerifier{}, ctx.Err()
		}
	}
	if err != nil {
		return PasswordVerifier{}, err
	}
	return PasswordVerifier{
		EncodedHash: "encoded-verifier",
		Scheme:      "test",
		Parameters:  "version=1",
	}, nil
}

func (*recordingPasswordManager) Verify(
	context.Context,
	string,
	PasswordVerifier,
) (bool, error) {
	return false, errors.New("not implemented by setup test")
}

func TestServiceInitializesExactlyOneNormalizedAdministrator(t *testing.T) {
	repository := &memoryRepository{}
	passwords := &recordingPasswordManager{}
	service := newTestService(t, repository, passwords)

	state, err := service.SetupState(context.Background())
	if err != nil || state != SetupRequired {
		t.Fatalf("initial SetupState() = %q, %v; want required", state, err)
	}
	administrator, err := service.Initialize(context.Background(), InitializeParams{
		Username:    "Ａdministrator",
		DisplayName: "  Home Admin  ",
		Password:    "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if administrator.Administrator.Username != "Administrator" ||
		administrator.Administrator.DisplayName != "Home Admin" {
		t.Fatalf("administrator = %#v", administrator)
	}
	if administrator.CookieToken == "" || administrator.CSRFToken == "" {
		t.Fatal("initialization did not establish a session")
	}
	if repository.created.UsernameKey != "administrator" {
		t.Fatalf("username key = %q, want administrator", repository.created.UsernameKey)
	}
	if repository.created.PasswordVerifier.EncodedHash != "encoded-verifier" {
		t.Fatalf("password verifier = %#v", repository.created.PasswordVerifier)
	}
	if passwords.password != "correct horse battery staple" {
		t.Fatal("password manager did not receive the original password")
	}

	state, err = service.SetupState(context.Background())
	if err != nil || state != SetupComplete {
		t.Fatalf("completed SetupState() = %q, %v; want complete", state, err)
	}
	if _, err := service.Initialize(context.Background(), InitializeParams{
		Username:    "other",
		DisplayName: "Other",
		Password:    "another valid password",
	}); !errors.Is(err, ErrSetupClosed) {
		t.Fatalf("second Initialize() error = %v, want ErrSetupClosed", err)
	}
	if passwords.calls != 1 {
		t.Fatalf("password hash calls = %d, want 1", passwords.calls)
	}
}

func TestServiceRejectsConcurrentSetupBeforeASecondPasswordHash(t *testing.T) {
	repository := &memoryRepository{}
	passwords := &recordingPasswordManager{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	service := newTestService(t, repository, passwords)
	params := InitializeParams{
		Username:    "admin",
		DisplayName: "Administrator",
		Password:    "correct horse battery staple",
	}

	firstResult := make(chan error, 1)
	go func() {
		_, initializeErr := service.Initialize(context.Background(), params)
		firstResult <- initializeErr
	}()
	<-passwords.started

	if _, err := service.Initialize(context.Background(), params); !errors.Is(err, ErrSetupInProgress) {
		t.Fatalf("concurrent Initialize() error = %v, want ErrSetupInProgress", err)
	}
	close(passwords.release)
	if err := <-firstResult; err != nil {
		t.Fatalf("first Initialize() error = %v", err)
	}
	if passwords.calls != 1 {
		t.Fatalf("password hash calls = %d, want 1", passwords.calls)
	}
}

func TestServiceDoesNotLeakPasswordInHasherFailure(t *testing.T) {
	password := "do not expose this password"
	service := newTestService(
		t,
		&memoryRepository{},
		&recordingPasswordManager{err: errors.New("password backend failed")},
	)

	_, err := service.Initialize(context.Background(), InitializeParams{
		Username:    "admin",
		DisplayName: "Administrator",
		Password:    password,
	})
	if err == nil {
		t.Fatal("Initialize() unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), password) {
		t.Fatalf("Initialize() error leaked password: %v", err)
	}
}

func newTestService(
	t *testing.T,
	repository Repository,
	passwords PasswordManager,
) *Service {
	t.Helper()
	service, err := NewService(
		repository,
		passwords,
		ServiceOptions{
			Random: bytes.NewReader(deterministicRandom(0x42)),
			Now: func() time.Time {
				return time.UnixMilli(1700000000000)
			},
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func deterministicRandom(seed byte) []byte {
	random := make([]byte, 4096)
	for index := range random {
		random[index] = seed + byte(index)
	}
	return random
}
