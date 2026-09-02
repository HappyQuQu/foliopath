package onnx

import (
	"math"
	"slices"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

const (
	FaceDetectorWidth          = 640
	FaceDetectorHeight         = 640
	FaceDetectorScoreThreshold = float32(0.9)
	FaceDetectorNMSThreshold   = float32(0.3)
	FaceDetectorTopK           = 5000
)

type faceDetectorHead struct {
	stride int
	cls    []float32
	obj    []float32
	bbox   []float32
	kps    []float32
}

type indexedFaceDetection struct {
	detection FaceDetection
	index     int
}

func decodeFaceDetectorHeads(heads []faceDetectorHead) ([]FaceDetection, error) {
	candidates := make([]indexedFaceDetection, 0, 256)
	index := 0
	for _, head := range heads {
		columns := FaceDetectorWidth / head.stride
		rows := FaceDetectorHeight / head.stride
		cells := columns * rows
		if head.stride != 8 && head.stride != 16 && head.stride != 32 ||
			len(head.cls) != cells || len(head.obj) != cells || len(head.bbox) != cells*4 || len(head.kps) != cells*10 {
			return nil, aimodel.ErrModelIncompatible
		}
		for row := 0; row < rows; row++ {
			for column := 0; column < columns; column++ {
				cell := row*columns + column
				classification := clampUnit(head.cls[cell])
				objectness := clampUnit(head.obj[cell])
				if !finite(classification) || !finite(objectness) {
					return nil, aimodel.ErrModelIncompatible
				}
				score := float32(math.Sqrt(float64(classification * objectness)))
				if score < FaceDetectorScoreThreshold {
					continue
				}
				boxOffset := cell * 4
				boxValues := head.bbox[boxOffset : boxOffset+4]
				if !allFinite(boxValues) {
					return nil, aimodel.ErrModelIncompatible
				}
				stride := float32(head.stride)
				centerX := (float32(column) + boxValues[0]) * stride
				centerY := (float32(row) + boxValues[1]) * stride
				width := float32(math.Exp(float64(boxValues[2]))) * stride
				height := float32(math.Exp(float64(boxValues[3]))) * stride
				if !finite(width) || !finite(height) || width <= 0 || height <= 0 {
					return nil, aimodel.ErrModelIncompatible
				}
				detection := FaceDetection{X: centerX - width/2, Y: centerY - height/2,
					Width: width, Height: height, Score: score}
				keypointOffset := cell * 10
				keypoints := head.kps[keypointOffset : keypointOffset+10]
				if !allFinite(keypoints) {
					return nil, aimodel.ErrModelIncompatible
				}
				for point := range detection.Landmarks {
					detection.Landmarks[point] = FacePoint{
						X: (keypoints[point*2] + float32(column)) * stride,
						Y: (keypoints[point*2+1] + float32(row)) * stride,
					}
				}
				candidates = append(candidates, indexedFaceDetection{detection: detection, index: index})
				index++
			}
		}
	}
	slices.SortStableFunc(candidates, func(left, right indexedFaceDetection) int {
		switch {
		case left.detection.Score > right.detection.Score:
			return -1
		case left.detection.Score < right.detection.Score:
			return 1
		case left.index < right.index:
			return -1
		case left.index > right.index:
			return 1
		default:
			return 0
		}
	})
	if len(candidates) > FaceDetectorTopK {
		candidates = candidates[:FaceDetectorTopK]
	}
	kept := make([]FaceDetection, 0, len(candidates))
	for _, candidate := range candidates {
		suppressed := false
		for _, existing := range kept {
			if integerBoxIOU(candidate.detection, existing) > FaceDetectorNMSThreshold {
				suppressed = true
				break
			}
		}
		if !suppressed {
			kept = append(kept, candidate.detection)
		}
	}
	return kept, nil
}

func clampUnit(value float32) float32 {
	return min(max(value, 0), 1)
}

func finite(value float32) bool {
	return !float32IsNaN(value) && !float32IsInf(value)
}

func allFinite(values []float32) bool {
	for _, value := range values {
		if !finite(value) {
			return false
		}
	}
	return true
}

func float32IsNaN(value float32) bool { return value != value }
func float32IsInf(value float32) bool {
	return value > math.MaxFloat32 || value < -math.MaxFloat32
}

func integerBoxIOU(left, right FaceDetection) float32 {
	lx, ly, lw, lh := int(left.X), int(left.Y), int(left.Width), int(left.Height)
	rx, ry, rw, rh := int(right.X), int(right.Y), int(right.Width), int(right.Height)
	intersectionWidth := max(0, min(lx+lw, rx+rw)-max(lx, rx))
	intersectionHeight := max(0, min(ly+lh, ry+rh)-max(ly, ry))
	intersection := intersectionWidth * intersectionHeight
	union := lw*lh + rw*rh - intersection
	if union <= 0 {
		return 0
	}
	return float32(intersection) / float32(union)
}
