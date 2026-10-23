package face

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
)

const (
	MaxPersonAssetPageSize       = 200
	MaxFaceClusterMemberPageSize = 200
)

var (
	ErrInvalidFaceProjection = errors.New("invalid face projection")
	ErrFaceProjectionStale   = errors.New("face projection cursor stale")
)

type CoarseRegion struct{ XPercent, YPercent, WidthPercent, HeightPercent int }
type AssetFaceView struct {
	FaceID            string
	AssetID, Revision int64
	Ordinal           int
	Region            CoarseRegion
	State             string
	PersonID          string
}
type AssetFacesRepository interface {
	ListAssetFaceViews(context.Context, int64) ([]AssetFaceView, error)
}
type AssetFacesService struct{ repository AssetFacesRepository }

func NewAssetFacesService(repository AssetFacesRepository) (*AssetFacesService, error) {
	if repository == nil {
		return nil, errors.New("asset faces repository is required")
	}
	return &AssetFacesService{repository: repository}, nil
}
func (s *AssetFacesService) List(ctx context.Context, assetID int64) ([]AssetFaceView, error) {
	if assetID < 1 {
		return nil, ErrInvalidFaceProjection
	}
	items, err := s.repository.ListAssetFaceViews(ctx, assetID)
	if len(items) > MaxCandidatesPerAsset {
		return nil, ErrInvalidFaceProjection
	}
	return items, err
}

type PersonAssetPosition struct{ MTimeNS, AssetID int64 }
type PersonAssetView struct {
	LibraryID, AssetID, MTimeNS int64
	FaceIDs                     []string
}
type PersonAssetSource struct {
	LibraryID int64
	Revision  int64
	Status    string
}
type PersonAssetSnapshot struct {
	Revision int64
	Sources  []PersonAssetSource
}
type PersonAssetQuery struct {
	PersonID string
	After    *PersonAssetPosition
	Limit    int
}
type PersonAssetRepository interface {
	GetPersonAssetSnapshot(context.Context, string) (PersonAssetSnapshot, error)
	ListPersonAssetViews(context.Context, PersonAssetQuery) ([]PersonAssetView, error)
}
type PersonAssetRequest struct {
	PersonID, Cursor string
	Limit            int
}
type PersonAssetPage struct {
	Items      []PersonAssetView
	NextCursor string
}
type personAssetCursor struct {
	Version     int    `json:"v"`
	PersonHash  string `json:"p"`
	Snapshot    int64  `json:"r"`
	SourcesHash string `json:"s"`
	MTimeNS     int64  `json:"m"`
	AssetID     int64  `json:"i"`
}
type PersonAssetService struct {
	repository PersonAssetRepository
	cursors    *cursorcodec.Codec
}

func NewPersonAssetService(repository PersonAssetRepository, key []byte) (*PersonAssetService, error) {
	if repository == nil {
		return nil, errors.New("person asset repository is required")
	}
	codec, err := cursorcodec.New(key)
	if err != nil {
		return nil, err
	}
	return &PersonAssetService{repository: repository, cursors: codec}, nil
}
func (s *PersonAssetService) List(ctx context.Context, request PersonAssetRequest) (PersonAssetPage, error) {
	if !validReviewID(request.PersonID) || request.Limit < 1 || request.Limit > MaxPersonAssetPageSize {
		return PersonAssetPage{}, ErrInvalidFaceProjection
	}
	snapshot, err := s.repository.GetPersonAssetSnapshot(ctx, request.PersonID)
	if err != nil {
		return PersonAssetPage{}, err
	}
	if snapshot.Revision < 1 {
		return PersonAssetPage{}, ErrInvalidFaceProjection
	}
	sourcesHash, ready := personAssetSourcesHash(snapshot.Sources)
	if sourcesHash == "" {
		return PersonAssetPage{}, ErrInvalidFaceProjection
	}
	if !ready {
		return PersonAssetPage{}, ErrFaceNotReady
	}
	hash := personProjectionHash(request.PersonID)
	var after *PersonAssetPosition
	if request.Cursor != "" {
		var value personAssetCursor
		if len(request.Cursor) > 4096 || s.cursors.Decode(request.Cursor, "foliopath:person-assets:v2", &value) != nil || value.Version != 2 || value.PersonHash != hash || value.MTimeNS < 0 || value.AssetID < 1 {
			return PersonAssetPage{}, ErrInvalidPeopleCursor
		}
		if value.Snapshot != snapshot.Revision || value.SourcesHash != sourcesHash {
			return PersonAssetPage{}, ErrFaceProjectionStale
		}
		after = &PersonAssetPosition{MTimeNS: value.MTimeNS, AssetID: value.AssetID}
	}
	items, err := s.repository.ListPersonAssetViews(ctx, PersonAssetQuery{PersonID: request.PersonID, After: after, Limit: request.Limit + 1})
	if err != nil {
		return PersonAssetPage{}, err
	}
	latest, err := s.repository.GetPersonAssetSnapshot(ctx, request.PersonID)
	if err != nil {
		return PersonAssetPage{}, err
	}
	latestHash, latestReady := personAssetSourcesHash(latest.Sources)
	if latestHash == "" || latest.Revision < 1 {
		return PersonAssetPage{}, ErrInvalidFaceProjection
	}
	if !latestReady {
		return PersonAssetPage{}, ErrFaceNotReady
	}
	if latest.Revision != snapshot.Revision || latestHash != sourcesHash {
		return PersonAssetPage{}, ErrFaceProjectionStale
	}
	result := PersonAssetPage{Items: items}
	if len(result.Items) > request.Limit {
		result.Items = result.Items[:request.Limit]
		last := result.Items[len(result.Items)-1]
		result.NextCursor, err = s.cursors.Encode(personAssetCursor{Version: 2, PersonHash: hash, Snapshot: snapshot.Revision, SourcesHash: sourcesHash, MTimeNS: last.MTimeNS, AssetID: last.AssetID}, "foliopath:person-assets:v2")
	}
	return result, err
}

