package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/releaseinfo"
)

const defaultURL = "https://api.github.com/repos/HappyQuQu/foliopath/releases"

type Source struct {
	client *http.Client
	url    string
}

func New(client *http.Client, url string) (*Source, error) {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	if url == "" {
		url = defaultURL
	}
	if client.Timeout <= 0 || client.Timeout > 30*time.Second {
		return nil, errors.New("release source requires a bounded HTTP timeout")
	}
	return &Source{client: client, url: url}, nil
}

func (source *Source) ListStableReleases(
	ctx context.Context,
	limit int,
) ([]releaseinfo.Release, error) {
	if limit < 1 || limit > 20 {
		return nil, errors.New("release limit must be between 1 and 20")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s?per_page=%d", source.url, limit), nil)
	if err != nil {
		return nil, fmt.Errorf("construct release request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "FolioPath-update-check")
	response, err := source.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("check releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("check releases: upstream status %d", response.StatusCode)
	}
	var payload []struct {
		TagName     string    `json:"tag_name"`
		Name        string    `json:"name"`
		Body        string    `json:"body"`
		HTMLURL     string    `json:"html_url"`
		Draft       bool      `json:"draft"`
		Prerelease  bool      `json:"prerelease"`
		PublishedAt time.Time `json:"published_at"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode releases: %w", err)
	}
	releases := make([]releaseinfo.Release, 0, len(payload))
	for _, item := range payload {
		if item.Draft || item.Prerelease || item.TagName == "" || item.PublishedAt.IsZero() {
			continue
		}
		releases = append(releases, releaseinfo.Release{
			Version: item.TagName, Name: fallbackName(item.Name, item.TagName),
			Summary: summary(item.Body), PublishedAt: item.PublishedAt.UTC(),
			URL: item.HTMLURL,
		})
	}
	return releases, nil
}

func fallbackName(name, version string) string {
	if strings.TrimSpace(name) == "" {
		return version
	}
	return strings.TrimSpace(name)
}

func summary(body string) string {
	for _, paragraph := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n\n") {
		paragraph = strings.TrimSpace(paragraph)
		paragraph = strings.TrimLeft(paragraph, "# ")
		if paragraph == "" {
			continue
		}
		if len(paragraph) > 500 {
			return paragraph[:500]
		}
		return paragraph
	}
	return ""
}
