package aimodel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

var ErrIdempotencyConflict = errors.New("AI model operation idempotency conflict")

type InstallWork struct {
	IdempotencyKey string
	RequestHash    string
	CandidateID    string
	Candidate      Candidate
	StorageMode    StorageMode
	Operation      Operation
}

type InstallQueue interface {
	FindAIModelInstall(context.Context, string) (InstallWork, bool, error)
	CreateAIModelInstall(context.Context, InstallWork) (InstallWork, bool, error)
	ClaimAIModelInstall(context.Context, time.Time) (InstallWork, bool, error)
}

func InstallRequestHash(candidateID string, storageMode StorageMode) string {
	digest := sha256.Sum256([]byte("foliopath:model-install:v1\x00" + candidateID + "\x00" + string(storageMode)))
	return hex.EncodeToString(digest[:])
}

type InstallAdmissionService struct {
	queue InstallQueue
	now   func() time.Time
	newID IDGenerator
	wake  interface{ Wake() }
}

func NewInstallAdmissionService(
	queue InstallQueue,
	wake interface{ Wake() },
	now func() time.Time,
	newID IDGenerator,
) (*InstallAdmissionService, error) {
	if queue == nil || wake == nil {
		return nil, errors.New("AI model install admission dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomOperationID
	}
	return &InstallAdmissionService{queue: queue, wake: wake, now: now, newID: newID}, nil
}

func (service *InstallAdmissionService) ReplayInstall(
	ctx context.Context,
	candidateID string,
	storageMode StorageMode,
	idempotencyKey string,
) (InstallResult, bool, error) {
	existing, found, err := service.queue.FindAIModelInstall(ctx, idempotencyKey)
	if err != nil || !found {
		return InstallResult{}, false, err
	}
	if existing.RequestHash != InstallRequestHash(candidateID, storageMode) {
		return InstallResult{}, false, ErrIdempotencyConflict
	}
	if err := validateInstallWork(existing, candidateID, storageMode, idempotencyKey); err != nil {
		return InstallResult{}, false, err
	}
	return InstallResult{Operation: existing.Operation, Replayed: true}, true, nil
}

func (service *InstallAdmissionService) StartInstall(
	ctx context.Context,
	candidate Candidate,
	storageMode StorageMode,
	idempotencyKey string,
) (InstallResult, error) {
	if candidate.ID == "" || candidate.Compatibility != "compatible" || ValidatePackage(candidate.Package) != nil ||
		validateManifest(candidate.Manifest) != nil || candidate.SourceIdentity == "" ||
		(storageMode != StorageManaged && storageMode != StorageDirect) ||
		len(idempotencyKey) < 8 || len(idempotencyKey) > 128 {
		return InstallResult{}, ErrInvalidModel
	}
	requestHash := InstallRequestHash(candidate.ID, storageMode)
	if replay, found, err := service.ReplayInstall(ctx, candidate.ID, storageMode, idempotencyKey); err != nil || found {
		return replay, err
	}
	id, err := service.newID()
	if err != nil {
		return InstallResult{}, err
	}
	now := service.now().UTC()
	requested := InstallWork{
		IdempotencyKey: idempotencyKey, RequestHash: requestHash,
		CandidateID: candidate.ID, Candidate: candidate, StorageMode: storageMode,
		Operation: Operation{
			ID: id, Kind: OperationModelInstall, State: OperationQueued, Phase: PhaseQueued,
			Revision: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
	work, created, err := service.queue.CreateAIModelInstall(ctx, requested)
	if err != nil {
		return InstallResult{}, err
	}
	if work.RequestHash != requestHash {
		return InstallResult{}, ErrIdempotencyConflict
	}
	if err := validateInstallWork(work, candidate.ID, storageMode, idempotencyKey); err != nil {
		return InstallResult{}, err
	}
	if !equalCandidate(work.Candidate, candidate) || (created && !sameOperationCreation(work.Operation, requested.Operation)) {
		return InstallResult{}, ErrRepositoryState
	}
	if created {
		service.wake.Wake()
	}
	return InstallResult{Operation: work.Operation, Created: created, Replayed: !created}, nil
}

func validateInstallWork(work InstallWork, candidateID string, storageMode StorageMode, idempotencyKey string) error {
	if work.IdempotencyKey != idempotencyKey || work.CandidateID != candidateID || work.StorageMode != storageMode ||
		work.Candidate.ID != candidateID || work.Candidate.Compatibility != "compatible" ||
		ValidatePackage(work.Candidate.Package) != nil || validateManifest(work.Candidate.Manifest) != nil ||
		work.Candidate.SourceIdentity == "" || validateOperation(work.Operation) != nil ||
		work.Operation.Kind != OperationModelInstall || work.Operation.ModelID != "" {
		return ErrRepositoryState
	}
	return nil
}

func equalCandidate(left, right Candidate) bool {
	return left.ID == right.ID && left.Package == right.Package && equalManifest(left.Manifest, right.Manifest) &&
		left.Compatibility == right.Compatibility && left.SourceIdentity == right.SourceIdentity
}
