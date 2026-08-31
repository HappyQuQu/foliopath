package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestVerifySignedCatalogSecurityMatrix(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	otherPublic, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0).UTC()
	payload := []byte(`{"schema_version":2,"models":[]}`)
	valid := signCatalogForTest(t, privateKey, signedCatalogEnvelope{
		SchemaVersion: 1, KeyID: "release-2026", Sequence: 7,
		IssuedAt: now.Add(-time.Hour).Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
	}, payload)
	keys := map[string]ed25519.PublicKey{"release-2026": publicKey}

	verified, err := verifySignedCatalog(valid, keys, now, catalogCheckpoint{})
	if err != nil {
		t.Fatal(err)
	}
	if verified.Sequence != 7 || string(verified.Payload) != string(payload) || verified.PayloadSHA256 == "" {
		t.Fatalf("unexpected verified catalog: %+v", verified)
	}
	if _, err := verifySignedCatalog(valid, keys, now, catalogCheckpoint{Sequence: 7, PayloadSHA256: verified.PayloadSHA256}); err != nil {
		t.Fatalf("idempotent checkpoint rejected: %v", err)
	}

	t.Run("unknown and wrong keys", func(t *testing.T) {
		if _, err := verifySignedCatalog(valid, map[string]ed25519.PublicKey{}, now, catalogCheckpoint{}); !errors.Is(err, errCatalogSignature) {
			t.Fatalf("unknown key: %v", err)
		}
		if _, err := verifySignedCatalog(valid, map[string]ed25519.PublicKey{"release-2026": otherPublic}, now, catalogCheckpoint{}); !errors.Is(err, errCatalogSignature) {
			t.Fatalf("wrong key: %v", err)
		}
	})

	t.Run("payload and metadata tampering", func(t *testing.T) {
		var envelope signedCatalogEnvelope
		if err := json.Unmarshal(valid, &envelope); err != nil {
			t.Fatal(err)
		}
		envelope.Payload = base64.StdEncoding.EncodeToString([]byte(`{"schema_version":2,"models":[{}]}`))
		tampered, _ := json.Marshal(envelope)
		if _, err := verifySignedCatalog(tampered, keys, now, catalogCheckpoint{}); !errors.Is(err, errCatalogSignature) {
			t.Fatalf("payload tamper: %v", err)
		}
		envelope = signedCatalogEnvelope{}
		if err := json.Unmarshal(valid, &envelope); err != nil {
			t.Fatal(err)
		}
		envelope.Sequence++
		tampered, _ = json.Marshal(envelope)
		if _, err := verifySignedCatalog(tampered, keys, now, catalogCheckpoint{}); !errors.Is(err, errCatalogSignature) {
			t.Fatalf("metadata tamper: %v", err)
		}
	})

	t.Run("validity window", func(t *testing.T) {
		if _, err := verifySignedCatalog(valid, keys, now.Add(-2*time.Hour), catalogCheckpoint{}); !errors.Is(err, errCatalogValidity) {
			t.Fatalf("future catalog: %v", err)
		}
		if _, err := verifySignedCatalog(valid, keys, now.Add(2*time.Hour), catalogCheckpoint{}); !errors.Is(err, errCatalogValidity) {
			t.Fatalf("expired catalog: %v", err)
		}
	})

	t.Run("rollback and equivocation", func(t *testing.T) {
		if _, err := verifySignedCatalog(valid, keys, now, catalogCheckpoint{Sequence: 8}); !errors.Is(err, errCatalogRollback) {
			t.Fatalf("rollback: %v", err)
		}
		if _, err := verifySignedCatalog(valid, keys, now, catalogCheckpoint{Sequence: 7, PayloadSHA256: strings.Repeat("0", 64)}); !errors.Is(err, errCatalogEquivocate) {
			t.Fatalf("equivocation: %v", err)
		}
	})

	t.Run("strict envelope and size boundary", func(t *testing.T) {
		withUnknown := append(valid[:len(valid)-1], []byte(`,"unknown":true}`)...)
		if _, err := verifySignedCatalog(withUnknown, keys, now, catalogCheckpoint{}); err == nil {
			t.Fatal("unknown envelope field accepted")
		}
		withTrailing := append(append([]byte(nil), valid...), []byte(` {}`)...)
		if _, err := verifySignedCatalog(withTrailing, keys, now, catalogCheckpoint{}); err == nil {
			t.Fatal("trailing JSON accepted")
		}
		oversized := make([]byte, maxSignedCatalogEnvelope+1)
		if _, err := verifySignedCatalog(oversized, keys, now, catalogCheckpoint{}); err == nil {
			t.Fatal("oversized envelope accepted")
		}
		invalidPayload := signCatalogForTest(t, privateKey, signedCatalogEnvelope{
			SchemaVersion: 1, KeyID: "release-2026", Sequence: 8,
			IssuedAt: now.Add(-time.Hour).Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
		}, []byte("not-json"))
		if _, err := verifySignedCatalog(invalidPayload, keys, now, catalogCheckpoint{}); err == nil {
			t.Fatal("signed non-JSON payload accepted")
		}
	})
}

func signCatalogForTest(t *testing.T, privateKey ed25519.PrivateKey, envelope signedCatalogEnvelope, payload []byte) []byte {
	t.Helper()
	envelope.Payload = base64.StdEncoding.EncodeToString(payload)
	message, err := catalogSignatureMessage(envelope, payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, message))
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
