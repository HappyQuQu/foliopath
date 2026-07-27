package scanner

import (
	"context"
	"errors"
	"time"
)

const scheduledLibraryPageSize = 64

type ScheduleRepository interface {
	GetScheduledScanIntervalHours(context.Context) (*int64, error)
	ListDueLibraryIDs(context.Context, int64, int64, int) ([]int64, error)
	AdmitFullScan(context.Context, int64, Trigger) (AdmissionResult, error)
}

type ScheduleSignal interface {
	Notifications() <-chan struct{}
}

type SchedulerOptions struct {
	PollInterval time.Duration
	Now          func() time.Time
}

type Scheduler struct {
	repository ScheduleRepository
	workWaker  WakeNotifier
	signal     ScheduleSignal
	poll       time.Duration
	now        func() time.Time
}

func NewScheduler(
	repository ScheduleRepository,
	workWaker WakeNotifier,
	signal ScheduleSignal,
	options SchedulerOptions,
) (*Scheduler, error) {
	if repository == nil || workWaker == nil || signal == nil {
		return nil, errors.New("scan scheduler dependencies are required")
	}
	if options.PollInterval == 0 {
		options.PollInterval = time.Minute
	}
	if options.PollInterval < time.Millisecond {
		return nil, errors.New("scan scheduler poll interval is too short")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Scheduler{
		repository: repository, workWaker: workWaker, signal: signal,
		poll: options.PollInterval, now: options.Now,
	}, nil
}

func (scheduler *Scheduler) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		case <-scheduler.signal.Notifications():
		}
		if err := scheduler.reconcile(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		timer.Reset(scheduler.poll)
	}
}

func (scheduler *Scheduler) reconcile(ctx context.Context) error {
	interval, err := scheduler.repository.GetScheduledScanIntervalHours(ctx)
	if err != nil || interval == nil {
		return err
	}
	cutoff := scheduler.now().Add(-time.Duration(*interval) * time.Hour).UnixMilli()
	var afterID int64
	for {
		ids, err := scheduler.repository.ListDueLibraryIDs(
			ctx, cutoff, afterID, scheduledLibraryPageSize,
		)
		if err != nil {
			return err
		}
		for _, libraryID := range ids {
			if libraryID <= afterID {
				return errors.New("scheduled library page is not strictly ordered")
			}
			result, err := scheduler.repository.AdmitFullScan(ctx, libraryID, TriggerScheduled)
			switch {
			case err == nil:
				if !result.Coalesced {
					scheduler.workWaker.Wake()
				}
			case errors.Is(err, ErrAdmissionConflict), errors.Is(err, ErrLibraryNotFound):
			case errors.Is(err, ErrAdmissionCapacity):
				return nil
			default:
				return err
			}
			afterID = libraryID
		}
		if len(ids) < scheduledLibraryPageSize {
			return nil
		}
	}
}
