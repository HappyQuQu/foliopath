//go:build libvips

// Opt-in synthetic resource probe. Never mount a real media library here.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"runtime"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/davidbyttow/govips/v2/vips"
)

type syntheticImage struct{}

func (syntheticImage) ColorModel() color.Model { return color.YCbCrModel }
func (syntheticImage) Bounds() image.Rectangle { return image.Rect(0, 0, 8800, 11000) }
func (syntheticImage) At(x, y int) color.Color {
	n := uint32(x)*0x9e3779b9 ^ uint32(y)*0x85ebca6b
	n ^= n >> 16
	n *= 0x7feb352d
	n ^= n >> 15
	return color.YCbCr{Y: byte(n), Cb: byte(n >> 8), Cr: byte(n >> 16)}
}

func main() {
	if os.Getenv("FOLIOPATH_SYNTHETIC_FACE_MEMORY") != "1" {
		panic("synthetic opt-in required")
	}
	vr := imagevips.NewRuntime()
	if err := vr.Start(); err != nil {
		panic(err)
	}
	defer vr.Shutdown()
	if len(os.Args) == 2 && os.Args[1] == "generate" {
		f, err := os.Create("/generated/source.jpg")
		if err != nil {
			panic(err)
		}
		if err := jpeg.Encode(f, syntheticImage{}, &jpeg.Options{Quality: 90}); err != nil {
			panic(err)
		}
		f.Close()
		im, err := vips.LoadImageFromFile("/generated/source.jpg", vips.NewImportParams())
		if err != nil {
			panic(err)
		}
		defer im.Close()
		p := vips.NewJpegExportParams()
		p.Quality = 90
		p.Interlace = true
		data, _, err := im.ExportJpeg(p)
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile("/generated/progressive.jpg", data, 0600); err != nil {
			panic(err)
		}
		fmt.Println("generated procedural 96.8 MP JPEG pair; no real media")
		return
	}
	r, err := files.OpenRoot("/library")
	if err != nil {
		panic(err)
	}
	defer r.Close()
	if ro, err := r.ReadOnly(); err != nil || !ro {
		panic("readonly synthetic root required")
	}
	name, expected := "progressive.jpg", "38276a09c8aba0676f368ceeeeac97d125b3fb7202267ca1b039bdfacf9b18f7"
	if os.Getenv("FOLIOPATH_FACE_MEMORY_BASELINE") == "1" {
		name, expected = "source.jpg", "4f2c440ac6bc1625331ba9d711967ad340448f091fb89969bb5422c93bfcb02f"
	}
	iterations := 15
	if raw := os.Getenv("FOLIOPATH_FACE_MEMORY_ITERATIONS"); raw != "" {
		iterations, err = strconv.Atoi(raw)
		if err != nil {
			panic(err)
		}
	}
	if iterations < 1 || iterations > 120 {
		panic("iteration bound")
	}
	retained := make([]byte, 256<<20)
	for i := 0; i < len(retained); i += 4096 {
		retained[i] = byte(i/4096 + 1)
	}
	for i := 0; i < iterations; i++ {
		func() {
			f, err := r.Open(name)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			got, err := imagevips.New().DecodeFaceImage(context.Background(), f, media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
			if err != nil {
				panic(err)
			}
			if got.Width != 1280 || got.Height != 1600 || fmt.Sprintf("%x", sha256.Sum256(got.RGB)) != expected {
				panic("synthetic pixel contract drift")
			}
			rgba := image.NewNRGBA(image.Rect(0, 0, got.Width, got.Height))
			for j := 0; j < len(got.RGB)/3; j++ {
				copy(rgba.Pix[j*4:j*4+3], got.RGB[j*3:j*3+3])
				rgba.Pix[j*4+3] = 255
			}
			var out bytes.Buffer
			encoder := png.Encoder{CompressionLevel: png.BestSpeed}
			if err := encoder.Encode(&out, rgba); err != nil {
				panic(err)
			}
		}()
		if i == 0 || i == iterations-1 {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			current, _ := os.ReadFile("/sys/fs/cgroup/memory.current")
			peak, _ := os.ReadFile("/sys/fs/cgroup/memory.peak")
			data, _ := json.Marshal(map[string]any{"iteration": i + 1, "heap": stats.HeapAlloc, "goSys": stats.Sys, "cgroup": string(current), "peak": string(peak), "rgb_sha256": expected, "refs": vips.OpenImageRefs()})
			fmt.Println(string(data))
		}
	}
	runtime.KeepAlive(retained)
	fmt.Println("synthetic face-memory probe passed")
}
