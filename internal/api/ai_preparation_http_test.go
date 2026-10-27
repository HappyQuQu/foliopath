package api

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type preparationFixture struct{}

func (preparationFixture) Preparation(context.Context) (aimodel.Preparation, error) {
	return aimodel.Preparation{State: "downloading", CompletedFiles: 1, TotalFiles: 2, CompletedBytes: 4, TotalBytes: 8}, nil
}
func TestPreparationWireContract(t *testing.T) {
	for _, reader := range []aimodel.PreparationReader{nil, preparationFixture{}} {
		mux := http.NewServeMux()
		registerAIPreparationRoute(mux, reader)
		result := httptest.NewRecorder()
		mux.ServeHTTP(result, httptest.NewRequest("GET", "/api/v1/ai/preparation", nil))
		state := "disabled"
		if reader != nil {
			state = "downloading"
		}
		if result.Code != 200 || !strings.Contains(result.Body.String(), `"state":"`+state+`"`) || result.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(result)
		}
	}
}

func TestPreparationRequiresAuthentication(t *testing.T) {
	mux := http.NewServeMux()
	registerAIPreparationRoute(mux, preparationFixture{})
	handler := requireAPIAuthentication(mux, acceptingAuthentication())
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, httptest.NewRequest("GET", "/api/v1/ai/preparation", nil))
	if result.Code != http.StatusUnauthorized {
		t.Fatal(result.Code)
	}
}
