package face

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"sort"
)

var ErrInvalidClusteringInput = errors.New("invalid face clustering input")

type VectorFace struct {
	ID     string
	Vector []float32
}

type FacePair struct{ Left, Right string }

type ClusterMember struct {
	FaceID     string
	Role       string
	Confidence float32
}

type Cluster struct {
	ID      string
	Role    string
	Members []ClusterMember
}

type ClusterProfile struct {
	CoreSimilarity float32
	EdgeSimilarity float32
	MinCoreSize    int
}

const (
	exactClusteringLimit = 4096
	lshTableCount        = 4
	lshBitsPerTable      = 12
	lshNeighborWindow    = 8
)

type indexedFace struct {
	index     int
	signature uint16
}

// ClusterFaces deterministically forms anchor-coherent core components, then
// attaches remaining faces to their nearest permitted core. Every core member
// must meet CoreSimilarity against the component's smallest-ID anchor, so a
// chain of pairwise-similar bridge faces cannot join dissimilar identities. The
// explicit cannot-link set always wins over model similarity.
func ClusterFaces(generationID string, faces []VectorFace, cannotLinks []FacePair, profile ClusterProfile) ([]Cluster, error) {
	if len(generationID) < 8 || len(generationID) > 128 || len(faces) > 1_000_000 ||
		profile.CoreSimilarity <= -1 || profile.CoreSimilarity > 1 ||
		profile.EdgeSimilarity <= -1 || profile.EdgeSimilarity > profile.CoreSimilarity || profile.MinCoreSize < 2 {
		return nil, ErrInvalidClusteringInput
	}
	ordered := append([]VectorFace(nil), faces...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	dimension := 0
	indexByID := make(map[string]int, len(ordered))
	for index := range ordered {
		if len(ordered[index].ID) < 8 || len(ordered[index].ID) > 128 || len(ordered[index].Vector) < 1 ||
			len(ordered[index].Vector) > MaxEmbeddingDimension || index > 0 && ordered[index-1].ID == ordered[index].ID {
			return nil, ErrInvalidClusteringInput
		}
		if dimension == 0 {
			dimension = len(ordered[index].Vector)
		}
		if len(ordered[index].Vector) != dimension || !normalizeInPlace(ordered[index].Vector) {
			return nil, ErrInvalidClusteringInput
		}
		indexByID[ordered[index].ID] = index
	}
	blocked := make(map[FacePair]struct{}, len(cannotLinks))
	for _, pair := range cannotLinks {
		if pair.Left >= pair.Right {
			return nil, ErrInvalidClusteringInput
		}
		if _, left := indexByID[pair.Left]; !left {
			return nil, ErrInvalidClusteringInput
		}
		if _, right := indexByID[pair.Right]; !right {
			return nil, ErrInvalidClusteringInput
		}
		blocked[pair] = struct{}{}
	}
	parent := make([]int, len(ordered))
	componentMembers := make([][]int, len(ordered))
	for index := range parent {
		parent[index] = index
		componentMembers[index] = []int{index}
	}
	var find func(int) int
	find = func(value int) int {
		if parent[value] != value {
			parent[value] = find(parent[value])
		}
		return parent[value]
	}
	union := func(a, b int) {
		a, b = find(a), find(b)
		if a != b {
			if a < b {
				parent[b] = a
				componentMembers[a] = append(componentMembers[a], componentMembers[b]...)
				componentMembers[b] = nil
			} else {
				parent[a] = b
				componentMembers[b] = append(componentMembers[b], componentMembers[a]...)
				componentMembers[a] = nil
			}
		}
	}
	blockedIndexes := make(map[int]map[int]struct{}, len(blocked)*2)
	for pair := range blocked {
		left, right := indexByID[pair.Left], indexByID[pair.Right]
		if blockedIndexes[left] == nil {
			blockedIndexes[left] = make(map[int]struct{})
		}
		if blockedIndexes[right] == nil {
			blockedIndexes[right] = make(map[int]struct{})
		}
		blockedIndexes[left][right] = struct{}{}
		blockedIndexes[right][left] = struct{}{}
	}
	canMerge := func(leftRoot, rightRoot int) bool {
		// union keeps the smallest ordered index as the component root and thus as
		// its stable anchor. Existing members already satisfy their current anchor;
		// only the component moving under a new smaller anchor needs revalidation.
		anchorRoot, movingRoot := leftRoot, rightRoot
		if anchorRoot > movingRoot {
			anchorRoot, movingRoot = movingRoot, anchorRoot
		}
		for _, member := range componentMembers[movingRoot] {
			if cosine(ordered[anchorRoot].Vector, ordered[member].Vector) < profile.CoreSimilarity {
				return false
			}
		}
		if len(componentMembers[leftRoot]) > len(componentMembers[rightRoot]) {
			leftRoot, rightRoot = rightRoot, leftRoot
		}
		for _, left := range componentMembers[leftRoot] {
			for right := range blockedIndexes[left] {
				if find(right) == rightRoot {
					return false
				}
			}
		}
		return true
	}
	considerPair := func(left, right int) {
		if left > right {
			left, right = right, left
		}
		if left == right {
			return
		}
		if _, denied := blocked[FacePair{ordered[left].ID, ordered[right].ID}]; denied {
			return
		}
		leftRoot, rightRoot := find(left), find(right)
		if leftRoot != rightRoot && cosine(ordered[left].Vector, ordered[right].Vector) >= profile.CoreSimilarity &&
			canMerge(leftRoot, rightRoot) {
			union(left, right)
		}
	}
	var largeNeighbors [][]int
	if len(ordered) <= exactClusteringLimit {
		for left := range ordered {
			for right := left + 1; right < len(ordered); right++ {
				considerPair(left, right)
			}
		}
	} else {
		largeNeighbors = make([][]int, len(ordered))
		for table := 0; table < lshTableCount; table++ {
			indexed := make([]indexedFace, len(ordered))
			for index := range ordered {
				indexed[index] = indexedFace{index: index, signature: lshSignature(ordered[index].Vector, table)}
			}
			sort.Slice(indexed, func(i, j int) bool {
				if indexed[i].signature != indexed[j].signature {
					return indexed[i].signature < indexed[j].signature
				}
				return ordered[indexed[i].index].ID < ordered[indexed[j].index].ID
			})
			for position := range indexed {
				start := max(0, position-lshNeighborWindow)
				for start < position && indexed[start].signature != indexed[position].signature {
					start++
				}
				for previous := start; previous < position; previous++ {
					left, right := indexed[previous].index, indexed[position].index
					largeNeighbors[left] = append(largeNeighbors[left], right)
					largeNeighbors[right] = append(largeNeighbors[right], left)
					considerPair(left, right)
				}
			}
		}
	}
	components := make(map[int][]int)
	for index := range ordered {
		components[find(index)] = append(components[find(index)], index)
	}
	clusters := make([]Cluster, 0)
	assigned := make(map[int]struct{})
	for _, indexes := range components {
		if len(indexes) < profile.MinCoreSize {
			continue
		}
		ids := make([]string, len(indexes))
		members := make([]ClusterMember, len(indexes))
		for position, index := range indexes {
			ids[position] = ordered[index].ID
			members[position] = ClusterMember{FaceID: ordered[index].ID, Role: "core", Confidence: 1}
			assigned[index] = struct{}{}
		}
		clusters = append(clusters, Cluster{ID: deterministicClusterID(generationID, ids), Role: "core", Members: members})
	}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].ID < clusters[j].ID })
	coreClusterByFace := make(map[int]int, len(assigned))
	for clusterIndex := range clusters {
		for _, member := range clusters[clusterIndex].Members {
			coreClusterByFace[indexByID[member.FaceID]] = clusterIndex
		}
	}
	coreClusterCount := len(clusters)
	for index := range ordered {
		if _, core := assigned[index]; core {
			continue
		}
		bestCluster, bestScore := -1, float32(-2)
		if largeNeighbors != nil {
			blockedClusters := make(map[int]struct{}, len(blockedIndexes[index]))
			for blockedIndex := range blockedIndexes[index] {
				if clusterIndex, core := coreClusterByFace[blockedIndex]; core {
					blockedClusters[clusterIndex] = struct{}{}
				}
			}
			for _, neighbor := range largeNeighbors[index] {
				clusterIndex, core := coreClusterByFace[neighbor]
				if !core {
					continue
				}
				if _, denied := blockedClusters[clusterIndex]; denied {
					continue
				}
				score := cosine(ordered[index].Vector, ordered[neighbor].Vector)
				if score >= profile.EdgeSimilarity &&
					(score > bestScore || score == bestScore && (bestCluster < 0 || clusterIndex < bestCluster)) {
					bestCluster, bestScore = clusterIndex, score
				}
			}
		} else {
			for clusterIndex := 0; clusterIndex < coreClusterCount; clusterIndex++ {
				permitted := true
				localBest := float32(-2)
				for _, member := range clusters[clusterIndex].Members {
					left, right := ordered[index].ID, member.FaceID
					if left > right {
						left, right = right, left
					}
					if _, denied := blocked[FacePair{left, right}]; denied {
						permitted = false
						break
					}
					score := cosine(ordered[index].Vector, ordered[indexByID[member.FaceID]].Vector)
					if score > localBest {
						localBest = score
					}
				}
				if permitted && localBest >= profile.EdgeSimilarity && localBest > bestScore {
					bestCluster, bestScore = clusterIndex, localBest
				}
			}
		}
		if bestCluster >= 0 {
			clusters[bestCluster].Members = append(clusters[bestCluster].Members,
				ClusterMember{FaceID: ordered[index].ID, Role: "edge", Confidence: bestScore})
		} else {
			clusters = append(clusters, Cluster{
				ID: deterministicClusterID(generationID, []string{ordered[index].ID}), Role: "edge",
				Members: []ClusterMember{{FaceID: ordered[index].ID, Role: "edge", Confidence: 0}},
			})
		}
	}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].ID < clusters[j].ID })
	return clusters, nil
}

