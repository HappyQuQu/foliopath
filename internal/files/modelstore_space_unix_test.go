//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package files

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"golang.org/x/sys/unix"
)

func TestManagedStoreWriteErrorMapsKernelSpaceExhaustion(t *testing.T) {
	for _, input := range []error{unix.ENOSPC, unix.EDQUOT} {
		mapped := managedStoreWriteError(input)
		if !errors.Is(mapped, aimodel.ErrInsufficientSpace) || !errors.Is(mapped, input) {
			t.Fatalf("mapped %v = %v", input, mapped)
		}
	}
	sentinel := errors.New("source read failed")
	if mapped := managedStoreWriteError(sentinel); !errors.Is(mapped, sentinel) || errors.Is(mapped, aimodel.ErrInsufficientSpace) {
		t.Fatalf("non-space error mapped = %v", mapped)
	}
}

func TestManagedTargetWriterMapsOnlyDestinationSpaceExhaustion(t *testing.T) {
	target := managedTargetWriter{writer: errorWriter{err: unix.ENOSPC}}
	if _, err := target.Write([]byte("x")); !errors.Is(err, aimodel.ErrInsufficientSpace) || !errors.Is(err, unix.ENOSPC) {
		t.Fatalf("target ENOSPC = %v", err)
	}

	expected := aimodel.ManifestFile{Name: "model.onnx", Size: 1, SHA256: "2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881"}
	err := copyManagedFile(filepath.Join(t.TempDir(), expected.Name), errorReader{err: unix.ENOSPC}, expected)
	if !errors.Is(err, unix.ENOSPC) || errors.Is(err, aimodel.ErrInsufficientSpace) {
		t.Fatalf("source ENOSPC classification = %v", err)
	}
}

type errorWriter struct{ err error }

func (writer errorWriter) Write([]byte) (int, error) { return 0, writer.err }

type errorReader struct{ err error }

func (reader errorReader) Read([]byte) (int, error) { return 0, reader.err }

var _ io.Reader = errorReader{}
