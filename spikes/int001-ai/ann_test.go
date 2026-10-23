package main

import "testing"

func TestANNBenchmarkRejectsCorruptIndexAndRecovers(t *testing.T) {
	report, err := BenchmarkANN(200, 16, 3, 5, 42, 8, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !report.CorruptLoadRejected || !report.RecoverySearchOK {
		t.Fatalf("recovery evidence failed: %#v", report)
	}
	if report.MeanRecallAtK < 0 || report.MeanRecallAtK > 1 {
		t.Fatalf("invalid recall: %#v", report)
	}
}
