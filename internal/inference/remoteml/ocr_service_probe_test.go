//go:build libvips && ocrprobe

package remoteml

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

func TestOCRProducerRealService(t *testing.T) {
	origin := os.Getenv("FOLIOPATH_OCR_TEST_ORIGIN")
	mode := os.Getenv("FOLIOPATH_OCR_TEST_MODE")
	if origin == "" || (mode != "baseline" && mode != "ocr" && mode != "missing") {
		t.Fatal("explicit isolated origin/mode required")
	}
	var identities struct {
		Semantic ServiceIdentity `json:"semantic"`
		Face     ServiceIdentity `json:"face"`
		Tag      TagIdentity     `json:"tag"`
	}
	data, err := os.ReadFile("/contracts/baseline-identities.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &identities); err != nil {
		t.Fatal(err)
	}
	runtime := imagevips.NewRuntime()
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown()
	client, err := NewConfigured(origin, identities.Semantic)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := client.VerifyIdentity(ctx); err != nil {
		t.Fatal("semantic identity", err)
	}
	faceClient, err := client.BindFace(identities.Face)
	if err != nil {
		t.Fatal(err)
	}
	if err := faceClient.VerifyIdentity(ctx); err != nil {
		t.Fatal("face identity", err)
	}
	tagClient, err := client.BindTag(identities.Tag)
	if err != nil {
		t.Fatal(err)
	}
	if err := tagClient.VerifyIdentity(ctx); err != nil {
		t.Fatal("tag identity", err)
	}
	fixture, err := os.ReadFile("/fixtures/bilingual.png")
	if err != nil {
		t.Fatal(err)
	}
	legacyImage, err := imagevips.New().PrepareRemoteSemanticImage(ctx, bytes.NewReader(fixture), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	text, err := client.EncodeText(ctx, "中文 OCR 测试")
	if err != nil {
		t.Fatal("legacy text", err)
	}
	visual, err := client.EncodeImage(ctx, legacyImage)
	if err != nil {
		t.Fatal("legacy image", err)
	}
	tags, err := tagClient.Analyze(ctx, legacyImage)
	if err != nil {
		t.Fatal("legacy tags", err)
	}
	faces, err := faceClient.Detect(ctx, legacyImage)
	if err != nil {
		t.Fatal("legacy faces", err)
	}
	observation, _ := json.Marshal(struct {
		Text, Visual []float32
		Tags         []TagLabel
		Faces        []DetectedFace
	}{text, visual, tags, faces})
	digest := sha256.Sum256(observation)
	legacyDigest := hex.EncodeToString(digest[:])
	if mode == "baseline" {
		if err := os.WriteFile("/observations/baseline.json", observation, 0600); err != nil {
			t.Fatal(err)
		}
		t.Log("legacy baseline", legacyDigest)
		return
	}
	old, err := os.ReadFile("/observations/baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(old, observation) {
		t.Fatal("legacy model output changed")
	}
	var expected ocr.Identity
	data, err = os.ReadFile("/contracts/ocr-identity.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	if expected.InputContractSHA256 != ocr.InputContractSHA256() || expected.OutputContractSHA256 != ocr.OutputContractSHA256() {
		t.Fatal("producer contracts not current")
	}
	ocrClient, err := client.BindOCR(expected)
	if err != nil {
		t.Fatal(err)
	}
	if mode == "missing" {
		if err := ocrClient.VerifyIdentity(ctx); !errors.Is(err, ErrUnavailable) {
			t.Fatal("missing OCR cache must be unavailable", err)
		}
		if _, err := ocrClient.Recognize(ctx, legacyImage); !errors.Is(err, ErrUnavailable) {
			t.Fatal("must not infer without ready identity", err)
		}
		t.Log("OCR unavailable while legacy output unchanged", legacyDigest)
		return
	}
	if err := ocrClient.VerifyIdentity(ctx); err != nil {
		t.Fatal("OCR identity", err)
	}
	normalize := func(text string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, text)
	}
	cases := map[string][]string{"bilingual.png": {"FolioPath OCR 2026", "中文文字识别测试", "订单编号123456", "LOCAL TEST ONLY"}, "exif-oriented.jpg": {"FolioPath OCR 2026", "中文文字识别测试", "订单编号123456", "LOCAL TEST ONLY"}, "wide.png": {"WIDE IMAGE OCR 2048"}, "blank.png": {}}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile("/fixtures/" + name)
			if err != nil {
				t.Fatal(err)
			}
			format := media.FormatPNG
			if strings.HasSuffix(name, "jpg") {
				format = media.FormatJPEG
			}
			prepared, err := imagevips.New().PrepareOCRImage(ctx, bytes.NewReader(source), format)
			if err != nil {
				t.Fatal(err)
			}
			result, err := ocrClient.Recognize(ctx, prepared.PNG)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Lines) != len(want) {
				t.Fatal(result)
			}
			for i, line := range result.Lines {
				if normalize(line.Text) != normalize(want[i]) {
					t.Fatal(result)
				}
			}
		})
	}
	if _, err := ocrClient.Recognize(ctx, []byte("synthetic invalid PNG")); !errors.Is(err, ocr.ErrInvalidMedia) {
		t.Fatal("source failure classification", err)
	}
	t.Log("unchanged legacy digest", legacyDigest, "actual OCR identity", expected)
}
