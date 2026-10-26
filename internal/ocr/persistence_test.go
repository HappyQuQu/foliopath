package ocr

import (
	"strings"
	"testing"
)

func TestPersistenceResultContractRejectsAmbiguousFailures(t *testing.T) {
	base := StoredResult{Candidate: Candidate{AssetID: 1, SourceFingerprint: "v1:10:20"}, Outcome: "ready", Width: 32, Height: 32, Result: Result{Lines: []Line{}}}
	for _, mutate := range []func(*StoredResult){
		func(v *StoredResult) { v.Outcome = "" }, func(v *StoredResult) { v.AssetID = 0 }, func(v *StoredResult) { v.SourceFingerprint = "/private/media" },
		func(v *StoredResult) { v.Width = 31 }, func(v *StoredResult) { v.Height = 2049 }, func(v *StoredResult) { v.ErrorCode = "inference_failed" }, func(v *StoredResult) { v.Result.Lines = nil },
		func(v *StoredResult) { v.Outcome = "failed"; v.ErrorCode = "invalid_media" },
	} {
		v := base
		mutate(&v)
		if _, err := ValidateStoredResult(v); err == nil {
			t.Fatal("accepted invalid persistent result")
		}
	}
	for _, code := range []string{"invalid_media", "input_limit"} {
		v := base
		v.Outcome = "failed"
		v.ErrorCode = code
		v.Width, v.Height = 0, 0
		v.Result.Lines = nil
		if _, err := ValidateStoredResult(v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ValidateStoredResult(base); err != nil {
		t.Fatal(err)
	}
}
func TestPersistenceRequestsBindEpochModelScopeAndMode(t *testing.T) {
	key, request, err := RequestHashes(1, 1, strings.Repeat("a", 64), Missing, "opaque-key")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []struct {
		lib, epoch  int64
		fingerprint string
		mode        Mode
	}{{2, 1, strings.Repeat("a", 64), Missing}, {1, 2, strings.Repeat("a", 64), Missing}, {1, 1, strings.Repeat("b", 64), Missing}, {1, 1, strings.Repeat("a", 64), RetryFailed}} {
		otherKey, otherRequest, err := RequestHashes(v.lib, v.epoch, v.fingerprint, v.mode, "opaque-key")
		if err != nil || key != otherKey || request == otherRequest {
			t.Fatal("scope collision", err)
		}
	}
	for _, key := range []string{"short", "a b c d e", "\x00opaque-key", strings.Repeat("x", 129), "不透明key"} {
		if ValidOpaque(key) {
			t.Fatal("invalid opaque key")
		}
	}
}
