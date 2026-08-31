package files

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

const (
	managedPublishKillHelperEnv = "FOLIOPATH_MANAGED_PUBLISH_KILL_HELPER"
	managedPublishKillRootEnv   = "FOLIOPATH_MANAGED_PUBLISH_KILL_ROOT"
	managedPublishKillModeEnv   = "FOLIOPATH_MANAGED_PUBLISH_KILL_MODE"
)

func TestManagedModelStoreReconcilesRealPublishAfterProcessKill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "models")
	helperContext, cancelHelper := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelHelper()
	command := exec.CommandContext(helperContext, os.Args[0], "-test.run=^TestManagedModelStorePublishKillHelper$")
	command.Env = append(os.Environ(), managedPublishKillHelperEnv+"=1", managedPublishKillRootEnv+"="+root, managedPublishKillModeEnv+"=copying")
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		_ = command.Wait()
		t.Fatalf("read managed publish helper: %v", err)
	}
	if strings.TrimSpace(line) != "COPYING" {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("unexpected managed publish helper output %q", line)
	}
	staging, err := filepath.Glob(filepath.Join(root, ".partial-*"))
	if err != nil || len(staging) != 1 {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("in-flight staging = %v err=%v", staging, err)
	}
	if err := command.Process.Kill(); err != nil {
		_ = command.Wait()
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("managed publish helper unexpectedly exited successfully")
	}

	store, err := NewManagedModelStore(root, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.Reconcile(context.Background())
	if err != nil || report.RemovedStaging != 1 || report.KnownFinals != 0 || report.UnknownEntries != 0 {
		t.Fatalf("post-kill reconcile = %#v err=%v", report, err)
	}
	staging, err = filepath.Glob(filepath.Join(root, ".partial-*"))
	if err != nil || len(staging) != 0 {
		t.Fatalf("post-reconcile staging = %v err=%v", staging, err)
	}
	finals, err := filepath.Glob(filepath.Join(root, "*.foliomodel"))
	if err != nil || len(finals) != 0 {
		t.Fatalf("partial publish became visible = %v err=%v", finals, err)
	}
}

func TestManagedModelStoreRetainsPublishedFinalAfterProcessKill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "models")
	helperContext, cancelHelper := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelHelper()
	command := exec.CommandContext(helperContext, os.Args[0], "-test.run=^TestManagedModelStorePublishKillHelper$")
	command.Env = append(os.Environ(), managedPublishKillHelperEnv+"=1", managedPublishKillRootEnv+"="+root, managedPublishKillModeEnv+"=published")
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		_ = command.Wait()
		t.Fatalf("read managed publish helper: %v", err)
	}
	if strings.TrimSpace(line) != "PUBLISHED" {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("unexpected managed publish helper output %q", line)
	}
	if err := command.Process.Kill(); err != nil {
		_ = command.Wait()
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("managed publish helper unexpectedly exited successfully")
	}

	store, err := NewManagedModelStore(root, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.Reconcile(context.Background())
	if err != nil || report.RemovedStaging != 0 || report.KnownFinals != 1 || report.UnknownEntries != 0 {
		t.Fatalf("published post-kill reconcile = %#v err=%v", report, err)
	}
	manifest, verified, contents := managedPublishKillFixture()
	identity, err := store.PublishModelPackage(context.Background(), verified, manifest, func(_ context.Context, name string) (io.ReadCloser, int64, error) {
		content, found := contents[name]
		if !found {
			return nil, 0, os.ErrNotExist
		}
		return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
	})
	if err != nil || identity != "managed:"+verified.ContentHash {
		t.Fatalf("verify retained final = %q err=%v", identity, err)
	}
}

func TestManagedModelStorePublishKillHelper(t *testing.T) {
	if os.Getenv(managedPublishKillHelperEnv) != "1" {
		return
	}
	root := os.Getenv(managedPublishKillRootEnv)
	store, err := NewManagedModelStore(root, 1<<30)
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	allowManagedPublishForTest(store)
	manifest, verified, contents := managedPublishKillFixture()
	mode := os.Getenv(managedPublishKillModeEnv)
	_, err = store.PublishModelPackage(context.Background(), verified, manifest, func(_ context.Context, name string) (io.ReadCloser, int64, error) {
		content, found := contents[name]
		if !found {
			return nil, 0, os.ErrNotExist
		}
		if mode == "published" {
			return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
		}
		return &publishKillReader{content: content}, int64(len(content)), nil
	})
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	if mode == "published" {
		fmt.Fprintln(os.Stdout, "PUBLISHED")
		select {}
	}
	os.Exit(3)
}

type publishKillReader struct {
	content  []byte
	offset   int
	signaled bool
}

func (reader *publishKillReader) Read(target []byte) (int, error) {
	if reader.offset < len(reader.content) {
		count := copy(target, reader.content[reader.offset:])
		reader.offset += count
		return count, nil
	}
	if !reader.signaled {
		reader.signaled = true
		fmt.Fprintln(os.Stdout, "COPYING")
	}
	select {}
}

func (*publishKillReader) Close() error { return nil }

func managedPublishKillFixture() (aimodel.Manifest, aimodel.VerifiedPackage, map[string][]byte) {
	contents := map[string][]byte{
		"image_encoder.onnx": bytesOf(0x11, 4096),
		"text_encoder.onnx":  bytesOf(0x22, 4096),
		"tokenizer.json":     bytesOf(0x33, 1024),
	}
	manifest := aimodel.Manifest{
		FormatVersion: 1,
		PackageID:     "semantic-publish-kill-v1",
		Purpose:       aimodel.PurposeSemanticImageText,
		Version:       "1.0.0",
		Architecture:  "portable-onnx",
		LicenseID:     "Apache-2.0",
	}
	roles := map[string]string{
		"image_encoder.onnx": "image_encoder",
		"text_encoder.onnx":  "text_encoder",
		"tokenizer.json":     "tokenizer",
	}
	var total int64
	for _, name := range []string{"image_encoder.onnx", "text_encoder.onnx", "tokenizer.json"} {
		digest := sha256.Sum256(contents[name])
		manifest.Files = append(manifest.Files, aimodel.ManifestFile{
			Name: name, Size: int64(len(contents[name])), SHA256: hex.EncodeToString(digest[:]), Role: roles[name],
		})
		total += int64(len(contents[name]))
	}
	return manifest, aimodel.VerifiedPackage{
		PackageID: manifest.PackageID, Purpose: manifest.Purpose, Version: manifest.Version,
		Architecture: "arm64", ContentHash: strings.Repeat("e", 64), LicenseID: manifest.LicenseID,
		PackageSizeByte: total,
	}, contents
}

func bytesOf(value byte, count int) []byte {
	result := make([]byte, count)
	for index := range result {
		result[index] = value
	}
	return result
}
