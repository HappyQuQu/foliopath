package sentencepiece

import "testing"

type boundaryModelFileStub struct {
	path string
	size int64
}

func (file *boundaryModelFileStub) RuntimePath() string { return file.path }
func (file *boundaryModelFileStub) Size() int64         { return file.size }
func (*boundaryModelFileStub) Close() error             { return nil }

func TestModelFileBoundaryRejectsPathsAndBoundsForEveryBuild(t *testing.T) {
	for _, file := range []*boundaryModelFileStub{
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
	if !validModelFile(&boundaryModelFileStub{path: "/proc/self/fd/9", size: 1}) {
		t.Fatal("rejected valid anchored file")
	}
}
