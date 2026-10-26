package ocr

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
)

func identityFixture() Identity {
	return Identity{Version: 1, Protocol: "foliopath-ocr-v1", Model: Model, DetectorSHA256: strings.Repeat("a", 64), RecognizerSHA256: strings.Repeat("b", 64),
		InputContractSHA256: strings.Repeat("c", 64), OutputContractSHA256: strings.Repeat("d", 64), RuntimeContractSHA256: strings.Repeat("e", 64), CacheManifestSHA256: strings.Repeat("f", 64)}
}

func lineFixture() Line {
	return Line{Text: "中文 Café 2026", Box: [8]float64{.1, .1, .9, .1, .9, .3, .1, .3}, DetectionScore: .9, RecognitionScore: .95}
}

func TestIdentityFingerprintCanonicalAndIndependent(t *testing.T) {
	i := identityFixture()
	fingerprint, err := i.Fingerprint()
	if err != nil || fingerprint != "b5f6e2e3042a9744bedc37ea4c5bccf2255123bf1f8fbd580571773ee892969e" {
		t.Fatal(fingerprint, err)
	}
	// The golden value is independently produced from sorted JSON in Python.
	for name, mutate := range map[string]func(*Identity){
		"detector":   func(i *Identity) { i.DetectorSHA256 = strings.Repeat("0", 64) },
		"recognizer": func(i *Identity) { i.RecognizerSHA256 = strings.Repeat("0", 64) },
		"input":      func(i *Identity) { i.InputContractSHA256 = strings.Repeat("0", 64) },
		"output":     func(i *Identity) { i.OutputContractSHA256 = strings.Repeat("0", 64) },
		"runtime":    func(i *Identity) { i.RuntimeContractSHA256 = strings.Repeat("0", 64) },
		"cache":      func(i *Identity) { i.CacheManifestSHA256 = strings.Repeat("0", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			changed := i
			mutate(&changed)
			other, err := changed.Fingerprint()
			if err != nil || other == fingerprint {
				t.Fatal(other, err)
			}
		})
	}
	for _, mutate := range []func(*Identity){func(i *Identity) { i.Version = 2 }, func(i *Identity) { i.Protocol = "foliopath-ml-v1" }, func(i *Identity) { i.Model = "JoyTag" }, func(i *Identity) { i.InputContractSHA256 = "" }, func(i *Identity) { i.DetectorSHA256 = strings.Repeat("A", 64) }, func(i *Identity) { i.OutputContractSHA256 = strings.Repeat("z", 64) }} {
		changed := i
		mutate(&changed)
		if _, err := changed.Fingerprint(); !errors.Is(err, ErrInvalidIdentity) {
			t.Fatal(err)
		}
	}
	data, _ := json.Marshal(i)
	var restored Identity
	_ = json.Unmarshal(data, &restored)
	other, _ := restored.Fingerprint()
	if other != fingerprint {
		t.Fatal("identity round trip")
	}
}

func TestNormalizePlaintextEmptyAndNFC(t *testing.T) {
	result, err := Normalize([]Line{})
	if err != nil || result.Lines == nil || len(result.Lines) != 0 {
		t.Fatal(result, err)
	}
	line := lineFixture()
	line.Text = "<b>中文 Cafe\u0301 😀</b>"
	result, err = Normalize([]Line{line})
	if err != nil || result.Lines[0].Text != "<b>中文 Café 😀</b>" || line.Text == result.Lines[0].Text {
		t.Fatal(result, err)
	}
}

func TestNormalizeRejectsWholeInvalidResult(t *testing.T) {
	for name, mutate := range map[string]func(*Line){
		"empty": func(l *Line) { l.Text = "" }, "whitespace": func(l *Line) { l.Text = " \t" }, "control": func(l *Line) { l.Text = "text\x00" },
		"invalidUTF8": func(l *Line) { l.Text = string([]byte{0xff}) }, "long": func(l *Line) { l.Text = strings.Repeat("字", MaxLineRunes+1) },
		"expandedNFC": func(l *Line) { l.Text = strings.Repeat("\u0344", MaxLineRunes) },
		"detection":   func(l *Line) { l.DetectionScore = .49 }, "recognition": func(l *Line) { l.RecognitionScore = .79 },
		"nan": func(l *Line) { l.Box[0] = math.NaN() }, "inf": func(l *Line) { l.RecognitionScore = math.Inf(1) },
		"outside": func(l *Line) { l.Box[0] = -1 }, "emptyBox": func(l *Line) { l.Box = [8]float64{} },
		"bowtie":  func(l *Line) { l.Box = [8]float64{.1, .1, .9, .3, .9, .1, .1, .3} },
		"concave": func(l *Line) { l.Box = [8]float64{0, 0, 1, 0, .2, .2, 0, 1} },
	} {
		t.Run(name, func(t *testing.T) {
			good := lineFixture()
			bad := good
			mutate(&bad)
			result, err := Normalize([]Line{good, bad})
			if !errors.Is(err, ErrInvalidResult) || result.Lines != nil {
				t.Fatal(result, err)
			}
		})
	}
	if _, err := Normalize(nil); !errors.Is(err, ErrInvalidResult) {
		t.Fatal(err)
	}
	for _, n := range []int{MaxLines + 1, 17} {
		lines := make([]Line, n)
		for i := range lines {
			lines[i] = lineFixture()
			if n == 17 {
				lines[i].Text = strings.Repeat("a", MaxLineRunes)
			}
		}
		if _, err := Normalize(lines); !errors.Is(err, ErrInvalidResult) {
			t.Fatal(n, err)
		}
	}
}

func TestNormalizeExactTextBudgetAndClockwiseBox(t *testing.T) {
	lines := make([]Line, 16)
	for i := range lines {
		lines[i] = lineFixture()
		lines[i].Text = strings.Repeat("字", MaxLineRunes)
		lines[i].Box = [8]float64{.1, .1, .1, .3, .9, .3, .9, .1}
	}
	result, err := Normalize(lines)
	if err != nil || len(result.Lines) != 16 {
		t.Fatal(result, err)
	}
}
