package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/face"
)

type mergeCheckHTTPStub struct {
	source, target string
	value          face.PersonMergeCheck
	err            error
	calls          int
}

func (s *mergeCheckHTTPStub) Check(_ context.Context, source, target string) (face.PersonMergeCheck, error) {
	s.source, s.target = source, target
	s.calls++
	return s.value, s.err
}
func TestPersonMergeCheckRouteUsesOpaqueIDsAndStrictQuery(t *testing.T) {
	stub := &mergeCheckHTTPStub{value: face.PersonMergeCheck{SourcePersonID: "person_source", TargetPersonID: "person_target", SourceRevision: 7, TargetRevision: 4, CannotLinkCount: 2, Examples: []face.MergeConflictExample{{SourceLibraryID: 1, SourceAssetID: 7, TargetLibraryID: 2, TargetAssetID: 8}}}}
	mux := http.NewServeMux()
	registerPersonMergeCheckRoute(mux, stub)
	for _, suffix := range []string{"", "?targetPersonId=", "?targetPersonId=a&targetPersonId=b", "?targetPersonId=a&url=http://example.invalid"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/people/person_source/merge-check"+suffix, nil))
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	if stub.calls != 0 {
		t.Fatal("malformed query reached service")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/people/person_source/merge-check?targetPersonId=person_target", nil))
	var value personMergeCheckJSON
	if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || value.CanMerge || value.CannotLinkCount != 2 || len(value.Examples) != 1 || value.Examples[0].SourceAssetID != "ast_7" || value.Examples[0].TargetLibraryID != "lib_2" || stub.source != "person_source" || stub.target != "person_target" {
		t.Fatalf("response=%+v %s", value, w.Body.String())
	}
	stub.value.Examples = nil
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/people/person_source/merge-check?targetPersonId=person_target", nil))
	if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil || value.Examples == nil {
		t.Fatal("empty examples must be []")
	}
	stub.err = face.ErrPersonNotFound
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/people/person_source/merge-check?targetPersonId=person_target", nil))
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}
