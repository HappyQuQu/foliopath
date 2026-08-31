package semantic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
)

const videoSearchCursorData = "foliopath:video-semantic-search:v1"

type VideoCoverage struct {
	Eligible int64
	Ready    int64
	Degraded int64
	Failed   int64
	Stale    int64
	Revision int64
}

func (value VideoCoverage) Complete() bool {
	return value.Eligible == value.Ready && value.Degraded == 0 && value.Failed == 0 && value.Stale == 0
}

type VideoSearchSnapshotMember struct {
	LibraryID         int64
	LibraryGeneration int64
	SettingsRevision  int64
	Coverage          VideoCoverage
}

type VideoSearchSnapshot struct {
	GenerationID    string
	CatalogRevision int64
	Members         []VideoSearchSnapshotMember
	Excluded        []SearchExclusion
}

type VideoSearchSnapshotRepository interface {
	GetVideoSemanticSearchSnapshot(context.Context, SearchScope) (VideoSearchSnapshot, error)
}

type VideoSearchResult struct {
	Matches      []VideoVectorMatch
	NextCursor   string
	GenerationID string
	Coverage     VideoCoverage
	Excluded     []SearchExclusion
}

type videoSearchCursor struct {
	Version             int    `json:"v"`
	QueryHash           string `json:"q"`
	LibraryID           int64  `json:"l"`
	DirectoryID         int64  `json:"d"`
	Recursive           bool   `json:"r"`
	GenerationID        string `json:"g"`
	CatalogRevision     int64  `json:"c"`
	SnapshotFingerprint string `json:"s"`
	ScoreBits           uint32 `json:"b"`
	AssetID             int64  `json:"a"`
}

type VideoSearchService struct {
	snapshots VideoSearchSnapshotRepository
	vectors   VideoVectorSearchRepository
	encoder   TextEncoder
	cursors   *cursorcodec.Codec
}

func NewVideoSearchService(snapshots VideoSearchSnapshotRepository, vectors VideoVectorSearchRepository, encoder TextEncoder, cursorKey []byte) (*VideoSearchService, error) {
	if snapshots == nil || vectors == nil || encoder == nil {
		return nil, errors.New("video semantic search dependencies are required")
	}
	cursors, err := cursorcodec.New(cursorKey)
	if err != nil {
		return nil, fmt.Errorf("create video semantic cursor codec: %w", err)
	}
	return &VideoSearchService{snapshots: snapshots, vectors: vectors, encoder: encoder, cursors: cursors}, nil
}

func (service *VideoSearchService) Search(ctx context.Context, request SearchRequest) (VideoSearchResult, error) {
	if request.LibraryID < 0 || request.DirectoryID < 0 || request.DirectoryID > 0 && request.LibraryID == 0 ||
		request.Limit < 1 || request.Limit > MaxSemanticSearchLimit {
		return VideoSearchResult{}, ErrInvalidVideoSemantic
	}
	if request.Cursor != "" && (len(request.Cursor) < 8 || len(request.Cursor) > MaxSemanticCursorBytes) {
		return VideoSearchResult{}, ErrInvalidSemanticCursor
	}
	canonical, err := CanonicalizeQuery(request.Query)
	if err != nil {
		return VideoSearchResult{}, err
	}
	scope := SearchScope{LibraryID: request.LibraryID, DirectoryID: request.DirectoryID, Recursive: request.Recursive}
	snapshot, err := service.snapshots.GetVideoSemanticSearchSnapshot(ctx, scope)
	if err != nil {
		return VideoSearchResult{}, err
	}
	fingerprint, coverage, err := VideoSearchSnapshotFingerprint(snapshot)
	if err != nil {
		return VideoSearchResult{}, err
	}
	queryHash := semanticSearchQueryHash(canonical)
	var after *SearchPosition
	if request.Cursor != "" {
		var payload videoSearchCursor
		if err := service.cursors.Decode(request.Cursor, videoSearchCursorData, &payload); err != nil {
			return VideoSearchResult{}, ErrInvalidSemanticCursor
		}
		if payload.Version != 1 || payload.QueryHash != queryHash || payload.LibraryID != scope.LibraryID ||
			payload.DirectoryID != scope.DirectoryID || payload.Recursive != scope.Recursive {
			return VideoSearchResult{}, ErrInvalidSemanticCursor
		}
		if payload.GenerationID != snapshot.GenerationID || payload.CatalogRevision != snapshot.CatalogRevision || payload.SnapshotFingerprint != fingerprint {
			return VideoSearchResult{}, ErrSemanticCursorStale
		}
		score := math.Float32frombits(payload.ScoreBits)
		if payload.AssetID < 1 || math.IsNaN(float64(score)) || math.IsInf(float64(score), 0) {
			return VideoSearchResult{}, ErrInvalidSemanticCursor
		}
		after = &SearchPosition{Score: score, AssetID: payload.AssetID}
	}
	query, err := service.encoder.EncodeSemanticText(ctx, snapshot.GenerationID, canonical)
	if err != nil {
		return VideoSearchResult{}, err
	}
	matches, err := service.vectors.SearchVideoSemanticVectors(ctx, VideoVectorSearchRequest{
		GenerationID: snapshot.GenerationID, LibraryID: scope.LibraryID, DirectoryID: scope.DirectoryID,
		Recursive: scope.Recursive, Query: query, After: after, Limit: request.Limit,
	})
	if err != nil {
		return VideoSearchResult{}, err
	}
	if err := validateVideoSearchMatches(matches, request.Limit, scope, snapshot, after); err != nil {
		return VideoSearchResult{}, err
	}
	result := VideoSearchResult{Matches: matches, GenerationID: snapshot.GenerationID, Coverage: coverage, Excluded: append([]SearchExclusion(nil), snapshot.Excluded...)}
	if len(matches) == request.Limit {
		last := matches[len(matches)-1]
		result.NextCursor, err = service.cursors.Encode(videoSearchCursor{Version: 1, QueryHash: queryHash,
			LibraryID: scope.LibraryID, DirectoryID: scope.DirectoryID, Recursive: scope.Recursive,
			GenerationID: snapshot.GenerationID, CatalogRevision: snapshot.CatalogRevision,
			SnapshotFingerprint: fingerprint, ScoreBits: math.Float32bits(last.Score), AssetID: last.AssetID}, videoSearchCursorData)
	}
	return result, err
}

