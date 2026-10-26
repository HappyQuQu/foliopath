package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	signedCatalogSchemaVersion = 1
	maxSignedCatalogEnvelope   = 6 << 20
	maxSignedCatalogPayload    = 4 << 20
)

var (
	errCatalogSignature  = errors.New("catalog signature verification failed")
	errCatalogValidity   = errors.New("catalog is outside its validity window")
	errCatalogRollback   = errors.New("catalog sequence would roll back trusted state")
	errCatalogEquivocate = errors.New("catalog sequence was reused for different content")
)

type signedCatalogEnvelope struct {
	SchemaVersion int    `json:"schema_version"`
	KeyID         string `json:"key_id"`
	Sequence      uint64 `json:"sequence"`
	IssuedAt      int64  `json:"issued_at"`
	ExpiresAt     int64  `json:"expires_at"`
	Payload       string `json:"payload"`
	Signature     string `json:"signature"`
}

type catalogCheckpoint struct {
	Sequence      uint64
	PayloadSHA256 string
}

type verifiedCatalog struct {
	KeyID         string
	Sequence      uint64
	IssuedAt      time.Time
	ExpiresAt     time.Time
	Payload       []byte
	PayloadSHA256 string
	authenticated bool
}

// verifySignedCatalog authenticates an exact payload and applies monotonic
// sequence/equivocation checks against the last durable checkpoint. Callers
// must validate the returned catalog schema before using any artifact entry.
func verifySignedCatalog(
	encoded []byte,
	trustedKeys map[string]ed25519.PublicKey,
	now time.Time,
	checkpoint catalogCheckpoint,
) (verifiedCatalog, error) {
	if len(encoded) == 0 || len(encoded) > maxSignedCatalogEnvelope {
		return verifiedCatalog{}, errors.New("signed catalog envelope exceeds its size boundary")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope signedCatalogEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return verifiedCatalog{}, fmt.Errorf("decode signed catalog: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return verifiedCatalog{}, err
	}
	if envelope.SchemaVersion != signedCatalogSchemaVersion || !packageSegmentPattern.MatchString(envelope.KeyID) ||
		envelope.Sequence == 0 || envelope.IssuedAt <= 0 || envelope.ExpiresAt <= envelope.IssuedAt {
		return verifiedCatalog{}, errors.New("signed catalog metadata is invalid")
	}
	publicKey, exists := trustedKeys[envelope.KeyID]
	if !exists || len(publicKey) != ed25519.PublicKeySize {
		return verifiedCatalog{}, errCatalogSignature
	}
	payload, err := base64.StdEncoding.Strict().DecodeString(envelope.Payload)
	if err != nil || len(payload) == 0 || len(payload) > maxSignedCatalogPayload || !json.Valid(payload) {
		return verifiedCatalog{}, errors.New("signed catalog payload is invalid")
	}
	signature, err := base64.StdEncoding.Strict().DecodeString(envelope.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return verifiedCatalog{}, errCatalogSignature
	}
	message, err := catalogSignatureMessage(envelope, payload)
	if err != nil || !ed25519.Verify(publicKey, message, signature) {
		return verifiedCatalog{}, errCatalogSignature
	}
	issuedAt := time.Unix(envelope.IssuedAt, 0).UTC()
	expiresAt := time.Unix(envelope.ExpiresAt, 0).UTC()
	now = now.UTC()
	if now.Before(issuedAt) || !now.Before(expiresAt) {
		return verifiedCatalog{}, errCatalogValidity
	}
	digestBytes := sha256.Sum256(payload)
	digest := hex.EncodeToString(digestBytes[:])
	if envelope.Sequence < checkpoint.Sequence {
		return verifiedCatalog{}, errCatalogRollback
	}
	if envelope.Sequence == checkpoint.Sequence && checkpoint.Sequence != 0 && digest != checkpoint.PayloadSHA256 {
		return verifiedCatalog{}, errCatalogEquivocate
	}
	return verifiedCatalog{
		KeyID: envelope.KeyID, Sequence: envelope.Sequence,
		IssuedAt: issuedAt, ExpiresAt: expiresAt,
		Payload: payload, PayloadSHA256: digest,
		authenticated: true,
	}, nil
}

func catalogSignatureMessage(envelope signedCatalogEnvelope, payload []byte) ([]byte, error) {
	if len(envelope.KeyID) > 255 || len(payload) > maxSignedCatalogPayload {
		return nil, errors.New("catalog signature input exceeds its boundary")
	}
	var message bytes.Buffer
	message.WriteString("FolioPath signed model catalog\x00")
	if err := binary.Write(&message, binary.BigEndian, uint16(envelope.SchemaVersion)); err != nil {
		return nil, err
	}
	message.WriteByte(byte(len(envelope.KeyID)))
	message.WriteString(envelope.KeyID)
	for _, value := range []uint64{envelope.Sequence, uint64(envelope.IssuedAt), uint64(envelope.ExpiresAt), uint64(len(payload))} {
		if err := binary.Write(&message, binary.BigEndian, value); err != nil {
			return nil, err
		}
	}
	message.Write(payload)
	return message.Bytes(), nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("signed catalog contains trailing JSON")
		}
		return fmt.Errorf("decode signed catalog trailing data: %w", err)
	}
	return nil
}
