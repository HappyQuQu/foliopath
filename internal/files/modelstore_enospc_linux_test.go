//go:build linux && fsboundary

package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"golang.org/x/sys/unix"
)

func TestManagedModelStoreActualENOSPCFailsClosed(t *testing.T) {
	root := filepath.Join(t.TempDir(), "managed-model-tmpfs")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mount("tmpfs", root, "tmpfs", 0, "size=128k,mode=0700"); err != nil {
		t.Fatalf("mount constrained tmpfs (requires isolated CAP_SYS_ADMIN): %v", err)
	}
	t.Cleanup(func() {
		if err := unix.Unmount(root, 0); err != nil {
			t.Errorf("unmount constrained tmpfs: %v", err)
		}
	})

	verified, manifest, contents := managedPackageFixture()
	contents["image_encoder.onnx"] = bytes.Repeat([]byte{0x5a}, 512<<10)
	var total int64
	for index := range manifest.Files {
		content := contents[manifest.Files[index].Name]
		digest := sha256.Sum256(content)
		manifest.Files[index].Size = int64(len(content))
		manifest.Files[index].SHA256 = hex.EncodeToString(digest[:])
		total += int64(len(content))
	}
	verified.PackageSizeByte = total

	store, err := NewManagedModelStore(root, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the post-preflight kernel failure rather than the already-covered
	// reserve calculation. All filesystem writes still use production code.
	allowManagedPublishForTest(store)
	_, err = store.PublishModelPackage(context.Background(), verified, manifest, func(_ context.Context, name string) (io.ReadCloser, int64, error) {
		content, found := contents[name]
		if !found {
			return nil, 0, os.ErrNotExist
		}
		return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
	})
	if !errors.Is(err, aimodel.ErrInsufficientSpace) || !errors.Is(err, unix.ENOSPC) {
		t.Fatalf("managed publish error=%v, want insufficient_space + ENOSPC", err)
	}
	finals, globErr := filepath.Glob(filepath.Join(root, "*.foliomodel"))
	if globErr != nil || len(finals) != 0 {
		t.Fatalf("visible finals after ENOSPC=%v err=%v", finals, globErr)
	}
	staging, globErr := filepath.Glob(filepath.Join(root, ".partial-*"))
	if globErr != nil || len(staging) != 0 {
		t.Fatalf("staging after ENOSPC=%v err=%v", staging, globErr)
	}
}
