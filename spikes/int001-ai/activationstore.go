package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"
)

type activationStore struct {
	db *sql.DB
}

type activationRequest struct {
	Channel       string
	ModelID       string
	Generation    string
	PackageDigest string
	Catalog       verifiedCatalog
}

type activationState struct {
	Generation    string
	PackageDigest string
	Sequence      uint64
	Available     bool
	SourceKind    string
}

func openActivationStore(path string) (*activationStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	statements := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA busy_timeout=5000`,
		`PRAGMA foreign_keys=ON`,
		`CREATE TABLE IF NOT EXISTS published_generations (
            model_id TEXT NOT NULL,
            generation TEXT NOT NULL,
            package_digest TEXT NOT NULL,
            source_kind TEXT NOT NULL CHECK (source_kind IN ('managed', 'direct')),
            available INTEGER NOT NULL DEFAULT 1 CHECK (available IN (0, 1)),
            published_at INTEGER NOT NULL,
            PRIMARY KEY (model_id, generation)
        ) STRICT`,
		`CREATE TABLE IF NOT EXISTS catalog_checkpoints (
            channel TEXT PRIMARY KEY,
            sequence INTEGER NOT NULL,
            payload_sha256 TEXT NOT NULL,
            key_id TEXT NOT NULL,
            accepted_at INTEGER NOT NULL
        ) STRICT`,
		`CREATE TABLE IF NOT EXISTS active_models (
            model_id TEXT PRIMARY KEY,
            generation TEXT NOT NULL,
            package_digest TEXT NOT NULL,
            checkpoint_channel TEXT NOT NULL,
            catalog_sequence INTEGER NOT NULL,
            activated_at INTEGER NOT NULL,
            FOREIGN KEY (model_id, generation)
                REFERENCES published_generations(model_id, generation)
        ) STRICT`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return nil, fmt.Errorf("initialize activation store: %w", err)
		}
	}
	return &activationStore{db: db}, nil
}

func (store *activationStore) Close() error {
	return store.db.Close()
}

// recordPublishedGeneration is called only after a complete package directory
// has been verified and published no-replace. Reusing an immutable generation
// name for different bytes fails closed.
func (store *activationStore) recordPublishedGeneration(ctx context.Context, modelID, generation, digest string) error {
	if !safeActivationSegment(modelID) || !safeActivationSegment(generation) || !validSHA256(digest) {
		return errors.New("published generation metadata is invalid")
	}
	result, err := store.db.ExecContext(ctx, `
		INSERT INTO published_generations(model_id, generation, package_digest, source_kind, available, published_at)
		VALUES (?, ?, ?, 'managed', 1, ?)
        ON CONFLICT(model_id, generation) DO NOTHING`,
		modelID, generation, digest, time.Now().UTC().Unix())
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted == 1 {
		return err
	}
	var existing, existingSource string
	if err := store.db.QueryRowContext(ctx, `
		SELECT package_digest, source_kind FROM published_generations
		WHERE model_id = ? AND generation = ?`, modelID, generation).Scan(&existing, &existingSource); err != nil {
		return err
	}
	if existing != digest || existingSource != "managed" {
		return errCatalogEquivocate
	}
	return nil
}

type verifiedPackageObservation struct {
	ModelID       string
	Generation    string
	PackageDigest string
	SourceKind    string
	authenticated bool
}

func (store *activationStore) reconcileVerifiedPackage(ctx context.Context, observation verifiedPackageObservation) error {
	if !observation.authenticated || !safeActivationSegment(observation.ModelID) ||
		!safeActivationSegment(observation.Generation) || !validSHA256(observation.PackageDigest) ||
		(observation.SourceKind != "managed" && observation.SourceKind != "direct") {
		return errors.New("verified package observation is invalid")
	}
	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	var existing, existingSource string
	err = transaction.QueryRowContext(ctx, `
		SELECT package_digest, source_kind FROM published_generations
		WHERE model_id = ? AND generation = ?`, observation.ModelID, observation.Generation).Scan(&existing, &existingSource)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = transaction.ExecContext(ctx, `
			INSERT INTO published_generations(model_id, generation, package_digest, source_kind, available, published_at)
			VALUES (?, ?, ?, ?, 1, ?)`, observation.ModelID, observation.Generation,
			observation.PackageDigest, observation.SourceKind, time.Now().UTC().Unix())
	} else if err == nil && (existing != observation.PackageDigest || existingSource != observation.SourceKind) {
		return errCatalogEquivocate
	} else if err == nil {
		_, err = transaction.ExecContext(ctx, `
            UPDATE published_generations SET available = 1
            WHERE model_id = ? AND generation = ?`, observation.ModelID, observation.Generation)
	}
	if err != nil {
		return err
	}
	return transaction.Commit()
}

func (store *activationStore) markGenerationUnavailable(ctx context.Context, modelID, generation, sourceKind string) error {
	if !safeActivationSegment(modelID) || !safeActivationSegment(generation) ||
		(sourceKind != "managed" && sourceKind != "direct") {
		return errors.New("generation identity is invalid")
	}
	_, err := store.db.ExecContext(ctx, `
		UPDATE published_generations SET available = 0
		WHERE model_id = ? AND generation = ? AND source_kind = ?`, modelID, generation, sourceKind)
	return err
}

// activate advances the durable catalog checkpoint and active model pointer in
// one SQLite transaction. Filesystem publication must happen first; therefore
// a crash can leave an inert published generation, but cannot commit an active
// pointer to a generation absent from this registry.
func (store *activationStore) activate(ctx context.Context, request activationRequest) error {
	if !safeActivationSegment(request.Channel) || !safeActivationSegment(request.ModelID) ||
		!safeActivationSegment(request.Generation) || !validSHA256(request.PackageDigest) ||
		request.Catalog.Sequence == 0 || request.Catalog.Sequence > math.MaxInt64 ||
		!request.Catalog.authenticated || request.Catalog.KeyID == "" || !validSHA256(request.Catalog.PayloadSHA256) {
		return errors.New("activation request is invalid")
	}
	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	var publishedDigest string
	if err := transaction.QueryRowContext(ctx, `
        SELECT package_digest FROM published_generations
        WHERE model_id = ? AND generation = ?`, request.ModelID, request.Generation).Scan(&publishedDigest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("activation generation is not published")
		}
		return err
	}
	if publishedDigest != request.PackageDigest {
		return errCatalogEquivocate
	}
	var checkpointSequence int64
	var checkpointDigest, checkpointKey string
	err = transaction.QueryRowContext(ctx, `
        SELECT sequence, payload_sha256, key_id FROM catalog_checkpoints
        WHERE channel = ?`, request.Channel).Scan(&checkpointSequence, &checkpointDigest, &checkpointKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		if request.Catalog.Sequence < uint64(checkpointSequence) {
			return errCatalogRollback
		}
		if request.Catalog.Sequence == uint64(checkpointSequence) &&
			(request.Catalog.PayloadSHA256 != checkpointDigest || request.Catalog.KeyID != checkpointKey) {
			return errCatalogEquivocate
		}
	}
	now := time.Now().UTC().Unix()
	if _, err := transaction.ExecContext(ctx, `
        INSERT INTO catalog_checkpoints(channel, sequence, payload_sha256, key_id, accepted_at)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(channel) DO UPDATE SET
            sequence = excluded.sequence,
            payload_sha256 = excluded.payload_sha256,
            key_id = excluded.key_id,
            accepted_at = excluded.accepted_at`,
		request.Channel, int64(request.Catalog.Sequence), request.Catalog.PayloadSHA256, request.Catalog.KeyID, now); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, `
        INSERT INTO active_models(model_id, generation, package_digest, checkpoint_channel, catalog_sequence, activated_at)
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(model_id) DO UPDATE SET
            generation = excluded.generation,
            package_digest = excluded.package_digest,
            checkpoint_channel = excluded.checkpoint_channel,
            catalog_sequence = excluded.catalog_sequence,
            activated_at = excluded.activated_at`,
		request.ModelID, request.Generation, request.PackageDigest, request.Channel, int64(request.Catalog.Sequence), now); err != nil {
		return err
	}
	return transaction.Commit()
}

func (store *activationStore) state(ctx context.Context, channel, modelID string) (catalogCheckpoint, activationState, error) {
	var checkpoint catalogCheckpoint
	var sequence int64
	if err := store.db.QueryRowContext(ctx, `
        SELECT sequence, payload_sha256 FROM catalog_checkpoints WHERE channel = ?`, channel).
		Scan(&sequence, &checkpoint.PayloadSHA256); err != nil {
		return checkpoint, activationState{}, err
	}
	checkpoint.Sequence = uint64(sequence)
	var active activationState
	if err := store.db.QueryRowContext(ctx, `
		SELECT active.generation, active.package_digest, active.catalog_sequence, published.available, published.source_kind
        FROM active_models AS active
        JOIN published_generations AS published
          ON published.model_id = active.model_id AND published.generation = active.generation
        WHERE active.model_id = ?`, modelID).
		Scan(&active.Generation, &active.PackageDigest, &sequence, &active.Available, &active.SourceKind); err != nil {
		return checkpoint, active, err
	}
	active.Sequence = uint64(sequence)
	return checkpoint, active, nil
}

func safeActivationSegment(value string) bool {
	return packageSegmentPattern.MatchString(value)
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}
