package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/ocr"
)

type OCRService interface {
	Get(context.Context, int64) (ocr.Status, error)
	Update(context.Context, int64, int64, bool) (ocr.Status, error)
	Request(context.Context, int64, int64, ocr.Mode, string) (ocr.Job, bool, error)
	Job(context.Context, int64, string) (ocr.Job, error)
	Cancel(context.Context, int64, string, int64) (ocr.Job, error)
	Clear(context.Context, int64, int64) (ocr.Status, error)
}
type ocrJobResponse struct {
	ID          string   `json:"id"`
	LibraryID   string   `json:"libraryId"`
	Epoch       int64    `json:"epoch"`
	Fingerprint string   `json:"fingerprint"`
	Mode        ocr.Mode `json:"mode"`
	State       string   `json:"state"`
	Revision    int64    `json:"revision"`
	Recorded    int64    `json:"recorded"`
	Failed      int64    `json:"failed"`
	Attempts    int64    `json:"attempts"`
	ErrorCode   *string  `json:"errorCode"`
}
type ocrCoverageResponse struct {
	Eligible int64 `json:"eligible"`
	Ready    int64 `json:"ready"`
	Empty    int64 `json:"empty"`
	Failed   int64 `json:"failed"`
	Pending  int64 `json:"pending"`
	Stale    int64 `json:"stale"`
}
type ocrStatusResponse struct {
	LibraryID           string              `json:"libraryId"`
	Enabled             bool                `json:"enabled"`
	Clearing            bool                `json:"clearing"`
	Revision            int64               `json:"revision"`
	Epoch               int64               `json:"epoch"`
	Fingerprint         *string             `json:"fingerprint"`
	ExpectedFingerprint string              `json:"expectedFingerprint"`
	State               string              `json:"state"`
	Coverage            ocrCoverageResponse `json:"coverage"`
	LatestJob           *ocrJobResponse     `json:"latestJob"`
}

