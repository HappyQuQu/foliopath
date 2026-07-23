package auth

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestArgon2idPasswordManagerHashesAndVerifies(t *testing.T) {
	salt := bytes.Repeat([]byte{0x7a}, argon2idSaltBytes)
	manager := NewArgon2idPasswordManager(bytes.NewReader(salt))
	password := "correct horse battery staple"

	verifier, err := manager.Hash(context.Background(), password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if verifier.Scheme != argon2idScheme || verifier.Parameters != argon2idParameters {
		t.Fatalf("verifier metadata = %#v", verifier)
	}
	if strings.Contains(verifier.EncodedHash, password) {
		t.Fatal("encoded verifier contains the plaintext password")
	}

	matches, err := manager.Verify(context.Background(), password, verifier)
	if err != nil {
		t.Fatalf("Verify(correct) error = %v", err)
	}
	if !matches {
		t.Fatal("Verify(correct) = false")
	}
	matches, err = manager.Verify(context.Background(), "incorrect password", verifier)
	if err != nil {
		t.Fatalf("Verify(incorrect) error = %v", err)
	}
	if matches {
		t.Fatal("Verify(incorrect) = true")
	}
}

func TestArgon2idPasswordManagerRejectsUnknownOrMalformedVerifier(t *testing.T) {
	manager := NewArgon2idPasswordManager(bytes.NewReader(
		bytes.Repeat([]byte{0x42}, argon2idSaltBytes),
	))
	verifier, err := manager.Hash(context.Background(), "correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	tests := []PasswordVerifier{
		{EncodedHash: verifier.EncodedHash, Scheme: "argon2i", Parameters: verifier.Parameters},
		{EncodedHash: verifier.EncodedHash, Scheme: verifier.Scheme, Parameters: "v=19,m=1,t=1,p=1"},
		{EncodedHash: "$argon2id$v=19$m=1,t=1,p=1$bad$bad", Scheme: verifier.Scheme, Parameters: verifier.Parameters},
		{EncodedHash: verifier.EncodedHash + "$extra", Scheme: verifier.Scheme, Parameters: verifier.Parameters},
	}
	for _, candidate := range tests {
		if _, err := manager.Verify(
			context.Background(),
			"correct horse battery staple",
			candidate,
		); !errors.Is(err, errInvalidPasswordVerifier) {
			t.Errorf("Verify(%#v) error = %v, want invalid verifier", candidate, err)
		}
	}
}

func TestArgon2idPasswordManagerHonorsPreCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	manager := NewArgon2idPasswordManager(bytes.NewReader(
		bytes.Repeat([]byte{0x42}, argon2idSaltBytes),
	))

	if _, err := manager.Hash(ctx, "correct horse battery staple"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Hash(cancelled) error = %v, want context.Canceled", err)
	}
	if _, err := manager.Verify(ctx, "password", PasswordVerifier{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Verify(cancelled) error = %v, want context.Canceled", err)
	}
}
