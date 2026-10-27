package api

import (
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"net/http"
)

func registerAIPreparationRoute(mux *http.ServeMux, service aimodel.PreparationReader) {
	mux.HandleFunc("GET /api/v1/ai/preparation", func(writer http.ResponseWriter, request *http.Request) {
		value := aimodel.Preparation{State: "disabled"}
		if service != nil {
			var err error
			value, err = service.Preparation(request.Context())
			if err != nil {
				value = aimodel.Preparation{State: "unavailable"}
			}
		}
		writer.Header().Set("Cache-Control", "no-store")
		writeJSON(writer, http.StatusOK, value)
	})
}
