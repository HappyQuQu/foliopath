package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

var aiOpaqueIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

const maxAIModelJSONBytes = 4096

type AIModelManagementService interface {
	List(context.Context) (aimodel.Snapshot, error)
	ScanCandidates(context.Context) (aimodel.CandidateScan, error)
	StartInstall(context.Context, string, aimodel.StorageMode, string) (aimodel.InstallResult, error)
	StartActivation(context.Context, string, int64, string) (aimodel.ActivationResult, error)
	GetOperation(context.Context, string) (aimodel.Operation, error)
	CancelOperation(context.Context, string, int64) (aimodel.Operation, error)
}

type installAIModelRequest struct {
	StorageMode aimodel.StorageMode `json:"storageMode"`
}

type aiModelPageResponse struct {
	Items             []aiModelResponse `json:"items"`
	ActiveModelID     *string           `json:"activeModelId"`
	ActiveFaceModelID *string           `json:"activeFaceModelId"`
	Revision          int64             `json:"revision"`
}

type aiModelResponse struct {
	ID                   string              `json:"id"`
	Purpose              string              `json:"purpose"`
	Version              string              `json:"version"`
	Architecture         string              `json:"architecture"`
	StorageMode          aimodel.StorageMode `json:"storageMode"`
	State                aimodel.State       `json:"state"`
	AvailabilityRevision int64               `json:"availabilityRevision"`
	PackageSizeBytes     int64               `json:"packageSizeBytes"`
	LicenseID            string              `json:"licenseId"`
	Active               bool                `json:"active"`
}

type aiModelCandidateScanResponse struct {
	Revision   int64                      `json:"revision"`
	ScannedAt  string                     `json:"scannedAt"`
	Candidates []aiModelCandidateResponse `json:"candidates"`
	Truncated  bool                       `json:"truncated"`
}

type aiModelCandidateResponse struct {
	ID               string `json:"id"`
	Purpose          string `json:"purpose"`
	Version          string `json:"version"`
	Architecture     string `json:"architecture"`
	PackageSizeBytes int64  `json:"packageSizeBytes"`
	LicenseID        string `json:"licenseId"`
	Compatibility    string `json:"compatibility"`
}

type aiOperationResponse struct {
	ID             string                 `json:"id"`
	Kind           aimodel.OperationKind  `json:"kind"`
	State          aimodel.OperationState `json:"state"`
	Phase          aimodel.OperationPhase `json:"phase"`
	CompletedItems int64                  `json:"completedItems"`
	TotalItems     *int64                 `json:"totalItems"`
	Revision       int64                  `json:"revision"`
	ErrorCode      *string                `json:"errorCode"`
	CreatedAt      string                 `json:"createdAt"`
	UpdatedAt      string                 `json:"updatedAt"`
}

