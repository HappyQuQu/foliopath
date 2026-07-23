package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"testing"
	"time"
)

type sessionRepository struct {
	credential       AdministratorCredential
	credentialErr    error
	session          StoredSession
	sessionErr       error
	created          CreateSessionParams
	obsoleteCutoffMS int64
	touched          TouchSessionParams
	touchResult      bool
	revoked          RevokeSessionParams
	revokeResult     bool
}

func (*sessionRepository) AdministratorInitialized(context.Context) (bool, error) {
	return true, nil
}

func (*sessionRepository) CreateAdministratorWithSession(
	context.Context,
	CreateAdministratorParams,
	CreateSessionParams,
) (Administrator, StoredSession, error) {
	return Administrator{}, StoredSession{}, errors.New("not implemented by session test")
}

func (repository *sessionRepository) FindAdministratorCredential(
	context.Context,
	string,
) (AdministratorCredential, error) {
	return repository.credential, repository.credentialErr
}

func (repository *sessionRepository) CreateSession(
	_ context.Context,
	params CreateSessionParams,
	obsoleteCutoffMS int64,
) (StoredSession, error) {
	repository.created = params
	repository.obsoleteCutoffMS = obsoleteCutoffMS
	created := repository.session
	created.ExpiresAtMS = params.ExpiresAtMS
	return created, repository.sessionErr
}

func (repository *sessionRepository) FindSession(
	context.Context,
	[sessionDigestBytes]byte,
) (StoredSession, error) {
	return repository.session, repository.sessionErr
}

func (repository *sessionRepository) TouchSession(
	_ context.Context,
	params TouchSessionParams,
) (bool, error) {
	repository.touched = params
	return repository.touchResult, repository.sessionErr
}

func (repository *sessionRepository) RevokeSession(
	_ context.Context,
	params RevokeSessionParams,
) (bool, error) {
	repository.revoked = params
	return repository.revokeResult, repository.sessionErr
}

type verifyingPasswordManager struct {
	matches  bool
	err      error
	calls    int
	password string
	verifier PasswordVerifier
}

func (*verifyingPasswordManager) Hash(
	context.Context,
	string,
) (PasswordVerifier, error) {
	return PasswordVerifier{}, errors.New("not implemented by session test")
}

func (manager *verifyingPasswordManager) Verify(
	_ context.Context,
	password string,
	verifier PasswordVerifier,
) (bool, error) {
	manager.calls++
	manager.password = password
	manager.verifier = verifier
	return manager.matches, manager.err
}

