package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

func TestAuthenticationRepositoryPersistsAdministratorAtomically(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	store, _ := openTestStoreWithOptions(t, Options{Now: func() time.Time { return now }})
	ctx := context.Background()

	initialized, err := store.AdministratorInitialized(ctx)
	if err != nil {
		t.Fatalf("AdministratorInitialized() error = %v", err)
	}
	if initialized {
		t.Fatal("new authentication repository is initialized")
	}

	params := auth.CreateAdministratorParams{
		Username:    "Administrator",
		UsernameKey: "administrator",
		DisplayName: "Home Admin",
		PasswordVerifier: auth.PasswordVerifier{
			EncodedHash: "$argon2id$encoded",
			Scheme:      "argon2id",
			Parameters:  "v=19,m=65536,t=3,p=4",
		},
	}
	sessionParams := auth.CreateSessionParams{
		AuthVersion:   1,
		TokenHash:     [32]byte{0x11},
		CSRFTokenHash: [32]byte{0x22},
		CreatedAtMS:   now.UnixMilli(),
		ExpiresAtMS:   now.Add(auth.SessionLifetime).UnixMilli(),
	}
	administrator, session, err := store.CreateAdministratorWithSession(
		ctx,
		params,
		sessionParams,
	)
	if err != nil {
		t.Fatalf("CreateAdministratorWithSession() error = %v", err)
	}
	if administrator.ID <= 0 ||
		administrator.Username != params.Username ||
		administrator.DisplayName != params.DisplayName ||
		administrator.CreatedAtMS != now.UnixMilli() ||
		administrator.UpdatedAtMS != now.UnixMilli() {
		t.Fatalf("administrator = %#v", administrator)
	}
	if session.ID <= 0 || session.ExpiresAtMS != sessionParams.ExpiresAtMS {
		t.Fatalf("session = %#v", session)
	}
	var sessionCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&sessionCount); err != nil {
		t.Fatalf("count initial sessions: %v", err)
	}
	if sessionCount != 1 {
		t.Fatalf("initial session count = %d, want 1", sessionCount)
	}

	var (
		usernameKey        string
		passwordHash       string
		passwordScheme     string
		passwordParameters string
		authVersion        int64
	)
	if err := store.db.QueryRowContext(
		ctx,
		`SELECT username_key, password_hash, password_scheme,
		        password_parameters, auth_version
		 FROM users WHERE singleton_key = 1`,
	).Scan(
		&usernameKey,
		&passwordHash,
		&passwordScheme,
		&passwordParameters,
		&authVersion,
	); err != nil {
		t.Fatalf("inspect administrator verifier: %v", err)
	}
	if usernameKey != params.UsernameKey ||
		passwordHash != params.PasswordVerifier.EncodedHash ||
		passwordScheme != params.PasswordVerifier.Scheme ||
		passwordParameters != params.PasswordVerifier.Parameters ||
		authVersion != 1 {
		t.Fatalf(
			"stored authentication record = (%q, %q, %q, %q, %d)",
			usernameKey,
			passwordHash,
			passwordScheme,
			passwordParameters,
			authVersion,
		)
	}

	initialized, err = store.AdministratorInitialized(ctx)
	if err != nil || !initialized {
		t.Fatalf("completed AdministratorInitialized() = %t, %v", initialized, err)
	}
	if _, _, err := store.CreateAdministratorWithSession(
		ctx,
		params,
		sessionParams,
	); !errors.Is(err, auth.ErrSetupClosed) {
		t.Fatalf("second CreateAdministratorWithSession() error = %v, want ErrSetupClosed", err)
	}
}

