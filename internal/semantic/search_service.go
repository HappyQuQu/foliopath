package semantic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
)

const semanticSearchCursorData = "foliopath:semantic-search:v1"

var (
	ErrInvalidSemanticCursor   = errors.New("invalid semantic search cursor")
	ErrSemanticCursorStale     = errors.New("semantic search cursor stale")
	ErrSemanticScopeNotFound   = errors.New("semantic search scope not found")
	ErrSemanticLibraryOffline  = errors.New("semantic library offline")
	ErrInvalidSemanticSnapshot = errors.New("invalid semantic search snapshot")
)

type SearchScope struct {
	LibraryID   int64
	DirectoryID int64
	Recursive   bool
}

type SearchSnapshotMember struct {
	LibraryID         int64
	LibraryGeneration int64
	SettingsRevision  int64
	Coverage          Coverage
}

type SearchExclusion struct {
	LibraryID        int64
	SettingsRevision int64
	Reason           string
}

type SearchSnapshot struct {
	GenerationID    string
	CatalogRevision int64
	Members         []SearchSnapshotMember
	Excluded        []SearchExclusion
}

type SearchSnapshotRepository interface {
	GetSemanticSearchSnapshot(context.Context, SearchScope) (SearchSnapshot, error)
}

type TextEncoder interface {
	EncodeSemanticText(context.Context, string, string) ([]float32, error)
}

type SearchRequest struct {
	Query       string
	LibraryID   int64
	DirectoryID int64
	Recursive   bool
	Cursor      string
	Limit       int
}

type SearchResult struct {
	Matches      []VectorMatch
	NextCursor   string
	GenerationID string
	Coverage     Coverage
	Excluded     []SearchExclusion
}

