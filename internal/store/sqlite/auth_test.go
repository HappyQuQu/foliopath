package sqlite

import (
	"context"
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
	administrator, err := store.CreateAdministrator(ctx, params)
	if err != nil {
		t.Fatalf("CreateAdministrator() error = %v", err)
	}
	if administrator.ID <= 0 ||
		administrator.Username != params.Username ||
		administrator.DisplayName != params.DisplayName ||
		administrator.CreatedAtMS != now.UnixMilli() ||
		administrator.UpdatedAtMS != now.UnixMilli() {
		t.Fatalf("administrator = %#v", administrator)
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
	if _, err := store.CreateAdministrator(ctx, params); !errors.Is(err, auth.ErrSetupClosed) {
		t.Fatalf("second CreateAdministrator() error = %v, want ErrSetupClosed", err)
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
