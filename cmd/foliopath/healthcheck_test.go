package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeReadinessAcceptsOnlyOKWithoutFollowingRedirects(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantError  bool
	}{
		{name: "ready", statusCode: http.StatusOK},
		{name: "not ready", statusCode: http.StatusServiceUnavailable, wantError: true},
		{name: "redirect", statusCode: http.StatusTemporaryRedirect, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.statusCode)
			}))
			defer server.Close()

			client := server.Client()
			client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			}
			err := probeReadiness(client, server.URL)
			if (err != nil) != test.wantError {
				t.Fatalf("probeReadiness() error = %v, wantError %t", err, test.wantError)
			}
		})
	}
}

func TestProbeReadinessMasksConnectionFailure(t *testing.T) {
	if err := probeReadiness(http.DefaultClient, "http://127.0.0.1:0/health/ready"); err == nil {
		t.Fatal("probeReadiness() error = nil, want unavailable")
	}
}
