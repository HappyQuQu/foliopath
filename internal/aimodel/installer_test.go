package aimodel

import (
	"bytes"
	"context"
	"io"
	"testing"
)

type installerSource struct {
	directValidated bool
}

func (source *installerSource) OpenModelPackageFile(context.Context, string, string) (io.ReadCloser, int64, error) {
	return io.NopCloser(bytes.NewReader(nil)), 0, nil
}

func (source *installerSource) ValidateDirectModelSource(context.Context, string) error {
	source.directValidated = true
	return nil
}

type installerPublisher struct {
	called bool
}

func (publisher *installerPublisher) PublishModelPackage(context.Context, VerifiedPackage, Manifest, PackageOpener) (string, error) {
	publisher.called = true
	return "managed:verified", nil
}

func TestInstallerKeepsManagedAndDirectOwnershipSeparate(t *testing.T) {
	catalog, manifest, facts := catalogFixture(t)
	encoded, err := jsonMarshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := catalog.Verify(encoded, facts, "arm64")
	if err != nil {
		t.Fatal(err)
	}
	candidate := Candidate{
		ID: "aic_install", Package: verified, Manifest: manifest,
		Compatibility: "compatible", SourceIdentity: "source:verified",
	}
	repository := &memoryRepository{snapshot: Snapshot{Revision: 1}}
	ids := []string{"aim_managed", "aim_direct"}
	service, err := NewService(repository, nil, func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	source := &installerSource{}
	publisher := &installerPublisher{}
	installer, err := NewInstaller(service, source, publisher)
	if err != nil {
		t.Fatal(err)
	}
	managed, created, err := installer.Install(context.Background(), candidate, StorageManaged)
	if err != nil || !created || !publisher.called || source.directValidated || managed.SourceIdentity != "managed:verified" {
		t.Fatalf("managed install = %#v, %v, %v", managed, created, err)
	}

	// Use a different package identity so repository idempotency does not mask
	// the direct-source validation path.
	candidate.Package.PackageID = "semantic-direct-v1"
	candidate.Manifest.PackageID = "semantic-direct-v1"
	candidate.Package.ContentHash = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	direct, created, err := installer.Install(context.Background(), candidate, StorageDirect)
	if err != nil || !created || !source.directValidated || direct.SourceIdentity != "source:verified" {
		t.Fatalf("direct install = %#v, %v, %v", direct, created, err)
	}
}