func TestAuthenticationRepositoryManagesSessionLifecycle(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	store, _ := openTestStoreWithOptions(t, Options{})
	ctx := context.Background()
	administrator, _, err := store.CreateAdministratorWithSession(
		ctx,
		testAdministratorParams(),
		testSessionParams(0, 0x11, now, now.Add(auth.SessionLifetime)),
	)
	if err != nil {
		t.Fatalf("create administrator: %v", err)
	}

	credential, err := store.FindAdministratorCredential(ctx, "administrator")
	if err != nil {
		t.Fatalf("FindAdministratorCredential() error = %v", err)
	}
	if credential.ID != administrator.ID ||
		credential.AuthVersion != 1 ||
		credential.Disabled ||
		credential.PasswordVerifier != testAdministratorParams().PasswordVerifier {
		t.Fatalf("credential = %#v", credential)
	}
	if _, err := store.FindAdministratorCredential(
		ctx,
		"missing",
	); !errors.Is(err, auth.ErrAdministratorNotFound) {
		t.Fatalf("missing credential error = %v", err)
	}

	oldSession := testSessionParams(
		administrator.ID,
		0x31,
		now.Add(-72*time.Hour),
		now.Add(-48*time.Hour),
	)
	if _, err := store.CreateSession(
		ctx,
		oldSession,
		now.Add(-96*time.Hour).UnixMilli(),
	); err != nil {
		t.Fatalf("create old session: %v", err)
	}
	activeParams := testSessionParams(
		administrator.ID,
		0x41,
		now,
		now.Add(auth.SessionLifetime),
	)
	active, err := store.CreateSession(
		ctx,
		activeParams,
		now.Add(-24*time.Hour).UnixMilli(),
	)
	if err != nil {
		t.Fatalf("create active session: %v", err)
	}

	record, err := store.FindSession(ctx, activeParams.TokenHash)
	if err != nil {
		t.Fatalf("FindSession() error = %v", err)
	}
	if record.ID != active.ID ||
		record.Administrator.ID != administrator.ID ||
		record.AuthVersion != 1 ||
		record.UserAuthVersion != 1 ||
		record.RevokedAtMS != nil {
		t.Fatalf("session record = %#v", record)
	}

	usedAt := now.Add(time.Minute).UnixMilli()
	touched, err := store.TouchSession(ctx, auth.TouchSessionParams{
		SessionID:       active.ID,
		TokenHash:       activeParams.TokenHash,
		ExpectedVersion: 1,
		UsedAtMS:        usedAt,
	})
	if err != nil || !touched {
		t.Fatalf("TouchSession() = %t, %v", touched, err)
	}
	var (
		storedCSRF []byte
		lastSeen   int64
	)
	if err := store.db.QueryRowContext(
		ctx,
		`SELECT csrf_token_hash, last_seen_at_ms FROM sessions WHERE id = ?`,
		active.ID,
	).Scan(&storedCSRF, &lastSeen); err != nil {
		t.Fatalf("inspect touched session: %v", err)
	}
	if string(storedCSRF) != string(activeParams.CSRFTokenHash[:]) || lastSeen != usedAt {
		t.Fatal("touch changed CSRF or did not persist last-seen time")
	}

	revokedAt := now.Add(2 * time.Minute).UnixMilli()
	revoked, err := store.RevokeSession(ctx, auth.RevokeSessionParams{
		SessionID:   active.ID,
		TokenHash:   activeParams.TokenHash,
		RevokedAtMS: revokedAt,
	})
	if err != nil || !revoked {
		t.Fatalf("RevokeSession() = %t, %v", revoked, err)
	}
	record, err = store.FindSession(ctx, activeParams.TokenHash)
	if err != nil || record.RevokedAtMS == nil || *record.RevokedAtMS != revokedAt {
		t.Fatalf("revoked session = %#v, %v", record, err)
	}
	if revoked, err := store.RevokeSession(ctx, auth.RevokeSessionParams{
		SessionID:   active.ID,
		TokenHash:   activeParams.TokenHash,
		RevokedAtMS: revokedAt,
	}); err != nil || revoked {
		t.Fatalf("second RevokeSession() = %t, %v; want false", revoked, err)
	}

	var obsoleteCount int
	if err := store.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sessions WHERE token_hash = ?`,
		oldSession.TokenHash[:],
	).Scan(&obsoleteCount); err != nil {
		t.Fatalf("inspect obsolete session: %v", err)
	}
	if obsoleteCount != 0 {
		t.Fatal("obsolete expired session was not cleaned up")
	}
}

func TestAuthenticationRepositoryRollsBackAdministratorWhenInitialSessionFails(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	store, _ := openTestStoreWithOptions(t, Options{})
	invalidSession := testSessionParams(0, 0x51, now, now)

	if _, _, err := store.CreateAdministratorWithSession(
		context.Background(),
		testAdministratorParams(),
		invalidSession,
	); err == nil {
		t.Fatal("CreateAdministratorWithSession() unexpectedly succeeded")
	}
	initialized, err := store.AdministratorInitialized(context.Background())
	if err != nil {
		t.Fatalf("AdministratorInitialized() error = %v", err)
	}
	if initialized {
		t.Fatal("failed initial session left an administrator behind")
	}
}

func TestTouchSessionQueuesBehindExistingWriteTransaction(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	store, _ := openTestStoreWithOptions(t, Options{BusyTimeout: 100 * time.Millisecond})
	_, session, err := store.CreateAdministratorWithSession(
		context.Background(),
		testAdministratorParams(),
		testSessionParams(0, 0x61, now, now.Add(auth.SessionLifetime)),
	)
	if err != nil {
		t.Fatalf("create administrator: %v", err)
	}

	writeStarted := make(chan struct{})
	releaseWrite := make(chan struct{})
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- store.withWriteTx(context.Background(), func(*sql.Tx) error {
			close(writeStarted)
			<-releaseWrite
			return nil
		})
	}()
	<-writeStarted

	touchDone := make(chan error, 1)
	go func() {
		touched, touchErr := store.TouchSession(context.Background(), auth.TouchSessionParams{
			SessionID:       session.ID,
			TokenHash:       [32]byte{0x61},
			ExpectedVersion: 1,
			UsedAtMS:        now.Add(time.Minute).UnixMilli(),
		})
		if touchErr == nil && !touched {
			touchErr = errors.New("session was not touched")
		}
		touchDone <- touchErr
	}()

	select {
	case touchErr := <-touchDone:
		close(releaseWrite)
		t.Fatalf("TouchSession returned before write gate released: %v", touchErr)
	case <-time.After(150 * time.Millisecond):
	}
	close(releaseWrite)
	if err := <-writeDone; err != nil {
		t.Fatalf("finish held write: %v", err)
	}
	if err := <-touchDone; err != nil {
		t.Fatalf("TouchSession after write gate release: %v", err)
	}
}

func testAdministratorParams() auth.CreateAdministratorParams {
	return auth.CreateAdministratorParams{
		Username:    "Administrator",
		UsernameKey: "administrator",
		DisplayName: "Home Admin",
		PasswordVerifier: auth.PasswordVerifier{
			EncodedHash: "$argon2id$encoded",
			Scheme:      "argon2id",
			Parameters:  "v=19,m=65536,t=3,p=4",
		},
	}
}

func testSessionParams(
	userID int64,
	value byte,
	createdAt time.Time,
	expiresAt time.Time,
) auth.CreateSessionParams {
	return auth.CreateSessionParams{
		UserID:        userID,
		AuthVersion:   1,
		TokenHash:     [32]byte{value},
		CSRFTokenHash: [32]byte{value + 1},
		CreatedAtMS:   createdAt.UnixMilli(),
		ExpiresAtMS:   expiresAt.UnixMilli(),
	}
}

func openTestStoreWithOptions(t *testing.T, options Options) (*Store, string) {
	t.Helper()
	filename := t.TempDir() + "/foliopath-test.db"
	store, err := Open(context.Background(), filename, options)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return store, filename
}
