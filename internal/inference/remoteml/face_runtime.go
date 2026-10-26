package remoteml

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/media"
)

// FaceRuntime adapts already-open media through the canonical bounded decoder.
// It owns neither files nor the client's lifetime. Activation must bind a face
// generation before passing this runtime to the existing face Processor.
type FaceRuntime struct {
	decoder face.ImageDecoder
	client  *FaceClient
}

func NewFaceRuntime(decoder face.ImageDecoder, client *FaceClient) (*FaceRuntime, error) {
	if decoder == nil || client == nil || client.client == nil {
		return nil, face.ErrInvalidInput
	}
	return &FaceRuntime{decoder: decoder, client: client}, nil
}

func (r *FaceRuntime) AnalyzeFaces(ctx context.Context, source io.ReadSeeker, format media.Format, maxBytes int64) ([]face.Candidate, error) {
	if source == nil || maxBytes < 1 || maxBytes > face.MaxInputBytes {
		return nil, face.ErrInvalidInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decoded, err := r.decoder.DecodeFaceImage(ctx, source, format, maxBytes, face.MaxDecodeDimension)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.Join(face.ErrInvalidInput, err)
	}
	if decoded.Width < 1 || decoded.Height < 1 || decoded.Width > face.MaxDecodeDimension ||
		decoded.Height > face.MaxDecodeDimension || len(decoded.RGB) != decoded.Width*decoded.Height*3 {
		return nil, face.ErrInvalidOutput
	}
	// Only reconstructed pixels leave the process, never original metadata or paths.
	pixels := image.NewNRGBA(image.Rect(0, 0, decoded.Width, decoded.Height))
	for y := 0; y < decoded.Height; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := 0; x < decoded.Width; x++ {
			src, dst := (y*decoded.Width+x)*3, y*pixels.Stride+x*4
			copy(pixels.Pix[dst:dst+3], decoded.RGB[src:src+3])
			pixels.Pix[dst+3] = 255
		}
	}
	w := &facePNGWriter{ctx: ctx}
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(w, pixels); err != nil {
		return nil, err
	}
	detections, err := r.client.Detect(ctx, w.buffer.Bytes())
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errors.Is(err, ErrBusy) {
			return nil, errors.Join(face.ErrRuntimeBusy, err)
		}
		return nil, errors.Join(face.ErrRuntimeUnavailable, err)
	}
	result := make([]face.Candidate, len(detections))
	for i, detection := range detections {
		quality, err := face.SizeWeightedDetectionQuality(detection.Detection,
			detection.Box.Width*float32(decoded.Width), detection.Box.Height*float32(decoded.Height))
		if err != nil {
			return nil, err
		}
		result[i] = face.Candidate{Box: detection.Box, Detection: detection.Detection, Quality: quality, Embedding: detection.Embedding}
	}
	return result, nil
}

type facePNGWriter struct {
	ctx    context.Context
	buffer bytes.Buffer
}

func (w *facePNGWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) > MaxImageBytes-w.buffer.Len() {
		return 0, face.ErrInvalidInput
	}
	return w.buffer.Write(p)
}

var _ face.Runtime = (*FaceRuntime)(nil)
