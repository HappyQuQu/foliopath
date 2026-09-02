// Package faceonnx composes the reviewed image decoder with bounded YuNet and
// face-embedding sessions. It does not own model activation or filesystem paths.
package faceonnx

import (
	"context"
	"errors"
	"io"
	"math"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/media"
)

const alignedFaceSize = 112

var alignmentReference = [5]onnx.FacePoint{
	{X: 38.2946, Y: 51.6963},
	{X: 73.5318, Y: 51.5014},
	{X: 56.0252, Y: 71.7366},
	{X: 41.5493, Y: 92.3655},
	{X: 70.7299, Y: 92.2041},
}

type Runtime struct {
	decoder  face.ImageDecoder
	detector onnx.FaceDetectorSession
	embedder onnx.FaceEmbeddingSession
}

func New(decoder face.ImageDecoder, detector onnx.FaceDetectorSession, embedder onnx.FaceEmbeddingSession) (*Runtime, error) {
	if decoder == nil || detector == nil || embedder == nil {
		return nil, face.ErrInvalidInput
	}
	return &Runtime{decoder: decoder, detector: detector, embedder: embedder}, nil
}

func (runtime *Runtime) Close() error {
	if runtime == nil {
		return nil
	}
	var detectorErr, embedderErr error
	if runtime.detector != nil {
		detectorErr = runtime.detector.Close()
	}
	if runtime.embedder != nil {
		embedderErr = runtime.embedder.Close()
	}
	return errors.Join(detectorErr, embedderErr)
}

func (runtime *Runtime) AnalyzeFaces(
	ctx context.Context,
	source io.ReadSeeker,
	format media.Format,
	maxBytes int64,
) ([]face.Candidate, error) {
	if runtime == nil || runtime.decoder == nil || runtime.detector == nil || runtime.embedder == nil ||
		source == nil || maxBytes < 1 || maxBytes > face.MaxInputBytes {
		return nil, face.ErrInvalidInput
	}
	decoded, err := runtime.decoder.DecodeFaceImage(ctx, source, format, maxBytes, face.MaxDecodeDimension)
	if err != nil {
		return nil, err
	}
	if decoded.Width < 1 || decoded.Height < 1 || len(decoded.RGB) != decoded.Width*decoded.Height*3 {
		return nil, face.ErrInvalidOutput
	}
	detectorInput, scale := prepareDetectorInput(decoded)
	detections, err := runtime.detector.DetectFaces(ctx, detectorInput)
	if err != nil {
		return nil, runtimeError(err)
	}
	if len(detections) > face.MaxCandidatesPerAsset {
		return nil, face.ErrInvalidOutput
	}
	result := make([]face.Candidate, 0, len(detections))
	for _, detection := range detections {
		candidate, ok, err := runtime.embedDetection(ctx, decoded, detection, scale)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, candidate)
		}
	}
	return result, nil
}

func (runtime *Runtime) embedDetection(
	ctx context.Context,
	image face.DecodedImage,
	detection onnx.FaceDetection,
	scale float32,
) (face.Candidate, bool, error) {
	if scale <= 0 || detection.Score < 0 || detection.Score > 1 || !finite(detection.Score) {
		return face.Candidate{}, false, face.ErrInvalidOutput
	}
	x0 := max(float32(0), detection.X/scale)
	y0 := max(float32(0), detection.Y/scale)
	x1 := min(float32(image.Width), (detection.X+detection.Width)/scale)
	y1 := min(float32(image.Height), (detection.Y+detection.Height)/scale)
	if !finite(x0) || !finite(y0) || !finite(x1) || !finite(y1) || x1 <= x0 || y1 <= y0 {
		return face.Candidate{}, false, nil
	}
	landmarks := detection.Landmarks
	for index := range landmarks {
		landmarks[index].X /= scale
		landmarks[index].Y /= scale
		if !finite(landmarks[index].X) || !finite(landmarks[index].Y) {
			return face.Candidate{}, false, face.ErrInvalidOutput
		}
	}
	aligned, ok := alignFace(image, landmarks)
	if !ok {
		return face.Candidate{}, false, nil
	}
	embedding, err := runtime.embedder.EmbedFace(ctx, aligned)
	if err != nil {
		return face.Candidate{}, false, runtimeError(err)
	}
	width, height := x1-x0, y1-y0
	quality := detection.Score * min(float32(1), min(width, height)/64)
	return face.Candidate{
		Box: face.Box{X: x0 / float32(image.Width), Y: y0 / float32(image.Height),
			Width: width / float32(image.Width), Height: height / float32(image.Height)},
		Detection: detection.Score,
		Quality:   quality,
		Embedding: embedding,
	}, true, nil
}

