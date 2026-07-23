package app

import (
	"context"
	"reflect"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/api"
)

func TestSystemStatusProviderUsesRuntimeAndCanonicalCapabilities(t *testing.T) {
	readiness := newReadinessState()
	provider := systemStatusProvider("v0.1.0", readiness)

	degraded, err := provider(context.Background())
	if err != nil {
		t.Fatalf("system status: %v", err)
	}
	if degraded.Version != "v0.1.0" ||
		degraded.APIVersion != "v1" ||
		degraded.RuntimeState != "degraded" ||
		degraded.Initialized ||
		!degraded.ReadOnlyMedia {
		t.Fatalf("degraded status = %#v, want safe pre-auth status", degraded)
	}
	if got, want := degraded.SupportedLocales, []string{"zh-CN", "en"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("supported locales = %q, want %q", got, want)
	}
	if got, want := degraded.SupportedMedia.ImageMIMETypes, []string{
		"image/jpeg",
		"image/png",
		"image/webp",
		"image/gif",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("image MIME types = %q, want %q", got, want)
	}
	if got, want := degraded.SupportedMedia.VideoMIMETypes, []string{
		"video/mp4",
		"video/quicktime",
		"video/x-matroska",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("video MIME types = %q, want %q", got, want)
	}
	if degraded.SupportedMedia.VideoTranscoding {
		t.Fatal("MVP status unexpectedly advertises video transcoding")
	}

	readiness.set(api.Readiness{Ready: true})
	ready, err := provider(context.Background())
	if err != nil {
		t.Fatalf("ready system status: %v", err)
	}
	if ready.RuntimeState != "ready" {
		t.Fatalf("runtime state = %q, want ready", ready.RuntimeState)
	}
}

func TestSystemStatusProviderDefaultsEmptyVersion(t *testing.T) {
	status, err := systemStatusProvider("", newReadinessState())(context.Background())
	if err != nil {
		t.Fatalf("system status: %v", err)
	}
	if status.Version != "dev" {
		t.Fatalf("version = %q, want dev", status.Version)
	}
}
