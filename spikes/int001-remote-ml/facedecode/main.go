// Opt-in diagnostic over already anonymized sample copies; no production state.
package main

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/davidbyttow/govips/v2/vips"
)

func main() {
	if run() != nil {
		fmt.Fprintln(os.Stderr, "bounded sample decode failed")
		os.Exit(1)
	}
}
func run() error {
	if os.Getenv("FOLIOPATH_FACE_SAMPLE_DIAGNOSTIC") != "1" {
		return fmt.Errorf("opt in required")
	}
	r, err := files.OpenRoot("/library")
	if err != nil {
		return err
	}
	defer r.Close()
	ro, err := r.ReadOnly()
	if err != nil || !ro {
		return fmt.Errorf("readonly required")
	}
	entries, err := r.ReadDir("")
	if err != nil {
		return err
	}
	if len(entries) > 100 {
		return fmt.Errorf("sample bound")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	vipsRuntime := imagevips.NewRuntime()
	if err = vipsRuntime.Start(); err != nil {
		return err
	}
	defer vipsRuntime.Shutdown()
	for _, entry := range entries {
		name := entry.Name()
		if selected := os.Getenv("FOLIOPATH_FACE_SAMPLE_ID"); selected != "" && name != selected {
			continue
		}
		if !strings.HasPrefix(name, "sample-") {
			return fmt.Errorf("anonymous sample required")
		}
		var format media.Format
		switch strings.ToLower(path.Ext(name)) {
		case ".jpg", ".jpeg":
			format = media.FormatJPEG
		case ".png":
			format = media.FormatPNG
		case ".webp":
			format = media.FormatWebP
		case ".gif":
			format = media.FormatGIF
		default:
			continue
		}
		fmt.Println("start", name)
		f, e := r.Open(name)
		if e != nil {
			return e
		}
		config, _, headerErr := image.DecodeConfig(f)
		if headerErr == nil {
			fmt.Println("dimensions", config.Width, config.Height)
		}
		im, e := func() (face.DecodedImage, error) {
			if os.Getenv("FOLIOPATH_DIAGNOSTIC_THREAD_CLEANUP") == "1" {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				defer vips.ShutdownThread()
			}
			return imagevips.New().DecodeFaceImage(context.Background(), f, format, face.MaxInputBytes, face.MaxDecodeDimension)
		}()
		f.Close()
		if e != nil {
			fmt.Println("rejected", name)
			continue
		}
		fmt.Println("decoded", name, im.Width, im.Height)
		if os.Getenv("FOLIOPATH_DIAGNOSTIC_TRIM") == "1" {
			trimDiagnosticMemory()
		}
		var goStats runtime.MemStats
		runtime.ReadMemStats(&goStats)
		var nativeStats vips.MemoryStats
		vips.ReadVipsMemStats(&nativeStats)
		current, _ := os.ReadFile("/sys/fs/cgroup/memory.current")
		fmt.Printf("memory goHeap=%d goSys=%d vips=%d refs=%d cgroup=%s", goStats.HeapAlloc, goStats.Sys, nativeStats.Mem, vips.OpenImageRefs(), current)
		stat, _ := os.ReadFile("/sys/fs/cgroup/memory.stat")
		fields := strings.Fields(string(stat))
		for i := 0; i+1 < len(fields); i += 2 {
			switch fields[i] {
			case "anon", "file", "unevictable", "inactive_file", "active_file":
				fmt.Printf("%s=%s ", fields[i], fields[i+1])
			}
		}
		fmt.Println()
	}
	return nil
}
