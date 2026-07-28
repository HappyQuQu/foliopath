package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	sessionSecretBytes   = 32
	sessionCookieBytes   = 2 * sessionSecretBytes
	sessionDigestBytes   = sha256.Size
	SessionLifetime      = 7 * 24 * time.Hour
	obsoleteSessionGrace = 24 * time.Hour
	sessionTouchInterval = 30 * time.Second
)

type AdministratorCredential struct {
	Administrator
	UsernameKey      string
	PasswordVerifier PasswordVerifier
	AuthVersion      int64
	Disabled         bool
}

type CreateSessionParams struct {
	UserID        int64
	AuthVersion   int64
	TokenHash     [sessionDigestBytes]byte
	CSRFTokenHash [sessionDigestBytes]byte
	CreatedAtMS   int64
	ExpiresAtMS   int64
}

type TouchSessionParams struct {
	SessionID       int64
	TokenHash       [sessionDigestBytes]byte
	ExpectedVersion int64
	UsedAtMS        int64
}

type RevokeSessionParams struct {
	SessionID   int64
	TokenHash   [sessionDigestBytes]byte
	RevokedAtMS int64
}

type StoredSession struct {
	ID              int64
	Administrator   Administrator
	AuthVersion     int64
	UserAuthVersion int64
	CreatedAtMS     int64
	LastSeenAtMS    int64
	ExpiresAtMS     int64
	CSRFTokenHash   [sessionDigestBytes]byte
	RevokedAtMS     *int64
	UserDisabled    bool
}

type EstablishedSession struct {
	Administrator Administrator
	CookieToken   string
	CSRFToken     string
	ExpiresAtMS   int64
}

type CurrentSession struct {
	Administrator Administrator
	CSRFToken     string
	ExpiresAtMS   int64
}

type LoginParams struct {
	Username string
	Password string
}

type sessionSecrets struct {
	cookieToken string
	csrfToken   string
	tokenHash   [sessionDigestBytes]byte
	csrfHash    [sessionDigestBytes]byte
}

func (service *Service) Login(
	ctx context.Context,
	params LoginParams,
) (EstablishedSession, error) {
	if ctx == nil {
		return EstablishedSession{}, errors.New("authentication context is nil")
	}

	_, usernameKey, normalizeErr := NormalizeUsername(params.Username)
	credential, credentialErr := AdministratorCredential{}, ErrAdministratorNotFound
	if normalizeErr == nil {
		credential, credentialErr = service.repository.FindAdministratorCredential(ctx, usernameKey)
	}
	if credentialErr != nil && !errors.Is(credentialErr, ErrAdministratorNotFound) {
		return EstablishedSession{}, fmt.Errorf("find administrator credential: %w", credentialErr)
	}

	verifier := dummyPasswordVerifier
	if credentialErr == nil {
		verifier = credential.PasswordVerifier
	}
	matches, err := service.passwords.Verify(ctx, params.Password, verifier)
	if err != nil {
		return EstablishedSession{}, fmt.Errorf("verify administrator password: %w", err)
	}
	if credentialErr != nil || credential.Disabled || !matches {
		return EstablishedSession{}, ErrInvalidCredentials
	}

	secrets, err := issueSessionSecrets(service.random)
	if err != nil {
		return EstablishedSession{}, err
	}
	now := service.now().UTC()
	session, err := service.repository.CreateSession(
		ctx,
		newCreateSessionParams(credential.ID, credential.AuthVersion, secrets, now),
		now.Add(-obsoleteSessionGrace).UnixMilli(),
	)
	if err != nil {
		return EstablishedSession{}, fmt.Errorf("persist administrator session: %w", err)
	}
	return establishedSession(credential.Administrator, session, secrets), nil
}

func (service *Service) Session(
	ctx context.Context,
	cookieToken string,
) (CurrentSession, error) {
	if ctx == nil {
		return CurrentSession{}, errors.New("authentication context is nil")
	}
	tokenHash, csrfToken, csrfHash, err := parseSessionCookie(cookieToken)
	if err != nil {
		return CurrentSession{}, ErrAuthenticationRequired
	}
	record, err := service.repository.FindSession(ctx, tokenHash)
	if errors.Is(err, ErrAuthenticationRequired) {
		return CurrentSession{}, ErrAuthenticationRequired
	}
	if err != nil {
		return CurrentSession{}, fmt.Errorf("find administrator session: %w", err)
	}

	now := service.now().UTC()
	if !sessionActive(record, now.UnixMilli()) {
		return CurrentSession{}, ErrSessionExpired
	}
	if subtle.ConstantTimeCompare(record.CSRFTokenHash[:], csrfHash[:]) != 1 {
		return CurrentSession{}, ErrSessionExpired
	}
	if record.LastSeenAtMS <= now.Add(-sessionTouchInterval).UnixMilli() {
		touched, err := service.repository.TouchSession(
			ctx,
			TouchSessionParams{
				SessionID:       record.ID,
				TokenHash:       tokenHash,
				ExpectedVersion: record.AuthVersion,
				UsedAtMS:        now.UnixMilli(),
			},
		)
		if err != nil {
			return CurrentSession{}, fmt.Errorf("touch administrator session: %w", err)
		}
		if !touched {
			return CurrentSession{}, ErrSessionExpired
		}
	}
	return CurrentSession{
		Administrator: record.Administrator,
		CSRFToken:     csrfToken,
		ExpiresAtMS:   record.ExpiresAtMS,
	}, nil
}

