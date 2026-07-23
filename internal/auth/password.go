package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2idScheme      = "argon2id"
	argon2idVersion     = argon2.Version
	argon2idMemoryKiB   = 64 * 1024
	argon2idIterations  = 3
	argon2idParallelism = 4
	argon2idSaltBytes   = 16
	argon2idKeyBytes    = 32
)

var (
	errInvalidPasswordVerifier = errors.New("invalid password verifier")
	argon2idParameters         = fmt.Sprintf(
		"v=%d,m=%d,t=%d,p=%d",
		argon2idVersion,
		argon2idMemoryKiB,
		argon2idIterations,
		argon2idParallelism,
	)
	dummyPasswordVerifier = PasswordVerifier{
		EncodedHash: "$argon2id$v=19$m=65536,t=3,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Scheme:      argon2idScheme,
		Parameters:  argon2idParameters,
	}
)

type Argon2idPasswordManager struct {
	random io.Reader
}

func NewArgon2idPasswordManager(random io.Reader) *Argon2idPasswordManager {
	if random == nil {
		random = rand.Reader
	}
	return &Argon2idPasswordManager{random: random}
}

func (manager *Argon2idPasswordManager) Hash(
	ctx context.Context,
	password string,
) (PasswordVerifier, error) {
	if ctx == nil {
		return PasswordVerifier{}, errors.New("password context is nil")
	}
	if ctx.Err() != nil {
		return PasswordVerifier{}, ctx.Err()
	}

	salt := make([]byte, argon2idSaltBytes)
	if _, err := io.ReadFull(manager.random, salt); err != nil {
		return PasswordVerifier{}, fmt.Errorf("read password salt: %w", err)
	}
	key := deriveArgon2id(password, salt)
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idVersion,
		argon2idMemoryKiB,
		argon2idIterations,
		argon2idParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	clear(key)

	if ctx.Err() != nil {
		return PasswordVerifier{}, ctx.Err()
	}
	return PasswordVerifier{
		EncodedHash: encoded,
		Scheme:      argon2idScheme,
		Parameters:  argon2idParameters,
	}, nil
}

func (manager *Argon2idPasswordManager) Verify(
	ctx context.Context,
	password string,
	verifier PasswordVerifier,
) (bool, error) {
	if ctx == nil {
		return false, errors.New("password context is nil")
	}
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	salt, expected, err := parseArgon2idVerifier(verifier)
	if err != nil {
		return false, err
	}
	candidate := deriveArgon2id(password, salt)
	matches := subtle.ConstantTimeCompare(candidate, expected) == 1
	clear(candidate)
	clear(expected)
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	return matches, nil
}

func deriveArgon2id(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		argon2idIterations,
		argon2idMemoryKiB,
		argon2idParallelism,
		argon2idKeyBytes,
	)
}

func parseArgon2idVerifier(verifier PasswordVerifier) ([]byte, []byte, error) {
	if verifier.Scheme != argon2idScheme || verifier.Parameters != argon2idParameters {
		return nil, nil, errInvalidPasswordVerifier
	}
	parts := strings.Split(verifier.EncodedHash, "$")
	if len(parts) != 6 ||
		parts[0] != "" ||
		parts[1] != argon2idScheme ||
		parts[2] != fmt.Sprintf("v=%d", argon2idVersion) ||
		parts[3] != fmt.Sprintf(
			"m=%d,t=%d,p=%d",
			argon2idMemoryKiB,
			argon2idIterations,
			argon2idParallelism,
		) {
		return nil, nil, errInvalidPasswordVerifier
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) != argon2idSaltBytes {
		return nil, nil, errInvalidPasswordVerifier
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(key) != argon2idKeyBytes {
		return nil, nil, errInvalidPasswordVerifier
	}
	return salt, key, nil
}
