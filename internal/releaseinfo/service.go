// Package releaseinfo owns the safe application-version and update projection.
package releaseinfo

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Release struct {
	Version     string
	Name        string
	Summary     string
	Notes       string
	PublishedAt time.Time
	URL         string
}

type Snapshot struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	CheckedAt       time.Time
	Releases        []Release
}

type Source interface {
	ListStableReleases(context.Context, int) ([]Release, error)
}

type Service struct {
	current        string
	source         Source
	now            func() time.Time
	ttl            time.Duration
	minimumRefresh time.Duration

	mutex  sync.Mutex
	cached Snapshot
}

func NewService(current string, source Source) (*Service, error) {
	if strings.TrimSpace(current) == "" || source == nil {
		return nil, errors.New("release information dependencies are required")
	}
	return &Service{
		current: current, source: source, now: time.Now, ttl: 6 * time.Hour,
		minimumRefresh: time.Minute,
	}, nil
}

func (service *Service) Get(ctx context.Context, refresh bool) (Snapshot, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	now := service.now().UTC()
	if !service.cached.CheckedAt.IsZero() {
		age := now.Sub(service.cached.CheckedAt)
		if (!refresh && age < service.ttl) ||
			(refresh && age < service.minimumRefresh) {
			return cloneSnapshot(service.cached), nil
		}
	}
	releases, err := service.source.ListStableReleases(ctx, 20)
	if err != nil {
		if !service.cached.CheckedAt.IsZero() {
			return cloneSnapshot(service.cached), nil
		}
		return Snapshot{}, err
	}
	snapshot := Snapshot{CurrentVersion: service.current, CheckedAt: now, Releases: releases}
	if len(releases) > 0 {
		snapshot.LatestVersion = releases[0].Version
		snapshot.UpdateAvailable = newer(releases[0].Version, service.current)
	}
	service.cached = snapshot
	return cloneSnapshot(snapshot), nil
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	snapshot.Releases = append([]Release(nil), snapshot.Releases...)
	return snapshot
}

func newer(candidate, current string) bool {
	candidateVersion, ok := parseVersion(candidate)
	if !ok {
		return false
	}
	currentVersion, ok := parseVersion(current)
	if !ok {
		return false
	}
	for index := range candidateVersion {
		if candidateVersion[index] != currentVersion[index] {
			return candidateVersion[index] > currentVersion[index]
		}
	}
	return false
}

func parseVersion(value string) ([3]int64, bool) {
	var result [3]int64
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	if strings.ContainsAny(value, "+-") {
		return result, false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return result, false
	}
	for index, part := range parts {
		parsed, err := strconv.ParseInt(part, 10, 32)
		if err != nil || parsed < 0 || (len(part) > 1 && part[0] == '0') {
			return result, false
		}
		result[index] = parsed
	}
	return result, true
}
