package app

import (
	"context"
	"crypto/sha256"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestComposedCurationAuthenticationPersistenceAndOriginalInvariance(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	mediaPath := filepath.Join(mediaRoot, "family", "photo.jpg")
	if err := os.MkdirAll(filepath.Dir(mediaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte("synthetic-curation-fixture")
	if err := os.WriteFile(mediaPath, original, 0o640); err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(mediaPath, modified, modified); err != nil {
		t.Fatal(err)
	}
	originalHash := sha256.Sum256(original)
	libraryID := seedRuntimeLibrary(t, dataRoot, mediaRoot)

	application, err := composeConfiguration(Input{Version: "integration"}, configuration{listenAddress: "127.0.0.1:0", mediaRoot: mediaRoot, dataRoot: dataRoot})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- application.run(ctx) }()
	defer func() {
		cancel()
		select {
		case runErr := <-result:
			if runErr != nil {
				t.Errorf("application.run() error = %v", runErr)
			}
		case <-time.After(runtimeIntegrationTimeout):
			t.Error("application did not stop")
		}
	}()

	address := waitForListenAddress(t, application.http)
	client := &http.Client{Timeout: runtimeIntegrationTimeout}
	setup := runtimeAuthenticationRequest(t, client, address, http.MethodPost, "/api/v1/auth/setup", `{"username":"Administrator","displayName":"Administrator","password":"correct horse battery staple"}`, "", "")
	if setup.StatusCode != http.StatusCreated {
		t.Fatalf("setup response = %#v", setup)
	}
	assetID := waitForRuntimeContentAsset(t, dataRoot, libraryID)
	favoritePath := "/api/v1/assets/ast_" + strconv.FormatInt(assetID, 10) + "/favorite"

	unauthorized := runtimeAuthenticationRequest(t, client, address, http.MethodPut, favoritePath, `{"favorite":true}`, "", "")
	if unauthorized.StatusCode != http.StatusUnauthorized || unauthorized.ErrorCode != "authentication_required" {
		t.Fatalf("unauthorized favorite = %#v", unauthorized)
	}
	favorited := runtimeAuthenticationRequest(t, client, address, http.MethodPut, favoritePath, `{"favorite":true}`, setup.Cookie, setup.CSRFToken)
	if favorited.StatusCode != http.StatusOK || favorited.ETag == "" || !strings.Contains(favorited.Body, `"favorite":true`) {
		t.Fatalf("favorite response = %#v", favorited)
	}
	createdTag := runtimeAuthenticationRequest(t, client, address, http.MethodPost, "/api/v1/tags", `{"name":"人物"}`, setup.Cookie, setup.CSRFToken)
	if createdTag.StatusCode != http.StatusCreated || !strings.Contains(createdTag.Body, `"name":"人物"`) {
		t.Fatalf("tag response = %#v", createdTag)
	}
	favorites := runtimeAuthenticationRequest(t, client, address, http.MethodGet, "/api/v1/favorites?libraryId=lib_"+strconv.FormatInt(libraryID, 10), "", setup.Cookie, "")
	if favorites.StatusCode != http.StatusOK || !strings.Contains(favorites.Body, `"assetId":"ast_`+strconv.FormatInt(assetID, 10)+`"`) {
		t.Fatalf("favorite list = %#v", favorites)
	}

	current, err := os.ReadFile(mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(current) != originalHash || !info.ModTime().Equal(modified) {
		t.Fatalf("curation changed original media: hash=%x mtime=%s", sha256.Sum256(current), info.ModTime())
	}
}