func normalizeInPlace(vector []float32) bool {
	var norm float64
	for _, value := range vector {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return false
		}
		norm += float64(value) * float64(value)
	}
	if norm == 0 || math.IsInf(norm, 0) {
		return false
	}
	scale := float32(1 / math.Sqrt(norm))
	for index := range vector {
		vector[index] *= scale
	}
	return true
}

func cosine(left, right []float32) float32 {
	var result float32
	for index := range left {
		result += left[index] * right[index]
	}
	return result
}

// lshSignature uses fixed, generation-independent random hyperplanes. It is
// deterministic across architectures and keeps large-library candidate work
// bounded; exact cosine and cannot-link checks still decide every merge.
func lshSignature(vector []float32, table int) uint16 {
	var signature uint16
	for bit := 0; bit < lshBitsPerTable; bit++ {
		var projection float64
		seed := uint64(table*lshBitsPerTable+bit+1) * 0x9e3779b97f4a7c15
		for dimension, value := range vector {
			x := seed + uint64(dimension+1)*0xbf58476d1ce4e5b9
			x ^= x >> 30
			x *= 0xbf58476d1ce4e5b9
			x ^= x >> 27
			x *= 0x94d049bb133111eb
			x ^= x >> 31
			if x&1 == 0 {
				projection += float64(value)
			} else {
				projection -= float64(value)
			}
		}
		if projection >= 0 {
			signature |= 1 << bit
		}
	}
	return signature
}

func deterministicClusterID(generationID string, faceIDs []string) string {
	hash := sha256.New()
	hash.Write([]byte(generationID))
	for _, id := range faceIDs {
		hash.Write([]byte{0})
		hash.Write([]byte(id))
	}
	return "fcluster_" + hex.EncodeToString(hash.Sum(nil)[:16])
}
