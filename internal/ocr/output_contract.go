package ocr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// OutputContractJSON is the canonical producer policy. Generated Python inputs
// consume these bytes; neither language owns a separate editable policy catalog.
func OutputContractJSON() string {
	data, err := json.Marshal(map[string]any{
		"profile": "ocr-horizontal-output-v2", "max_lines": MaxLines, "max_line_runes": MaxLineRunes, "max_text_runes": MaxTextRunes,
		"min_detection": MinDetectionScore, "min_recognition": MinRecognitionScore, "normalization": "NFC",
		"box": "normalized-convex-quad", "reading_order": "horizontal-row-left-v1", "rotation_detection": false,
		"max_contours": 4096, "max_crop_pixels": 16 * 1024 * 1024, "detector_resolution": 736, "recognition_batch": 1,
	})
	if err != nil {
		panic("static OCR output contract invalid")
	}
	return string(data)
}

func OutputContractSHA256() string {
	digest := sha256.Sum256([]byte(OutputContractJSON()))
	return hex.EncodeToString(digest[:])
}
