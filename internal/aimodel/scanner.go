package aimodel

import (
	"context"
	"errors"
	"sync"
)

const MaxScanCandidates = 64

var ErrCandidateStale = errors.New("AI model candidate is stale")

type RawCandidate struct {
	Manifest       []byte
	Files          []FileFact
	SourceIdentity string
	Failure        error
}

type CandidateSource interface {
	ScanModelPackages(context.Context, int, int, int64) ([]RawCandidate, bool, error)
}

type Candidate struct {
	ID             string
	Package        VerifiedPackage
	Manifest       Manifest
	Compatibility  string
	SourceIdentity string
}

type CandidateScan struct {
	Revision   int64
	Candidates []Candidate
	Truncated  bool
}

type Scanner struct {
	mu           sync.RWMutex
	source       CandidateSource
	catalog      *Catalog
	architecture string
	newID        IDGenerator
	revision     int64
	candidates   map[string]Candidate
}

func NewScanner(source CandidateSource, catalog *Catalog, architecture string, newID IDGenerator) (*Scanner, error) {
	if source == nil || catalog == nil ||
		(architecture != "amd64" && architecture != "arm64") {
		return nil, ErrInvalidModel
	}
	if newID == nil {
		newID = randomCandidateID
	}
	return &Scanner{
		source:       source,
		catalog:      catalog,
		architecture: architecture,
		newID:        newID,
		candidates:   map[string]Candidate{},
	}, nil
}

func (scanner *Scanner) Scan(ctx context.Context) (CandidateScan, error) {
	raw, truncated, err := scanner.source.ScanModelPackages(ctx, MaxScanCandidates, MaxPackageFiles, MaxPackageBytes)
	if err != nil {
		return CandidateScan{}, err
	}
	items := make([]Candidate, 0, len(raw))
	lookup := make(map[string]Candidate, len(raw))
	for _, current := range raw {
		id, idErr := scanner.newID()
		if idErr != nil {
			return CandidateScan{}, idErr
		}
		candidate := Candidate{ID: id, Compatibility: "incompatible"}
		if current.Failure == nil {
			verified, verifyErr := scanner.catalog.Verify(current.Manifest, current.Files, scanner.architecture)
			if verifyErr == nil {
				candidate.Package = verified
				candidate.Manifest, _ = scanner.catalog.Manifest(verified.PackageID)
				candidate.SourceIdentity = current.SourceIdentity
				candidate.Compatibility = "compatible"
			}
		}
		items = append(items, candidate)
		lookup[id] = candidate
	}

	scanner.mu.Lock()
	scanner.revision++
	scanner.candidates = lookup
	revision := scanner.revision
	scanner.mu.Unlock()

	return CandidateScan{Revision: revision, Candidates: items, Truncated: truncated}, nil
}

func (scanner *Scanner) Resolve(candidateID string, revision int64) (Candidate, error) {
	scanner.mu.RLock()
	defer scanner.mu.RUnlock()
	if revision < 1 || revision != scanner.revision {
		return Candidate{}, ErrCandidateStale
	}
	candidate, exists := scanner.candidates[candidateID]
	if !exists || candidate.Compatibility != "compatible" {
		return Candidate{}, ErrCandidateStale
	}
	return candidate, nil
}

// ResolveCurrent resolves only candidates from the latest completed scan.
// Candidate IDs are regenerated for every scan, so callers do not need to
// accept or trust a separately supplied revision.
func (scanner *Scanner) ResolveCurrent(candidateID string) (Candidate, error) {
	scanner.mu.RLock()
	defer scanner.mu.RUnlock()
	if scanner.revision < 1 || candidateID == "" {
		return Candidate{}, ErrCandidateStale
	}
	candidate, exists := scanner.candidates[candidateID]
	if !exists || candidate.Compatibility != "compatible" {
		return Candidate{}, ErrCandidateStale
	}
	return candidate, nil
}

func randomCandidateID() (string, error) {
	id, err := randomModelID()
	if err != nil {
		return "", err
	}
	return "aic_" + id[4:], nil
}