func validateVideoSearchMatches(matches []VideoVectorMatch, limit int, scope SearchScope, snapshot VideoSearchSnapshot, after *SearchPosition) error {
	if len(matches) > limit {
		return ErrInvalidVideoSemantic
	}
	allowed := make(map[int64]struct{}, len(snapshot.Members))
	for _, member := range snapshot.Members {
		allowed[member.LibraryID] = struct{}{}
	}
	seen := make(map[int64]struct{}, len(matches))
	for index, match := range matches {
		if _, ok := allowed[match.LibraryID]; !ok || scope.LibraryID > 0 && match.LibraryID != scope.LibraryID ||
			match.AssetID < 1 || (match.PlanSize != 4 && match.PlanSize != 10) || match.Ordinal < 0 ||
			match.Ordinal >= match.PlanSize || match.TimestampMS < 0 || math.IsNaN(float64(match.Score)) || math.IsInf(float64(match.Score), 0) {
			return ErrInvalidVideoSemantic
		}
		if _, duplicate := seen[match.AssetID]; duplicate {
			return ErrInvalidVideoSemantic
		}
		seen[match.AssetID] = struct{}{}
		if after != nil && !(match.Score < after.Score || match.Score == after.Score && match.AssetID > after.AssetID) {
			return ErrInvalidVideoSemantic
		}
		if index > 0 {
			previous := matches[index-1]
			if match.Score > previous.Score || match.Score == previous.Score && match.AssetID <= previous.AssetID {
				return ErrInvalidVideoSemantic
			}
		}
	}
	return nil
}

func VideoSearchSnapshotFingerprint(snapshot VideoSearchSnapshot) (string, VideoCoverage, error) {
	if len(snapshot.GenerationID) < 8 || len(snapshot.GenerationID) > 128 || snapshot.CatalogRevision < 1 || len(snapshot.Members) == 0 {
		return "", VideoCoverage{}, ErrInvalidSemanticSnapshot
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("foliopath:video-semantic-snapshot:v1\x00" + snapshot.GenerationID + "\x00" + strconv.FormatInt(snapshot.CatalogRevision, 10)))
	coverage := VideoCoverage{Revision: 1}
	previousID := int64(0)
	memberIDs := make(map[int64]struct{}, len(snapshot.Members))
	for _, member := range snapshot.Members {
		value := member.Coverage
		if member.LibraryID <= previousID || member.LibraryGeneration < 0 || member.SettingsRevision < 1 || value.Revision < 1 ||
			value.Eligible < 0 || value.Ready < 0 || value.Degraded < 0 || value.Failed < 0 || value.Stale < 0 ||
			value.Ready+value.Degraded+value.Failed+value.Stale > value.Eligible {
			return "", VideoCoverage{}, ErrInvalidSemanticSnapshot
		}
		previousID = member.LibraryID
		memberIDs[member.LibraryID] = struct{}{}
		for _, pair := range []int64{member.LibraryID, member.LibraryGeneration, member.SettingsRevision,
			value.Eligible, value.Ready, value.Degraded, value.Failed, value.Stale, value.Revision} {
			_, _ = hash.Write([]byte("\x00" + strconv.FormatInt(pair, 10)))
		}
		if !addVideoCoverage(&coverage, value) {
			return "", VideoCoverage{}, ErrInvalidSemanticSnapshot
		}
	}
	previousID = 0
	for _, excluded := range snapshot.Excluded {
		if excluded.LibraryID <= previousID || excluded.SettingsRevision < 1 ||
			(excluded.Reason != "disabled" && excluded.Reason != "offline") {
			return "", VideoCoverage{}, ErrInvalidSemanticSnapshot
		}
		if _, included := memberIDs[excluded.LibraryID]; included {
			return "", VideoCoverage{}, ErrInvalidSemanticSnapshot
		}
		previousID = excluded.LibraryID
		_, _ = hash.Write([]byte("\x00excluded\x00" + strconv.FormatInt(excluded.LibraryID, 10) + "\x00" +
			strconv.FormatInt(excluded.SettingsRevision, 10) + "\x00" + excluded.Reason))
	}
	return hex.EncodeToString(hash.Sum(nil)), coverage, nil
}

func addVideoCoverage(target *VideoCoverage, value VideoCoverage) bool {
	for destination, addition := range map[*int64]int64{
		&target.Eligible: value.Eligible, &target.Ready: value.Ready, &target.Degraded: value.Degraded,
		&target.Failed: value.Failed, &target.Stale: value.Stale,
	} {
		if addition > math.MaxInt64-*destination {
			return false
		}
		*destination += addition
	}
	if value.Revision > target.Revision {
		target.Revision = value.Revision
	}
	return true
}
