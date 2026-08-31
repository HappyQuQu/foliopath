package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func (s *Store) FindAIModelActivation(ctx context.Context, key string) (aimodel.ActivationWork, bool, error) {
	work, err := scanAIModelActivation(s.db.QueryRowContext(ctx, aiModelActivationSelect+` WHERE request.idempotency_key = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return aimodel.ActivationWork{}, false, nil
	}
	return work, err == nil, err
}

func (s *Store) CreateAIModelActivation(ctx context.Context, work aimodel.ActivationWork) (aimodel.ActivationWork, bool, error) {
	if work.IdempotencyKey == "" || work.RequestHash == "" || work.ModelID == "" ||
		work.ExpectedAvailabilityRevision < 1 || validateActivationOperation(work.Operation, work.ModelID) != nil {
		return aimodel.ActivationWork{}, false, aimodel.ErrInvalidModel
	}
	created := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var state string
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT state, availability_revision FROM ai_models WHERE id = ?`, work.ModelID).Scan(&state, &revision); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return aimodel.ErrModelNotFound
			}
			return err
		}
		if revision != work.ExpectedAvailabilityRevision {
			return aimodel.ErrPreconditionFailed
		}
		if state != string(aimodel.StateAvailable) {
			return aimodel.ErrModelUnavailable
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO ai_model_operations(id, kind, state, phase, model_id, completed_items, revision, created_at_ms, updated_at_ms)
            VALUES(?, 'model_activate', 'queued', 'queued', ?, 0, 1, ?, ?)`,
			work.Operation.ID, work.ModelID, work.Operation.CreatedAt.UTC().UnixMilli(), work.Operation.UpdatedAt.UTC().UnixMilli()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO ai_model_activation_requests(
                idempotency_key, request_hash, operation_id, model_id, expected_availability_revision, created_at_ms
            ) VALUES(?, ?, ?, ?, ?, ?)`, work.IdempotencyKey, work.RequestHash, work.Operation.ID,
			work.ModelID, work.ExpectedAvailabilityRevision, work.Operation.CreatedAt.UTC().UnixMilli()); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		if !isUniqueConstraint(err) {
			return aimodel.ActivationWork{}, false, err
		}
		existing, found, findErr := s.FindAIModelActivation(ctx, work.IdempotencyKey)
		return existing, false, firstErrorIfMissing(found, findErr)
	}
	stored, found, err := s.FindAIModelActivation(ctx, work.IdempotencyKey)
	return stored, created, firstErrorIfMissing(found, err)
}

func (s *Store) ClaimAIModelActivation(ctx context.Context, now time.Time) (aimodel.ActivationWork, bool, error) {
	var claimed aimodel.ActivationWork
	found := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		work, err := scanAIModelActivation(tx.QueryRowContext(ctx, aiModelActivationSelect+`
            WHERE operation.state = 'queued' ORDER BY operation.created_at_ms, operation.id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE ai_model_operations SET state = 'running', phase = 'loading', revision = revision + 1, updated_at_ms = ?
            WHERE id = ? AND state = 'queued' AND revision = ?`, now.UTC().UnixMilli(), work.Operation.ID, work.Operation.Revision)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return aimodel.ErrPreconditionFailed
		}
		work.Operation.State, work.Operation.Phase = aimodel.OperationRunning, aimodel.PhaseLoading
		work.Operation.Revision++
		work.Operation.UpdatedAt = now.UTC()
		claimed, found = work, true
		return nil
	})
	return claimed, found, err
}

