// Package ocr owns independent OCR contracts, source analysis and bounded task
// use cases. App owns composition; media/files own original filesystem access.
package ocr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	Model               = "PP-OCRv5_server"
	MaxImageBytes       = 8 << 20
	MaxLines            = 128
	MaxLineRunes        = 512
	MaxTextRunes        = 8192
	MinDetectionScore   = 0.5
	MinRecognitionScore = 0.8
)

var (
	ErrInvalidIdentity = errors.New("invalid OCR identity")
	ErrInvalidResult   = errors.New("invalid OCR result")
	ErrInputLimit      = errors.New("OCR input exceeds contract")
	ErrInvalidMedia    = errors.New("invalid OCR media input")
)

// Identity is supplied independently by trusted deployment configuration, not
// accepted on first use from an identity endpoint. Fingerprint is compatibility,
// not authentication or authorization to activate an index generation.
type Identity struct {
	Version               int    `json:"version"`
	Protocol              string `json:"protocol"`
	Model                 string `json:"model"`
	DetectorSHA256        string `json:"detector_sha256"`
	RecognizerSHA256      string `json:"recognizer_sha256"`
	InputContractSHA256   string `json:"input_contract_sha256"`
	OutputContractSHA256  string `json:"output_contract_sha256"`
	RuntimeContractSHA256 string `json:"runtime_contract_sha256"`
	CacheManifestSHA256   string `json:"cache_manifest_sha256"`
}

func (i Identity) Fingerprint() (string, error) {
	if i.Version != 1 || i.Protocol != "foliopath-ocr-v1" || i.Model != Model {
		return "", ErrInvalidIdentity
	}
	for _, value := range []string{i.DetectorSHA256, i.RecognizerSHA256, i.InputContractSHA256,
		i.OutputContractSHA256, i.RuntimeContractSHA256, i.CacheManifestSHA256} {
		if len(value) != 64 || value != strings.ToLower(value) {
			return "", ErrInvalidIdentity
		}
		if _, err := hex.DecodeString(value); err != nil {
			return "", ErrInvalidIdentity
		}
	}
	// Marshal a map so key order is canonical across language implementations.
	data, err := json.Marshal(map[string]any{
		"version": i.Version, "protocol": i.Protocol, "model": i.Model,
		"detector_sha256": i.DetectorSHA256, "recognizer_sha256": i.RecognizerSHA256,
		"input_contract_sha256": i.InputContractSHA256, "output_contract_sha256": i.OutputContractSHA256,
		"runtime_contract_sha256": i.RuntimeContractSHA256, "cache_manifest_sha256": i.CacheManifestSHA256,
	})
	if err != nil {
		return "", ErrInvalidIdentity
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

// Line contains plaintext and a convex quadrilateral in the preprocessed image's
// normalized coordinates. Scores are uncalibrated model values, not probabilities.
type Line struct {
	Text             string
	Box              [8]float64
	DetectionScore   float64
	RecognitionScore float64
}

type Result struct{ Lines []Line }

// Normalize rejects an entire invalid result instead of truncating it or
// converting corruption to a successful zero-line result. It owns NFC policy.
func Normalize(lines []Line) (Result, error) {
	if lines == nil || len(lines) > MaxLines {
		return Result{}, ErrInvalidResult
	}
	result := Result{Lines: make([]Line, 0, len(lines))}
	total := 0
	for _, line := range lines {
		if !utf8.ValidString(line.Text) || strings.TrimSpace(line.Text) == "" || utf8.RuneCountInString(line.Text) > MaxLineRunes {
			return Result{}, ErrInvalidResult
		}
		for _, r := range line.Text {
			if unicode.IsControl(r) {
				return Result{}, ErrInvalidResult
			}
		}
		line.Text = norm.NFC.String(line.Text)
		count := utf8.RuneCountInString(line.Text)
		total += count
		if count > MaxLineRunes || total > MaxTextRunes || !score(line.DetectionScore, MinDetectionScore) || !score(line.RecognitionScore, MinRecognitionScore) {
			return Result{}, ErrInvalidResult
		}
		for _, value := range line.Box {
			if !score(value, 0) {
				return Result{}, ErrInvalidResult
			}
		}
		sign := 0.0
		for n := range 4 {
			a, b, c := n*2, ((n+1)%4)*2, ((n+2)%4)*2
			cross := (line.Box[b]-line.Box[a])*(line.Box[c+1]-line.Box[b+1]) - (line.Box[b+1]-line.Box[a+1])*(line.Box[c]-line.Box[b])
			if cross == 0 || (sign != 0 && math.Signbit(cross) != math.Signbit(sign)) {
				return Result{}, ErrInvalidResult
			}
			sign = cross
		}
		result.Lines = append(result.Lines, line)
	}
	return result, nil
}

func score(value, minimum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= 1
}
