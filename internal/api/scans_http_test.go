package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type scanQueryStub struct {
	page      scanner.Page
	details   scanner.Details
	cancelled scanner.ScanRun
	err       error
}

func (stub scanQueryStub) List(context.Context, int64, string, int) (scanner.Page, error) {
	return stub.page, stub.err
}

func (stub scanQueryStub) Get(context.Context, int64) (scanner.Details, error) {
	return stub.details, stub.err
}

func (stub scanQueryStub) Cancel(context.Context, int64) (scanner.ScanRun, error) {
	return stub.cancelled, stub.err
}

func TestScanQueryRoutesListPollAndCancel(t *testing.T) {
	run := scanner.ScanRun{
		ID: 4, LibraryID: 7, Generation: 1,
		Trigger: scanner.TriggerManual, Status: scanner.RunStatusRunning,
		Phase: "walking", CreatedAtMS: 1_000, Revision: 3,
	}
	sample := "album"
	details := scanner.Details{
		Run: run,
		Issues: []scanner.Issue{{
			Code: "unreadable_directory", Count: 2, SampleRelativePath: &sample,
		}},
	}
	mux := http.NewServeMux()
	registerScanQueryRoutes(mux, scanQueryStub{
		page:    scanner.Page{Items: []scanner.Details{details}, NextCursor: "next"},
		details: details, cancelled: run,
	})

	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/libraries/lib_7/scans?limit=1", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", list.Code, list.Body.String())
	}
	var page scanPageResponse
	if err := json.Unmarshal(list.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || len(page.Items[0].Issues) != 1 ||
		page.Items[0].Issues[0].Message == "" || page.NextCursor == nil {
		t.Fatalf("scan page = %#v", page)
	}

	get := httptest.NewRecorder()
	mux.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/scans/scan_4", nil))
	if get.Code != http.StatusOK || get.Header().Get("ETag") != `"scan_4-r3"` {
		t.Fatalf("get status/header = %d/%q", get.Code, get.Header().Get("ETag"))
	}
	notModifiedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/scans/scan_4", nil)
	notModifiedRequest.Header.Set("If-None-Match", `"scan_4-r3"`)
	notModified := httptest.NewRecorder()
	mux.ServeHTTP(notModified, notModifiedRequest)
	if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 {
		t.Fatalf("conditional get = %d, body %q", notModified.Code, notModified.Body.String())
	}

	cancel := httptest.NewRecorder()
	mux.ServeHTTP(cancel, httptest.NewRequest(http.MethodPost, "/api/v1/scans/scan_4/cancel", nil))
	if cancel.Code != http.StatusAccepted ||
		cancel.Header().Get("Location") != "/api/v1/scans/scan_4" {
		t.Fatalf("cancel status/location = %d/%q", cancel.Code, cancel.Header().Get("Location"))
	}
}

func TestScanQueryRoutesMapStableErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "missing", err: scanner.ErrScanRunNotFound, status: 404, code: "scan_not_found"},
		{name: "cursor", err: scanner.ErrInvalidScanCursor, status: 400, code: "invalid_cursor"},
		{name: "finished", err: scanner.ErrScanAlreadyFinished, status: 409, code: "scan_already_finished"},
		{name: "internal", err: errors.New("secret database failure"), status: 500, code: "internal_error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/scans/scan_1", nil)
			writeScanQueryError(response, request, test.err)
			if response.Code != test.status || !jsonErrorHasCode(response.Body.Bytes(), test.code) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if test.name == "internal" && errors.Is(test.err, nil) {
				t.Fatal("unreachable")
			}
		})
	}
}

func jsonErrorHasCode(body []byte, code string) bool {
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal(body, &payload) == nil && payload.Error.Code == code
}
