//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"github.com/davidbyttow/govips/v2/vips"
)

func TestOCRMalformedJPEGEntropyIsSourceFailure(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var source bytes.Buffer
	if err := jpeg.Encode(&source, syntheticImage(256, 192), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	encoded := source.Bytes()
	scan := bytes.Index(encoded, []byte{0xff, 0xda})
	if scan < 0 {
		t.Fatal("missing JPEG scan")
	}
	start := scan + 2 + int(binary.BigEndian.Uint16(encoded[scan+2:scan+4]))
	stuffed := bytes.Index(encoded[start:], []byte{0xff, 0x00})
	if stuffed < 0 {
		t.Fatal("fixture has no stuffed entropy byte")
	}
	encoded[start+stuffed+1] = 0x7f
	if _, err := jpeg.DecodeConfig(bytes.NewReader(encoded)); err != nil {
		t.Fatal("fixture must retain a valid header", err)
	}
	before := sha256.Sum256(encoded)
	_, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(encoded), media.FormatJPEG)
	if !errors.Is(err, ocr.ErrInvalidMedia) {
		t.Fatalf("malformed entropy = %v, want invalid media", err)
	}
	if sha256.Sum256(encoded) != before {
		t.Fatal("source changed")
	}
}

func TestOCRNativeErrorDoesNotBlameValidJPEG(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, syntheticImage(96, 64), nil); err != nil {
		t.Fatal(err)
	}
	if err := ocrNativeImageError(context.Background(), source.Bytes(), media.FormatJPEG); !errors.Is(err, ocr.ErrPreprocessFailed) {
		t.Fatalf("valid JPEG native fault = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ocrNativeImageError(ctx, source.Bytes(), media.FormatJPEG); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled classification = %v", err)
	}
}

func ocrPNG(t *testing.T, src image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, src); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func assertOCRPNG(t *testing.T, result ocr.PreparedImage, width, height int) {
	t.Helper()
	config, err := png.DecodeConfig(bytes.NewReader(result.PNG))
	if err != nil || config.Width != width || config.Height != height || result.Width != width || result.Height != height || result.ContractSHA256 != ocr.InputContractSHA256() {
		t.Fatal(config, result.Width, result.Height, err)
	}
	if result.PNG[24] != 8 || result.PNG[25] != 2 {
		t.Fatal("not RGB8 PNG", result.PNG[24:26])
	}
	for pos := 8; pos+12 <= len(result.PNG); {
		length := int(binary.BigEndian.Uint32(result.PNG[pos:]))
		if pos+12+length > len(result.PNG) {
			t.Fatal("invalid PNG chunk")
		}
		name := string(result.PNG[pos+4 : pos+8])
		switch name {
		case "eXIf", "tEXt", "zTXt", "iTXt", "iCCP":
			t.Fatal("metadata leaked", name)
		}
		pos += 12 + length
	}
}

func TestOCRImageFormatsAndLegacyResolutionRemainIndependent(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	input := ocrPNG(t, syntheticImage(3000, 1200))
	before := sha256.Sum256(input)
	result, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(input), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	assertOCRPNG(t, result, 2048, 819)
	old, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(input), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	config, _ := png.DecodeConfig(bytes.NewReader(old))
	if config.Width != 1024 {
		t.Fatal("legacy profile changed", config)
	}
	if sha256.Sum256(input) != before {
		t.Fatal("source changed")
	}
	var jpg, animated bytes.Buffer
	_ = jpeg.Encode(&jpg, syntheticImage(96, 64), nil)
	palette := color.Palette{color.Black, color.White}
	first := image.NewPaletted(image.Rect(0, 0, 96, 64), palette)
	second := image.NewPaletted(first.Bounds(), palette)
	for i := range second.Pix {
		second.Pix[i] = 1
	}
	if err := gif.EncodeAll(&animated, &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{1, 1}}); err != nil {
		t.Fatal(err)
	}
	webpImage, err := vips.LoadImageFromBuffer(ocrPNG(t, syntheticImage(96, 64)), nil)
	if err != nil {
		t.Fatal(err)
	}
	encodedWebP, _, err := webpImage.ExportWebp(vips.NewWebpExportParams())
	webpImage.Close()
	if err != nil {
		t.Fatal(err)
	}
	for format, data := range map[media.Format][]byte{media.FormatJPEG: jpg.Bytes(), media.FormatGIF: animated.Bytes(), media.FormatWebP: encodedWebP} {
		result, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(data), format)
		if err != nil {
			t.Fatal(format, err)
		}
		assertOCRPNG(t, result, 96, 64)
		if format == media.FormatGIF {
			decoded, _ := png.Decode(bytes.NewReader(result.PNG))
			r, _, _, _ := decoded.At(20, 20).RGBA()
			if r != 0 {
				t.Fatal("not first GIF frame")
			}
		}
	}
}

