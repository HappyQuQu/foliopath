// Synthetic-only input bridge. Never composed into the application.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

func main() {
	runtime := imagevips.NewRuntime()
	if err := runtime.Start(); err != nil {
		panic("native runtime unavailable")
	}
	defer runtime.Shutdown()
	results := []map[string]any{}
	for _, fixture := range []struct {
		name          string
		format        media.Format
		width, height int
	}{{"bilingual.png", media.FormatPNG, 1500, 720}, {"exif-oriented.jpg", media.FormatJPEG, 1500, 720}, {"wide.png", media.FormatPNG, 2048, 410}} {
		path := "/fixtures/" + fixture.name
		source, err := os.Open(path)
		if err != nil {
			panic("synthetic fixture missing")
		}
		before, err := os.ReadFile(path)
		if err != nil {
			panic("synthetic fixture unreadable")
		}
		result, err := imagevips.New().PrepareOCRImage(context.Background(), source, fixture.format)
		source.Close()
		if err != nil || result.Width != fixture.width || result.Height != fixture.height || result.ContractSHA256 != ocr.InputContractSHA256() {
			panic(fmt.Sprintf("fixture transform failed: %s %v %dx%d", fixture.name, err, result.Width, result.Height))
		}
		after, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
			panic("synthetic source modified")
		}
		if err := os.WriteFile("/output/"+fixture.name, result.PNG, 0600); err != nil {
			panic("test output unavailable")
		}
		digest := sha256.Sum256(result.PNG)
		results = append(results, map[string]any{"fixture": fixture.name, "width": result.Width, "height": result.Height, "pngSHA256": hex.EncodeToString(digest[:]), "bytes": len(result.PNG), "sourceUnchanged": true})
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "passed", "inputContractSHA256": ocr.InputContractSHA256(), "cases": results, "scope": "synthetic input bridge only; no production OCR producer/index"})
}