func TestServiceLoginCreatesOpaqueAbsoluteSession(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	administrator := Administrator{
		ID:          7,
		Username:    "Administrator",
		DisplayName: "Home Admin",
	}
	repository := &sessionRepository{
		credential: AdministratorCredential{
			Administrator: administrator,
			UsernameKey:   "administrator",
			PasswordVerifier: PasswordVerifier{
				EncodedHash: "password-verifier",
				Scheme:      "test",
				Parameters:  "v=1",
			},
			AuthVersion: 3,
		},
		session: StoredSession{
			ID:              11,
			Administrator:   administrator,
			AuthVersion:     3,
			UserAuthVersion: 3,
		},
	}
	passwords := &verifyingPasswordManager{matches: true}
	service := newSessionTestService(t, repository, passwords, now, 0x31)

	established, err := service.Login(context.Background(), LoginParams{
		Username: "Ａdministrator",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if established.Administrator != administrator ||
		established.ExpiresAtMS != now.Add(SessionLifetime).UnixMilli() {
		t.Fatalf("established session = %#v", established)
	}
	if len(established.CookieToken) != 86 || len(established.CSRFToken) != 43 {
		t.Fatalf(
			"secret lengths = (%d, %d), want (86, 43)",
			len(established.CookieToken),
			len(established.CSRFToken),
		)
	}
	if repository.created.UserID != administrator.ID ||
		repository.created.AuthVersion != 3 ||
		repository.created.CreatedAtMS != now.UnixMilli() ||
		repository.created.ExpiresAtMS != now.Add(SessionLifetime).UnixMilli() {
		t.Fatalf("created session params = %#v", repository.created)
	}
	if repository.created.TokenHash != mustDigestSecret(t, established.CookieToken) ||
		repository.created.CSRFTokenHash != mustDigestSecret(t, established.CSRFToken) {
		t.Fatal("repository did not receive the issued secret digests")
	}
	if repository.obsoleteCutoffMS != now.Add(-obsoleteSessionGrace).UnixMilli() {
		t.Fatalf("obsolete cutoff = %d", repository.obsoleteCutoffMS)
	}
	if passwords.password != "correct horse battery staple" ||
		passwords.verifier != repository.credential.PasswordVerifier {
		t.Fatal("login did not verify the stored credential")
	}
}

func TestServiceLoginUsesGenericFailureAndDummyVerifier(t *testing.T) {
	repository := &sessionRepository{credentialErr: ErrAdministratorNotFound}
	passwords := &verifyingPasswordManager{matches: true}
	service := newSessionTestService(
		t,
		repository,
		passwords,
		time.UnixMilli(1700000000000),
		0x32,
	)

	_, err := service.Login(context.Background(), LoginParams{
		Username: "missing",
		Password: "incorrect password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want invalid credentials", err)
	}
	if passwords.calls != 1 || passwords.verifier != dummyPasswordVerifier {
		t.Fatal("unknown administrator did not exercise the dummy verifier")
	}
}

func TestServiceLoginRejectsWrongPasswordOrDisabledAdministrator(t *testing.T) {
	for name, disabled := range map[string]bool{
		"wrong password": false,
		"disabled":       true,
	} {
		t.Run(name, func(t *testing.T) {
			repository := &sessionRepository{
				credential: AdministratorCredential{
					Administrator: Administrator{ID: 7},
					PasswordVerifier: PasswordVerifier{
						EncodedHash: "password-verifier",
						Scheme:      "test",
						Parameters:  "v=1",
					},
					AuthVersion: 1,
					Disabled:    disabled,
				},
			}
			passwords := &verifyingPasswordManager{matches: disabled}
			service := newSessionTestService(
				t,
				repository,
				passwords,
				time.UnixMilli(1700000000000),
				0x33,
			)
			if _, err := service.Login(
				context.Background(),
				LoginParams{Username: "admin", Password: "password"},
			); !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want invalid credentials", err)
			}
			if repository.created != (CreateSessionParams{}) {
				t.Fatal("invalid credentials created a session")
			}
		})
	}
}

func TestServiceSessionRecoversCSRFAndTouchesWithoutExtendingExpiry(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	cookieToken := encodedSessionCookie(0x71, 0x72)
	administrator := Administrator{ID: 7, Username: "admin", DisplayName: "Admin"}
	repository := &sessionRepository{
		session: StoredSession{
			ID:              11,
			Administrator:   administrator,
			AuthVersion:     2,
			UserAuthVersion: 2,
			CreatedAtMS:     now.Add(-time.Hour).UnixMilli(),
			LastSeenAtMS:    now.Add(-time.Minute).UnixMilli(),
			ExpiresAtMS:     now.Add(time.Hour).UnixMilli(),
			CSRFTokenHash:   mustDigestSecret(t, encodedSecret(0x72)),
		},
		touchResult: true,
	}
	service := newSessionTestService(
		t,
		repository,
		&verifyingPasswordManager{},
		now,
		0x41,
	)

	current, err := service.Session(context.Background(), cookieToken)
	if err != nil {
		t.Fatalf("Session() error = %v", err)
	}
	if current.Administrator != administrator ||
		current.ExpiresAtMS != repository.session.ExpiresAtMS ||
		current.CSRFToken != encodedSecret(0x72) {
		t.Fatalf("current session = %#v", current)
	}
	if repository.touched.SessionID != repository.session.ID ||
		repository.touched.TokenHash != mustDigestSecret(t, cookieToken) ||
		repository.touched.UsedAtMS != now.UnixMilli() {
		t.Fatalf("touch params = %#v", repository.touched)
	}
}

func TestServiceRejectsInactiveSessionsBeforeTouch(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	revokedAt := now.Add(-time.Minute).UnixMilli()
	tests := map[string]StoredSession{
		"expired": {
			AuthVersion:     1,
			UserAuthVersion: 1,
			ExpiresAtMS:     now.UnixMilli(),
		},
		"revoked": {
			AuthVersion:     1,
			UserAuthVersion: 1,
			ExpiresAtMS:     now.Add(time.Hour).UnixMilli(),
			RevokedAtMS:     &revokedAt,
		},
		"version changed": {
			AuthVersion:     1,
			UserAuthVersion: 2,
			ExpiresAtMS:     now.Add(time.Hour).UnixMilli(),
		},
		"administrator disabled": {
			AuthVersion:     1,
			UserAuthVersion: 1,
			ExpiresAtMS:     now.Add(time.Hour).UnixMilli(),
			UserDisabled:    true,
		},
	}
	for name, record := range tests {
		t.Run(name, func(t *testing.T) {
			repository := &sessionRepository{session: record, touchResult: true}
			service := newSessionTestService(
				t,
				repository,
				&verifyingPasswordManager{},
				now,
				0x51,
			)
			if _, err := service.Session(
				context.Background(),
				encodedSessionCookie(0x72, 0x73),
			); !errors.Is(err, ErrSessionExpired) {
				t.Fatalf("Session() error = %v, want session expired", err)
			}
			if repository.touched != (TouchSessionParams{}) {
				t.Fatal("inactive session was touched")
			}
		})
	}
}

func TestServiceRejectsSessionWithMismatchedCSRFDigest(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	repository := &sessionRepository{
		session: StoredSession{
			ID:              11,
			AuthVersion:     1,
			UserAuthVersion: 1,
			ExpiresAtMS:     now.Add(time.Hour).UnixMilli(),
			CSRFTokenHash:   mustDigestSecret(t, encodedSecret(0x99)),
		},
		touchResult: true,
	}
	service := newSessionTestService(
		t,
		repository,
		&verifyingPasswordManager{},
		now,
		0x52,
	)

	if _, err := service.Session(
		context.Background(),
		encodedSessionCookie(0x80, 0x81),
	); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("Session() error = %v, want session expired", err)
	}
	if repository.touched != (TouchSessionParams{}) {
		t.Fatal("corrupt session was touched")
	}
}

