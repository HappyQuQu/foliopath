// Package systemlog owns the bounded, sanitized operational event history
// exposed to administrators. It never stores arbitrary errors, paths, SQL,
// stack traces, headers, or subprocess output.
package systemlog

import (
	"context"
	"errors"
)

const (
	MaxPageSize = 100
	MaxEntries  = 5_000
)

type Level string

const (
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

type Event struct {
	ID           int64
	OccurredAtMS int64
	Level        Level
	Module       string
	EventCode    string
	RequestID    string
	Method       string
	RoutePattern string
	StatusCode   int
	DurationMS   int64
}

type Query struct {
	Level    Level
	Module   string
	BeforeID int64
	Limit    int
}

type Repository interface {
	AppendSystemEvent(context.Context, Event, int) error
	ListSystemEvents(context.Context, Query) ([]Event, error)
}

type Service struct {
	repository Repository
}

var ErrInvalidRequest = errors.New("invalid system log request")

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("system log repository is required")
	}
	return &Service{repository: repository}, nil
}

func (service *Service) Record(ctx context.Context, event Event) error {
	if !validLevel(event.Level) || len(event.Module) < 1 || len(event.Module) > 64 ||
		len(event.EventCode) < 1 || len(event.EventCode) > 128 ||
		len(event.RequestID) > 128 || len(event.Method) > 16 ||
		len(event.RoutePattern) > 512 || event.StatusCode < 0 ||
		event.StatusCode > 599 || event.DurationMS < 0 {
		return ErrInvalidRequest
	}
	return service.repository.AppendSystemEvent(ctx, event, MaxEntries)
}

func (service *Service) List(ctx context.Context, query Query) ([]Event, error) {
	if query.BeforeID < 0 || query.Limit < 1 || query.Limit > MaxPageSize ||
		(query.Level != "" && !validLevel(query.Level)) || len(query.Module) > 64 {
		return nil, ErrInvalidRequest
	}
	return service.repository.ListSystemEvents(ctx, query)
}

func validLevel(level Level) bool {
	return level == LevelInfo || level == LevelWarning || level == LevelError
}
