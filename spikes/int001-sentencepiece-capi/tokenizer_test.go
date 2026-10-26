//go:build linux && cgo && sentencepiece

package sentencepiececapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

var modelPath = flag.String("sentencepiece-model", "", "fixed SigLIP spiece.model")

type tokenizerReferenceFixture struct {
	SchemaVersion int `json:"schema_version"`
	Generator     struct {
		Python        string `json:"python"`
		Transformers  string `json:"transformers"`
		SentencePiece string `json:"sentencepiece"`
	} `json:"generator"`
	Model struct {
		ID        string `json:"id"`
		Revision  string `json:"revision"`
		Filename  string `json:"filename"`
		SizeBytes int64  `json:"size_bytes"`
		SHA256    string `json:"sha256"`
	} `json:"model"`
	Contract struct {
		SequenceLength int   `json:"sequence_length"`
		EOSID          int64 `json:"eos_id"`
		PadID          int64 `json:"pad_id"`
	} `json:"contract"`
	Cases []struct {
		Name      string  `json:"name"`
		Query     string  `json:"query"`
		Canonical string  `json:"canonical"`
		InputIDs  []int64 `json:"input_ids"`
	} `json:"cases"`
}

func TestPinnedSigLIPReferenceFixture(t *testing.T) {
	if *modelPath == "" {
		t.Fatal("sentencepiece-model is required")
	}
	fixtureBytes, err := os.ReadFile("testdata/siglip-tokenizer-reference-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture tokenizerReferenceFixture
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != 1 || fixture.Generator.Transformers != "4.56.2" ||
		fixture.Generator.SentencePiece != "0.2.1" || fixture.Model.ID != "google/siglip-base-patch16-224" ||
		fixture.Model.Revision != "7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed" ||
		fixture.Contract.SequenceLength != semantic.TextSequenceLength ||
		fixture.Contract.EOSID != semantic.SigLIPEOSTokenID || fixture.Contract.PadID != semantic.SigLIPPadTokenID ||
		len(fixture.Cases) < 30 {
		t.Fatalf("fixture metadata does not match pinned contract: %+v", fixture)
	}
	modelBytes, err := os.ReadFile(*modelPath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(modelBytes)
	if int64(len(modelBytes)) != fixture.Model.SizeBytes || hex.EncodeToString(digest[:]) != fixture.Model.SHA256 {
		t.Fatal("model bytes do not match reference fixture")
	}
	tokenizer, err := open(*modelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenizer.close()
	seen := make(map[string]struct{}, len(fixture.Cases))
	for _, test := range fixture.Cases {
		if _, exists := seen[test.Name]; exists || test.Name == "" {
			t.Fatalf("duplicate or empty fixture case name %q", test.Name)
		}
		seen[test.Name] = struct{}{}
		canonical, err := semantic.CanonicalizeQuery(test.Query)
		if err != nil || canonical != test.Canonical {
			t.Errorf("canonicalize %q = %q, %v; want %q", test.Name, canonical, err, test.Canonical)
			continue
		}
		if len(test.InputIDs) != semantic.TextSequenceLength {
			t.Errorf("fixture %q has %d IDs", test.Name, len(test.InputIDs))
			continue
		}
		ids, err := tokenizer.encode(test.Query)
		if err != nil || !slices.Equal(ids[:], test.InputIDs) {
			t.Errorf("encode %q = %v, %v; want %v", test.Name, ids, err, test.InputIDs)
		}
	}
}

func TestPinnedSigLIPLoadsFromOpenFileDescriptor(t *testing.T) {
	if *modelPath == "" {
		t.Fatal("sentencepiece-model is required")
	}
	file, err := os.Open(*modelPath)
	if err != nil {
		t.Fatal(err)
	}
	tokenizer, err := openFile(file)
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	defer tokenizer.close()
	ids, err := tokenizer.encode("红色 portrait")
	if err != nil || ids[0] == 0 || ids[len(ids)-1] != 1 {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}

func TestPinnedSigLIPConcurrentEncodeAndClosedHandle(t *testing.T) {
	if *modelPath == "" {
		t.Fatal("sentencepiece-model is required")
	}
	tokenizer, err := open(*modelPath)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	errorsSeen := make(chan error, 32)
	for index := 0; index < 32; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			ids, err := tokenizer.encode("并发 concurrent portrait")
			if err != nil || ids[len(ids)-1] != 1 {
				errorsSeen <- errors.Join(err, errTokenizer)
			}
		}()
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatal(err)
	}
	tokenizer.close()
	tokenizer.close()
	if _, err := tokenizer.encode("closed"); !errors.Is(err, errTokenizer) {
		t.Fatalf("closed encode error = %v", err)
	}
}

func TestPinnedSigLIPCancellationAndRepeatedLoadClose(t *testing.T) {
	if *modelPath == "" {
		t.Fatal("sentencepiece-model is required")
	}
	loadClose := func(cycles int) {
		t.Helper()
		for cycle := 0; cycle < cycles; cycle++ {
			tokenizer, err := open(*modelPath)
			if err != nil {
				t.Fatalf("cycle %d: %v", cycle, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := tokenizer.encodeContext(ctx, "cancelled query"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cycle %d cancellation error = %v", cycle, err)
			}
			tokenizer.close()
		}
	}
	loadClose(10)
	debug.FreeOSMemory()
	before := linuxResidentBytes(t)
	loadClose(100)
	debug.FreeOSMemory()
	after := linuxResidentBytes(t)
	increase := after - before
	if increase < 0 {
		increase = 0
	}
	t.Logf("resident bytes before=%d after=%d retained_increase=%d", before, after, increase)
	if increase > 64<<20 {
		t.Fatalf("100 load/close cycles retained %d bytes", increase)
	}
}

func linuxResidentBytes(t *testing.T) int64 {
	t.Helper()
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		t.Fatalf("invalid statm: %q", data)
	}
	residentPages, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return residentPages * int64(os.Getpagesize())
}

func TestPinnedSigLIPPreCancelledContextDoesNotEnterNativeEncode(t *testing.T) {
	if *modelPath == "" {
		t.Fatal("sentencepiece-model is required")
	}
	tokenizer, err := open(*modelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenizer.close()
	for cycle := 0; cycle < 100; cycle++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := tokenizer.encodeContext(ctx, "cancelled query"); !errors.Is(err, context.Canceled) {
			t.Fatalf("cycle %d cancellation error = %v", cycle, err)
		}
	}
}

func TestRejectsMalformedTruncatedAndNonRegularModels(t *testing.T) {
	for name, content := range map[string][]byte{
		"empty.model":     {},
		"truncated.model": {0x0a, 0x04, 0x74},
		"oversized.model": make([]byte, maximumSentencePieceModelBytes+1),
	} {
		path := t.TempDir() + "/" + name
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		tokenizer, openErr := openFile(file)
		_ = file.Close()
		if openErr == nil {
			tokenizer.close()
			t.Fatalf("%s was accepted", name)
		}
	}
	directory, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	if tokenizer, err := openFile(directory); err == nil {
		tokenizer.close()
		t.Fatal("directory model was accepted")
	}
}

func TestPinnedSigLIPTruncatesAndTerminatesAtFixedLength(t *testing.T) {
	if *modelPath == "" {
		t.Fatal("sentencepiece-model is required")
	}
	tokenizer, err := open(*modelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenizer.close()

	ids, err := tokenizer.encode(strings.Repeat("🦊", 512))
	if err != nil {
		t.Fatal(err)
	}
	if ids[len(ids)-1] != 1 {
		t.Fatalf("final token = %d, want EOS/pad 1", ids[len(ids)-1])
	}
	for index, id := range ids[:len(ids)-1] {
		if id == 1 {
			t.Fatalf("token %d was padded before the 63-token truncation boundary", index)
		}
	}
}
