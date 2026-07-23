package fs03vips

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

func TestMain(m *testing.M) {
	vips.Startup(&vips.Config{
		ConcurrencyLevel: 1,
		MaxCacheFiles:    0,
		MaxCacheMem:      32 << 20,
		MaxCacheSize:     32,
		ReportLeaks:      true,
	})
	code := m.Run()
	vips.Shutdown()
	os.Exit(code)
}

func TestStillImageFormatsAndThumbnailBounds(t *testing.T) {
	source := syntheticImage(96, 64)

	tests := []struct {
		name      string
		export    func(*vips.ImageRef) ([]byte, *vips.ImageMetadata, error)
		wantAlpha bool
	}{
		{
			name: "jpeg",
			export: func(ref *vips.ImageRef) ([]byte, *vips.ImageMetadata, error) {
				return ref.ExportJpeg(&vips.JpegExportParams{Quality: 85})
			},
		},
		{
			name: "png",
			export: func(ref *vips.ImageRef) ([]byte, *vips.ImageMetadata, error) {
				return ref.ExportPng(vips.NewPngExportParams())
			},
			wantAlpha: true,
		},
		{
			name: "webp",
			export: func(ref *vips.ImageRef) ([]byte, *vips.ImageMetadata, error) {
				return ref.ExportWebp(&vips.WebpExportParams{Quality: 85})
			},
			wantAlpha: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := vips.NewImageFromGoImage(source)
			if err != nil {
				t.Fatalf("create source: %v", err)
			}
			defer ref.Close()

			encoded, metadata, err := tt.export(ref)
			if err != nil {
				t.Fatalf("export: %v", err)
			}
			if metadata.Width != 96 || metadata.Height != 64 {
				t.Fatalf("export metadata = %dx%d, want 96x64", metadata.Width, metadata.Height)
			}

			loaded, err := vips.NewImageFromBuffer(encoded)
			if err != nil {
				t.Fatalf("load exported image: %v", err)
			}
			defer loaded.Close()
			if loaded.Width() != 96 || loaded.Height() != 64 {
				t.Fatalf("loaded dimensions = %dx%d, want 96x64", loaded.Width(), loaded.Height())
			}
			if loaded.HasAlpha() != tt.wantAlpha {
				t.Fatalf("HasAlpha() = %t, want %t", loaded.HasAlpha(), tt.wantAlpha)
			}

			thumbnail, err := vips.NewThumbnailFromBuffer(encoded, 48, 32, vips.InterestingCentre)
			if err != nil {
				t.Fatalf("thumbnail: %v", err)
			}
			defer thumbnail.Close()
			if thumbnail.Width() > 48 || thumbnail.Height() > 32 {
				t.Fatalf("thumbnail dimensions = %dx%d, exceed 48x32", thumbnail.Width(), thumbnail.Height())
			}
		})
	}
}

func TestOrientationCanBeAppliedAndRemoved(t *testing.T) {
	ref, err := vips.NewImageFromGoImage(syntheticImage(96, 64))
	if err != nil {
		t.Fatal(err)
	}
	defer ref.Close()
	if err := ref.SetOrientation(6); err != nil {
		t.Fatalf("set orientation: %v", err)
	}
	if err := ref.AutoRotate(); err != nil {
		t.Fatalf("auto rotate: %v", err)
	}
	if ref.Width() != 64 || ref.Height() != 96 {
		t.Fatalf("rotated dimensions = %dx%d, want 64x96", ref.Width(), ref.Height())
	}
	if orientation := ref.GetOrientation(); orientation != 1 && orientation != 0 {
		t.Fatalf("orientation after autorotate = %d, want normalized", orientation)
	}
}

func TestAnimatedGIFMetadataAndFirstFramePolicy(t *testing.T) {
	var encoded bytes.Buffer
	frames := make([]*image.Paletted, 4)
	delays := make([]int, len(frames))
	palette := color.Palette{color.Black, color.White, color.RGBA{R: 255, A: 255}}
	for i := range frames {
		frame := image.NewPaletted(image.Rect(0, 0, 96, 64), palette)
		for y := 0; y < 64; y++ {
			for x := 0; x < 96; x++ {
				frame.SetColorIndex(x, y, uint8((x+y+i)%len(palette)))
			}
		}
		frames[i] = frame
		delays[i] = 5
	}
	if err := gif.EncodeAll(&encoded, &gif.GIF{Image: frames, Delay: delays, LoopCount: 0}); err != nil {
		t.Fatal(err)
	}

	ref, err := vips.NewImageFromBuffer(encoded.Bytes())
	if err != nil {
		t.Fatalf("load animated GIF: %v", err)
	}
	defer ref.Close()
	if ref.Pages() != 4 {
		t.Fatalf("Pages() = %d, want 4", ref.Pages())
	}

	params := vips.NewImportParams()
	params.NumPages.Set(1)
	firstFrame, err := vips.LoadImageFromBuffer(encoded.Bytes(), params)
	if err != nil {
		t.Fatalf("load first frame: %v", err)
	}
	defer firstFrame.Close()
	if firstFrame.Width() != 96 || firstFrame.Height() != 64 {
		t.Fatalf("first frame dimensions = %dx%d, want 96x64", firstFrame.Width(), firstFrame.Height())
	}
}

func TestTruncatedImageIsRejected(t *testing.T) {
	ref, err := vips.NewImageFromBuffer([]byte("\x89PNG\r\n\x1a\ntruncated"))
	if ref != nil {
		ref.Close()
	}
	if err == nil {
		t.Fatal("truncated PNG unexpectedly loaded")
	}
}

func syntheticImage(width, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: uint8((x + y) % 256),
				A: uint8(64 + (x+y)%192),
			})
		}
	}
	return img
}
