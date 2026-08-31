package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestActivationStoreAtomicCheckpointAndRecovery(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "activation.db")
	store, err := openActivationStore(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	modelID := "semantic-siglip"
	digestOne := digestText("package-one")
	digestTwo := digestText("package-two")
	if err := store.recordPublishedGeneration(ctx, modelID, "generation-1", digestOne); err != nil {
		t.Fatal(err)
	}
	if err := store.recordPublishedGeneration(ctx, modelID, "generation-2", digestTwo); err != nil {
		t.Fatal(err)
	}
	if err := store.recordPublishedGeneration(ctx, modelID, "generation-1", digestText("different")); !errors.Is(err, errCatalogEquivocate) {
		t.Fatalf("immutable generation replacement was not rejected: %v", err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0).UTC()
	verifiedForSequence := func(sequence uint64, name string) verifiedCatalog {
		t.Helper()
		payload := []byte(fmt.Sprintf(`{"schema_version":2,"name":%q}`, name))
		envelope := signCatalogForTest(t, privateKey, signedCatalogEnvelope{
			SchemaVersion: 1, KeyID: "release-2026", Sequence: sequence,
			IssuedAt: now.Add(-time.Hour).Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
		}, payload)
		verified, err := verifySignedCatalog(envelope, map[string]ed25519.PublicKey{"release-2026": publicKey}, now, catalogCheckpoint{})
		if err != nil {
			t.Fatal(err)
		}
		return verified
	}
	catalogSeven := verifiedForSequence(7, "catalog-seven")
	catalogEight := verifiedForSequence(8, "catalog-eight")
	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: modelID, Generation: "generation-1",
		PackageDigest: digestOne, Catalog: catalogSeven,
	}); err != nil {
		t.Fatal(err)
	}
	assertActivationState(t, store, 7, "generation-1", digestOne)

	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: modelID, Generation: "missing-generation",
		PackageDigest: digestTwo, Catalog: catalogEight,
	}); err == nil {
		t.Fatal("unpublished generation activated")
	}
	assertActivationState(t, store, 7, "generation-1", digestOne)

	if _, err := store.db.Exec(`
        CREATE TRIGGER inject_activation_abort
        BEFORE UPDATE ON active_models
        BEGIN SELECT RAISE(ABORT, 'injected activation failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: modelID, Generation: "generation-2",
		PackageDigest: digestTwo, Catalog: catalogEight,
	}); err == nil {
		t.Fatal("injected activation failure unexpectedly committed")
	}
	assertActivationState(t, store, 7, "generation-1", digestOne)
	if _, err := store.db.Exec(`DROP TRIGGER inject_activation_abort`); err != nil {
		t.Fatal(err)
	}
	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: modelID, Generation: "generation-2",
		PackageDigest: digestTwo, Catalog: catalogEight,
	}); err != nil {
		t.Fatal(err)
	}
	assertActivationState(t, store, 8, "generation-2", digestTwo)
	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: modelID, Generation: "generation-2",
		PackageDigest: digestTwo, Catalog: catalogEight,
	}); err != nil {
		t.Fatalf("idempotent activation failed: %v", err)
	}

	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: modelID, Generation: "generation-1",
		PackageDigest: digestOne, Catalog: catalogSeven,
	}); !errors.Is(err, errCatalogRollback) {
		t.Fatalf("stale catalog activation was not rejected: %v", err)
	}
	equivocation := verifiedForSequence(8, "different-catalog-eight")
	if err := store.activate(ctx, activationRequest{
		Channel: "stable", ModelID: modelID, Generation: "generation-1",
		PackageDigest: digestOne, Catalog: equivocation,
	}); !errors.Is(err, errCatalogEquivocate) {
		t.Fatalf("same-sequence catalog equivocation was not rejected: %v", err)
	}
	assertActivationState(t, store, 8, "generation-2", digestTwo)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := openActivationStore(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	assertActivationState(t, reopened, 8, "generation-2", digestTwo)
}

func assertActivationState(t *testing.T, store *activationStore, sequence uint64, generation, digest string) {
	t.Helper()
	checkpoint, active, err := store.state(context.Background(), "stable", "semantic-siglip")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			t.Fatal("activation state unexpectedly absent")
		}
		t.Fatal(err)
	}
	if checkpoint.Sequence != sequence || active.Sequence != sequence ||
		active.Generation != generation || active.PackageDigest != digest {
		t.Fatalf("unexpected state: checkpoint=%+v active=%+v", checkpoint, active)
	}
}

func digestText(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
