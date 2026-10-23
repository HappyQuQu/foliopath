//go:build !linux || !cgo || !sentencepiece

package sentencepiece

import (
	"context"
	"errors"
	"testing"
)

type modelFileStub struct {
	path   string
	size   int64
	closed bool
}

func (file *modelFileStub) RuntimePath() string { return file.path }
func (file *modelFileStub) Size() int64         { return file.size }
func (file *modelFileStub) Close() error        { file.closed = true; return nil }

func TestRuntimeFailsClosedWithoutNativeSentencePiece(t *testing.T) {
	file := &modelFileStub{path: "/proc/self/fd/9", size: 128}
	session, err := New().Open(context.Background(), file)
	if session != nil || !errors.Is(err, ErrTokenizerUnavailable) {
		t.Fatalf("session=%#v error=%v", session, err)
	}
	if !file.closed {
		t.Fatal("unavailable runtime did not close transferred file")
	}
}

func TestModelFileBoundaryRejectsPathsAndBounds(t *testing.T) {
	for _, file := range []*modelFileStub{
		{path: "/tmp/model", size: 1},
		{path: "/proc/self/fd/2", size: 1},
		{path: "/proc/self/fd/9/extra", size: 1},
		{path: "/proc/self/fd/9", size: 0},
		{path: "/proc/self/fd/9", size: MaxModelBytes + 1},
	} {
		if validModelFile(file) {
			t.Fatalf("accepted %#v", file)
		}
	}
	if !validModelFile(&modelFileStub{path: "/proc/self/fd/9", size: 1}) {
		t.Fatal("rejected valid anchored file")
	}
}
