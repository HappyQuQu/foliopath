package main

import (
	"math"
	"testing"
)

func TestHalfRoundTrip(t *testing.T) {
	for _, value := range []float32{-1, -0.1, 0, 0.001, 0.1, 1} {
		actual := halfToFloat32(float32ToHalf(value))
		if math.Abs(float64(actual-value)) > 0.001 {
			t.Fatalf("round trip %f = %f", value, actual)
		}
	}
}

func TestQuantizationBenchmark(t *testing.T) {
	report, err := BenchmarkQuantization(100, 16, 2, 5, 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Formats) != 2 || report.Formats[0].StorageBytes >= report.Baseline.StorageBytes {
		t.Fatalf("unexpected quantization report: %#v", report)
	}
	for _, format := range report.Formats {
		if format.MeanRecallAtK < 0 || format.MeanRecallAtK > 1 {
			t.Fatalf("invalid recall: %#v", format)
		}
	}
}
