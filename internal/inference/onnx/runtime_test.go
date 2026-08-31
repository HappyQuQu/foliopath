package onnx

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestValidRuntimePath(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		value string
		want  bool
	}{
		{value: "/proc/self/fd/3", want: true},
		{value: "/proc/self/fd/2048", want: true},
		{value: "/proc/self/fd/2", want: false},
		{value: "/proc/self/fd/-1", want: false},
		{value: "/proc/self/fd/3/model.onnx", want: false},
		{value: "/models/model.onnx", want: false},
	} {
		if got := validRuntimePath(test.value); got != test.want {
			t.Errorf("validRuntimePath(%q) = %t, want %t", test.value, got, test.want)
		}
	}
}

func TestOpenModelFilesRequiresKernelHandleAndExpectedSize(t *testing.T) {
	t.Parallel()
	manifest := testManifest()
	opened := make([]*runtimeFileStub, 0, 3)
	files, err := openModelFiles(context.Background(), manifest, func(_ context.Context, name string) (aimodel.RuntimeModelFile, error) {
		for index, file := range manifest.Files {
			if file.Name == name {
				value := &runtimeFileStub{path: fmt.Sprintf("/proc/self/fd/%d", 3+index), size: file.Size}
				opened = append(opened, value)
				return value, nil
			}
		}
		return nil, errors.New("unexpected file")
	})
	if err != nil {
		t.Fatalf("openModelFiles: %v", err)
	}
	files.close()
	for _, file := range opened {
		if !file.closed {
			t.Fatal("runtime file was not closed")
		}
	}

	_, err = openModelFiles(context.Background(), manifest, func(_ context.Context, name string) (aimodel.RuntimeModelFile, error) {
		return &runtimeFileStub{path: "/models/" + name, size: 1}, nil
	})
	if !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatalf("unsafe runtime path error = %v", err)
	}
}

func TestOpenModelRoleKeepsOnlyRequestedKernelHandle(t *testing.T) {
	t.Parallel()
	manifest := testManifest()
	var openedName string
	file, err := openModelRole(context.Background(), manifest, "image_encoder", func(_ context.Context, name string) (aimodel.RuntimeModelFile, error) {
		openedName = name
		return &runtimeFileStub{path: "/proc/self/fd/9", size: 10}, nil
	})
	if err != nil || openedName != "image.onnx" || file == nil {
		t.Fatalf("open role = %q, %#v, %v", openedName, file, err)
	}
	_ = file.Close()
	invalid := testManifest()
	invalid.Files = append(invalid.Files, invalid.Files[0])
	if _, err := openModelRole(context.Background(), invalid, "image_encoder", nil); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatalf("duplicate role error = %v", err)
	}
}

func testManifest() aimodel.Manifest {
	return aimodel.Manifest{Files: []aimodel.ManifestFile{
		{Name: "image.onnx", Role: "image_encoder", Size: 10},
		{Name: "text.onnx", Role: "text_encoder", Size: 20},
		{Name: "tokenizer.json", Role: "tokenizer", Size: 30},
	}}
}

type runtimeFileStub struct {
	path   string
	size   int64
	closed bool
}

func (file *runtimeFileStub) Close() error        { file.closed = true; return nil }
func (file *runtimeFileStub) RuntimePath() string { return file.path }
func (file *runtimeFileStub) Size() int64         { return file.size }