func (s *Store) CommitAIModelActivation(ctx context.Context, commit aimodel.ActivationCommit) (aimodel.Operation, error) {
	if commit.OperationID == "" || commit.ExpectedRevision < 1 || commit.ExpectedAvailabilityRevision < 1 || commit.UpdatedAt.IsZero() ||
		aimodel.ValidateGeneration(commit.Generation) != nil {
		return aimodel.Operation{}, aimodel.ErrInvalidTransition
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var modelID, operationState string
		var operationRevision int64
		if err := tx.QueryRowContext(ctx, `SELECT model_id, state, revision FROM ai_model_operations WHERE id = ?`, commit.OperationID).Scan(&modelID, &operationState, &operationRevision); err != nil {
			return err
		}
		if operationRevision != commit.ExpectedRevision {
			return aimodel.ErrPreconditionFailed
		}
		if operationState != string(aimodel.OperationRunning) || modelID != commit.Generation.ModelID {
			return aimodel.ErrInvalidTransition
		}
		var modelState string
		var availabilityRevision int64
		if err := tx.QueryRowContext(ctx, `SELECT state, availability_revision FROM ai_models WHERE id = ?`, modelID).Scan(&modelState, &availabilityRevision); err != nil {
			return err
		}
		if availabilityRevision != commit.ExpectedAvailabilityRevision {
			return aimodel.ErrPreconditionFailed
		}
		if modelState != string(aimodel.StateAvailable) {
			return aimodel.ErrModelUnavailable
		}
		if _, err := tx.ExecContext(ctx, `UPDATE semantic_generations SET state = 'retired', updated_at_ms = ? WHERE state = 'active'`, commit.UpdatedAt.UTC().UnixMilli()); err != nil {
			return err
		}
		activatedAt := commit.Generation.ActivatedAt.UTC().UnixMilli()
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO semantic_generations(
                id, model_id, transform_version, output_schema_version, index_format_version,
                embedding_dimension, state, created_at_ms, activated_at_ms, updated_at_ms
            ) VALUES(?, ?, ?, ?, ?, ?, 'active', ?, ?, ?)`,
			commit.Generation.ID, modelID, commit.Generation.TransformVersion, commit.Generation.OutputSchemaVersion,
			commit.Generation.IndexFormatVersion, commit.Generation.EmbeddingDimension,
			commit.Generation.CreatedAt.UTC().UnixMilli(), activatedAt, commit.UpdatedAt.UTC().UnixMilli()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE ai_model_state SET revision = revision + 1, active_model_id = ?, active_generation_id = ? WHERE singleton_key = 1`,
			modelID, commit.Generation.ID); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE ai_model_operations SET state = 'succeeded', phase = 'completed', revision = revision + 1,
                updated_at_ms = ?, finished_at_ms = ?, error_code = NULL WHERE id = ? AND revision = ?`,
			commit.UpdatedAt.UTC().UnixMilli(), commit.UpdatedAt.UTC().UnixMilli(), commit.OperationID, commit.ExpectedRevision)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return aimodel.ErrPreconditionFailed
		}
		return nil
	})
	if err != nil {
		return aimodel.Operation{}, fmt.Errorf("commit AI model activation: %w", err)
	}
	return s.GetAIOperation(ctx, commit.OperationID)
}

const aiModelActivationSelect = `
    SELECT request.idempotency_key, request.request_hash, request.model_id,
           request.expected_availability_revision, operation.id, operation.kind,
           operation.state, operation.phase, operation.completed_items, operation.total_items,
           operation.error_code, operation.revision, operation.created_at_ms,
           operation.updated_at_ms, operation.finished_at_ms
    FROM ai_model_activation_requests AS request
    JOIN ai_model_operations AS operation ON operation.id = request.operation_id`

func scanAIModelActivation(row aiModelRowScanner) (aimodel.ActivationWork, error) {
	var work aimodel.ActivationWork
	var kind, state, phase string
	var totalItems, finishedAt sql.NullInt64
	var errorCode sql.NullString
	var createdAt, updatedAt int64
	if err := row.Scan(&work.IdempotencyKey, &work.RequestHash, &work.ModelID,
		&work.ExpectedAvailabilityRevision, &work.Operation.ID, &kind, &state, &phase,
		&work.Operation.CompletedItems, &totalItems, &errorCode, &work.Operation.Revision,
		&createdAt, &updatedAt, &finishedAt); err != nil {
		return aimodel.ActivationWork{}, err
	}
	work.Operation.Kind, work.Operation.State, work.Operation.Phase = aimodel.OperationKind(kind), aimodel.OperationState(state), aimodel.OperationPhase(phase)
	work.Operation.ModelID, work.Operation.ErrorCode = work.ModelID, errorCode.String
	work.Operation.CreatedAt, work.Operation.UpdatedAt = time.UnixMilli(createdAt).UTC(), time.UnixMilli(updatedAt).UTC()
	if totalItems.Valid {
		work.Operation.TotalItems = &totalItems.Int64
	}
	if finishedAt.Valid {
		value := time.UnixMilli(finishedAt.Int64).UTC()
		work.Operation.FinishedAt = &value
	}
	return work, nil
}

func validateActivationOperation(operation aimodel.Operation, modelID string) error {
	if operation.ID == "" || operation.Kind != aimodel.OperationModelActivate || operation.ModelID != modelID ||
		operation.State != aimodel.OperationQueued || operation.Phase != aimodel.PhaseQueued || operation.Revision != 1 ||
		operation.CreatedAt.IsZero() || !operation.UpdatedAt.Equal(operation.CreatedAt) {
		return aimodel.ErrInvalidTransition
	}
	return nil
}
