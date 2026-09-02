package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestAIModelInstallQueuePersistsIdempotencyAndClaimsQueuedWork(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 16, 0, 0, 0, time.UTC)
	work := installWorkFixture(now)
	created, wasCreated, err := store.CreateAIModelInstall(context.Background(), work)
	if err != nil || !wasCreated || created.Operation.ID != work.Operation.ID {
		t.Fatalf("create = %#v, %v, %v", created, wasCreated, err)
	}
	replayed, wasCreated, err := store.CreateAIModelInstall(context.Background(), work)
	if err != nil || wasCreated || replayed.Operation.ID != work.Operation.ID {
		t.Fatalf("replay = %#v, %v, %v", replayed, wasCreated, err)
	}
	claimed, found, err := store.ClaimAIModelInstall(context.Background(), now.Add(time.Second))
	if err != nil || !found || claimed.Operation.State != aimodel.OperationRunning ||
		claimed.Operation.Phase != aimodel.PhaseVerifying || claimed.Operation.Revision != 2 ||
		claimed.Candidate.Package != work.Candidate.Package {
		t.Fatalf("claim = %#v, %v, %v", claimed, found, err)
	}
	if _, found, err := store.ClaimAIModelInstall(context.Background(), now.Add(2*time.Second)); err != nil || found {
		t.Fatalf("second claim found/error = %v/%v", found, err)
	}
}

func TestAIModelRecoveryPreservesDurableQueuedInstall(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 17, 0, 0, 0, time.UTC)
	work := installWorkFixture(now)
	if _, _, err := store.CreateAIModelInstall(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	count, err := store.RecoverInterruptedAIOperations(context.Background(), now.Add(time.Minute))
	if err != nil || count != 0 {
		t.Fatalf("recovery = %d, %v", count, err)
	}
	reloaded, found, err := store.FindAIModelInstall(context.Background(), work.IdempotencyKey)
	if err != nil || !found || reloaded.Operation.State != aimodel.OperationQueued {
		t.Fatalf("queued install = %#v, %v, %v", reloaded, found, err)
	}
}

func TestAIModelInstallClaimClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 9, 1, 12, 30, 0, 0, time.UTC)
	work := installWorkFixture(now)
	work.Operation.ID = "aio_install_clock_rollback"
	work.IdempotencyKey = "install-clock-rollback"
	if _, _, err := store.CreateAIModelInstall(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimAIModelInstall(context.Background(), now.Add(-time.Minute))
	if err != nil || !found || claimed.Operation.UpdatedAt.Before(claimed.Operation.CreatedAt) {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
}

func installWorkFixture(now time.Time) aimodel.InstallWork {
	manifest := aimodel.Manifest{
		FormatVersion: 1, PackageID: "semantic-test-v1", Purpose: aimodel.PurposeSemanticImageText,
		Version: "1.0.0", Architecture: "portable-onnx", LicenseID: "Apache-2.0",
		Files: []aimodel.ManifestFile{
			{Name: "image.onnx", Size: 1, SHA256: strings.Repeat("a", 64), Role: "image_encoder"},
			{Name: "text.onnx", Size: 1, SHA256: strings.Repeat("b", 64), Role: "text_encoder"},
			{Name: "tokenizer.json", Size: 1, SHA256: strings.Repeat("c", 64), Role: "tokenizer"},
		},
	}
	candidate := aimodel.Candidate{
		ID: "aic_persisted_candidate", Compatibility: "compatible", SourceIdentity: "source:persisted",
		Manifest: manifest,
		Package: aimodel.VerifiedPackage{
			PackageID: manifest.PackageID, Purpose: manifest.Purpose, Version: manifest.Version,
			Architecture: "arm64", ContentHash: strings.Repeat("d", 64), LicenseID: manifest.LicenseID,
			PackageSizeByte: 3,
		},
	}
	return aimodel.InstallWork{
		IdempotencyKey: "install-request-persisted", CandidateID: candidate.ID,
		RequestHash: aimodel.InstallRequestHash(candidate.ID, aimodel.StorageManaged),
		Candidate:   candidate, StorageMode: aimodel.StorageManaged,
		Operation: aimodel.Operation{
			ID: "aio_persisted_install", Kind: aimodel.OperationModelInstall,
			State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued,
			Revision: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
}