func TestServiceLogoutRevokesActiveSession(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	cookieToken := encodedSessionCookie(0x73, 0x74)
	repository := &sessionRepository{
		session: StoredSession{
			ID:              11,
			AuthVersion:     1,
			UserAuthVersion: 1,
			ExpiresAtMS:     now.Add(time.Hour).UnixMilli(),
		},
		revokeResult: true,
	}
	service := newSessionTestService(
		t,
		repository,
		&verifyingPasswordManager{},
		now,
		0x61,
	)

	if err := service.Logout(context.Background(), cookieToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if repository.revoked.SessionID != repository.session.ID ||
		repository.revoked.TokenHash != mustDigestSecret(t, cookieToken) ||
		repository.revoked.RevokedAtMS != now.UnixMilli() {
		t.Fatalf("revoke params = %#v", repository.revoked)
	}
}

func TestServiceRejectsMalformedOrUnknownSessionSecret(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	repository := &sessionRepository{sessionErr: ErrAuthenticationRequired}
	service := newSessionTestService(
		t,
		repository,
		&verifyingPasswordManager{},
		now,
		0x62,
	)

	for _, token := range []string{"", "not-base64!", encodedSessionCookie(0x74, 0x75)} {
		if _, err := service.Session(
			context.Background(),
			token,
		); !errors.Is(err, ErrAuthenticationRequired) {
			t.Errorf("Session(%q) error = %v, want authentication required", token, err)
		}
	}
}

func TestSessionSecretGenerationFailureDoesNotReturnPartialSecrets(t *testing.T) {
	service, err := NewService(
		&sessionRepository{
			credential: AdministratorCredential{
				Administrator:    Administrator{ID: 1},
				PasswordVerifier: dummyPasswordVerifier,
				AuthVersion:      1,
			},
		},
		&verifyingPasswordManager{matches: true},
		ServiceOptions{
			Random: io.LimitReader(bytes.NewReader(bytes.Repeat([]byte{0x01}, 40)), 40),
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	established, err := service.Login(context.Background(), LoginParams{
		Username: "admin",
		Password: "correct horse battery staple",
	})
	if err == nil {
		t.Fatal("Login() unexpectedly succeeded with short randomness")
	}
	if established.CookieToken != "" || established.CSRFToken != "" {
		t.Fatalf("partial secrets escaped: %#v", established)
	}
}

func newSessionTestService(
	t *testing.T,
	repository Repository,
	passwords PasswordManager,
	now time.Time,
	randomByte byte,
) *Service {
	t.Helper()
	service, err := NewService(
		repository,
		passwords,
		ServiceOptions{
			Random: bytes.NewReader(deterministicRandom(randomByte)),
			Now:    func() time.Time { return now },
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func encodedSecret(value byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{value}, sessionSecretBytes))
}

func encodedSessionCookie(sessionValue byte, csrfValue byte) string {
	raw := append(
		bytes.Repeat([]byte{sessionValue}, sessionSecretBytes),
		bytes.Repeat([]byte{csrfValue}, sessionSecretBytes)...,
	)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func mustDigestSecret(t *testing.T, encoded string) [sessionDigestBytes]byte {
	t.Helper()
	raw, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode secret: %v", err)
	}
	return sha256.Sum256(raw)
}
