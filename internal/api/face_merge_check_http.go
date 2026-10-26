package api

import (
	"context"
	"net/http"

	"github.com/HappyQuQu/foliopath/internal/face"
)

type PersonMergeCheckQueryService interface {
	Check(context.Context, string, string) (face.PersonMergeCheck, error)
}
type personMergeConflictExampleJSON struct {
	SourceLibraryID string `json:"sourceLibraryId"`
	SourceAssetID   string `json:"sourceAssetId"`
	TargetLibraryID string `json:"targetLibraryId"`
	TargetAssetID   string `json:"targetAssetId"`
}
type personMergeCheckJSON struct {
	SourcePersonID           string                           `json:"sourcePersonId"`
	TargetPersonID           string                           `json:"targetPersonId"`
	SourceRevision           int64                            `json:"sourceRevision"`
	TargetRevision           int64                            `json:"targetRevision"`
	CanMerge                 bool                             `json:"canMerge"`
	CannotLinkCount          int64                            `json:"cannotLinkCount"`
	DuplicateAssignmentCount int64                            `json:"duplicateAssignmentCount"`
	SourceLimitExceeded      bool                             `json:"sourceLimitExceeded"`
	Examples                 []personMergeConflictExampleJSON `json:"examples"`
}

func registerPersonMergeCheckRoute(mux *http.ServeMux, service PersonMergeCheckQueryService) {
	mux.HandleFunc("GET /api/v1/people/{personId}/merge-check", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if len(query) != 1 || len(query["targetPersonId"]) != 1 || query.Get("targetPersonId") == "" {
			writeFaceError(w, r, face.ErrInvalidPerson)
			return
		}
		value, err := service.Check(r.Context(), r.PathValue("personId"), query.Get("targetPersonId"))
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		examples := make([]personMergeConflictExampleJSON, 0, len(value.Examples))
		for _, example := range value.Examples {
			examples = append(examples, personMergeConflictExampleJSON{SourceLibraryID: libraryID(example.SourceLibraryID), SourceAssetID: assetID(example.SourceAssetID), TargetLibraryID: libraryID(example.TargetLibraryID), TargetAssetID: assetID(example.TargetAssetID)})
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, personMergeCheckJSON{SourcePersonID: value.SourcePersonID, TargetPersonID: value.TargetPersonID, SourceRevision: value.SourceRevision, TargetRevision: value.TargetRevision, CanMerge: value.CanMerge(), CannotLinkCount: value.CannotLinkCount, DuplicateAssignmentCount: value.DuplicateAssignmentCount, SourceLimitExceeded: value.SourceLimitExceeded, Examples: examples})
	})
}