func prepareDetectorInput(image face.DecodedImage) ([]float32, float32) {
	scale := min(float32(1), min(float32(onnx.FaceDetectorWidth)/float32(image.Width),
		float32(onnx.FaceDetectorHeight)/float32(image.Height)))
	width := max(1, int(math.Round(float64(float32(image.Width)*scale))))
	height := max(1, int(math.Round(float64(float32(image.Height)*scale))))
	result := make([]float32, 3*onnx.FaceDetectorWidth*onnx.FaceDetectorHeight)
	plane := onnx.FaceDetectorWidth * onnx.FaceDetectorHeight
	for y := 0; y < height; y++ {
		sourceY := (float32(y)+.5)/scale - .5
		for x := 0; x < width; x++ {
			sourceX := (float32(x)+.5)/scale - .5
			r, g, b := bilinearRGB(image, sourceX, sourceY)
			offset := y*onnx.FaceDetectorWidth + x
			result[offset] = b
			result[plane+offset] = g
			result[2*plane+offset] = r
		}
	}
	return result, scale
}

func alignFace(image face.DecodedImage, source [5]onnx.FacePoint) ([]float32, bool) {
	var sourceX, sourceY, targetX, targetY float64
	for index := range source {
		sourceX += float64(source[index].X)
		sourceY += float64(source[index].Y)
		targetX += float64(alignmentReference[index].X)
		targetY += float64(alignmentReference[index].Y)
	}
	sourceX, sourceY, targetX, targetY = sourceX/5, sourceY/5, targetX/5, targetY/5
	var numeratorA, numeratorB, denominator float64
	for index := range source {
		dx := float64(source[index].X) - sourceX
		dy := float64(source[index].Y) - sourceY
		du := float64(alignmentReference[index].X) - targetX
		dv := float64(alignmentReference[index].Y) - targetY
		numeratorA += dx*du + dy*dv
		numeratorB += dx*dv - dy*du
		denominator += dx*dx + dy*dy
	}
	if denominator <= 1e-9 {
		return nil, false
	}
	a, b := numeratorA/denominator, numeratorB/denominator
	tx := targetX - a*sourceX + b*sourceY
	ty := targetY - b*sourceX - a*sourceY
	inverse := a*a + b*b
	if inverse <= 1e-12 || math.IsNaN(inverse) || math.IsInf(inverse, 0) {
		return nil, false
	}
	result := make([]float32, 3*alignedFaceSize*alignedFaceSize)
	plane := alignedFaceSize * alignedFaceSize
	for y := 0; y < alignedFaceSize; y++ {
		for x := 0; x < alignedFaceSize; x++ {
			u, v := float64(x)-tx, float64(y)-ty
			sourcePixelX := float32((a*u + b*v) / inverse)
			sourcePixelY := float32((-b*u + a*v) / inverse)
			r, g, blue := bilinearRGB(image, sourcePixelX, sourcePixelY)
			offset := y*alignedFaceSize + x
			result[offset] = (r - 127.5) / 127.5
			result[plane+offset] = (g - 127.5) / 127.5
			result[2*plane+offset] = (blue - 127.5) / 127.5
		}
	}
	return result, true
}

func bilinearRGB(image face.DecodedImage, x, y float32) (float32, float32, float32) {
	if x < 0 || y < 0 || x > float32(image.Width-1) || y > float32(image.Height-1) {
		return 0, 0, 0
	}
	x0, y0 := int(math.Floor(float64(x))), int(math.Floor(float64(y)))
	x1, y1 := min(x0+1, image.Width-1), min(y0+1, image.Height-1)
	fx, fy := x-float32(x0), y-float32(y0)
	value := func(channel int) float32 {
		topLeft := float32(image.RGB[(y0*image.Width+x0)*3+channel])
		topRight := float32(image.RGB[(y0*image.Width+x1)*3+channel])
		bottomLeft := float32(image.RGB[(y1*image.Width+x0)*3+channel])
		bottomRight := float32(image.RGB[(y1*image.Width+x1)*3+channel])
		return (topLeft*(1-fx)+topRight*fx)*(1-fy) + (bottomLeft*(1-fx)+bottomRight*fx)*fy
	}
	return value(0), value(1), value(2)
}

func runtimeError(err error) error {
	if errors.Is(err, aimodel.ErrInferenceRuntimeUnavailable) || errors.Is(err, aimodel.ErrModelIncompatible) {
		return errors.Join(face.ErrRuntimeUnavailable, err)
	}
	return err
}

func finite(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

var _ face.Runtime = (*Runtime)(nil)
