// Package cursor owns the authenticated opaque-token mechanism. Resource
// capabilities still own cursor payloads, query binding, and validation.
package cursor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var ErrInvalid = errors.New("invalid cursor token")

type Codec struct{ aead cipher.AEAD }

func New(key []byte) (*Codec, error) {
	key = append([]byte(nil), key...)
	if len(key) == 0 {
		key = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("generate cursor key: %w", err)
		}
	}
	if len(key) != 32 {
		return nil, errors.New("cursor key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("construct cursor cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("construct cursor AEAD: %w", err)
	}
	return &Codec{aead: aead}, nil
}

func (codec *Codec) Encode(value any, associatedData string) (string, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode cursor payload: %w", err)
	}
	nonce := make([]byte, codec.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate cursor nonce: %w", err)
	}
	sealed := codec.aead.Seal(nonce, nonce, plaintext, []byte(associatedData))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (codec *Codec) Decode(encoded, associatedData string, target any) error {
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(sealed) <= codec.aead.NonceSize() {
		return ErrInvalid
	}
	if base64.RawURLEncoding.EncodeToString(sealed) != encoded {
		return ErrInvalid
	}
	nonce := sealed[:codec.aead.NonceSize()]
	plaintext, err := codec.aead.Open(
		nil, nonce, sealed[codec.aead.NonceSize():], []byte(associatedData),
	)
	if err != nil || json.Unmarshal(plaintext, target) != nil {
		return ErrInvalid
	}
	return nil
}
