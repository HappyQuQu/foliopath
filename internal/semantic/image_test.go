package semantic

import (
	"errors"
	"math"
	"testing"
)

func TestPrepareSigLIPImageTensorMatchesReferenceAndCHWOrder(t *testing.T) {
	rgb := make([]byte, SigLIPImageValues)
	copy(rgb, []byte{0, 128, 255, 255, 127, 1})
	tensor, err := PrepareSigLIPImageTensor(rgb)
	if err != nil {
		t.Fatal(err)
	}
	if len(tensor) != SigLIPImageValues {
		t.Fatalf("tensor length = %d, want %d", len(tensor), SigLIPImageValues)
	}
	for _, test := range []struct {
		index int
		bits  uint32
	}{
		{index: 0, bits: 0xbf800000},
		{index: 1, bits: 0x3f800000},
		{index: SigLIPImagePixels, bits: 0x3b808100},
		{index: SigLIPImagePixels + 1, bits: 0xbb808080},
		{index: 2 * SigLIPImagePixels, bits: 0x3f800000},
		{index: 2*SigLIPImagePixels + 1, bits: 0xbf7dfdfe},
	} {
		if actual := math.Float32bits(tensor[test.index]); actual != test.bits {
			t.Errorf("tensor[%d] bits = %#08x, want %#08x", test.index, actual, test.bits)
		}
	}
	if rgb[0] != 0 || rgb[1] != 128 || rgb[2] != 255 {
		t.Fatal("input was mutated")
	}
}

func TestPrepareSigLIPImageTensorRejectsWrongLength(t *testing.T) {
	for _, rgb := range [][]byte{
		nil,
		make([]byte, SigLIPImageValues-1),
		make([]byte, SigLIPImageValues+1),
	} {
		if _, err := PrepareSigLIPImageTensor(rgb); !errors.Is(err, ErrInvalidImageInput) {
			t.Fatalf("length %d error = %v", len(rgb), err)
		}
	}
}