func TestOCRImageAlphaEXIFAndMetadata(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	transparent := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for i := 0; i < len(transparent.Pix); i += 4 {
		transparent.Pix[i] = 255
	}
	result, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(ocrPNG(t, transparent)), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	assertOCRPNG(t, result, 64, 32)
	decoded, _ := png.Decode(bytes.NewReader(result.PNG))
	r, g, b, _ := decoded.At(20, 20).RGBA()
	if r != 65535 || g != 65535 || b != 65535 {
		t.Fatal("transparent must become white", r, g, b)
	}
	var base bytes.Buffer
	_ = jpeg.Encode(&base, syntheticImage(96, 64), nil)
	exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	marker := []byte{255, 225, 0, 0}
	binary.BigEndian.PutUint16(marker[2:], uint16(len(exif)+2))
	input := append([]byte{}, base.Bytes()[:2]...)
	input = append(input, marker...)
	input = append(input, exif...)
	input = append(input, base.Bytes()[2:]...)
	result, err = New().PrepareOCRImage(context.Background(), bytes.NewReader(input), media.FormatJPEG)
	if err != nil {
		t.Fatal(err)
	}
	assertOCRPNG(t, result, 64, 96)
}

func TestOCRImageInputBoundsAndFailuresReturnNoPartialImage(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	small := ocrPNG(t, syntheticImage(64, 64))
	oversizedHeader := append([]byte{}, small...)
	binary.BigEndian.PutUint32(oversizedHeader[16:20], 10000)
	binary.BigEndian.PutUint32(oversizedHeader[20:24], 4001)
	binary.BigEndian.PutUint32(oversizedHeader[29:33], crc32.ChecksumIEEE(oversizedHeader[12:29]))
	for _, tc := range []struct {
		name   string
		source io.ReadSeeker
		format media.Format
		want   error
	}{
		{"nil", nil, media.FormatPNG, ocr.ErrInvalidImageInput}, {"unsupported", bytes.NewReader(small), media.FormatMP4, ocr.ErrInvalidImageInput},
		{"empty", bytes.NewReader(nil), media.FormatPNG, ocr.ErrInvalidMedia}, {"corrupt", bytes.NewReader([]byte("synthetic invalid")), media.FormatPNG, ocr.ErrInvalidMedia},
		{"mismatch", bytes.NewReader(small), media.FormatJPEG, ocr.ErrInvalidMedia}, {"pixels", bytes.NewReader(oversizedHeader), media.FormatPNG, ocr.ErrInputLimit},
		{"tiny", bytes.NewReader(ocrPNG(t, syntheticImage(64, 20))), media.FormatPNG, ocr.ErrInputLimit},
		{"slender", bytes.NewReader(ocrPNG(t, syntheticImage(10000, 32))), media.FormatPNG, ocr.ErrInputLimit},
		{"sourceBytes", &ocrSizeReader{Reader: bytes.NewReader(small), size: ocr.MaxSourceBytes + 1}, media.FormatPNG, ocr.ErrInputLimit},
		{"shortRead", &ocrSizeReader{Reader: bytes.NewReader(small), size: int64(len(small) + 5)}, media.FormatPNG, ocr.ErrPreprocessFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := New().PrepareOCRImage(context.Background(), tc.source, tc.format)
			if !errors.Is(err, tc.want) || result.PNG != nil || result.ContractSHA256 != "" {
				t.Fatal(result.Width, result.Height, err)
			}
		})
	}
	// A valid JPEG with more than the allowed metadata header budget is limited,
	// not labelled corrupt. No native load is needed to reject it.
	var jpg bytes.Buffer
	_ = jpeg.Encode(&jpg, syntheticImage(64, 64), nil)
	header := append([]byte{}, jpg.Bytes()[:2]...)
	for range 17 {
		header = append(header, 0xff, 0xee, 0xff, 0xff)
		header = append(header, make([]byte, 65533)...)
	}
	header = append(header, jpg.Bytes()[2:]...)
	if _, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(header), media.FormatJPEG); !errors.Is(err, ocr.ErrInputLimit) {
		t.Fatal(err)
	}
}

