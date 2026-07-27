package cursor

import (
	"errors"
	"strings"
	"testing"
)

type testPayload struct {
	Version int    `json:"v"`
	Value   string `json:"s"`
}

func TestCodecRoundTripAndBinding(t *testing.T) {
	t.Parallel()

	codec, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := codec.Encode(testPayload{Version: 1, Value: "media"}, "resource:a")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(encoded, "media") {
		t.Fatal("opaque cursor exposed its payload")
	}

	var decoded testPayload
	if err := codec.Decode(encoded, "resource:a", &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != (testPayload{Version: 1, Value: "media"}) {
		t.Fatalf("unexpected decoded payload: %#v", decoded)
	}
	if err := codec.Decode(encoded, "resource:b", &decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong binding error = %v, want ErrInvalid", err)
	}
}

func TestCodecRejectsTamperingAndInvalidKey(t *testing.T) {
	t.Parallel()

	if _, err := New([]byte("short")); err == nil {
		t.Fatal("New accepted a non-32-byte key")
	}
	codec, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := codec.Encode(testPayload{Version: 1}, "resource")
	if err != nil {
		t.Fatal(err)
	}
	replacement := "A"
	if encoded[0] == 'A' {
		replacement = "B"
	}
	tampered := replacement + encoded[1:]
	var decoded testPayload
	if err := codec.Decode(tampered, "resource", &decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("tampered token error = %v, want ErrInvalid", err)
	}
}
