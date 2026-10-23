package api

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"net/http"
)

type ModelProxyService interface {
	Get(context.Context) (aimodel.ProxySnapshot, error)
	Save(context.Context, aimodel.ProxySettings) (aimodel.ProxySnapshot, error)
	Test(context.Context, aimodel.ProxySettings) (bool, error)
}
type modelProxyRequest struct {
	Mode     string `json:"mode"`
	ProxyURL string `json:"proxyUrl"`
	Revision int64  `json:"revision"`
}
type modelProxyResponse struct {
	modelProxyRequest
	DeploymentConfigured bool   `json:"deploymentConfigured"`
	EffectiveMode        string `json:"effectiveMode"`
}

func registerModelProxyRoutes(mux *http.ServeMux, s ModelProxyService) {
	wire := func(w http.ResponseWriter, v aimodel.ProxySnapshot) {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, modelProxyResponse{modelProxyRequest{v.Mode, v.URL, v.Revision}, v.DeploymentConfigured, v.EffectiveMode})
	}
	mux.HandleFunc("GET /api/v1/ai/model-proxy", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.Get(r.Context())
		if err != nil {
			writeAIModelError(w, r, err)
			return
		}
		wire(w, v)
	})
	for _, method := range []string{"PUT", "POST"} {
		path := "/api/v1/ai/model-proxy"
		if method == "POST" {
			path += "/test"
		}
		mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			var body struct {
				Mode     string  `json:"mode"`
				ProxyURL *string `json:"proxyUrl"`
				Revision int64   `json:"revision"`
			}
			if decodeAIModelJSON(r, &body) != nil || body.ProxyURL == nil || body.Revision < 1 {
				writeAIModelError(w, r, aimodel.ErrInvalidProxySettings)
				return
			}
			v := aimodel.ProxySettings{Mode: body.Mode, URL: *body.ProxyURL, Revision: body.Revision}
			if r.Method == "POST" {
				ok, err := s.Test(r.Context(), v)
				if err != nil {
					writeAIModelError(w, r, err)
					return
				}
				writeJSON(w, 200, struct {
					Available bool `json:"available"`
				}{ok})
				return
			}
			saved, err := s.Save(r.Context(), v)
			if err != nil {
				writeAIModelError(w, r, err)
				return
			}
			wire(w, saved)
		})
	}
}
