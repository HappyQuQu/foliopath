package semantic

import "testing"

func TestRemoteImageProfilesAreExplicitAndDistinct(t *testing.T) {
	legacy, ok := RemoteImageLegacy.Contract()
	if !ok || legacy != RemoteImagePreprocessingContract {
		t.Fatal("default no longer preserves v3")
	}
	current, ok := RemoteImagePNG.Contract()
	if !ok || current != RemotePNGPreprocessingContract || current == legacy {
		t.Fatal("PNG profile must have a distinct input contract")
	}
	for _, name := range []RemoteImageProfile{"v3", "latest", "PNG-v4", "png-v4 "} {
		if _, ok := name.Contract(); ok {
			t.Fatalf("unreviewed profile accepted: %q", name)
		}
	}
}