type ocrSizeReader struct {
	*bytes.Reader
	size int64
}

func (r *ocrSizeReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekEnd && offset == 0 {
		return r.size, nil
	}
	return r.Reader.Seek(offset, whence)
}

type ocrReadObserver struct {
	*bytes.Reader
	reads  int
	cancel context.CancelFunc
	empty  bool
}

func (r *ocrReadObserver) Read(p []byte) (int, error) {
	r.reads++
	if r.cancel != nil {
		r.cancel()
		return 0, io.ErrUnexpectedEOF
	}
	if r.empty {
		return 0, nil
	}
	return r.Reader.Read(p)
}

func TestOCRImageSharedAdmissionCancellationAndNoProgress(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	input := ocrPNG(t, syntheticImage(64, 64))
	release, err := beginLargeImageDecode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	source := &ocrReadObserver{Reader: bytes.NewReader(input)}
	_, err = New().PrepareOCRImage(ctx, source, media.FormatPNG)
	cancel()
	release()
	if !errors.Is(err, context.DeadlineExceeded) || source.reads != 0 {
		t.Fatal("did not wait on shared native admission", err, source.reads)
	}
	ctx, cancel = context.WithCancel(context.Background())
	source = &ocrReadObserver{Reader: bytes.NewReader(input), cancel: cancel}
	result, err := New().PrepareOCRImage(ctx, source, media.FormatPNG)
	cancel()
	if !errors.Is(err, context.Canceled) || source.reads != 1 || result.PNG != nil {
		t.Fatal(err, source.reads)
	}
	source = &ocrReadObserver{Reader: bytes.NewReader(input), empty: true}
	_, err = New().PrepareOCRImage(context.Background(), source, media.FormatPNG)
	if !errors.Is(err, ocr.ErrPreprocessFailed) || source.reads != 100 {
		t.Fatal(err, source.reads)
	}
	if _, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(input), media.FormatPNG); err != nil {
		t.Fatal("admission not released", err)
	}
}

func TestOCRImageLargeFileAndOutputBudget(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var big bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&big, generatedRGB{image.Rect(0, 0, 5460, 3072)}); err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/synthetic.png"
	if err := os.WriteFile(path, big.Bytes(), 0400); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	result, err := New().PrepareOCRImage(context.Background(), f, media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	assertOCRPNG(t, result, 2048, 1152)
	noise := image.NewRGBA(image.Rect(0, 0, 2048, 2048))
	state := uint32(1234)
	for i := 0; i < len(noise.Pix); i += 4 {
		for c := range 3 {
			state = state*1664525 + 1013904223
			noise.Pix[i+c] = byte(state >> 24)
		}
		noise.Pix[i+3] = 255
	}
	output, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(ocrPNG(t, noise)), media.FormatPNG)
	if !errors.Is(err, ocr.ErrInputLimit) || output.PNG != nil {
		t.Fatal("oversized PNG output accepted", len(output.PNG), err)
	}
}

