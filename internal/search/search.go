// Package search owns fused retrieval, evidence and snapshot-bound pagination.
package search

import (
	"container/heap"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/cursor"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const MinSemanticScore float32 = 0.20
const cursorAudience = "foliopath:fused-search:v2"

var (
	ErrInvalid     = errors.New("invalid fused search")
	ErrCursor      = errors.New("invalid fused search cursor")
	ErrStale       = errors.New("fused search snapshot changed")
	ErrUnavailable = errors.New("search text encoder unavailable")
)

type Source = string

const (
	Filename   Source = "filename"
	ParentPath Source = "path"
	Visual     Source = "semantic"
	OCR        Source = "ocr"
)

type Request struct {
	Query            string
	Mode             string
	LibraryID        int64
	DirectoryID      int64
	Recursive        bool
	Kinds            []catalog.AssetKind
	ModifiedFromNS   *int64
	ModifiedBeforeNS *int64
	Cursor           string
	Limit            int
}
type Candidate struct {
	SourceFingerprint string
	LibraryID         int64
	AssetID           int64
	Name              string
	RelativePath      string
	Vector            []byte
	OCRLines          []string
}
type Scan struct {
	Request        Request
	Terms          []string
	GenerationID   string
	OCRFingerprint string
}
type Repository interface {
	ResolveScope(context.Context, int64, int64) (catalog.Scope, error)
	ResolveGlobalCatalogRevision(context.Context) (int64, error)
	GetSemanticSearchSnapshot(context.Context, semantic.SearchScope) (semantic.SearchSnapshot, error)
	StreamSearchCandidates(context.Context, Scan, func(Candidate) error) error
}
type OCRCoverage struct {
	Eligible int64 `json:"eligible"`
	Ready    int64 `json:"ready"`
	Failed   int64 `json:"failed"`
}
type OCRExcerpt struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}
type OCRRepository interface {
	GetOCRSearchCoverage(context.Context, Scan) (OCRCoverage, error)
}
type Hit struct {
	OCR           []OCRExcerpt
	Evidence      string
	LibraryID     int64
	AssetID       int64
	Sources       []Source
	SemanticScore *float32
	Rank          float32
}
type Result struct {
	Snapshot      string
	OCRState      string
	OCRCoverage   *OCRCoverage
	Hits          []Hit
	NextCursor    string
	SemanticState string
	Coverage      *semantic.Coverage
	Excluded      []semantic.SearchExclusion
}
type Service struct {
	repo           Repository
	encoder        semantic.TextEncoder
	codec          *cursor.Codec
	ocrFingerprint string
}

func New(repo Repository, encoder semantic.TextEncoder, key []byte, ocrFingerprint ...string) (*Service, error) {
	if repo == nil || encoder == nil {
		return nil, ErrInvalid
	}
	c, e := cursor.New(key)
	if e != nil {
		return nil, e
	}
	fp := ""
	if len(ocrFingerprint) > 1 {
		return nil, ErrInvalid
	}
	if len(ocrFingerprint) == 1 {
		fp = ocrFingerprint[0]
		if len(fp) != 64 {
			return nil, ErrInvalid
		}
		if _, err := hex.DecodeString(fp); err != nil {
			return nil, ErrInvalid
		}
		if _, ok := repo.(OCRRepository); !ok {
			return nil, ErrInvalid
		}
	}
	return &Service{repo: repo, encoder: encoder, codec: c, ocrFingerprint: fp}, nil
}

type snapshot struct {
	fingerprint string
	state       string
	semantic    semantic.SearchSnapshot
	coverage    *semantic.Coverage
	ocrCoverage *OCRCoverage
	ocrState    string
}
type token struct {
	Version  int     `json:"v"`
	Query    string  `json:"q"`
	Snapshot string  `json:"s"`
	State    string  `json:"t"`
	Vector   string  `json:"e"`
	Rank     float32 `json:"r"`
	AssetID  int64   `json:"a"`
}

