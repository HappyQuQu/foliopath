package onnx

import (
	"errors"
	"math"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestDecodeFaceDetectorHeadsFiltersAndSuppresses(t *testing.T) {
	heads := emptyFaceDetectorHeads()
	first := &heads[0]
	first.cls[0], first.obj[0] = 1, 1
	first.bbox[2], first.bbox[3] = float32(math.Log(4)), float32(math.Log(4))
	for index := 0; index < 10; index++ {
		first.kps[index] = 1
	}
	first.cls[1], first.obj[1] = .95, .95
	first.bbox[4], first.bbox[6], first.bbox[7] = -1, float32(math.Log(4)), float32(math.Log(4))
	for index := 10; index < 20; index++ {
		first.kps[index] = 1
	}

	got, err := decodeFaceDetectorHeads(heads)
	if err != nil || len(got) != 1 {
		t.Fatalf("detections=%#v err=%v", got, err)
	}
	if got[0].Score != 1 || got[0].Width != 32 || got[0].Height != 32 {
		t.Fatalf("detection=%#v", got[0])
	}
}

func TestDecodeFaceDetectorHeadsRejectsInvalidOutput(t *testing.T) {
	heads := emptyFaceDetectorHeads()
	heads[0].cls[0] = float32(math.NaN())
	if _, err := decodeFaceDetectorHeads(heads); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatalf("non-finite error=%v", err)
	}
	heads = emptyFaceDetectorHeads()
	heads[0].bbox = heads[0].bbox[:1]
	if _, err := decodeFaceDetectorHeads(heads); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatalf("shape error=%v", err)
	}
}

func emptyFaceDetectorHeads() []faceDetectorHead {
	result := make([]faceDetectorHead, 0, 3)
	for _, stride := range []int{8, 16, 32} {
		cells := FaceDetectorWidth / stride * (FaceDetectorHeight / stride)
		result = append(result, faceDetectorHead{stride: stride, cls: make([]float32, cells), obj: make([]float32, cells),
			bbox: make([]float32, cells*4), kps: make([]float32, cells*10)})
	}
	return result
}
