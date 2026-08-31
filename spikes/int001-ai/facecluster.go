package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
)

type FaceClusterState struct {
	Persons         map[string]struct{}
	Assignments     map[string]string
	CannotLinks     map[[2]string]struct{}
	ModelGeneration string
}

type AnonymousFaceCluster struct {
	ID   string   `json:"id"`
	Core []string `json:"core"`
	Edge []string `json:"edge"`
}

type FaceClusterPlan struct {
	ModelGeneration string                 `json:"model_generation"`
	Clusters        []AnonymousFaceCluster `json:"clusters"`
	Unclustered     []string               `json:"unclustered"`
	Assigned        map[string]string      `json:"assigned"`
}

func NewFaceClusterState(modelGeneration string) *FaceClusterState {
	return &FaceClusterState{
		Persons:         make(map[string]struct{}),
		Assignments:     make(map[string]string),
		CannotLinks:     make(map[[2]string]struct{}),
		ModelGeneration: modelGeneration,
	}
}

func PlanAnonymousFaceClusters(
	observations []FaceObservation,
	state *FaceClusterState,
	modelGeneration string,
	coreThreshold float64,
	edgeThreshold float64,
) (FaceClusterPlan, error) {
	if state == nil {
		return FaceClusterPlan{}, errors.New("face cluster state is required")
	}
	if coreThreshold < -1 || coreThreshold > 1 || edgeThreshold < -1 || edgeThreshold > coreThreshold {
		return FaceClusterPlan{}, errors.New("face thresholds must satisfy -1 <= edge <= core <= 1")
	}
	ordered := append([]FaceObservation(nil), observations...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	seen := make(map[string]struct{}, len(ordered))
	dimensions := 0
	for index, observation := range ordered {
		if observation.ID == "" || len(observation.Embedding) == 0 {
			return FaceClusterPlan{}, errors.New("every face requires an id and embedding")
		}
		if _, exists := seen[observation.ID]; exists {
			return FaceClusterPlan{}, fmt.Errorf("duplicate face %q", observation.ID)
		}
		seen[observation.ID] = struct{}{}
		if index == 0 {
			dimensions = len(observation.Embedding)
		} else if len(observation.Embedding) != dimensions {
			return FaceClusterPlan{}, errors.New("face embeddings must have consistent dimensions")
		}
	}

	plan := FaceClusterPlan{
		ModelGeneration: modelGeneration,
		Assigned:        make(map[string]string),
	}
	var candidates []FaceObservation
	for _, observation := range ordered {
		if personID := state.Assignments[observation.ID]; personID != "" {
			plan.Assigned[observation.ID] = personID
			continue
		}
		candidates = append(candidates, observation)
	}

	var groups [][]FaceObservation
	for _, candidate := range candidates {
		joined := false
		for groupIndex := range groups {
			if canJoinCore(candidate, groups[groupIndex], state, coreThreshold) {
				groups[groupIndex] = append(groups[groupIndex], candidate)
				joined = true
				break
			}
		}
		if !joined {
			groups = append(groups, []FaceObservation{candidate})
		}
	}

	var singletons []FaceObservation
	for _, group := range groups {
		if len(group) == 1 {
			singletons = append(singletons, group[0])
			continue
		}
		ids := faceIDs(group)
		plan.Clusters = append(plan.Clusters, AnonymousFaceCluster{ID: anonymousClusterID(ids), Core: ids})
	}
	for _, singleton := range singletons {
		bestIndex := -1
		bestSimilarity := -2.0
		for clusterIndex, cluster := range plan.Clusters {
			members := observationsByID(ordered, cluster.Core)
			if conflictsWithAny(singleton.ID, cluster.Core, state) {
				continue
			}
			var total float64
			for _, member := range members {
				total += cosine(singleton.Embedding, member.Embedding)
			}
			average := total / float64(len(members))
			if average >= edgeThreshold && (average > bestSimilarity || average == bestSimilarity && cluster.ID < plan.Clusters[bestIndex].ID) {
				bestIndex = clusterIndex
				bestSimilarity = average
			}
		}
		if bestIndex < 0 {
			plan.Unclustered = append(plan.Unclustered, singleton.ID)
			continue
		}
		plan.Clusters[bestIndex].Edge = append(plan.Clusters[bestIndex].Edge, singleton.ID)
	}
	for index := range plan.Clusters {
		sort.Strings(plan.Clusters[index].Edge)
	}
	sort.Slice(plan.Clusters, func(i, j int) bool { return plan.Clusters[i].ID < plan.Clusters[j].ID })
	sort.Strings(plan.Unclustered)
	return plan, nil
}

func (state *FaceClusterState) CreatePersonFromAnonymous(
	plan FaceClusterPlan,
	clusterID string,
	personID string,
) error {
	if personID == "" {
		return errors.New("person id is required")
	}
	if _, exists := state.Persons[personID]; exists {
		return errors.New("person already exists")
	}
	cluster, err := findAnonymousCluster(plan, clusterID)
	if err != nil {
		return err
	}
	if err := state.validateAnonymousClusterAssignment(cluster, personID); err != nil {
		return err
	}
	state.Persons[personID] = struct{}{}
	state.commitAnonymousClusterAssignment(cluster, personID)
	return nil
}

func (state *FaceClusterState) MergeAnonymousIntoPerson(
	plan FaceClusterPlan,
	clusterID string,
	personID string,
) error {
	if _, exists := state.Persons[personID]; !exists {
		return errors.New("person does not exist")
	}
	cluster, err := findAnonymousCluster(plan, clusterID)
	if err != nil {
		return err
	}
	if err := state.validateAnonymousClusterAssignment(cluster, personID); err != nil {
		return err
	}
	state.commitAnonymousClusterAssignment(cluster, personID)
	return nil
}

func (state *FaceClusterState) AssignUnassignedFace(faceID string, personID string) error {
	if faceID == "" {
		return errors.New("face id is required")
	}
	if _, exists := state.Persons[personID]; !exists {
		return errors.New("person does not exist")
	}
	if assigned := state.Assignments[faceID]; assigned != "" && assigned != personID {
		return errors.New("automatic overwrite of a manual assignment is forbidden")
	}
	for otherFaceID, assignedPersonID := range state.Assignments {
		if assignedPersonID != personID {
			continue
		}
		if _, blocked := state.CannotLinks[orderedFacePair(faceID, otherFaceID)]; blocked {
			return errors.New("manual assignment conflicts with a cannot-link")
		}
	}
	state.Assignments[faceID] = personID
	return nil
}

func (state *FaceClusterState) AddCannotLink(left string, right string) error {
	if left == "" || right == "" || left == right {
		return errors.New("cannot-link requires two distinct faces")
	}
	if state.Assignments[left] != "" && state.Assignments[left] == state.Assignments[right] {
		return errors.New("cannot-link conflicts with an existing manual person assignment")
	}
	state.CannotLinks[orderedFacePair(left, right)] = struct{}{}
	return nil
}

func (state *FaceClusterState) validateAnonymousClusterAssignment(
	cluster AnonymousFaceCluster,
	personID string,
) error {
	// Edge faces are suggestions, not confirmed group membership. A user must
	// assign them individually rather than inheriting a bulk group action.
	for _, faceID := range cluster.Core {
		if assigned := state.Assignments[faceID]; assigned != "" && assigned != personID {
			return errors.New("anonymous cluster contains an already assigned face")
		}
		for otherFaceID, assignedPersonID := range state.Assignments {
			if assignedPersonID != personID {
				continue
			}
			if _, blocked := state.CannotLinks[orderedFacePair(faceID, otherFaceID)]; blocked {
				return errors.New("anonymous cluster conflicts with a cannot-link")
			}
		}
	}
	return nil
}

func (state *FaceClusterState) commitAnonymousClusterAssignment(
	cluster AnonymousFaceCluster,
	personID string,
) {
	for _, faceID := range cluster.Core {
		state.Assignments[faceID] = personID
	}
}

func findAnonymousCluster(plan FaceClusterPlan, clusterID string) (AnonymousFaceCluster, error) {
	for _, cluster := range plan.Clusters {
		if cluster.ID == clusterID {
			return cluster, nil
		}
	}
	return AnonymousFaceCluster{}, errors.New("anonymous cluster does not exist in this plan")
}

func canJoinCore(
	candidate FaceObservation,
	group []FaceObservation,
	state *FaceClusterState,
	threshold float64,
) bool {
	for _, member := range group {
		if _, blocked := state.CannotLinks[orderedFacePair(candidate.ID, member.ID)]; blocked {
			return false
		}
		if cosine(candidate.Embedding, member.Embedding) < threshold {
			return false
		}
	}
	return true
}

func conflictsWithAny(faceID string, members []string, state *FaceClusterState) bool {
	for _, member := range members {
		if _, blocked := state.CannotLinks[orderedFacePair(faceID, member)]; blocked {
			return true
		}
	}
	return false
}

func orderedFacePair(left string, right string) [2]string {
	if right < left {
		left, right = right, left
	}
	return [2]string{left, right}
}

func faceIDs(observations []FaceObservation) []string {
	ids := make([]string, len(observations))
	for index, observation := range observations {
		ids[index] = observation.ID
	}
	sort.Strings(ids)
	return ids
}

func observationsByID(observations []FaceObservation, ids []string) []FaceObservation {
	selected := make([]FaceObservation, 0, len(ids))
	for _, id := range ids {
		for _, observation := range observations {
			if observation.ID == id {
				selected = append(selected, observation)
				break
			}
		}
	}
	return selected
}

func anonymousClusterID(ids []string) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("face-cluster-v1:%q", ids)))
	return fmt.Sprintf("anonymous-%x", digest[:8])
}
