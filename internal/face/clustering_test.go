package face

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestClusterFaces100KCapacity(t *testing.T) {
	if os.Getenv("FOLIOPATH_RUN_CAPACITY_TEST") != "1" {
		t.Skip("set FOLIOPATH_RUN_CAPACITY_TEST=1")
	}
	const faceCount, dimension = 100_000, 512
	started := time.Now()
	faces := make([]VectorFace, faceCount)
	for pair := 0; pair < faceCount/2; pair++ {
		vector := make([]float32, dimension)
		state := uint64(pair+1) * 0x9e3779b97f4a7c15
		for index := range vector {
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			vector[index] = float32(int64(state%20001) - 10000)
		}
		for member := 0; member < 2; member++ {
			id := pair*2 + member
			faces[id] = VectorFace{ID: fmt.Sprintf("face_%08d", id), Vector: append([]float32(nil), vector...)}
		}
	}
	clusters, err := ClusterFaces("face_generation_capacity", faces, nil, ClusterProfile{CoreSimilarity: .9999, EdgeSimilarity: .99, MinCoreSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	memberCount := 0
	fingerprint := sha256.New()
	fingerprint.Write([]byte("paired"))
	fingerprint.Write([]byte{0})
	for _, cluster := range clusters {
		fingerprint.Write([]byte(cluster.ID))
		fingerprint.Write([]byte{0})
		memberCount += len(cluster.Members)
		for _, member := range cluster.Members {
			fingerprint.Write([]byte(member.FaceID))
			fingerprint.Write([]byte{0})
		}
	}
	if len(clusters) != faceCount/2 || memberCount != faceCount {
		t.Fatalf("members=%d clusters=%d", memberCount, len(clusters))
	}
	pairedClusters := len(clusters)
	pairedMembers := memberCount

	faces, clusters = nil, nil
	runtime.GC()
	singletons := make([]VectorFace, faceCount)
	for item := range singletons {
		vector := make([]float32, dimension)
		state := uint64(item+1) * 0x9e3779b97f4a7c15
		for index := range vector {
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			vector[index] = float32(int64(state%20001) - 10000)
		}
		singletons[item] = VectorFace{ID: fmt.Sprintf("face_singleton_%08d", item), Vector: vector}
	}
	singletonClusters, err := ClusterFaces("face_generation_singleton_capacity", singletons, nil,
		ClusterProfile{CoreSimilarity: 1, EdgeSimilarity: 1, MinCoreSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	singletonMembers := 0
	fingerprint.Write([]byte("singletons"))
	fingerprint.Write([]byte{0})
	for _, cluster := range singletonClusters {
		if cluster.Role != "edge" || len(cluster.Members) != 1 || cluster.Members[0].Role != "edge" {
			t.Fatalf("non-core capacity cluster=%+v", cluster)
		}
		singletonMembers++
		fingerprint.Write([]byte(cluster.ID))
		fingerprint.Write([]byte{0})
		fingerprint.Write([]byte(cluster.Members[0].FaceID))
		fingerprint.Write([]byte{0})
	}
	if len(singletonClusters) != faceCount || singletonMembers != faceCount {
		t.Fatalf("singleton members=%d clusters=%d", singletonMembers, len(singletonClusters))
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	t.Logf("face_capacity_metrics face_count=%d embedding_dimension=%d paired_cluster_count=%d paired_member_count=%d singleton_cluster_count=%d singleton_member_count=%d goos=%s goarch=%s deterministic_sha256=%s elapsed_ms=%d memory_sys_bytes=%d",
		faceCount, dimension, pairedClusters, pairedMembers, len(singletonClusters), singletonMembers, runtime.GOOS, runtime.GOARCH,
		hex.EncodeToString(fingerprint.Sum(nil)), time.Since(started).Milliseconds(), memory.Sys)
}

func TestClusterFacesIsDeterministicAndHonorsCannotLinks(t *testing.T) {
	faces := []VectorFace{
		{ID: "face_item_0003", Vector: []float32{.99, .01}},
		{ID: "face_item_0001", Vector: []float32{1, 0}},
		{ID: "face_item_0004", Vector: []float32{0, 1}},
		{ID: "face_item_0002", Vector: []float32{.98, .02}},
	}
	profile := ClusterProfile{CoreSimilarity: .99, EdgeSimilarity: .95, MinCoreSize: 2}
	first, err := ClusterFaces("generation_test", faces, []FacePair{{Left: "face_item_0001", Right: "face_item_0003"}}, profile)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ClusterFaces("generation_test", []VectorFace{faces[3], faces[2], faces[1], faces[0]},
		[]FacePair{{Left: "face_item_0001", Right: "face_item_0003"}}, profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	for index := range first {
		if first[index].ID != second[index].ID {
			t.Fatalf("non-deterministic first=%+v second=%+v", first, second)
		}
		if first[index].Role == "core" {
			for _, member := range first[index].Members {
				if member.FaceID == "face_item_0003" {
					t.Fatal("cannot-linked face attached to core")
				}
			}
		}
	}
}

func TestClusterFacesRejectsTransitiveCoreBridge(t *testing.T) {
	faces := []VectorFace{
		{ID: "face_bridge_0001", Vector: []float32{1, 0}},
		{ID: "face_bridge_0002", Vector: []float32{.8, .6}},
		{ID: "face_bridge_0003", Vector: []float32{.28, .96}},
	}
	profile := ClusterProfile{CoreSimilarity: .75, EdgeSimilarity: .75, MinCoreSize: 2}
	for _, input := range [][]VectorFace{faces, {faces[2], faces[0], faces[1]}} {
		clusters, err := ClusterFaces("generation_bridge", input, nil, profile)
		if err != nil {
			t.Fatal(err)
		}
		cluster, member, found := findClusterMember(clusters, "face_bridge_0003")
		if !found || member.Role != "edge" {
			t.Fatalf("bridge face must not become core: cluster=%+v member=%+v found=%t", cluster, member, found)
		}
		for _, candidate := range clusters {
			coreMembers := 0
			for _, candidateMember := range candidate.Members {
				if candidateMember.Role == "core" {
					coreMembers++
				}
			}
			if coreMembers > 2 {
				t.Fatalf("transitive bridge formed oversized core: %+v", candidate)
			}
		}
	}
}

func TestClusterFacesDoesNotSkipNeighborsAtLSHBucketBoundary(t *testing.T) {
	const faceCount = exactClusteringLimit + 2
	faces := make([]VectorFace, faceCount)
	for index := range faces {
		vector := []float32{1, 0}
		if index >= faceCount/2 {
			vector = []float32{-1, 0}
		}
		faces[index] = VectorFace{ID: fmt.Sprintf("face_boundary_%08d", index), Vector: vector}
	}
	clusters, err := ClusterFaces("generation_bucket_boundary", faces, nil,
		ClusterProfile{CoreSimilarity: .9999, EdgeSimilarity: .99, MinCoreSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 2 {
		t.Fatalf("clusters=%d want=2", len(clusters))
	}
	members := 0
	for _, cluster := range clusters {
		if cluster.Role != "core" {
			t.Fatalf("cluster %s role=%s", cluster.ID, cluster.Role)
		}
		for _, member := range cluster.Members {
			if member.Role != "core" {
				t.Fatalf("member %s role=%s; LSH bucket boundary skipped a core neighbor", member.FaceID, member.Role)
			}
			members++
		}
	}
	if members != faceCount {
		t.Fatalf("members=%d want=%d", members, faceCount)
	}
}

func TestClusterFacesLargeEdgeUsesBoundedCoreNeighborsAndCannotLink(t *testing.T) {
	fixture := func() []VectorFace {
		faces := make([]VectorFace, exactClusteringLimit+2)
		faces[0] = VectorFace{ID: "face_large_edge_0000", Vector: []float32{1, 0}}
		faces[1] = VectorFace{ID: "face_large_edge_0001", Vector: []float32{1, 0}}
		faces[2] = VectorFace{ID: "face_large_edge_0002", Vector: []float32{1, .01}}
		for index := 3; index < len(faces); index++ {
			faces[index] = VectorFace{ID: fmt.Sprintf("face_large_edge_%04d", index),
				Vector: []float32{float32(index%97 + 2), float32(index%89 + 3)}}
		}
		return faces
	}
	sharedTable := false
	for table := 0; table < lshTableCount; table++ {
		if lshSignature(fixture()[0].Vector, table) == lshSignature(fixture()[2].Vector, table) {
			sharedTable = true
			break
		}
	}
	if !sharedTable {
		t.Fatal("test fixture does not share an LSH table")
	}
	profile := ClusterProfile{CoreSimilarity: 1, EdgeSimilarity: .9, MinCoreSize: 2}
	clusters, err := ClusterFaces("generation_large_edge", fixture(), nil, profile)
	if err != nil {
		t.Fatal(err)
	}
	cluster, member, found := findClusterMember(clusters, "face_large_edge_0002")
	if !found || cluster.Role != "core" || member.Role != "edge" {
		t.Fatalf("unconstrained edge cluster=%+v member=%+v found=%t", cluster, member, found)
	}

	clusters, err = ClusterFaces("generation_large_edge", fixture(), []FacePair{{
		Left: "face_large_edge_0000", Right: "face_large_edge_0002",
	}}, profile)
	if err != nil {
		t.Fatal(err)
	}
	cluster, member, found = findClusterMember(clusters, "face_large_edge_0002")
	if !found || cluster.Role != "edge" || member.Role != "edge" || len(cluster.Members) != 1 {
		t.Fatalf("cannot-linked edge cluster=%+v member=%+v found=%t", cluster, member, found)
	}
}

func findClusterMember(clusters []Cluster, faceID string) (Cluster, ClusterMember, bool) {
	for _, cluster := range clusters {
		for _, member := range cluster.Members {
			if member.FaceID == faceID {
				return cluster, member, true
			}
		}
	}
	return Cluster{}, ClusterMember{}, false
}

func TestClusterFacesMarksNearestNonCoreAsEdge(t *testing.T) {
	clusters, err := ClusterFaces("generation_edges", []VectorFace{
		{ID: "face_edge_0001", Vector: []float32{1, 0}},
		{ID: "face_edge_0002", Vector: []float32{.999, .001}},
		{ID: "face_edge_0003", Vector: []float32{.94, .34}},
	}, nil, ClusterProfile{CoreSimilarity: .999, EdgeSimilarity: .9, MinCoreSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 || len(clusters[0].Members) != 3 || clusters[0].Members[2].Role != "edge" {
		t.Fatalf("clusters=%+v", clusters)
	}
}

func TestClusterFacesDoesNotAttachEdgeToEdgeSingleton(t *testing.T) {
	clusters, err := ClusterFaces("generation_edge_singletons", []VectorFace{
		{ID: "face_singleton_0001", Vector: []float32{1, 0}},
		{ID: "face_singleton_0002", Vector: []float32{.99, .01}},
	}, nil, ClusterProfile{CoreSimilarity: 1, EdgeSimilarity: .9, MinCoreSize: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 2 {
		t.Fatalf("clusters=%+v; edge-only members must remain separate without a core", clusters)
	}
	for _, cluster := range clusters {
		if cluster.Role != "edge" || len(cluster.Members) != 1 || cluster.Members[0].Role != "edge" {
			t.Fatalf("cluster=%+v", cluster)
		}
	}
}
