package semantic

import (
	"errors"
	"strings"
	"testing"
)

func testSpaceIdentity() SpaceIdentity {
	return SpaceIdentity{Version: 1, Protocol: "immich-ml-v3.1.0", Model: "ViT-B-32__openai", Dimension: 512,
		RuntimeSHA256: strings.Repeat("a", 64), CacheManifestSHA256: strings.Repeat("b", 64),
		PreprocessingSHA256: strings.Repeat("c", 64)}
}

func TestSpaceCompatibility(t *testing.T) {
	identity := testSpaceIdentity()
	fingerprint, err := identity.Fingerprint()
	// Pinned independently using Node crypto over the documented ordered JSON.
	const expected = "04a9ac6de10bef84314aa4f9593677224b0704b33e7ec768e450fd3be2d9a672"
	if err != nil || fingerprint != expected {
		t.Fatalf("fingerprint: %q %v", fingerprint, err)
	}
	if err := RequireSpace(fingerprint, identity); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*SpaceIdentity){
		"model":         func(i *SpaceIdentity) { i.Model = "another-model" },
		"protocol":      func(i *SpaceIdentity) { i.Protocol = "immich-ml-v4" },
		"dimension":     func(i *SpaceIdentity) { i.Dimension = 768 },
		"runtime":       func(i *SpaceIdentity) { i.RuntimeSHA256 = strings.Repeat("d", 64) },
		"cache":         func(i *SpaceIdentity) { i.CacheManifestSHA256 = strings.Repeat("d", 64) },
		"preprocessing": func(i *SpaceIdentity) { i.PreprocessingSHA256 = strings.Repeat("d", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			changed := identity
			change(&changed)
			if err := RequireSpace(fingerprint, changed); !errors.Is(err, ErrIncompatibleSpace) {
				t.Fatalf("accepted changed space: %v", err)
			}
		})
	}
	for _, legacy := range []string{"", "old-model-id", strings.ToUpper(fingerprint), strings.Repeat("0", 64)} {
		if err := RequireSpace(legacy, identity); !errors.Is(err, ErrIncompatibleSpace) {
			t.Fatalf("accepted unbound generation: %v", err)
		}
	}
}

func TestSpaceIdentityInvalid(t *testing.T) {
	for name, change := range map[string]func(*SpaceIdentity){
		"version":    func(i *SpaceIdentity) { i.Version = 2 },
		"empty":      func(i *SpaceIdentity) { i.Model = "" },
		"path":       func(i *SpaceIdentity) { i.Model = "/private/model" },
		"whitespace": func(i *SpaceIdentity) { i.Protocol = " protocol" },
		"long":       func(i *SpaceIdentity) { i.Model = strings.Repeat("a", 129) },
		"zero":       func(i *SpaceIdentity) { i.Dimension = 0 },
		"large":      func(i *SpaceIdentity) { i.Dimension = 65537 },
		"short-hash": func(i *SpaceIdentity) { i.RuntimeSHA256 = "a" },
		"uppercase":  func(i *SpaceIdentity) { i.CacheManifestSHA256 = strings.Repeat("B", 64) },
		"nonhex":     func(i *SpaceIdentity) { i.PreprocessingSHA256 = strings.Repeat("z", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			identity := testSpaceIdentity()
			change(&identity)
			if got, err := identity.Fingerprint(); got != "" || !errors.Is(err, ErrInvalidSpaceIdentity) {
				t.Fatalf("invalid identity accepted: %q %v", got, err)
			}
			if err := RequireSpace("", identity); !errors.Is(err, ErrInvalidSpaceIdentity) {
				t.Fatalf("invalid runtime accepted: %v", err)
			}
		})
	}
}
