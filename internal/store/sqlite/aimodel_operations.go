package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func (s *Store) CreateAIOperation(ctx context.Context, operation aimodel.Operation) (aimodel.Operation, error) {
	if err := validateNewAIOperation(operation); err != nil {
		return aimodel.Operation{}, err
	}
	var modelID, libraryID, totalItems any
	if operation.ModelID != "" {
		modelID = operation.ModelID
	}
	if operation.LibraryID > 0 {
		libraryID = operation.LibraryID
	}
	if operation.TotalItems != nil {
		totalItems = *operation.TotalItems
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO ai_model_operations(
                id, kind, state, phase, model_id, library_id,
                completed_items, total_items, error_code, lease_expires_ms,
                revision, created_at_ms, updated_at_ms, finished_at_ms
            ) VALUES(?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, ?, NULL)`,
			operation.ID, operation.Kind, operation.State, operation.Phase, modelID, libraryID,
			operation.CompletedItems, totalItems, operation.Revision,
			operation.CreatedAt.UTC().UnixMilli(), operation.UpdatedAt.UTC().UnixMilli(),
		); err != nil {
			return fmt.Errorf("create AI operation: %w", err)
		}
		return nil
	})
	if err != nil {
		return aimodel.Operation{}, err
	}
	return s.GetAIOperation(ctx, operation.ID)
}

func (s *Store) GetAIOperation(ctx context.Context, operationID string) (aimodel.Operation, error) {
	operation, err := scanAIOperation(s.db.QueryRowContext(ctx, `
        SELECT id, kind, state, phase, model_id, library_id,
               completed_items, total_items, error_code, revision,
               created_at_ms, updated_at_ms, finished_at_ms
        FROM ai_model_operations WHERE id = ?`, operationID))
	if errors.Is(err, sql.ErrNoRows) {
		return aimodel.Operation{}, aimodel.ErrOperationNotFound
	}
	return operation, err
}

func (s *Store) TransitionAIOperation(
	ctx context.Context,
	operationID string,
	transition aimodel.OperationTransition,
) (aimodel.Operation, error) {
	if operationID == "" || transition.ExpectedRevision < 1 || transition.UpdatedAt.IsZero() {
		return aimodel.Operation{}, aimodel.ErrInvalidTransition
	}
	var totalItems, errorCode, finishedAt any
	if transition.TotalItems != nil {
		totalItems = *transition.TotalItems
	}
	if transition.ErrorCode != "" {
		errorCode = transition.ErrorCode
	}
	if transition.FinishedAt != nil {
		finishedAt = transition.FinishedAt.UTC().UnixMilli()
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            UPDATE ai_model_operations
            SET state = ?, phase = ?, completed_items = ?, total_items = ?,
				error_code = ?, revision = revision + 1, updated_at_ms = MAX(created_at_ms, ?),
				finished_at_ms = CASE WHEN ? IS NULL THEN NULL ELSE MAX(created_at_ms, ?) END
            WHERE id = ? AND revision = ?`,
			transition.State, transition.Phase, transition.CompletedItems, totalItems,
			errorCode, transition.UpdatedAt.UTC().UnixMilli(), finishedAt, finishedAt,
			operationID, transition.ExpectedRevision,
		)
		if err != nil {
			return fmt.Errorf("transition AI operation: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read AI operation transition: %w", err)
		}
		if rows == 1 {
			return nil
		}
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM ai_model_operations WHERE id = ?`, operationID).Scan(&revision); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return aimodel.ErrOperationNotFound
			}
			return fmt.Errorf("resolve AI operation transition conflict: %w", err)
		}
		return aimodel.ErrPreconditionFailed
	})
	if err != nil {
		return aimodel.Operation{}, err
	}
	return s.GetAIOperation(ctx, operationID)
}

func (s *Store) RecoverInterruptedAIOperations(ctx context.Context, now time.Time) (int64, error) {
	if now.IsZero() {
		return 0, aimodel.ErrInvalidTransition
	}
	var recovered int64
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            UPDATE ai_model_operations
            SET state = 'failed', phase = 'completed', error_code = 'operation_interrupted',
				revision = revision + 1, updated_at_ms = MAX(created_at_ms, ?),
				finished_at_ms = MAX(created_at_ms, ?), lease_expires_ms = NULL
			WHERE state IN ('running', 'cancelling')
			  AND kind NOT IN ('semantic_missing', 'semantic_rebuild', 'semantic_clear')`,
			now.UTC().UnixMilli(), now.UTC().UnixMilli(),
		)
		if err != nil {
			return fmt.Errorf("recover interrupted AI operations: %w", err)
		}
		recovered, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read recovered AI operation count: %w", err)
		}
		return nil
	})
	return recovered, err
}

func validateNewAIOperation(operation aimodel.Operation) error {
	if operation.ID == "" || operation.State != aimodel.OperationQueued || operation.Phase != aimodel.PhaseQueued ||
		operation.Revision != 1 || operation.CompletedItems != 0 || operation.ErrorCode != "" ||
		operation.CreatedAt.IsZero() || !operation.UpdatedAt.Equal(operation.CreatedAt) || operation.FinishedAt != nil ||
		operation.LibraryID < 0 || (operation.TotalItems != nil && *operation.TotalItems < 0) {
		return aimodel.ErrInvalidTransition
	}
	return nil
}

func scanAIOperation(row aiModelRowScanner) (aimodel.Operation, error) {
	var operation aimodel.Operation
	var kind, state, phase string
	var modelID, errorCode sql.NullString
	var libraryID, totalItems, finishedAt sql.NullInt64
	var createdAt, updatedAt int64
	if err := row.Scan(
		&operation.ID, &kind, &state, &phase, &modelID, &libraryID,
		&operation.CompletedItems, &totalItems, &errorCode, &operation.Revision,
		&createdAt, &updatedAt, &finishedAt,
	); err != nil {
		return aimodel.Operation{}, err
	}
	operation.Kind = aimodel.OperationKind(kind)
	operation.State = aimodel.OperationState(state)
	operation.Phase = aimodel.OperationPhase(phase)
	operation.ModelID = modelID.String
	if libraryID.Valid {
		operation.LibraryID = libraryID.Int64
	}
	if totalItems.Valid {
		operation.TotalItems = &totalItems.Int64
	}
	operation.ErrorCode = errorCode.String
	operation.CreatedAt = time.UnixMilli(createdAt).UTC()
	operation.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	if finishedAt.Valid {
		value := time.UnixMilli(finishedAt.Int64).UTC()
		operation.FinishedAt = &value
	}
	return operation, nil
}
