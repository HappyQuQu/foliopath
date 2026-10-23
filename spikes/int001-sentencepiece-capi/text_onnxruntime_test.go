//go:build linux && cgo && sentencepiece && onnxruntime

package sentencepiececapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"math"
	"os"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

var textModelPath = flag.String("text-onnx-model", "", "fixed SigLIP text_encoder.onnx")

type textEmbeddingReference struct {
	SchemaVersion int `json:"schema_version"`
	Generator     struct {
		ONNXRuntime string `json:"onnxruntime"`
		NumPy       string `json:"numpy"`
	} `json:"generator"`
	Graph struct {
		SizeBytes int64  `json:"size_bytes"`
		SHA256    string `json:"sha256"`
	} `json:"graph"`
	TokenFixture struct {
		SHA256 string `json:"sha256"`
	} `json:"token_fixture"`
	Cases []struct {
		Name          string  `json:"name"`
		Float32Base64 string  `json:"float32_le_base64"`
		SHA256        string  `json:"sha256"`
		L2Norm        float64 `json:"l2_norm"`
	} `json:"cases"`
}

func TestPinnedSigLIPTokenizerToTextEncoderParity(t *testing.T) {
	if *modelPath == "" || *textModelPath == "" {
		t.Fatal("sentencepiece-model and text-onnx-model are required")
	}
	tokenFixtureBytes, err := os.ReadFile("testdata/siglip-tokenizer-reference-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var tokenFixture tokenizerReferenceFixture
	if err := json.Unmarshal(tokenFixtureBytes, &tokenFixture); err != nil {
		t.Fatal(err)
	}
	embeddingBytes, err := os.ReadFile("testdata/siglip-text-embedding-reference-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if digest := sha256.Sum256(embeddingBytes); hex.EncodeToString(digest[:]) != "943c05755587be5092570063c8dcadf910fc6ba06dd6e917f285b38e68f40225" {
		t.Fatal("embedding fixture hash differs from pinned contract")
	}
	var reference textEmbeddingReference
	if err := json.Unmarshal(embeddingBytes, &reference); err != nil {
		t.Fatal(err)
	}
	tokenDigest := sha256.Sum256(tokenFixtureBytes)
	if reference.SchemaVersion != 1 || reference.Generator.ONNXRuntime != "1.29.0" || reference.Generator.NumPy != "2.5.2" ||
		reference.TokenFixture.SHA256 != hex.EncodeToString(tokenDigest[:]) || len(reference.Cases) != len(tokenFixture.Cases) {
		t.Fatal("embedding fixture metadata differs from pinned contract")
	}
	graphDigest, graphSize := hashFile(t, *textModelPath)
	if graphSize != reference.Graph.SizeBytes || graphDigest != reference.Graph.SHA256 {
		t.Fatal("text graph differs from pinned reference")
	}
	tokenizer, err := open(*modelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenizer.close()
	session, err := openTextSession(context.Background(), *textModelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer session.close()
	queries := make(map[string]string, len(tokenFixture.Cases))
	for _, item := range tokenFixture.Cases {
		queries[item.Name] = item.Query
	}
	maximumDifference := float64(0)
	for _, item := range reference.Cases {
		query, exists := queries[item.Name]
		if !exists {
			t.Fatalf("embedding case %q has no token fixture", item.Name)
		}
		ids, err := tokenizer.encode(query)
		if err != nil {
			t.Fatalf("tokenize %q: %v", item.Name, err)
		}
		actual, err := session.encode(context.Background(), ids)
		if err != nil {
			t.Fatalf("encode %q: %v", item.Name, err)
		}
		expected := decodeFloat32Reference(t, item.Float32Base64, item.SHA256)
		for index, want := range expected {
			got := actual[index]
			difference := math.Abs(float64(got - want))
			if difference > maximumDifference {
				maximumDifference = difference
			}
			if math.IsNaN(float64(got)) || math.IsInf(float64(got), 0) || difference > 1e-4+1e-4*math.Abs(float64(want)) {
				t.Fatalf("%s[%d] got=%g want=%g difference=%g", item.Name, index, got, want, difference)
			}
		}
	}
	t.Logf("cases=%d dimensions=%d max_abs_difference=%g", len(reference.Cases), textEmbeddingDimension, maximumDifference)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := session.encode(ctx, [64]int64{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancel error = %v", err)
	}
	session.close()
	if _, err := session.encode(context.Background(), [64]int64{}); !errors.Is(err, errTextRuntime) {
		t.Fatalf("closed session error = %v", err)
	}
}

func TestPinnedSigLIPTextSessionConcurrencyCancellationAndReuse(t *testing.T) {
	if *modelPath == "" || *textModelPath == "" {
		t.Fatal("sentencepiece-model and text-onnx-model are required")
	}
	tokenizer, err := open(*modelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenizer.close()
	ids, err := tokenizer.encode("红色盔甲 portrait")
	if err != nil {
		t.Fatal(err)
	}
	session, err := openTextSession(context.Background(), *textModelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer session.close()

	ctx, cancel := context.WithCancel(context.Background())
	cancelled := make(chan error, 1)
	go func() {
		_, err := session.encode(ctx, ids)
		cancelled <- err
	}()
	deadline := time.Now().Add(2 * time.Second)
	for !session.running() && time.Now().Before(deadline) {
		time.Sleep(100 * time.Microsecond)
	}
	if !session.running() {
		cancel()
		t.Fatal("text run did not enter native inference")
	}
	cancel()
	if err := <-cancelled; !errors.Is(err, context.Canceled) {
		t.Fatalf("active-run cancellation error = %v", err)
	}

	errorsSeen := make(chan error, 8)
	var group sync.WaitGroup
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			output, err := session.encode(context.Background(), ids)
			if err != nil || !finiteNonzero(output[:]) {
				errorsSeen <- errors.Join(err, errTextRuntime)
			}
		}()
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatal(err)
	}
}

func TestPinnedSigLIPTextSessionRepeatedLoadCloseRSS(t *testing.T) {
	if *textModelPath == "" {
		t.Fatal("text-onnx-model is required")
	}
	loadClose := func(cycles int) {
		t.Helper()
		for cycle := 0; cycle < cycles; cycle++ {
			session, err := openTextSession(context.Background(), *textModelPath)
			if err != nil {
				t.Fatalf("cycle %d: %v", cycle, err)
			}
			session.close()
		}
	}
	debug.FreeOSMemory()
	cold := linuxResidentBytes(t)
	loadClose(10)
	debug.FreeOSMemory()
	warmed := linuxResidentBytes(t)
	loadClose(10)
	debug.FreeOSMemory()
	after := linuxResidentBytes(t)
	increase := after - warmed
	if increase < 0 {
		increase = 0
	}
	t.Logf("text session resident bytes cold=%d warmed=%d after=%d measured_increase=%d cold_to_stable=%d", cold, warmed, after, increase, after-cold)
	if increase > 128<<20 {
		t.Fatalf("10 measured text session load/close cycles retained %d bytes after 10 warm-up cycles", increase)
	}
}

func finiteNonzero(values []float32) bool {
	nonzero := false
	for _, value := range values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return false
		}
		nonzero = nonzero || value != 0
	}
	return nonzero
}

func hashFile(t *testing.T, path string) (string, int64) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(hash.Sum(nil)), size
}

func decodeFloat32Reference(t *testing.T, encoded, expectedHash string) [textEmbeddingDimension]float32 {
	t.Helper()
	var result [textEmbeddingDimension]float32
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) != len(result)*4 {
		t.Fatalf("invalid embedding reference length=%d err=%v", len(raw), err)
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != expectedHash {
		t.Fatal("embedding case hash differs")
	}
	for index := range result {
		result[index] = math.Float32frombits(binary.LittleEndian.Uint32(raw[index*4:]))
	}
	return result
}
