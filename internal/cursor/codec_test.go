package cursor

import (
	"bytes"
	"encoding/base64"
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

func TestCodecRejectsNonCanonicalBase64Alias(t *testing.T) {
	t.Parallel()

	codec, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	var encoded string
	for padding := 0; ; padding++ {
		encoded, err = codec.Encode(testPayload{Version: 1, Value: strings.Repeat("x", padding)}, "resource")
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded)%4 == 2 || len(encoded)%4 == 3 {
			break
		}
	}
	alias := nonCanonicalRawURLAlias(t, encoded)
	canonicalBytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	aliasBytes, err := base64.RawURLEncoding.DecodeString(alias)
	if err != nil || !bytes.Equal(aliasBytes, canonicalBytes) {
		t.Fatalf("test alias did not decode to canonical bytes: %v", err)
	}
	var decoded testPayload
	if err := codec.Decode(alias, "resource", &decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("non-canonical alias error = %v, want ErrInvalid", err)
	}
}

func nonCanonicalRawURLAlias(t *testing.T, encoded string) string {
	t.Helper()
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := strings.IndexByte(alphabet, encoded[len(encoded)-1])
	if last < 0 {
		t.Fatal("encoded cursor ended in a non-base64url character")
	}
	var aliasIndex int
	switch len(encoded) % 4 {
	case 2:
		aliasIndex = last&0x30 | (last+1)&0x0f
	case 3:
		aliasIndex = last&0x3c | (last+1)&0x03
	default:
		t.Fatal("encoded cursor has no unused trailing base64 bits")
	}
	return encoded[:len(encoded)-1] + string(alphabet[aliasIndex])
}
