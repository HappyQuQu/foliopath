package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPNGProfileKeepsSemanticAndTagInputOwnersSeparate(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repositoryRoot(t), name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	composition := read("internal/app/run.go")
	for _, required := range []string{
		"imagevips.NewRemoteImagePreprocessor(config.ImagePreprocessing)",
		"automaticTagRuntime{content: contentService, preprocessor: imagevips.New()}",
	} {
		if !strings.Contains(composition, required) {
			t.Fatal("review semantic/tag preprocessor ownership: " + required)
		}
	}
	profile := read("internal/media/imagevips/remote_profile.go")
	if !strings.Contains(profile, "semantic.RemoteImageLegacy") {
		t.Fatal("default processor must preserve legacy tag input")
	}
	decoder := read("internal/media/imagevips/remote_semantic_libvips.go")
	if !strings.Contains(decoder, "beginLargeImageDecode(ctx)") || strings.Contains(decoder, "make(chan") {
		t.Fatal("PNG decoder must reuse the process-wide native admission")
	}
	if strings.Count(decoder, "func encodeRemoteSemanticImage(") != 1 {
		t.Fatal("profiles must share one canonical output transform")
	}
}