func normalize(r Request) (Request, []string, error) {
	if r.Mode == "" {
		r.Mode = "all"
	}
	if r.Mode != "all" && r.Mode != "filename" && r.Mode != "semantic" && r.Mode != "ocr" {
		return r, nil, ErrInvalid
	}
	if r.Limit < 1 || r.Limit > 200 || r.LibraryID < 0 || r.DirectoryID < 0 || r.DirectoryID > 0 && r.LibraryID == 0 || r.Recursive && r.DirectoryID == 0 || len(r.Kinds) > 3 {
		return r, nil, ErrInvalid
	}
	terms, e := catalog.NormalizeSearchTerms(r.Query)
	if e != nil {
		return r, nil, ErrInvalid
	}
	r.Kinds = slices.Clone(r.Kinds)
	slices.Sort(r.Kinds)
	for i, k := range r.Kinds {
		if k != catalog.KindImage && k != catalog.KindAnimated && k != catalog.KindVideo || i > 0 && r.Kinds[i-1] == k {
			return r, nil, ErrInvalid
		}
	}
	if r.ModifiedFromNS != nil && r.ModifiedBeforeNS != nil && *r.ModifiedFromNS >= *r.ModifiedBeforeNS {
		return r, nil, ErrInvalid
	}
	if r.Cursor != "" && (len(r.Cursor) < 8 || len(r.Cursor) > 2048) {
		return r, nil, ErrCursor
	}
	return r, terms, nil
}
func hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (s *Service) capture(ctx context.Context, r Request) (snapshot, error) {
	var out snapshot
	if r.LibraryID > 0 {
		if _, e := s.repo.ResolveScope(ctx, r.LibraryID, r.DirectoryID); e != nil {
			return out, e
		}
	}
	rev, e := s.repo.ResolveGlobalCatalogRevision(ctx)
	if e != nil {
		return out, e
	}
	if rev < 1 {
		return out, ErrInvalid
	}
	out.state = "not_requested"
	semanticFP := ""
	if r.Mode == "all" || r.Mode == "semantic" {
		out.semantic, e = s.repo.GetSemanticSearchSnapshot(ctx, semantic.SearchScope{LibraryID: r.LibraryID, DirectoryID: r.DirectoryID, Recursive: r.Recursive})
		switch {
		case e == nil:
			var coverage semantic.Coverage
			semanticFP, coverage, e = semantic.SemanticSearchSnapshotFingerprint(out.semantic)
			if e != nil {
				return out, e
			}
			out.coverage = &coverage
			out.state = "ready"
		case errors.Is(e, semantic.ErrSemanticDisabled):
			out.state = "disabled"
		case errors.Is(e, semantic.ErrSemanticLibraryOffline):
			out.state = "offline"
		case errors.Is(e, semantic.ErrSemanticGenerationUnavailable):
			out.state = "unavailable"
		default:
			return out, e
		}
	}
	out.ocrState = "not_requested"
	if r.Mode == "all" || r.Mode == "ocr" {
		out.ocrState = "disabled"
		if s.ocrFingerprint != "" {
			coverage, err := s.repo.(OCRRepository).GetOCRSearchCoverage(ctx, Scan{Request: r, OCRFingerprint: s.ocrFingerprint})
			if err != nil {
				return out, err
			}
			out.ocrCoverage = &coverage
			out.ocrState = "available"
		}
	}
	out.fingerprint = hash(struct {
		Revision             int64
		Semantic, State, OCR string
	}{rev, semanticFP, out.state, s.ocrFingerprint})
	return out, nil
}
func (s *Service) Search(ctx context.Context, input Request) (Result, error) {
	r, terms, e := normalize(input)
	if e != nil {
		return Result{}, e
	}
	if e = ctx.Err(); e != nil {
		return Result{}, e
	}
	snap, e := s.capture(ctx, r)
	if e != nil {
		return Result{}, e
	}
	state := snap.state
	var query []float32
	if state == "ready" {
		canonical, err := semantic.CanonicalizeQuery(r.Query)
		if err != nil {
			state = "unsupported_query"
		} else {
			query, err = s.encoder.EncodeSemanticText(ctx, snap.semantic.GenerationID, canonical)
			if err != nil {
				if ctx.Err() != nil {
					return Result{}, ctx.Err()
				}

				// Some legacy adapters join an availability marker with the
				// original failure. Invalid data/identity must win over it.
				if errors.Is(err, semantic.ErrInvalidEmbedding) || errors.Is(err, semantic.ErrInvalidEmbeddingRecord) || errors.Is(err, semantic.ErrInvalidSpaceIdentity) || errors.Is(err, semantic.ErrIncompatibleSpace) {
					return Result{}, err
				}
				if errors.Is(err, ErrUnavailable) || errors.Is(err, semantic.ErrSemanticGenerationUnavailable) {
					state = "unavailable"
				} else {
					return Result{}, err
				}
			}
			if state == "ready" {
				query, err = semantic.NormalizeEmbedding(query, len(query))
				if err != nil {
					return Result{}, err
				}
			}
		}
	}
	if r.Mode == "semantic" && state != "ready" {
		if state == "disabled" {
			return Result{}, semantic.ErrSemanticDisabled
		}
		if state == "offline" {
			return Result{}, semantic.ErrSemanticLibraryOffline
		}
		if state == "unsupported_query" {
			return Result{}, ErrInvalid
		}
		return Result{}, ErrUnavailable
	}
	binding := r
	binding.Query = ""
	binding.Cursor = ""
	binding.Limit = 0
	queryHash := hash(struct {
		Terms    []string
		Original string
		Request  Request
	}{terms, catalog.SearchTextKey(strings.TrimSpace(r.Query)), binding})
	vectorHash := hash(query)
	var after *token
	if r.Cursor != "" {
		var v token
		if s.codec.Decode(r.Cursor, cursorAudience, &v) != nil || v.Version != 2 || v.Query != queryHash || v.AssetID < 1 || math.IsNaN(float64(v.Rank)) || math.IsInf(float64(v.Rank), 0) {
			return Result{}, ErrCursor
		}
		if v.Snapshot != snap.fingerprint || v.State != state || v.Vector != vectorHash {
			return Result{}, ErrStale
		}
		after = &v
	}
	values := &hitHeap{}
	heap.Init(values)
	scan := Scan{Request: r, Terms: terms}
	if snap.ocrState == "available" {
		scan.OCRFingerprint = s.ocrFingerprint
	}
	if state == "ready" {
		scan.GenerationID = snap.semantic.GenerationID
	}
	e = s.repo.StreamSearchCandidates(ctx, scan, func(c Candidate) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.LibraryID < 1 || c.AssetID < 1 {
			return ErrInvalid
		}
		sources := LiteralSources(terms, c.Name, c.RelativePath)
		if r.Mode == "ocr" {
			sources = nil
		}
		var excerpts []OCRExcerpt
		if scan.OCRFingerprint != "" {
			excerpts = OCRExcerpts(terms, c.OCRLines)
			if len(excerpts) > 0 {
				sources = append(sources, OCR)
			}
		}
		literal := len(sources) > 0
		var score *float32
		if state == "ready" && len(c.Vector) > 0 {
			v, err := semantic.DecodeEmbedding(c.Vector, len(query))
			if err != nil {
				return err
			}
			var n float64
			for i, x := range v {
				n += float64(x) * float64(query[i])
			}
			f := float32(n)
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return semantic.ErrInvalidEmbeddingRecord
			}
			if f >= MinSemanticScore {
				score = &f
				sources = append(sources, Visual)
			}
		}
		if r.Mode == "filename" && !literal || r.Mode == "semantic" && score == nil || r.Mode == "ocr" && len(excerpts) == 0 || !literal && score == nil {
			return nil
		}
		rank := float32(0)
		if score != nil {
			rank = *score
		}
		if literal {
			rank = 4
			if score != nil {
				rank = 6 + *score
			}
		}
		if after != nil && !(rank < after.Rank || rank == after.Rank && c.AssetID > after.AssetID) {
			return nil
		}
		hit := Hit{OCR: excerpts, Evidence: Evidence(c.SourceFingerprint, c.Name, c.RelativePath), LibraryID: c.LibraryID, AssetID: c.AssetID, Sources: sources, SemanticScore: score, Rank: rank}
		if values.Len() < r.Limit+1 {
			heap.Push(values, hit)
		} else if better(hit, (*values)[0]) {
			heap.Pop(values)
			heap.Push(values, hit)
		}
		return nil
	})
	if e != nil {
		return Result{}, e
	}
	latest, e := s.capture(ctx, r)
	if e != nil {
		return Result{}, e
	}
	if latest.fingerprint != snap.fingerprint {
		return Result{}, ErrStale
	}
	hits := make([]Hit, values.Len())
	for i := len(hits) - 1; i >= 0; i-- {
		hits[i] = heap.Pop(values).(Hit)
	}
	result := Result{Snapshot: snap.fingerprint, OCRState: snap.ocrState, OCRCoverage: snap.ocrCoverage, Hits: hits, SemanticState: state, Coverage: snap.coverage, Excluded: snap.semantic.Excluded}
	if len(hits) > r.Limit {
		result.Hits = hits[:r.Limit]
		last := result.Hits[len(result.Hits)-1]
		result.NextCursor, e = s.codec.Encode(token{2, queryHash, snap.fingerprint, state, vectorHash, last.Rank, last.AssetID}, cursorAudience)
		if e != nil {
			return Result{}, e
		}
	}
	return result, nil
}