func TestOCRImageProgressiveAndSixteenBit(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	src := image.NewRGBA64(image.Rect(0, 0, 96, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 96; x++ {
			src.SetRGBA64(x, y, color.RGBA64{R: 32768, G: 16384, B: 8192, A: 65535})
		}
	}
	encoded := ocrPNG(t, src)
	result, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(encoded), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	assertOCRPNG(t, result, 96, 64)
	decoded, _ := png.Decode(bytes.NewReader(result.PNG))
	r, g, b, _ := decoded.At(20, 20).RGBA()
	if r < 32000 || r > 34000 || g < 15000 || g > 17500 || b < 7000 || b > 9500 {
		t.Fatal("16-bit range lost", r, g, b)
	}
	native, err := vips.LoadImageFromBuffer(encoded, imageImportParams())
	if err != nil {
		t.Fatal(err)
	}
	params := vips.NewJpegExportParams()
	params.Interlace = true
	progressive, _, err := native.ExportJpeg(params)
	native.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(progressive, []byte{0xff, 0xc2}) {
		t.Fatal("not progressive JPEG")
	}
	result, err = New().PrepareOCRImage(context.Background(), bytes.NewReader(progressive), media.FormatJPEG)
	if err != nil {
		t.Fatal(err)
	}
	assertOCRPNG(t, result, 96, 64)
}

func TestOCRImageExactSourcePixelBoundary(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	input := ocrPNG(t, image.NewGray(image.Rect(0, 0, 6400, 6250)))
	result, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(input), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	assertOCRPNG(t, result, 2048, 2000)
}

func TestOCRPNGFilteringKeepsPixelsAndAcceptsTexturedJPEG(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	texture := image.NewNRGBA(image.Rect(0, 0, 1900, 1900))
	state := uint32(1)
	for y := range 1900 {
		for x := range 1900 {
			for c := range 3 {
				state ^= state << 13
				state ^= state >> 17
				state ^= state << 5
				texture.Pix[y*texture.Stride+x*4+c] = uint8((x/4+y/7+c*53)%240) + uint8(state%16)
			}
			texture.Pix[y*texture.Stride+x*4+3] = 255
		}
	}
	var source bytes.Buffer
	if err := jpeg.Encode(&source, texture, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	result, err := New().PrepareOCRImage(context.Background(), bytes.NewReader(source.Bytes()), media.FormatJPEG)
	if err != nil {
		t.Fatal(err)
	}
	assertOCRPNG(t, result, 1900, 1900)
	native, err := vips.LoadImageFromBuffer(source.Bytes(), imageImportParams())
	if err != nil {
		t.Fatal(err)
	}
	defer native.Close()
	if err = native.ToColorSpace(vips.InterpretationSRGB); err != nil {
		t.Fatal(err)
	}
	export := vips.NewPngExportParams()
	export.StripMetadata = true
	legacy, _, err := native.ExportPng(export)
	if err != nil {
		t.Fatal(err)
	}
	if len(legacy) <= ocr.MaxImageBytes {
		t.Fatalf("fixture does not exceed old encoding limit: %d", len(legacy))
	}
	before, err := png.Decode(bytes.NewReader(legacy))
	if err != nil {
		t.Fatal(err)
	}
	after, err := png.Decode(bytes.NewReader(result.PNG))
	if err != nil {
		t.Fatal(err)
	}
	for y := range 1900 {
		for x := range 1900 {
			if before.At(x, y) != after.At(x, y) {
				t.Fatalf("PNG pixels changed at %d,%d", x, y)
			}
		}
	}
	t.Logf("unfiltered=%d filtered=%d input_contract=%s", len(legacy), len(result.PNG), result.ContractSHA256)
}