func personAssetSourcesHash(sources []PersonAssetSource) (string, bool) {
	hash := sha256.New()
	ready := true
	previous := int64(0)
	for _, source := range sources {
		if source.LibraryID < 1 || source.LibraryID <= previous || source.Revision < 1 ||
			(source.Status != "pending" && source.Status != "scanning" && source.Status != "ready" && source.Status != "offline" && source.Status != "error") {
			return "", false
		}
		previous = source.LibraryID
		ready = ready && source.Status == "ready"
		var encoded [16]byte
		binary.BigEndian.PutUint64(encoded[:8], uint64(source.LibraryID))
		binary.BigEndian.PutUint64(encoded[8:], uint64(source.Revision))
		_, _ = hash.Write(encoded[:])
		_, _ = hash.Write([]byte(source.Status))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), ready
}
func personProjectionHash(id string) string {
	return projectionHash("foliopath:person-assets:v1", id)
}

type FaceClusterDetailSnapshot struct {
	Cluster               FaceClusterView
	GenerationID, BuildID string
}
type FaceClusterMemberPosition struct{ Role, FaceID string }
type FaceClusterMemberView struct {
	FaceID            string
	AssetID, Revision int64
	Role              string
	Region            CoarseRegion
}
type FaceClusterMemberQuery struct {
	LibraryID          int64
	ClusterID, BuildID string
	After              *FaceClusterMemberPosition
	Limit              int
}
type FaceClusterDetailRepository interface {
	GetFaceClusterDetailSnapshot(context.Context, int64, string) (FaceClusterDetailSnapshot, error)
	ListFaceClusterMemberViews(context.Context, FaceClusterMemberQuery) ([]FaceClusterMemberView, error)
}
type FaceClusterDetailRequest struct {
	LibraryID         int64
	ClusterID, Cursor string
	Limit             int
}
type FaceClusterDetailPage struct {
	Cluster    FaceClusterView
	Items      []FaceClusterMemberView
	NextCursor string
}
type faceClusterDetailCursor struct {
	Version     int    `json:"v"`
	LibraryID   int64  `json:"l"`
	ClusterHash string `json:"c"`
	BuildID     string `json:"b"`
	Revision    int64  `json:"r"`
	Role        string `json:"k"`
	FaceID      string `json:"i"`
}
type FaceClusterDetailService struct {
	repository FaceClusterDetailRepository
	cursors    *cursorcodec.Codec
}

func NewFaceClusterDetailService(repository FaceClusterDetailRepository, key []byte) (*FaceClusterDetailService, error) {
	if repository == nil {
		return nil, errors.New("face cluster detail repository is required")
	}
	codec, err := cursorcodec.New(key)
	if err != nil {
		return nil, err
	}
	return &FaceClusterDetailService{repository: repository, cursors: codec}, nil
}
func (s *FaceClusterDetailService) List(ctx context.Context, request FaceClusterDetailRequest) (FaceClusterDetailPage, error) {
	if request.LibraryID < 1 || !validReviewID(request.ClusterID) || request.Limit < 1 || request.Limit > MaxFaceClusterMemberPageSize {
		return FaceClusterDetailPage{}, ErrInvalidFaceProjection
	}
	snapshot, err := s.repository.GetFaceClusterDetailSnapshot(ctx, request.LibraryID, request.ClusterID)
	if err != nil {
		return FaceClusterDetailPage{}, err
	}
	hash := projectionHash("foliopath:face-cluster-detail:v1", request.ClusterID)
	var after *FaceClusterMemberPosition
	if request.Cursor != "" {
		var value faceClusterDetailCursor
		if len(request.Cursor) > 4096 || s.cursors.Decode(request.Cursor, "foliopath:face-cluster-detail:v1", &value) != nil || value.Version != 1 || value.LibraryID != request.LibraryID || value.ClusterHash != hash || (value.Role != "core" && value.Role != "edge") || !validReviewID(value.FaceID) {
			return FaceClusterDetailPage{}, ErrInvalidFaceClusterCursor
		}
		if value.BuildID != snapshot.BuildID || value.Revision != snapshot.Cluster.Revision {
			return FaceClusterDetailPage{}, ErrFaceProjectionStale
		}
		after = &FaceClusterMemberPosition{Role: value.Role, FaceID: value.FaceID}
	}
	items, err := s.repository.ListFaceClusterMemberViews(ctx, FaceClusterMemberQuery{LibraryID: request.LibraryID, ClusterID: request.ClusterID, BuildID: snapshot.BuildID, After: after, Limit: request.Limit + 1})
	if err != nil {
		return FaceClusterDetailPage{}, err
	}
	result := FaceClusterDetailPage{Cluster: snapshot.Cluster, Items: items}
	if len(result.Items) > request.Limit {
		result.Items = result.Items[:request.Limit]
		last := result.Items[len(result.Items)-1]
		result.NextCursor, err = s.cursors.Encode(faceClusterDetailCursor{Version: 1, LibraryID: request.LibraryID, ClusterHash: hash, BuildID: snapshot.BuildID, Revision: snapshot.Cluster.Revision, Role: last.Role, FaceID: last.FaceID}, "foliopath:face-cluster-detail:v1")
	}
	return result, err
}

func projectionHash(namespace, id string) string {
	sum := sha256.Sum256([]byte(namespace + "\x00" + id))
	return hex.EncodeToString(sum[:])
}
