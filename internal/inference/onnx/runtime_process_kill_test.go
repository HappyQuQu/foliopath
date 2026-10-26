//go:build linux && cgo && onnxruntime && inferencekill

package onnx

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

const (
	runtimeKillHelperEnv = "FOLIOPATH_ORT_KILL_HELPER"
	runtimeKillModelEnv  = "FOLIOPATH_ORT_KILL_MODEL"
)

func TestNativeImageInferenceRecoversAfterProcessKill(t *testing.T) {
	modelPath := os.Getenv(runtimeKillModelEnv)
	if os.Getenv(runtimeKillHelperEnv) != "" {
		runRuntimeKillHelper(t, modelPath, os.Getenv(runtimeKillHelperEnv))
		return
	}
	if modelPath == "" {
		t.Skip("set FOLIOPATH_ORT_KILL_MODEL to the reviewed image encoder fixture")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeImageInferenceRecoversAfterProcessKill$")
	command.Env = append(os.Environ(), runtimeKillHelperEnv+"=kill")
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "RUN_ACTIVE" {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("native inference helper output=%q err=%v", line, err)
	}
	if err := command.Process.Kill(); err != nil {
		_ = command.Wait()
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("native inference helper unexpectedly exited successfully")
	}

	recovery := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeImageInferenceRecoversAfterProcessKill$")
	recovery.Env = append(os.Environ(), runtimeKillHelperEnv+"=recover")
	output, err := recovery.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "RECOVERED:768") {
		t.Fatalf("native inference recovery output=%q err=%v", output, err)
	}
}

func runRuntimeKillHelper(t *testing.T, modelPath, mode string) {
	t.Helper()
	file, err := os.Open(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	manifest := aimodel.Manifest{Files: []aimodel.ManifestFile{{Name: "image_encoder.onnx", Role: "image_encoder", Size: info.Size()}}}
	session, err := New().OpenImageSession(context.Background(), manifest, func(context.Context, string) (aimodel.RuntimeModelFile, error) {
		return &nativeRuntimeFile{File: file, size: info.Size()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	input := make([]float32, imageTensorElements)
	if mode == "kill" {
		done := make(chan struct{})
		go func() {
			select {
			case <-time.After(20 * time.Millisecond):
				fmt.Fprintln(os.Stdout, "RUN_ACTIVE")
				select {}
			case <-done:
				fmt.Fprintln(os.Stdout, "RUN_FINISHED_EARLY")
			}
		}()
		_, _ = session.Encode(context.Background(), input)
		close(done)
		select {}
	}
	output, err := session.Encode(context.Background(), input)
	if err != nil || len(output) != int(EmbeddingDimension) {
		t.Fatalf("recovery inference dimension=%d err=%v", len(output), err)
	}
	for _, value := range output {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			t.Fatal("recovery inference returned non-finite output")
		}
	}
	fmt.Fprintln(os.Stdout, "RECOVERED:"+strconv.Itoa(len(output)))
}

type nativeRuntimeFile struct {
	*os.File
	size int64
}

func (file *nativeRuntimeFile) RuntimePath() string {
	return "/proc/self/fd/" + strconv.FormatUint(uint64(file.Fd()), 10)
}
func (file *nativeRuntimeFile) Size() int64 { return file.size }