func (service *Service) Logout(ctx context.Context, cookieToken string) error {
	if ctx == nil {
		return errors.New("authentication context is nil")
	}
	tokenHash, _, _, err := parseSessionCookie(cookieToken)
	if err != nil {
		return ErrAuthenticationRequired
	}
	record, err := service.repository.FindSession(ctx, tokenHash)
	if errors.Is(err, ErrAuthenticationRequired) {
		return ErrAuthenticationRequired
	}
	if err != nil {
		return fmt.Errorf("find administrator session: %w", err)
	}
	nowMS := service.now().UTC().UnixMilli()
	if !sessionActive(record, nowMS) {
		return ErrSessionExpired
	}
	revoked, err := service.repository.RevokeSession(
		ctx,
		RevokeSessionParams{
			SessionID:   record.ID,
			TokenHash:   tokenHash,
			RevokedAtMS: nowMS,
		},
	)
	if err != nil {
		return fmt.Errorf("revoke administrator session: %w", err)
	}
	if !revoked {
		return ErrSessionExpired
	}
	return nil
}

func newCreateSessionParams(
	userID int64,
	authVersion int64,
	secrets sessionSecrets,
	now time.Time,
) CreateSessionParams {
	return CreateSessionParams{
		UserID:        userID,
		AuthVersion:   authVersion,
		TokenHash:     secrets.tokenHash,
		CSRFTokenHash: secrets.csrfHash,
		CreatedAtMS:   now.UnixMilli(),
		ExpiresAtMS:   now.Add(SessionLifetime).UnixMilli(),
	}
}

func establishedSession(
	administrator Administrator,
	session StoredSession,
	secrets sessionSecrets,
) EstablishedSession {
	return EstablishedSession{
		Administrator: administrator,
		CookieToken:   secrets.cookieToken,
		CSRFToken:     secrets.csrfToken,
		ExpiresAtMS:   session.ExpiresAtMS,
	}
}

func sessionActive(session StoredSession, nowMS int64) bool {
	return session.RevokedAtMS == nil &&
		!session.UserDisabled &&
		session.AuthVersion == session.UserAuthVersion &&
		session.ExpiresAtMS > nowMS
}

func issueSessionSecrets(random io.Reader) (sessionSecrets, error) {
	raw := make([]byte, sessionCookieBytes)
	if _, err := io.ReadFull(random, raw); err != nil {
		return sessionSecrets{}, fmt.Errorf("read session secrets: %w", err)
	}
	if subtle.ConstantTimeCompare(raw[:sessionSecretBytes], raw[sessionSecretBytes:]) == 1 {
		clear(raw)
		return sessionSecrets{}, errors.New("session secrets are not independent")
	}
	tokenHash := sha256.Sum256(raw)
	csrfHash := sha256.Sum256(raw[sessionSecretBytes:])
	cookieToken := base64.RawURLEncoding.EncodeToString(raw)
	csrfToken := base64.RawURLEncoding.EncodeToString(raw[sessionSecretBytes:])
	clear(raw)
	return sessionSecrets{
		cookieToken: cookieToken,
		csrfToken:   csrfToken,
		tokenHash:   tokenHash,
		csrfHash:    csrfHash,
	}, nil
}

func parseSessionCookie(
	encoded string,
) ([sessionDigestBytes]byte, string, [sessionDigestBytes]byte, error) {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil || len(raw) != sessionCookieBytes {
		return [sessionDigestBytes]byte{}, "", [sessionDigestBytes]byte{},
			errors.New("invalid session secret")
	}
	tokenHash := sha256.Sum256(raw)
	csrfHash := sha256.Sum256(raw[sessionSecretBytes:])
	csrfToken := base64.RawURLEncoding.EncodeToString(raw[sessionSecretBytes:])
	clear(raw)
	return tokenHash, csrfToken, csrfHash, nil
}
