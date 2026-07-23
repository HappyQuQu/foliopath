package media

import (
	"reflect"
	"testing"
)

func TestSupportedMIMETypesMatchMVPContract(t *testing.T) {
	if got, want := ImageMIMETypes(), []string{
		"image/jpeg",
		"image/png",
		"image/webp",
		"image/gif",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ImageMIMETypes() = %q, want %q", got, want)
	}
	if got, want := VideoMIMETypes(), []string{
		"video/mp4",
		"video/quicktime",
		"video/x-matroska",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("VideoMIMETypes() = %q, want %q", got, want)
	}
}

func TestSupportedMIMETypesReturnOwnedCopies(t *testing.T) {
	images := ImageMIMETypes()
	images[0] = "modified"
	if ImageMIMETypes()[0] != "image/jpeg" {
		t.Fatal("caller mutation changed the canonical image MIME registry")
	}

	videos := VideoMIMETypes()
	videos[0] = "modified"
	if VideoMIMETypes()[0] != "video/mp4" {
		t.Fatal("caller mutation changed the canonical video MIME registry")
	}
}
