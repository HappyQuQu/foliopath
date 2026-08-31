package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestPublishPackageDirectoryIsAtomicAndNoReplace(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip(errAtomicPackagePublishUnsupported)
	}
	parent := t.TempDir()
	staged := "incoming-semantic-v1"
	final := "semantic-v1-digest"
	if err := os.Mkdir(filepath.Join(parent, staged), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, staged, "model.bin"), []byte("complete"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := PublishPackageDirectory(parent, staged, final); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(parent, staged)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staging directory still visible: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(parent, final, "model.bin"))
	if err != nil || string(contents) != "complete" {
		t.Fatalf("published package is incomplete: %q, %v", contents, err)
	}

	secondStaged := "incoming-semantic-v2"
	if err := os.Mkdir(filepath.Join(parent, secondStaged), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := PublishPackageDirectory(parent, secondStaged, final); err == nil {
		t.Fatal("expected an existing generation to reject replacement")
	}
	if _, err := os.Stat(filepath.Join(parent, secondStaged)); err != nil {
		t.Fatalf("failed publish must preserve staging for diagnosis: %v", err)
	}
}

func TestPublishPackageDirectoryRejectsUnsafeNames(t *testing.T) {
	if err := PublishPackageDirectory(t.TempDir(), "../incoming", "model"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestPublishPackageDirectoryStrongKillBoundaries(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip(errAtomicPackagePublishUnsupported)
	}
	if os.Getenv("INT001_PACKAGE_PUBLISH_HELPER") == "1" {
		runPackagePublishKillHelper(t)
		return
	}
	for _, phase := range []packagePublishPhase{
		packagePublishBeforeRename,
		packagePublishAfterRename,
		packagePublishAfterSync,
	} {
		t.Run(string(phase), func(t *testing.T) {
			parent := t.TempDir()
			staged := "incoming-semantic-v1"
			final := "semantic-v1-digest"
			if err := os.Mkdir(filepath.Join(parent, staged), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(
				filepath.Join(parent, staged, "model.bin"),
				[]byte("complete-package"),
				0o600,
			); err != nil {
				t.Fatal(err)
			}
			helper := exec.Command(
				os.Args[0],
				"-test.run=^TestPublishPackageDirectoryStrongKillBoundaries$",
			)
			helper.Env = append(os.Environ(),
				"INT001_PACKAGE_PUBLISH_HELPER=1",
				"INT001_PACKAGE_PUBLISH_PARENT="+parent,
				"INT001_PACKAGE_PUBLISH_STAGED="+staged,
				"INT001_PACKAGE_PUBLISH_FINAL="+final,
				"INT001_PACKAGE_PUBLISH_PHASE="+string(phase),
			)
			if err := helper.Start(); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(parent, "kill-phase-"+string(phase))
			deadline := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Stat(marker); err == nil {
					break
				}
				if time.Now().After(deadline) {
					_ = helper.Process.Kill()
					_, _ = helper.Process.Wait()
					t.Fatal("publish helper did not reach kill boundary")
				}
				time.Sleep(time.Millisecond)
			}
			if err := helper.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err := helper.Wait(); err == nil {
				t.Fatal("strong-killed publish helper exited successfully")
			}

			stagedPath := filepath.Join(parent, staged)
			finalPath := filepath.Join(parent, final)
			if phase == packagePublishBeforeRename {
				assertCompletePackage(t, stagedPath)
				if _, err := os.Stat(finalPath); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("pre-rename kill exposed final generation: %v", err)
				}
				if err := PublishPackageDirectory(parent, staged, final); err != nil {
					t.Fatalf("restart could not publish preserved staging: %v", err)
				}
			} else {
				if _, err := os.Stat(stagedPath); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("post-rename kill retained staging generation: %v", err)
				}
			}
			assertCompletePackage(t, finalPath)
		})
	}
}

func TestPublishPackageDirectoryActualENOSPCBoundary(t *testing.T) {
	parent := os.Getenv("INT001_PACKAGE_PUBLISH_ENOSPC_DIR")
	if parent == "" {
		t.Skip("set INT001_PACKAGE_PUBLISH_ENOSPC_DIR to an empty size-limited filesystem")
	}
	staged := "incoming-model"
	final := "published-model-generation"
	if err := os.Mkdir(filepath.Join(parent, staged), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(parent, staged, "model.bin"),
		[]byte("complete-package"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(parent, "active-generation")
	if err := os.WriteFile(active, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	filler, err := os.OpenFile(
		filepath.Join(parent, "space-filler"),
		os.O_CREATE|os.O_WRONLY|os.O_EXCL,
		0o600,
	)
	if err != nil {
		t.Fatal(err)
	}
	block := make([]byte, 4096)
	for {
		_, err = filler.Write(block)
		if err != nil {
			break
		}
	}
	if closeErr := filler.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("expected kernel ENOSPC before publish, got %v", err)
	}

	publishErr := PublishPackageDirectory(parent, staged, final)
	stagedPath := filepath.Join(parent, staged)
	finalPath := filepath.Join(parent, final)
	if publishErr == nil {
		t.Log("full-filesystem no-replace rename and parent sync completed atomically")
		if _, err := os.Stat(stagedPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("successful disk-full publish retained staging: %v", err)
		}
		assertCompletePackage(t, finalPath)
	} else {
		t.Logf("full-filesystem publish failed closed: %v", publishErr)
		if !errors.Is(publishErr, syscall.ENOSPC) {
			t.Fatalf("disk-full publish returned unexpected error: %v", publishErr)
		}
		assertCompletePackage(t, stagedPath)
		if _, err := os.Stat(finalPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failed disk-full publish exposed final: %v", err)
		}
	}
	current, err := os.ReadFile(active)
	if err != nil || string(current) != "current" {
		t.Fatalf("active changed at disk-full publish boundary: %q, %v", current, err)
	}
}

func runPackagePublishKillHelper(t *testing.T) {
	t.Helper()
	targetPhase := packagePublishPhase(os.Getenv("INT001_PACKAGE_PUBLISH_PHASE"))
	err := publishPackageDirectoryWithHook(
		os.Getenv("INT001_PACKAGE_PUBLISH_PARENT"),
		os.Getenv("INT001_PACKAGE_PUBLISH_STAGED"),
		os.Getenv("INT001_PACKAGE_PUBLISH_FINAL"),
		func(phase packagePublishPhase) error {
			if phase != targetPhase {
				return nil
			}
			marker := filepath.Join(
				os.Getenv("INT001_PACKAGE_PUBLISH_PARENT"),
				"kill-phase-"+string(phase),
			)
			if err := os.WriteFile(marker, []byte("ready"), 0o600); err != nil {
				return err
			}
			select {}
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func assertCompletePackage(t *testing.T, directory string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(directory, "model.bin"))
	if err != nil || string(contents) != "complete-package" {
		t.Fatalf("package is incomplete: %q, %v", contents, err)
	}
}
