package app

import (
	"errors"
	"strings"
	"testing"
)

func TestExplicitModelProxyConfiguration(t *testing.T) {
	config, err := loadConfiguration(Input{Environ: []string{"FOLIOPATH_MODEL_DOWNLOAD_PROXY=http://192.168.2.231:7890"}})
	if err != nil || config.modelDownloadProxy != "http://192.168.2.231:7890" {
		t.Fatalf("model proxy configuration: %v", err)
	}
	for _, environ := range [][]string{
		{"FOLIOPATH_MODEL_DOWNLOAD_PROXY"},
		{"FOLIOPATH_MODEL_DOWNLOAD_PROXY=", "FOLIOPATH_MODEL_DOWNLOAD_PROXY="},
		{"FOLIOPATH_MODEL_DOWNLOAD_PROXY=http://user:secret@proxy.example"},
	} {
		_, err := loadConfiguration(Input{Environ: environ})
		if !errors.Is(err, errInvalidConfiguration) || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe config error: %v", err)
		}
	}
}