// LiteralSources shares catalog normalization, and excludes the basename from path evidence.
func LiteralSources(terms []string, name, relative string) []Source {
	return catalog.LiteralSources(terms, name, relative)
}
func better(a, b Hit) bool { return a.Rank > b.Rank || a.Rank == b.Rank && a.AssetID < b.AssetID }

type hitHeap []Hit

func (h hitHeap) Len() int           { return len(h) }
func (h hitHeap) Less(i, j int) bool { return better(h[j], h[i]) }
func (h hitHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *hitHeap) Push(v any)        { *h = append(*h, v.(Hit)) }
func (h *hitHeap) Pop() any          { a := *h; v := a[len(a)-1]; *h = a[:len(a)-1]; return v }

// Evidence binds later asset hydration to the exact source and literal fields
// used for ranking. It is internal evidence, never a public path or cursor field.
func Evidence(fingerprint, name, relative string) string {
	return hash([]string{fingerprint, name, relative})
}

// Verify rechecks source visibility after the HTTP asset hydration boundary.
func (s *Service) Verify(ctx context.Context, r Request, result Result) error {
	r, _, err := normalize(r)
	if err != nil {
		return err
	}
	snap, err := s.capture(ctx, r)
	if err != nil {
		return err
	}
	if snap.fingerprint != result.Snapshot {
		return ErrStale
	}
	return nil
}

