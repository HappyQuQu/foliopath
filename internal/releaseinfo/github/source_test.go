package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSourceReturnsOnlyStableBoundedReleaseFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("User-Agent") != "FolioPath-update-check" ||
			request.URL.Query().Get("per_page") != "20" {
			t.Errorf("request headers/query = %#v, %s", request.Header, request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
          {"tag_name":"v1.2.0","name":"FolioPath 1.2","body":"# Faster browsing\n\nDetails","html_url":"https://example.test/v1.2.0","draft":false,"prerelease":false,"published_at":"2026-08-03T00:00:00Z"},
          {"tag_name":"v1.3.0-beta.1","name":"Beta","body":"beta","html_url":"https://example.test/beta","draft":false,"prerelease":true,"published_at":"2026-08-03T00:00:00Z"}
        ]`))
	}))
	defer server.Close()
	source, err := New(&http.Client{Timeout: time.Second}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	releases, err := source.ListStableReleases(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].Version != "v1.2.0" ||
		releases[0].Summary != "Faster browsing" ||
		releases[0].Notes != "# Faster browsing\n\nDetails" {
		t.Fatalf("releases = %#v", releases)
	}
}

func TestReleaseNotesAreBoundedWithoutBreakingUnicode(t *testing.T) {
	body := "  # 更新\r\n\r\n" + strings.Repeat("更", 20_001)
	notes := releaseNotes(body)
	if len([]rune(notes)) > 20_000 {
		t.Fatalf("notes length = %d", len([]rune(notes)))
	}
}
