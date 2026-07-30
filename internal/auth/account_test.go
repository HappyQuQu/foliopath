package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

type accountTestRepository struct {
	sessionRepository
	account     Account
	update      UpdateAccountParams
	change      ChangePasswordParams
	updateErr   error
	changeErr   error
	changeCalls int
}

func (repository *accountTestRepository) GetAccount(context.Context, int64) (Account, error) {
	return repository.account, nil
}

func (repository *accountTestRepository) UpdateAccount(
	_ context.Context,
	params UpdateAccountParams,
) (Account, error) {
	repository.update = params
	return repository.account, repository.updateErr
}

func (repository *accountTestRepository) ChangePassword(
	_ context.Context,
	params ChangePasswordParams,
) (Account, error) {
	repository.changeCalls++
	repository.change = params
	return repository.account, repository.changeErr
}

type accountPasswordManager struct {
	matches bool
	hashErr error
}

func (manager *accountPasswordManager) Hash(
	context.Context,
	string,
) (PasswordVerifier, error) {
	if manager.hashErr != nil {
		return PasswordVerifier{}, manager.hashErr
	}
	return PasswordVerifier{
		EncodedHash: "replacement-verifier",
		Scheme:      "test",
		Parameters:  "v=2",
	}, nil
}

func (manager *accountPasswordManager) Verify(
	context.Context,
	string,
	PasswordVerifier,
) (bool, error) {
	return manager.matches, nil
}

func TestAccountServiceNormalizesProfileAndFailsClosedDuringPasswordChange(t *testing.T) {
	now := time.UnixMilli(1_700_000_000_000)
	raw := append(
		bytes.Repeat([]byte{0x42}, sessionSecretBytes),
		bytes.Repeat([]byte{0x43}, sessionSecretBytes)...,
	)
	secrets, err := issueSessionSecrets(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	administrator := Administrator{
		ID: 1, Username: "Administrator", DisplayName: "Admin",
	}
	repository := &accountTestRepository{
		sessionRepository: sessionRepository{
			credential: AdministratorCredential{
				Administrator: administrator,
				UsernameKey:   "administrator",
				PasswordVerifier: PasswordVerifier{
					EncodedHash: "current-verifier",
					Scheme:      "test",
					Parameters:  "v=1",
				},
				AuthVersion: 1,
			},
			session: StoredSession{
				ID: 9, Administrator: administrator,
				AuthVersion: 1, UserAuthVersion: 1,
				CreatedAtMS:   now.Add(-time.Hour).UnixMilli(),
				LastSeenAtMS:  now.UnixMilli(),
				ExpiresAtMS:   now.Add(time.Hour).UnixMilli(),
				CSRFTokenHash: secrets.csrfHash,
			},
		},
		account: Account{
			ID: 1, Username: "Administrator", DisplayName: "Home Admin",
			Revision: 2, UpdatedAtMS: now.UnixMilli(),
		},
	}
	passwords := &accountPasswordManager{}
	service, err := NewService(repository, passwords, ServiceOptions{
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.UpdateProfile(context.Background(), secrets.cookieToken, ProfileUpdate{
		DisplayName: "  Ho\u0301me Admin  ", ExpectedRevision: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if repository.update.DisplayName != "Hóme Admin" ||
		repository.update.ExpectedRevision != 1 {
		t.Fatalf("profile update = %#v", repository.update)
	}

	change := PasswordChange{
		CurrentPassword:  "wrong password",
		NewPassword:      "replacement password",
		ExpectedRevision: 2,
	}
	if _, err := service.ChangeAccountPassword(
		context.Background(), secrets.cookieToken, change,
	); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong current password error = %v", err)
	}
	if repository.changeCalls != 0 {
		t.Fatal("wrong password reached the transaction boundary")
	}

	passwords.matches = true
	passwords.hashErr = errors.New("injected hash failure")
	if _, err := service.ChangeAccountPassword(
		context.Background(), secrets.cookieToken, change,
	); err == nil {
		t.Fatal("hash failure unexpectedly succeeded")
	}
	if repository.changeCalls != 0 {
		t.Fatal("hash failure reached the transaction boundary")
	}

	passwords.hashErr = nil
	repository.changeErr = ErrSessionExpired
	if _, err := service.ChangeAccountPassword(
		context.Background(), secrets.cookieToken, change,
	); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("session race error = %v", err)
	}
	if repository.changeCalls != 1 ||
		repository.change.CurrentSessionID != 9 ||
		repository.change.ExpectedAuthVersion != 1 {
		t.Fatalf("password transaction params = %#v", repository.change)
	}
}