// OCRExcerpts never treats a model score as relevance. All terms must occur in
// the document; only bounded, original-text line windows leave this boundary.
func OCRExcerpts(terms []string, lines []string) []OCRExcerpt {
	key := catalog.SearchTextKey(strings.Join(lines, "\n"))
	for _, term := range terms {
		if !strings.Contains(key, term) {
			return nil
		}
	}
	var out []OCRExcerpt
	for i, line := range lines {
		folded := catalog.SearchTextKey(line)
		at := -1
		for _, term := range terms {
			if p := strings.Index(folded, term); p >= 0 && (at < 0 || p < at) {
				at = p
			}
		}
		if at < 0 {
			continue
		}
		runes := []rune(line)
		start := 0
		if len(runes) > 160 {
			pos := 0
			for pos < len(runes) && len(catalog.SearchTextKey(string(runes[:pos]))) < at {
				pos++
			}
			start = max(0, pos-40)
			start = min(start, len(runes)-160)
		}
		end := min(len(runes), start+160)
		text := string(runes[start:end])
		if start > 0 {
			text = "…" + text
		}
		if end < len(runes) {
			text += "…"
		}
		out = append(out, OCRExcerpt{Line: i + 1, Text: text})
		if len(out) == 8 {
			break
		}
	}
	return out
}
