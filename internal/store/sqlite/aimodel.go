package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func (s *Store) ListAIModels(ctx context.Context) (aimodel.Snapshot, error) {
	var snapshot aimodel.Snapshot
	var activeID sql.NullString
	if err := s.db.QueryRowContext(ctx, `
        SELECT revision, active_model_id
        FROM ai_model_state
        WHERE singleton_key = 1`,
	).Scan(&snapshot.Revision, &activeID); err != nil {
		return aimodel.Snapshot{}, fmt.Errorf("read AI model state: %w", err)
	}
	if activeID.Valid {
		snapshot.ActiveModelID = activeID.String
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, package_id, purpose, version, architecture, content_hash,
               license_id, package_size_bytes, storage_mode, state,
               source_identity, availability_revision, created_at_ms, updated_at_ms
        FROM ai_models
        ORDER BY package_id, version, architecture, id`)
	if err != nil {
		return aimodel.Snapshot{}, fmt.Errorf("list AI models: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		model, scanErr := scanAIModel(rows)
		if scanErr != nil {
			return aimodel.Snapshot{}, scanErr
		}
		model.Active = model.ID == snapshot.ActiveModelID
		snapshot.Items = append(snapshot.Items, model)
	}
	if err := rows.Err(); err != nil {
		return aimodel.Snapshot{}, fmt.Errorf("iterate AI models: %w", err)
	}
	return snapshot, nil
}

func (s *Store) RegisterAIModel(ctx context.Context, model aimodel.Model) (aimodel.Model, bool, error) {
	if err := aimodel.ValidateModel(model); err != nil {
		return aimodel.Model{}, false, err
	}
	created := false
	var registered aimodel.Model
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
            SELECT id, package_id, purpose, version, architecture, content_hash,
                   license_id, package_size_bytes, storage_mode, state,
                   source_identity, availability_revision, created_at_ms, updated_at_ms
            FROM ai_models
            WHERE purpose = ? AND package_id = ? AND version = ?
              AND architecture = ? AND content_hash = ?`,
			model.Package.Purpose, model.Package.PackageID, model.Package.Version,
			model.Package.Architecture, model.Package.ContentHash,
		)
		existing, scanErr := scanAIModel(row)
		if scanErr == nil {
			registered = existing
			return nil
		}
		if !errors.Is(scanErr, sql.ErrNoRows) {
			return scanErr
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO ai_models(
                id, purpose, package_id, version, architecture, content_hash,
                license_id, package_size_bytes, storage_mode, state,
                source_identity, availability_revision, created_at_ms, updated_at_ms
            ) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			model.ID, model.Package.Purpose, model.Package.PackageID, model.Package.Version,
			model.Package.Architecture, model.Package.ContentHash, model.Package.LicenseID,
			model.Package.PackageSizeByte, model.StorageMode, model.State, model.SourceIdentity,
			model.AvailabilityRevision, model.CreatedAt.UTC().UnixMilli(), model.UpdatedAt.UTC().UnixMilli(),
		); err != nil {
			return fmt.Errorf("insert AI model: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE ai_model_state SET revision = revision + 1 WHERE singleton_key = 1`,
		); err != nil {
			return fmt.Errorf("advance AI model revision: %w", err)
		}
		registered = model
		created = true
		return nil
	})
	if err != nil {
		return aimodel.Model{}, false, err
	}
	return registered, created, nil
}

func (s *Store) SetAIModelAvailability(
	ctx context.Context,
	modelID string,
	expectedRevision int64,
	state aimodel.State,
	now time.Time,
) (aimodel.Model, error) {
	if modelID == "" || expectedRevision < 1 ||
		(state != aimodel.StateAvailable && state != aimodel.StateUnavailable) || now.IsZero() {
		return aimodel.Model{}, aimodel.ErrInvalidModel
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var currentState string
		var revision int64
		if err := tx.QueryRowContext(ctx, `
            SELECT state, availability_revision FROM ai_models WHERE id = ?`, modelID,
		).Scan(&currentState, &revision); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return aimodel.ErrModelNotFound
			}
			return fmt.Errorf("read AI model availability: %w", err)
		}
		if revision != expectedRevision {
			return aimodel.ErrPreconditionFailed
		}
		if currentState == string(state) {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE ai_models
            SET state = ?, availability_revision = availability_revision + 1, updated_at_ms = ?
            WHERE id = ?`, state, now.UTC().UnixMilli(), modelID,
		); err != nil {
			return fmt.Errorf("update AI model availability: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE ai_model_state SET revision = revision + 1 WHERE singleton_key = 1`,
		); err != nil {
			return fmt.Errorf("advance AI model revision: %w", err)
		}
		return nil
	})
	if err != nil {
		return aimodel.Model{}, err
	}
	model, err := s.getAIModel(ctx, modelID)
	if err != nil {
		return aimodel.Model{}, err
	}
	return model, nil
}

func (s *Store) getAIModel(ctx context.Context, modelID string) (aimodel.Model, error) {
	model, err := scanAIModel(s.db.QueryRowContext(ctx, `
        SELECT id, package_id, purpose, version, architecture, content_hash,
               license_id, package_size_bytes, storage_mode, state,
               source_identity, availability_revision, created_at_ms, updated_at_ms
        FROM ai_models WHERE id = ?`, modelID))
	if errors.Is(err, sql.ErrNoRows) {
		return aimodel.Model{}, aimodel.ErrModelNotFound
	}
	return model, err
}

func (s *Store) GetAIModel(ctx context.Context, modelID string) (aimodel.Model, error) {
	model, err := s.getAIModel(ctx, modelID)
	if err != nil {
		return aimodel.Model{}, err
	}
	var activeID sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT active_model_id FROM ai_model_state WHERE singleton_key = 1`).Scan(&activeID); err != nil {
		return aimodel.Model{}, fmt.Errorf("read active AI model: %w", err)
	}
	model.Active = activeID.Valid && activeID.String == model.ID
	return model, nil
}

type aiModelRowScanner interface {
	Scan(...any) error
}

func scanAIModel(row aiModelRowScanner) (aimodel.Model, error) {
	var model aimodel.Model
	var storageMode, state string
	var createdAtMS, updatedAtMS int64
	if err := row.Scan(
		&model.ID, &model.Package.PackageID, &model.Package.Purpose, &model.Package.Version,
		&model.Package.Architecture, &model.Package.ContentHash, &model.Package.LicenseID,
		&model.Package.PackageSizeByte, &storageMode, &state, &model.SourceIdentity,
		&model.AvailabilityRevision, &createdAtMS, &updatedAtMS,
	); err != nil {
		return aimodel.Model{}, err
	}
	model.StorageMode = aimodel.StorageMode(storageMode)
	model.State = aimodel.State(state)
	model.CreatedAt = time.UnixMilli(createdAtMS).UTC()
	model.UpdatedAt = time.UnixMilli(updatedAtMS).UTC()
	if err := aimodel.ValidateModel(model); err != nil {
		return aimodel.Model{}, err
	}
	return model, nil
}
