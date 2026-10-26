package semantic

import (
	"encoding/hex"
	"errors"
	"math"
	"testing"
)

func TestEmbeddingEncodingIsDeterministicLittleEndianFloat16(t *testing.T) {
	encoded, err := EncodeEmbedding([]float32{3, 4}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := hex.EncodeToString(encoded), "cd38663a"; got != want {
		t.Fatalf("encoded = %s, want %s", got, want)
	}
	decoded, err := DecodeEmbedding(encoded, 2)
	if err != nil {
		t.Fatal(err)
	}
	if difference := math.Abs(float64(decoded[0] - 0.6)); difference > 0.001 {
		t.Fatalf("decoded[0] = %f", decoded[0])
	}
	if norm := math.Hypot(float64(decoded[0]), float64(decoded[1])); math.Abs(norm-1) > 1e-6 {
		t.Fatalf("decoded norm = %f", norm)
	}
}

func TestEmbeddingEncodingRejectsInvalidValues(t *testing.T) {
	for _, test := range []struct {
		name      string
		vector    []float32
		dimension int
	}{
		{name: "wrong dimension", vector: []float32{1}, dimension: 2},
		{name: "zero norm", vector: []float32{0, 0}, dimension: 2},
		{name: "nan", vector: []float32{1, float32(math.NaN())}, dimension: 2},
		{name: "infinity", vector: []float32{1, float32(math.Inf(1))}, dimension: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := EncodeEmbedding(test.vector, test.dimension); !errors.Is(err, ErrInvalidEmbedding) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestEmbeddingDecodeRejectsMalformedStorage(t *testing.T) {
	for _, encoded := range [][]byte{{}, {0, 0}, {0, 124, 0, 0}} {
		if _, err := DecodeEmbedding(encoded, 2); !errors.Is(err, ErrInvalidEmbedding) {
			t.Fatalf("DecodeEmbedding(%x) error = %v", encoded, err)
		}
	}
}

func TestFloat16BoundaryValues(t *testing.T) {
	for _, test := range []struct {
		value float32
		half  uint16
	}{
		{value: 1, half: 0x3c00},
		{value: -2, half: 0xc000},
		{value: 0, half: 0x0000},
		{value: float32(math.Ldexp(1, -24)), half: 0x0001},
	} {
		half, ok := float32ToHalf(test.value)
		if !ok || half != test.half || halfToFloat32(half) != test.value {
			t.Fatalf("round trip %g = %#04x/%g, want %#04x", test.value, half, halfToFloat32(half), test.half)
		}
	}
}