type searchCursor struct {
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

type SearchService struct {
	snapshots SearchSnapshotRepository
	vectors   VectorSearchRepository
	encoder   TextEncoder
	cursors   *cursorcodec.Codec
}

func NewSearchService(snapshots SearchSnapshotRepository, vectors VectorSearchRepository, encoder TextEncoder, cursorKey []byte) (*SearchService, error) {
	if snapshots == nil || vectors == nil || encoder == nil {
		return nil, errors.New("semantic search dependencies are required")
	}
	cursors, err := cursorcodec.New(cursorKey)
	if err != nil {
		return nil, fmt.Errorf("create semantic cursor codec: %w", err)
	}
	return &SearchService{snapshots: snapshots, vectors: vectors, encoder: encoder, cursors: cursors}, nil
}

func (service *SearchService) Search(ctx context.Context, request SearchRequest) (SearchResult, error) {
	if request.LibraryID < 0 || request.DirectoryID < 0 || request.DirectoryID > 0 && request.LibraryID == 0 ||
		request.Limit < 1 || request.Limit > MaxSemanticSearchLimit {
		return SearchResult{}, ErrInvalidSemanticSearch
	}
	if request.Cursor != "" && (len(request.Cursor) < 8 || len(request.Cursor) > MaxSemanticCursorBytes) {
		return SearchResult{}, ErrInvalidSemanticCursor
	}
	canonical, err := CanonicalizeQuery(request.Query)
	if err != nil {
		return SearchResult{}, err
	}
	scope := SearchScope{LibraryID: request.LibraryID, DirectoryID: request.DirectoryID, Recursive: request.Recursive}
	snapshot, err := service.snapshots.GetSemanticSearchSnapshot(ctx, scope)
	if err != nil {
		return SearchResult{}, err
	}
	fingerprint, coverage, err := SemanticSearchSnapshotFingerprint(snapshot)
	if err != nil {
		return SearchResult{}, err
	}
	queryHash := semanticSearchQueryHash(canonical)
	var after *SearchPosition
	if request.Cursor != "" {
		var payload searchCursor
		if err := service.cursors.Decode(request.Cursor, semanticSearchCursorData, &payload); err != nil {
			return SearchResult{}, ErrInvalidSemanticCursor
		}
		if payload.Version != 1 || payload.QueryHash != queryHash || payload.LibraryID != scope.LibraryID ||
			payload.DirectoryID != scope.DirectoryID || payload.Recursive != scope.Recursive {
			return SearchResult{}, ErrInvalidSemanticCursor
		}
		if payload.GenerationID != snapshot.GenerationID || payload.CatalogRevision != snapshot.CatalogRevision ||
			payload.SnapshotFingerprint != fingerprint {
			return SearchResult{}, ErrSemanticCursorStale
		}
		score := math.Float32frombits(payload.ScoreBits)
		if payload.AssetID < 1 || math.IsNaN(float64(score)) || math.IsInf(float64(score), 0) {
			return SearchResult{}, ErrInvalidSemanticCursor
		}
		after = &SearchPosition{Score: score, AssetID: payload.AssetID}
	}
	queryVector, err := service.encoder.EncodeSemanticText(ctx, snapshot.GenerationID, canonical)
	if err != nil {
		return SearchResult{}, err
	}
	matches, err := service.vectors.SearchSemanticVectors(ctx, VectorSearchRequest{
		GenerationID: snapshot.GenerationID,
		LibraryID:    scope.LibraryID,
		DirectoryID:  scope.DirectoryID,
		Recursive:    scope.Recursive,
		Query:        queryVector,
		After:        after,
		Limit:        request.Limit,
	})
	if err != nil {
		return SearchResult{}, err
	}
	if err := validateSearchMatches(matches, request.Limit, scope, snapshot, after); err != nil {
		return SearchResult{}, err
	}
	result := SearchResult{Matches: matches, GenerationID: snapshot.GenerationID, Coverage: coverage, Excluded: append([]SearchExclusion(nil), snapshot.Excluded...)}
	if len(matches) == request.Limit {
		last := matches[len(matches)-1]
		result.NextCursor, err = service.cursors.Encode(searchCursor{
			Version: 1, QueryHash: queryHash, LibraryID: scope.LibraryID, DirectoryID: scope.DirectoryID,
			Recursive: scope.Recursive, GenerationID: snapshot.GenerationID, CatalogRevision: snapshot.CatalogRevision,
			SnapshotFingerprint: fingerprint, ScoreBits: math.Float32bits(last.Score), AssetID: last.AssetID,
		}, semanticSearchCursorData)
		if err != nil {
			return SearchResult{}, fmt.Errorf("encode semantic search cursor: %w", err)
		}
	}
	return result, nil
}

func validateSearchMatches(matches []VectorMatch, limit int, scope SearchScope, snapshot SearchSnapshot, after *SearchPosition) error {
	if len(matches) > limit {
		return ErrInvalidEmbeddingRecord
	}
	allowedLibraries := make(map[int64]struct{}, len(snapshot.Members))
	for _, member := range snapshot.Members {
		allowedLibraries[member.LibraryID] = struct{}{}
	}
	seenAssets := make(map[int64]struct{}, len(matches))
	for index, match := range matches {
		if match.LibraryID < 1 || match.AssetID < 1 || math.IsNaN(float64(match.Score)) || math.IsInf(float64(match.Score), 0) {
			return ErrInvalidEmbeddingRecord
		}
		if _, ok := allowedLibraries[match.LibraryID]; !ok {
			return ErrInvalidEmbeddingRecord
		}
		if scope.LibraryID > 0 && match.LibraryID != scope.LibraryID {
			return ErrInvalidEmbeddingRecord
		}
		if _, duplicate := seenAssets[match.AssetID]; duplicate {
			return ErrInvalidEmbeddingRecord
		}
		seenAssets[match.AssetID] = struct{}{}
		if after != nil && !(match.Score < after.Score || match.Score == after.Score && match.AssetID > after.AssetID) {
			return ErrInvalidEmbeddingRecord
		}
		if index == 0 {
			continue
		}
		previous := matches[index-1]
		if match.Score > previous.Score || match.Score == previous.Score && match.AssetID <= previous.AssetID {
			return ErrInvalidEmbeddingRecord
		}
	}
	return nil
}

func SemanticSearchSnapshotFingerprint(snapshot SearchSnapshot) (string, Coverage, error) {
	if len(snapshot.GenerationID) < 8 || len(snapshot.GenerationID) > 128 || snapshot.CatalogRevision < 1 || len(snapshot.Members) == 0 {
		return "", Coverage{}, ErrInvalidSemanticSnapshot
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("foliopath:semantic-search-snapshot:v1\x00" + snapshot.GenerationID + "\x00" + strconv.FormatInt(snapshot.CatalogRevision, 10)))
	coverage := Coverage{Revision: 1}
	memberIDs := make(map[int64]struct{}, len(snapshot.Members))
	previousID := int64(0)
	for _, member := range snapshot.Members {
		if member.LibraryID <= previousID || member.LibraryGeneration < 0 || member.SettingsRevision < 1 || member.Coverage.Revision < 1 ||
			member.Coverage.Eligible < 0 || member.Coverage.Completed < 0 || member.Coverage.Failed < 0 || member.Coverage.Stale < 0 ||
			member.Coverage.Completed > member.Coverage.Eligible ||
			member.Coverage.Failed > member.Coverage.Eligible-member.Coverage.Completed ||
			member.Coverage.Stale > member.Coverage.Eligible-member.Coverage.Completed-member.Coverage.Failed {
			return "", Coverage{}, ErrInvalidSemanticSnapshot
		}
		previousID = member.LibraryID
		memberIDs[member.LibraryID] = struct{}{}
		if !addCoverage(&coverage, member.Coverage) {
			return "", Coverage{}, ErrInvalidSemanticSnapshot
		}
		fields := []int64{member.LibraryID, member.LibraryGeneration, member.SettingsRevision, member.Coverage.Revision,
			member.Coverage.Eligible, member.Coverage.Completed, member.Coverage.Failed, member.Coverage.Stale}
		for _, field := range fields {
			_, _ = hash.Write([]byte("\x00" + strconv.FormatInt(field, 10)))
		}
	}
	previousID = 0
	for _, excluded := range snapshot.Excluded {
		if excluded.LibraryID <= previousID || excluded.SettingsRevision < 1 ||
			(excluded.Reason != "disabled" && excluded.Reason != "offline") {
			return "", Coverage{}, ErrInvalidSemanticSnapshot
		}
		if _, included := memberIDs[excluded.LibraryID]; included {
			return "", Coverage{}, ErrInvalidSemanticSnapshot
		}
		previousID = excluded.LibraryID
		_, _ = hash.Write([]byte("\x00excluded\x00" + strconv.FormatInt(excluded.LibraryID, 10) + "\x00" +
			strconv.FormatInt(excluded.SettingsRevision, 10) + "\x00" + excluded.Reason))
	}
	return hex.EncodeToString(hash.Sum(nil)), coverage, nil
}

func addCoverage(total *Coverage, value Coverage) bool {
	if total == nil || value.Eligible > math.MaxInt64-total.Eligible || value.Completed > math.MaxInt64-total.Completed ||
		value.Failed > math.MaxInt64-total.Failed || value.Stale > math.MaxInt64-total.Stale ||
		value.Revision > math.MaxInt64-total.Revision {
		return false
	}
	total.Eligible += value.Eligible
	total.Completed += value.Completed
	total.Failed += value.Failed
	total.Stale += value.Stale
	total.Revision += value.Revision
	return true
}

func semanticSearchQueryHash(canonical string) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{"foliopath:semantic-query:v1", canonical}, "\x00")))
	return hex.EncodeToString(digest[:])
}
