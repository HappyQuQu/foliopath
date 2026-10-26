package main

import "testing"

func TestFaceScoreGateAndConstraints(t *testing.T) {
	dataset, err := ReadFaceScoreDataset("testdata/face-score-synthetic.json")
	if err != nil {
		t.Fatal(err)
	}
	report, err := ScoreFaces(dataset)
	if err != nil {
		t.Fatal(err)
	}
	if !report.GatePass || report.CoreClusterPairPrecision != 1 {
		t.Fatalf("expected clean synthetic fixture to pass: %#v", report)
	}
	dataset.Observations[3].ClusterID = "cluster-a"
	dataset.Observations[3].Tier = "core"
	report, err = ScoreFaces(dataset)
	if err != nil {
		t.Fatal(err)
	}
	if report.GatePass || report.CannotLinkViolations == 0 {
		t.Fatalf("expected cross-identity merge to fail: %#v", report)
	}
}
