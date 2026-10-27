package remoteml

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPreparationSanitizesRemoteStatus(t *testing.T) {
	for _, test := range []struct{ body, state string }{
		{`{"state":"downloading","completedFiles":1,"totalFiles":5,"completedBytes":42,"totalBytes":100,"privatePath":"/private/models"}`, "downloading"},
		{`{"state":"ready","completedFiles":-1}`, "unavailable"},
		{`{"state":"exception /private/models"}`, "unavailable"},
		{`{"state":"downloading","totalBytes":999999999999}`, "unavailable"},
		{`invalid`, "unavailable"},
	} {
		t.Run(test.state, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/preparation" {
					t.Error(r.URL.Path)
				}
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client, err := New(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			value, err := client.Preparation(context.Background())
			if err != nil || value.State != test.state {
				t.Fatal(value, err)
			}
		})
	}
}

func TestPreparationRespectsSharedAdmissionAndCancellation(t *testing.T) {
	client, _ := New("http://127.0.0.1:1")
	defer client.Close()
	client.slots <- struct{}{}
	client.slots <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	value, err := client.Preparation(ctx)
	if err != nil || value.State != "unavailable" {
		t.Fatal(value, err)
	}
	if len(client.slots) != 2 {
		t.Fatal("changed inference admission")
	}
}
