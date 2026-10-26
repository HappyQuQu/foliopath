package main

import "testing"

func TestAnonymousClusterToManualPersonLifecycle(t *testing.T) {
	observations := []FaceObservation{
		{ID: "a-1", Embedding: []float32{1, 0, 0}},
		{ID: "a-2", Embedding: []float32{0.999, 0.02, 0}},
		{ID: "a-edge", Embedding: []float32{0.93, 0.36, 0}},
		{ID: "a-pose-1", Embedding: []float32{0.82, 0, 0.57}},
		{ID: "a-pose-2", Embedding: []float32{0.80, 0.02, 0.59}},
		{ID: "b-1", Embedding: []float32{0, 1, 0}},
		{ID: "b-2", Embedding: []float32{0.02, 0.999, 0}},
		{ID: "single", Embedding: []float32{0, 0, 1}},
	}
	state := NewFaceClusterState("face-model-v1")
	if err := state.AddCannotLink("a-1", "b-1"); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanAnonymousFaceClusters(observations, state, "face-model-v1", 0.97, 0.90)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Clusters) != 3 {
		t.Fatalf("expected three anonymous groups, got %#v", plan.Clusters)
	}
	var primary, pose, other AnonymousFaceCluster
	for _, cluster := range plan.Clusters {
		switch cluster.Core[0] {
		case "a-1":
			primary = cluster
		case "a-pose-1":
			pose = cluster
		case "b-1":
			other = cluster
		}
	}
	if len(primary.Edge) != 1 || primary.Edge[0] != "a-edge" {
		t.Fatalf("expected edge suggestion on primary group: %#v", primary)
	}
	if err := state.CreatePersonFromAnonymous(plan, primary.ID, "person-a"); err != nil {
		t.Fatal(err)
	}
	if state.Assignments["a-edge"] != "" {
		t.Fatal("edge suggestion must not inherit a bulk person assignment")
	}
	if err := state.AssignUnassignedFace("a-edge", "person-a"); err != nil {
		t.Fatal(err)
	}
	if err := state.MergeAnonymousIntoPerson(plan, pose.ID, "person-a"); err != nil {
		t.Fatal(err)
	}
	if err := state.AssignUnassignedFace("single", "person-a"); err != nil {
		t.Fatal(err)
	}
	if err := state.CreatePersonFromAnonymous(plan, other.ID, "person-b"); err != nil {
		t.Fatal(err)
	}

	reclustered, err := PlanAnonymousFaceClusters(observations, state, "face-model-v2", 0.85, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	if len(reclustered.Clusters) != 0 || len(reclustered.Assigned) != len(observations) {
		t.Fatalf("manual assignments must be excluded from later clustering: %#v", reclustered)
	}
	for _, faceID := range []string{"a-1", "a-2", "a-edge", "a-pose-1", "a-pose-2", "single"} {
		if reclustered.Assigned[faceID] != "person-a" {
			t.Fatalf("manual person lost for %s: %#v", faceID, reclustered.Assigned)
		}
	}
	if reclustered.Assigned["b-1"] != "person-b" || reclustered.Assigned["b-2"] != "person-b" {
		t.Fatalf("second named person was overwritten: %#v", reclustered.Assigned)
	}
}

func TestFaceClusterConstraintsFailClosed(t *testing.T) {
	observations := []FaceObservation{
		{ID: "a", Embedding: []float32{1, 0}},
		{ID: "b", Embedding: []float32{0.999, 0.01}},
		{ID: "c", Embedding: []float32{0.998, 0.02}},
	}
	state := NewFaceClusterState("v1")
	if err := state.AddCannotLink("a", "b"); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanAnonymousFaceClusters(observations, state, "v1", 0.95, 0.90)
	if err != nil {
		t.Fatal(err)
	}
	for _, cluster := range plan.Clusters {
		if contains(cluster.Core, "a") && contains(cluster.Core, "b") {
			t.Fatalf("cannot-link faces joined a core: %#v", cluster)
		}
	}
	state.Persons["person-a"] = struct{}{}
	if err := state.AssignUnassignedFace("a", "person-a"); err != nil {
		t.Fatal(err)
	}
	state.Persons["person-b"] = struct{}{}
	if err := state.AssignUnassignedFace("a", "person-b"); err == nil {
		t.Fatal("expected automatic overwrite of manual assignment to fail")
	}
	if err := state.AssignUnassignedFace("b", "person-a"); err == nil {
		t.Fatal("expected assignment across a cannot-link to fail")
	}
	if err := state.AssignUnassignedFace("c", "person-a"); err != nil {
		t.Fatal(err)
	}
	if err := state.AddCannotLink("a", "c"); err == nil {
		t.Fatal("expected a cannot-link inside one named person to fail")
	}
}

func TestCreatePersonFromStaleAnonymousPlanIsAtomic(t *testing.T) {
	observations := []FaceObservation{
		{ID: "a", Embedding: []float32{1, 0}},
		{ID: "b", Embedding: []float32{0.999, 0.01}},
	}
	state := NewFaceClusterState("v1")
	plan, err := PlanAnonymousFaceClusters(observations, state, "v1", 0.95, 0.90)
	if err != nil {
		t.Fatal(err)
	}
	state.Persons["existing"] = struct{}{}
	state.Assignments["a"] = "existing"
	if err := state.CreatePersonFromAnonymous(plan, plan.Clusters[0].ID, "new-person"); err == nil {
		t.Fatal("expected stale anonymous plan to fail")
	}
	if _, exists := state.Persons["new-person"]; exists {
		t.Fatal("failed create left an empty person")
	}
	if state.Assignments["a"] != "existing" || state.Assignments["b"] != "" {
		t.Fatalf("failed create partially changed assignments: %#v", state.Assignments)
	}
}

func TestAnonymousClusterIDsAreInputOrderStable(t *testing.T) {
	left := []FaceObservation{
		{ID: "b", Embedding: []float32{1, 0}},
		{ID: "a", Embedding: []float32{0.999, 0.01}},
	}
	right := []FaceObservation{left[1], left[0]}
	state := NewFaceClusterState("v1")
	first, err := PlanAnonymousFaceClusters(left, state, "v1", 0.95, 0.90)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanAnonymousFaceClusters(right, state, "v1", 0.95, 0.90)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Clusters) != 1 || len(second.Clusters) != 1 || first.Clusters[0].ID != second.Clusters[0].ID {
		t.Fatalf("cluster id depends on input order: %#v %#v", first, second)
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