func ocrJobWire(j ocr.Job) ocrJobResponse {
	v := ocrJobResponse{ID: j.ID, LibraryID: fmt.Sprintf("lib_%d", j.LibraryID), Epoch: j.Epoch, Fingerprint: j.Fingerprint, Mode: j.Mode, State: j.State, Revision: j.ClaimRevision + 1, Recorded: j.Completed, Failed: j.Failed, Attempts: j.Attempts}
	if j.ErrorCode != "" {
		v.ErrorCode = &j.ErrorCode
	}
	return v
}
func ocrStatusWire(v ocr.Status) ocrStatusResponse {
	s, c := v.Settings, v.Coverage
	out := ocrStatusResponse{LibraryID: fmt.Sprintf("lib_%d", s.LibraryID), Enabled: s.Enabled, Clearing: s.Clearing, Revision: s.Revision, Epoch: s.Epoch, ExpectedFingerprint: v.ExpectedFingerprint, State: v.State, Coverage: ocrCoverageResponse{c.Eligible, c.Ready, c.Empty, c.Failed, c.Pending, c.Stale}}
	if s.Fingerprint != "" {
		out.Fingerprint = &s.Fingerprint
	}
	if v.LatestJob != nil {
		j := ocrJobWire(*v.LatestJob)
		out.LatestJob = &j
	}
	return out
}
func ocrStatusETag(v ocr.Status) string {
	return fmt.Sprintf(`"ocr-library-%d-r%d"`, v.Settings.LibraryID, v.Settings.Revision)
}
func ocrJobETag(j ocr.Job) string { return fmt.Sprintf(`"ocr-job-%s-r%d"`, j.ID, j.ClaimRevision+1) }
func writeOCRError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ocr.ErrInvalidPersistence):
		writePublicError(w, r, 400, codeInvalidRequest, "The request is invalid.")
	case errors.Is(err, ocr.ErrNotFound):
		writePublicError(w, r, 404, "ocr_not_found", "The OCR resource was not found.")
	case errors.Is(err, ocr.ErrDisabled):
		writePublicError(w, r, 409, "ocr_disabled", "OCR is disabled for this library.")
	case errors.Is(err, ocr.ErrOffline):
		writePublicError(w, r, 409, "source_offline", "The media library is not ready.")
	case errors.Is(err, ocr.ErrRevision):
		writePublicError(w, r, 412, "precondition_failed", "The OCR revision changed.")
	case errors.Is(err, ocr.ErrConflict):
		writePublicError(w, r, 409, "operation_conflict", "The OCR state changed or an incompatible operation is active.")
	default:
		writeAIModelError(w, r, err)
	}
}
func registerOCRRoutes(mux *http.ServeMux, service OCRService) {
	path := "/api/v1/libraries/{libraryId}/ai/ocr"
	parse := func(w http.ResponseWriter, r *http.Request) (int64, bool) {
		id, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		if err != nil || r.URL.RawQuery != "" {
			writeOCRError(w, r, ocr.ErrInvalidPersistence)
			return 0, false
		}
		return id, true
	}
	revision := func(w http.ResponseWriter, r *http.Request, resource string) (int64, bool) {
		v, err := parseAIRevisionETag(r.Header.Get("If-Match"), resource)
		if err != nil {
			writeAIModelError(w, r, err)
			return 0, false
		}
		return v, true
	}
	for _, method := range []string{"GET", "PUT"} {
		mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) {
			id, ok := parse(w, r)
			if !ok {
				return
			}
			var v ocr.Status
			var err error
			if r.Method == "PUT" {
				rev, ok := revision(w, r, fmt.Sprintf("ocr-library-%d", id))
				if !ok {
					return
				}
				var body struct {
					Enabled *bool `json:"enabled"`
				}
				if decodeAIModelJSON(r, &body) != nil || body.Enabled == nil {
					writeOCRError(w, r, ocr.ErrInvalidPersistence)
					return
				}
				v, err = service.Update(r.Context(), id, rev, *body.Enabled)
			} else {
				v, err = service.Get(r.Context(), id)
			}
			if err != nil {
				writeOCRError(w, r, err)
				return
			}
			w.Header().Set("ETag", ocrStatusETag(v))
			writeJSON(w, 200, ocrStatusWire(v))
		})
	}
	mux.HandleFunc("POST "+path+"/jobs", func(w http.ResponseWriter, r *http.Request) {
		id, ok := parse(w, r)
		if !ok {
			return
		}
		rev, ok := revision(w, r, fmt.Sprintf("ocr-library-%d", id))
		if !ok {
			return
		}
		var body struct {
			Mode         ocr.Mode `json:"mode"`
			Confirmation string   `json:"confirmation"`
		}
		key := r.Header.Get("Idempotency-Key")
		if !ocr.ValidOpaque(key) || decodeAIModelJSON(r, &body) != nil || !ocr.ValidMode(body.Mode) || (body.Mode == ocr.RetryFailed && body.Confirmation != "retry_failed_ocr") || (body.Mode == ocr.Missing && body.Confirmation != "") {
			writeOCRError(w, r, ocr.ErrInvalidPersistence)
			return
		}
		j, replay, err := service.Request(r.Context(), id, rev, body.Mode, key)
		if err != nil {
			writeOCRError(w, r, err)
			return
		}
		w.Header().Set("Location", fmt.Sprintf("/api/v1/libraries/lib_%d/ai/ocr/jobs/%s", id, j.ID))
		w.Header().Set("ETag", ocrJobETag(j))
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replay))
		writeJSON(w, 202, ocrJobWire(j))
	})
	for _, method := range []string{"GET", "POST"} {
		target := path + "/jobs/{jobId}"
		if method == "POST" {
			target += "/cancel"
		}
		mux.HandleFunc(method+" "+target, func(w http.ResponseWriter, r *http.Request) {
			id, ok := parse(w, r)
			if !ok {
				return
			}
			jobID := r.PathValue("jobId")
			if !ocr.ValidOpaque(jobID) {
				writeOCRError(w, r, ocr.ErrInvalidPersistence)
				return
			}
			var j ocr.Job
			var err error
			if r.Method == "POST" {
				rev, ok := revision(w, r, "ocr-job-"+jobID)
				if !ok {
					return
				}
				var body *struct{}
				if decodeAIModelJSON(r, &body) != nil || body == nil {
					writeOCRError(w, r, ocr.ErrInvalidPersistence)
					return
				}
				j, err = service.Cancel(r.Context(), id, jobID, rev-1)
			} else {
				j, err = service.Job(r.Context(), id, jobID)
			}
			if err != nil {
				writeOCRError(w, r, err)
				return
			}
			w.Header().Set("ETag", ocrJobETag(j))
			writeJSON(w, 200, ocrJobWire(j))
		})
	}
	mux.HandleFunc("POST "+path+"/clear", func(w http.ResponseWriter, r *http.Request) {
		id, ok := parse(w, r)
		if !ok {
			return
		}
		rev, ok := revision(w, r, fmt.Sprintf("ocr-library-%d", id))
		if !ok {
			return
		}
		var body struct {
			Confirmation string `json:"confirmation"`
		}
		if decodeAIModelJSON(r, &body) != nil || body.Confirmation != "clear_ocr_results" {
			writeOCRError(w, r, ocr.ErrInvalidPersistence)
			return
		}
		v, err := service.Clear(r.Context(), id, rev)
		if err != nil {
			writeOCRError(w, r, err)
			return
		}
		w.Header().Set("ETag", ocrStatusETag(v))
		writeJSON(w, 202, ocrStatusWire(v))
	})
}
