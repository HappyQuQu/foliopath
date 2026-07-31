// Package resourcecontrol owns the instance-wide media-source concurrency policy.
package resourcecontrol

import (
	"context"
	"errors"
	"sync"
)

type Profile string

const (
	ProfileEco         Profile = "eco"
	ProfileBalanced    Profile = "balanced"
	ProfilePerformance Profile = "performance"
)

type limits struct {
	background int
	content    int
}

var profileLimits = map[Profile]limits{
	ProfileEco:         {background: 1, content: 4},
	ProfileBalanced:    {background: 2, content: 8},
	ProfilePerformance: {background: 4, content: 16},
}

func ValidateProfile(profile Profile) error {
	if _, ok := profileLimits[profile]; !ok {
		return errors.New("invalid resource profile")
	}
	return nil
}

// Controller applies one shared background budget across scans, reconciliation,
// and derived-media work, plus an independent cap for long-lived content reads.
// Existing holders are never cancelled when a limit is lowered.
type Controller struct {
	background *limiter
	content    *limiter
}

type Processor[T any] interface {
	Process(context.Context, T) error
}

type BackgroundProcessor[T any] struct {
	controller *Controller
	next       Processor[T]
}

func LimitBackground[T any](
	controller *Controller,
	next Processor[T],
) (*BackgroundProcessor[T], error) {
	if controller == nil || next == nil {
		return nil, errors.New("background processor dependencies are required")
	}
	return &BackgroundProcessor[T]{controller: controller, next: next}, nil
}

func (processor *BackgroundProcessor[T]) Process(ctx context.Context, item T) error {
	release, err := processor.controller.AcquireBackground(ctx)
	if err != nil {
		return err
	}
	defer release()
	return processor.next.Process(ctx, item)
}

func NewController(profile Profile) (*Controller, error) {
	selected, ok := profileLimits[profile]
	if !ok {
		return nil, errors.New("invalid resource profile")
	}
	return &Controller{
		background: newLimiter(selected.background),
		content:    newLimiter(selected.content),
	}, nil
}

func (controller *Controller) ApplyResourceProfile(profile Profile) error {
	selected, ok := profileLimits[profile]
	if !ok {
		return errors.New("invalid resource profile")
	}
	controller.background.setLimit(selected.background)
	controller.content.setLimit(selected.content)
	return nil
}

func (controller *Controller) AcquireBackground(ctx context.Context) (func(), error) {
	return controller.background.acquire(ctx)
}

func (controller *Controller) TryAcquireContent() (func(), bool) {
	return controller.content.tryAcquire()
}

type limiter struct {
	mutex   sync.Mutex
	limit   int
	inUse   int
	changed chan struct{}
}

func newLimiter(limit int) *limiter {
	return &limiter{limit: limit, changed: make(chan struct{})}
}

func (limiter *limiter) acquire(ctx context.Context) (func(), error) {
	for {
		limiter.mutex.Lock()
		if limiter.inUse < limiter.limit {
			limiter.inUse++
			limiter.mutex.Unlock()
			return limiter.releaseFunc(), nil
		}
		changed := limiter.changed
		limiter.mutex.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

func (limiter *limiter) tryAcquire() (func(), bool) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	if limiter.inUse >= limiter.limit {
		return nil, false
	}
	limiter.inUse++
	return limiter.releaseFunc(), true
}

func (limiter *limiter) releaseFunc() func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			limiter.mutex.Lock()
			limiter.inUse--
			limiter.notifyLocked()
			limiter.mutex.Unlock()
		})
	}
}

func (limiter *limiter) setLimit(limit int) {
	limiter.mutex.Lock()
	limiter.limit = limit
	limiter.notifyLocked()
	limiter.mutex.Unlock()
}

func (limiter *limiter) notifyLocked() {
	close(limiter.changed)
	limiter.changed = make(chan struct{})
}