func registerAIModelRoutes(mux *http.ServeMux, service AIModelManagementService) {
	mux.HandleFunc("GET /api/v1/ai/models", func(writer http.ResponseWriter, request *http.Request) {
		snapshot, err := service.List(request.Context())
		if err != nil {
			writeAIModelError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", aiModelsETag(snapshot.Revision))
		writeJSON(writer, http.StatusOK, aiModelPageWire(snapshot))
	})
	mux.HandleFunc("POST /api/v1/ai/model-candidate-scans", func(writer http.ResponseWriter, request *http.Request) {
		scan, err := service.ScanCandidates(request.Context())
		if err != nil {
			writeAIModelError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", aiCandidateScanETag(scan.Revision))
		writeJSON(writer, http.StatusOK, aiCandidateScanWire(scan, time.Now().UTC()))
	})
	mux.HandleFunc("POST /api/v1/ai/model-candidates/{candidateId}/install", func(writer http.ResponseWriter, request *http.Request) {
		candidateID := request.PathValue("candidateId")
		key := request.Header.Get("Idempotency-Key")
		if !aiOpaqueIDPattern.MatchString(candidateID) || !idempotencyKeyPattern.MatchString(key) {
			writePublicError(writer, request, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		var payload installAIModelRequest
		if err := decodeAIModelJSON(request, &payload); err != nil ||
			(payload.StorageMode != aimodel.StorageManaged && payload.StorageMode != aimodel.StorageDirect) {
			writePublicError(writer, request, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		result, err := service.StartInstall(request.Context(), candidateID, payload.StorageMode, key)
		if err != nil {
			writeAIModelError(writer, request, err)
			return
		}
		location := "/api/v1/ai/operations/" + result.Operation.ID
		writer.Header().Set("Location", location)
		writer.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		writer.Header().Set("ETag", aiOperationETag(result.Operation))
		writeJSON(writer, http.StatusAccepted, aiOperationWire(result.Operation))
	})
	mux.HandleFunc("POST /api/v1/ai/models/{modelId}/activate", func(writer http.ResponseWriter, request *http.Request) {
		modelID := request.PathValue("modelId")
		key := request.Header.Get("Idempotency-Key")
		if !aiOpaqueIDPattern.MatchString(modelID) || !idempotencyKeyPattern.MatchString(key) {
			writePublicError(writer, request, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		revision, err := parseAIModelIfMatch(request.Header.Get("If-Match"), modelID)
		if err != nil {
			writeAIModelError(writer, request, err)
			return
		}
		result, err := service.StartActivation(request.Context(), modelID, revision, key)
		if err != nil {
			writeAIModelError(writer, request, err)
			return
		}
		writer.Header().Set("Location", "/api/v1/ai/operations/"+result.Operation.ID)
		writer.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		writer.Header().Set("ETag", aiOperationETag(result.Operation))
		writeJSON(writer, http.StatusAccepted, aiOperationWire(result.Operation))
	})
	mux.HandleFunc("GET /api/v1/ai/operations/{operationId}", func(writer http.ResponseWriter, request *http.Request) {
		operationID := request.PathValue("operationId")
		if !aiOpaqueIDPattern.MatchString(operationID) {
			writeAIModelError(writer, request, aimodel.ErrOperationNotFound)
			return
		}
		operation, err := service.GetOperation(request.Context(), operationID)
		if err != nil {
			writeAIModelError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", aiOperationETag(operation))
		writeJSON(writer, http.StatusOK, aiOperationWire(operation))
	})
	mux.HandleFunc("POST /api/v1/ai/operations/{operationId}/cancel", func(writer http.ResponseWriter, request *http.Request) {
		operationID := request.PathValue("operationId")
		if !aiOpaqueIDPattern.MatchString(operationID) {
			writeAIModelError(writer, request, aimodel.ErrOperationNotFound)
			return
		}
		revision, err := parseAIOperationIfMatch(request.Header.Get("If-Match"), operationID)
		if err != nil {
			writeAIModelError(writer, request, err)
			return
		}
		operation, err := service.CancelOperation(request.Context(), operationID, revision)
		if err != nil {
			writeAIModelError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", aiOperationETag(operation))
		writeJSON(writer, http.StatusAccepted, aiOperationWire(operation))
	})
}

func decodeAIModelJSON(request *http.Request, target any) error {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if request.Body == nil || err != nil || contentType != "application/json" {
		return errors.New("invalid JSON request")
	}
	limited := &io.LimitedReader{R: request.Body, N: maxAIModelJSONBytes + 1}
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("invalid trailing JSON")
	}
	if limited.N == 0 {
		return errors.New("JSON request exceeds size limit")
	}
	return nil
}

func aiModelPageWire(snapshot aimodel.Snapshot) aiModelPageResponse {
	response := aiModelPageResponse{Revision: snapshot.Revision, Items: make([]aiModelResponse, 0, len(snapshot.Items))}
	if snapshot.ActiveModelID != "" {
		value := snapshot.ActiveModelID
		response.ActiveModelID = &value
	}
	if snapshot.ActiveFaceModelID != "" {
		value := snapshot.ActiveFaceModelID
		response.ActiveFaceModelID = &value
	}
	for _, model := range snapshot.Items {
		response.Items = append(response.Items, aiModelResponse{
			ID: model.ID, Purpose: model.Package.Purpose, Version: model.Package.Version,
			Architecture: model.Package.Architecture, StorageMode: model.StorageMode, State: model.State,
			AvailabilityRevision: model.AvailabilityRevision, PackageSizeBytes: model.Package.PackageSizeByte,
			LicenseID: model.Package.LicenseID, Active: model.Active,
		})
	}
	return response
}

func aiCandidateScanWire(scan aimodel.CandidateScan, scannedAt time.Time) aiModelCandidateScanResponse {
	response := aiModelCandidateScanResponse{
		Revision: scan.Revision, ScannedAt: scannedAt.Format(time.RFC3339Nano),
		Candidates: make([]aiModelCandidateResponse, 0, len(scan.Candidates)), Truncated: scan.Truncated,
	}
	for _, candidate := range scan.Candidates {
		item := aiModelCandidateResponse{
			ID: candidate.ID, Purpose: "unknown", Version: "unknown", Architecture: "unknown",
			LicenseID: "unknown", Compatibility: "invalid_manifest",
		}
		if candidate.Compatibility == "compatible" {
			item.Purpose = candidate.Package.Purpose
			item.Version = candidate.Package.Version
			item.Architecture = candidate.Package.Architecture
			item.PackageSizeBytes = candidate.Package.PackageSizeByte
			item.LicenseID = candidate.Package.LicenseID
			item.Compatibility = "compatible"
		}
		response.Candidates = append(response.Candidates, item)
	}
	return response
}

func aiOperationWire(operation aimodel.Operation) aiOperationResponse {
	var errorCode *string
	if operation.ErrorCode != "" {
		value := operation.ErrorCode
		errorCode = &value
	}
	return aiOperationResponse{
		ID: operation.ID, Kind: operation.Kind, State: operation.State, Phase: operation.Phase,
		CompletedItems: operation.CompletedItems, TotalItems: operation.TotalItems,
		Revision: operation.Revision, ErrorCode: errorCode,
		CreatedAt: operation.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: operation.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func parseAIOperationIfMatch(value, operationID string) (int64, error) {
	return parseAIRevisionETag(value, operationID)
}

func parseAIModelIfMatch(value, modelID string) (int64, error) {
	return parseAIRevisionETag(value, modelID)
}

func parseAIRevisionETag(value, resourceID string) (int64, error) {
	if value == "" {
		return 0, errPreconditionRequired
	}
	prefix := `"` + resourceID + "-r"
	if !stringsHasETagShape(value, prefix) {
		return 0, aimodel.ErrPreconditionFailed
	}
	revisionText := value[len(prefix) : len(value)-1]
	revision, err := strconv.ParseInt(revisionText, 10, 64)
	if err != nil || revision < 1 {
		return 0, aimodel.ErrPreconditionFailed
	}
	if revisionText != strconv.FormatInt(revision, 10) {
		return 0, aimodel.ErrPreconditionFailed
	}
	return revision, nil
}

func stringsHasETagShape(value, prefix string) bool {
	return len(value) > len(prefix)+1 && value[:len(prefix)] == prefix && value[len(value)-1] == '"'
}

func aiModelsETag(revision int64) string {
	return `"ai-models-r` + strconv.FormatInt(revision, 10) + `"`
}
func aiCandidateScanETag(revision int64) string {
	return `"ai-candidates-r` + strconv.FormatInt(revision, 10) + `"`
}
func aiOperationETag(operation aimodel.Operation) string {
	return `"` + operation.ID + "-r" + strconv.FormatInt(operation.Revision, 10) + `"`
}

func writeAIModelError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, errPreconditionRequired):
		writePublicError(writer, request, http.StatusPreconditionRequired, "precondition_required", "A current resource validator is required.")
	case errors.Is(err, aimodel.ErrPreconditionFailed):
		writePublicError(writer, request, http.StatusPreconditionFailed, "precondition_failed", "The AI resource has changed.")
	case errors.Is(err, aimodel.ErrCandidateStale):
		writePublicError(writer, request, http.StatusConflict, "model_candidate_stale", "The model candidate scan is stale.")
	case errors.Is(err, aimodel.ErrIdempotencyConflict):
		writePublicError(writer, request, http.StatusConflict, "idempotency_conflict", "The idempotency key was used for a different request.")
	case errors.Is(err, aimodel.ErrOperationNotFound):
		writePublicError(writer, request, http.StatusNotFound, "ai_operation_not_found", "The AI operation was not found.")
	case errors.Is(err, aimodel.ErrOperationAlreadyFinished):
		writePublicError(writer, request, http.StatusConflict, "ai_operation_already_finished", "The AI operation has already finished.")
	case errors.Is(err, aimodel.ErrModelIncompatible):
		writePublicError(writer, request, http.StatusUnprocessableEntity, "model_incompatible", "The model package is incompatible.")
	case errors.Is(err, aimodel.ErrInsufficientSpace):
		writePublicError(writer, request, http.StatusUnprocessableEntity, "insufficient_space", "There is not enough safe managed-model space.")
	case errors.Is(err, aimodel.ErrModelSourceUnavailable):
		writePublicError(writer, request, http.StatusConflict, "model_source_unavailable", "The configured model source is unavailable.")
	case errors.Is(err, aimodel.ErrModelUnavailable), errors.Is(err, aimodel.ErrModelNotFound):
		writePublicError(writer, request, http.StatusConflict, "model_unavailable", "The AI model is unavailable.")
	default:
		writeInternalError(writer, request)
	}
}
