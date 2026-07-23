package scanner

import "testing"

func TestClassifyPathUsesFixedMVPFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path      string
		supported bool
		kind      AssetKind
		format    MediaFormat
		mime      string
	}{
		{path: "photo.JPG", supported: true, kind: AssetKindImage, format: MediaFormatJPEG, mime: "image/jpeg"},
		{path: "photo.jpeg", supported: true, kind: AssetKindImage, format: MediaFormatJPEG, mime: "image/jpeg"},
		{path: "photo.png", supported: true, kind: AssetKindImage, format: MediaFormatPNG, mime: "image/png"},
		{path: "photo.webp", supported: true, kind: AssetKindImage, format: MediaFormatWebP, mime: "image/webp"},
		{path: "animation.gif", supported: true, kind: AssetKindAnimated, format: MediaFormatGIF, mime: "image/gif"},
		{path: "clip.mp4", supported: true, kind: AssetKindVideo, format: MediaFormatMP4, mime: "video/mp4"},
		{path: "clip.mov", supported: true, kind: AssetKindVideo, format: MediaFormatMOV, mime: "video/quicktime"},
		{path: "clip.MKV", supported: true, kind: AssetKindVideo, format: MediaFormatMKV, mime: "video/x-matroska"},
		{path: "photo.heic", supported: false},
		{path: "photo.heif", supported: false},
		{path: "photo.avif", supported: false},
		{path: "drawing.svg", supported: false},
		{path: "raw.dng", supported: false},
		{path: "raw.cr3", supported: false},
		{path: "raw.nef", supported: false},
		{path: "raw.arw", supported: false},
		{path: "raw.raf", supported: false},
		{path: "raw.rw2", supported: false},
		{path: "no-extension", supported: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			t.Parallel()
			kind, format, mime, supported := ClassifyPath(test.path)
			if supported != test.supported ||
				kind != test.kind ||
				format != test.format ||
				mime != test.mime {
				t.Fatalf(
					"ClassifyPath(%q) = (%q, %q, %q, %v), want (%q, %q, %q, %v)",
					test.path,
					kind,
					format,
					mime,
					supported,
					test.kind,
					test.format,
					test.mime,
					test.supported,
				)
			}
		})
	}
}

func TestSystemDirectoryList(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"@eaDir", ".thumbnails", "__MACOSX", "$RECYCLE.BIN"} {
		if !IsSystemDirectory(name) {
			t.Fatalf("expected %q to be skipped", name)
		}
	}
	if IsSystemDirectory(".private-photos") {
		t.Fatal("ordinary hidden directories must remain scannable")
	}
}
