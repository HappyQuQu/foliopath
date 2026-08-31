package aimodel

import (
	"context"
	"io"
	"strings"
	"testing"
)

type directActivationStub struct {
	validated bool
	opened    bool
}

func (stub *directActivationStub) ValidateDirectModelPackage(context.Context, string, Manifest) error {
	stub.validated = true
	return nil
}
func (stub *directActivationStub) OpenDirectRuntimeModelFile(context.Context, string, string) (RuntimeModelFile, error) {
	stub.opened = true
	return &runtimeModelFileStub{ReadCloser: io.NopCloser(strings.NewReader("d")), path: "/proc/self/fd/8", size: 1}, nil
}

type managedActivationStub struct {
	validated bool
	opened    bool
}

func (stub *managedActivationStub) ValidateManagedModelPackage(context.Context, Model, Manifest) error {
	stub.validated = true
	return nil
}
func (stub *managedActivationStub) OpenManagedRuntimeModelFile(context.Context, Model, string) (RuntimeModelFile, error) {
	stub.opened = true
	return &runtimeModelFileStub{ReadCloser: io.NopCloser(strings.NewReader("m")), path: "/proc/self/fd/9", size: 1}, nil
}

func TestActivationSourceRouterKeepsManagedAndDirectOwnershipDistinct(t *testing.T) {
	direct, managed := &directActivationStub{}, &managedActivationStub{}
	router, err := NewActivationSourceRouter(direct, managed)
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{PackageID: "reviewed"}
	directModel := Model{StorageMode: StorageDirect, SourceIdentity: "source:test"}
	if err := router.ValidateActivationSource(context.Background(), directModel, manifest); err != nil {
		t.Fatal(err)
	}
	reader, err := router.OpenActivationModelFile(context.Background(), directModel, "image.onnx")
	if err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	if !direct.validated || !direct.opened || managed.validated || managed.opened {
		t.Fatalf("direct routing = %#v/%#v", direct, managed)
	}
	managedModel := Model{StorageMode: StorageManaged, SourceIdentity: "managed:test"}
	if err := router.ValidateActivationSource(context.Background(), managedModel, manifest); err != nil {
		t.Fatal(err)
	}
	reader, err = router.OpenActivationModelFile(context.Background(), managedModel, "image.onnx")
	if err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	if !managed.validated || !managed.opened {
		t.Fatalf("managed routing = %#v", managed)
	}
}

var _ DirectActivationSource = (*directActivationStub)(nil)
var _ ManagedActivationSource = (*managedActivationStub)(nil)
