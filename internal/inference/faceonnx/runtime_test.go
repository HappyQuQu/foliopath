package faceonnx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/media"
)

type decoderStub struct{ image face.DecodedImage }

func (stub decoderStub) DecodeFaceImage(context.Context, io.ReadSeeker, media.Format, int64, int) (face.DecodedImage, error) {
	return stub.image, nil
}

type detectorStub struct {
	detection []onnx.FaceDetection
	err       error
	input     []float32
	closed    bool
}

func (stub *detectorStub) DetectFaces(_ context.Context, input []float32) ([]onnx.FaceDetection, error) {
	stub.input = append([]float32(nil), input...)
	return stub.detection, stub.err
}
func (stub *detectorStub) Close() error { stub.closed = true; return nil }

type embedderStub struct {
	input  []float32
	output []float32
	err    error
	closed bool
}

func (stub *embedderStub) EmbedFace(_ context.Context, input []float32) ([]float32, error) {
	stub.input = append([]float32(nil), input...)
	if stub.output != nil {
		return append([]float32(nil), stub.output...), stub.err
	}
	return []float32{3, 4}, stub.err
}
func (stub *embedderStub) Close() error { stub.closed = true; return nil }

func TestRuntimeComposesBoundedDetectorAlignmentAndEmbedding(t *testing.T) {
	image := face.DecodedImage{Width: 112, Height: 112, RGB: make([]byte, 112*112*3)}
	for offset := 0; offset < len(image.RGB); offset += 3 {
		image.RGB[offset], image.RGB[offset+1], image.RGB[offset+2] = 255, 64, 0
	}
	scale := float32(1)
	detection := onnx.FaceDetection{X: 10 * scale, Y: 12 * scale, Width: 80 * scale, Height: 84 * scale, Score: .95}
	for index := range detection.Landmarks {
		detection.Landmarks[index] = onnx.FacePoint{X: alignmentReference[index].X * scale, Y: alignmentReference[index].Y * scale}
	}
	detector := &detectorStub{detection: []onnx.FaceDetection{detection}}
	embedder := &embedderStub{}
	runtime, err := New(decoderStub{image: image}, detector, embedder)
	if err != nil {
		t.Fatal(err)
	}
	got, err := runtime.AnalyzeFaces(context.Background(), bytes.NewReader([]byte("fixture")), media.FormatJPEG, 1024)
	if err != nil || len(got) != 1 {
		t.Fatalf("candidates=%#v err=%v", got, err)
	}
	if len(detector.input) != 3*onnx.FaceDetectorWidth*onnx.FaceDetectorHeight || len(embedder.input) != 3*112*112 ||
		got[0].Embedding[0] != 3 || got[0].Embedding[1] != 4 || math.Abs(float64(got[0].Box.X-10.0/112)) > 1e-5 ||
		got[0].Detection != .95 || got[0].Quality <= 0 || got[0].Quality > got[0].Detection {
		t.Fatalf("candidate=%#v detector=%d embedder=%d", got[0], len(detector.input), len(embedder.input))
	}
	if detector.input[0] != 0 || detector.input[onnx.FaceDetectorWidth*onnx.FaceDetectorHeight] != 64 ||
		detector.input[2*onnx.FaceDetectorWidth*onnx.FaceDetectorHeight] != 255 {
		t.Fatal("detector input does not use BGR planar order")
	}
	if err := runtime.Close(); err != nil || !detector.closed || !embedder.closed {
		t.Fatalf("close error=%v detector=%t embedder=%t", err, detector.closed, embedder.closed)
	}
}

func TestRuntimeMapsModelFailureAndRejectsBounds(t *testing.T) {
	image := face.DecodedImage{Width: 1, Height: 1, RGB: []byte{0, 0, 0}}
	detector := &detectorStub{err: aimodel.ErrModelIncompatible}
	runtime, err := New(decoderStub{image: image}, detector, &embedderStub{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.AnalyzeFaces(context.Background(), bytes.NewReader(nil), media.FormatJPEG, 1); !errors.Is(err, face.ErrRuntimeUnavailable) {
		t.Fatalf("runtime error=%v", err)
	}
	if _, err := runtime.AnalyzeFaces(context.Background(), bytes.NewReader(nil), media.FormatJPEG, face.MaxInputBytes+1); !errors.Is(err, face.ErrInvalidInput) {
		t.Fatalf("bounds error=%v", err)
	}
}

func TestAlignFaceRejectsDegenerateLandmarks(t *testing.T) {
	image := face.DecodedImage{Width: 2, Height: 2, RGB: make([]byte, 12)}
	if _, ok := alignFace(image, [5]onnx.FacePoint{}); ok {
		t.Fatal("degenerate alignment succeeded")
	}
}
