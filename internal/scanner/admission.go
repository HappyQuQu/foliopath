package scanner

import (
	"context"
	"errors"
	"time"
)

var (
	ErrAdmissionConflict = errors.New("scan admission conflicts with library removal")
	ErrAdmissionCapacity = errors.New("scan admission capacity reached")
)

const (
	MaxActiveFullScans     = 256
	startupLibraryPageSize = 64
	startupCapacityRetry   = 25 * time.Millisecond
)

type AdmissionResult struct {
	Run       ScanRun
	Coalesced bool
}

type StartupAdmissionSummary struct {
	Admitted  int64
	Coalesced int64
	Skipped   int64
}

type AdmissionRepository interface {
	AdmitFullScan(context.Context, int64, Trigger) (AdmissionResult, error)
	ListStartupLibraryIDs(context.Context, int64, int) ([]int64, error)
}

type WakeNotifier interface {
	Wake()
}

type AdmissionService struct {
	repository AdmissionRepository
	waker      WakeNotifier
}

func NewAdmissionService(
	repository AdmissionRepository,
	waker WakeNotifier,
) (*AdmissionService, error) {
	if repository == nil || waker == nil {
		return nil, errors.New("scan admission dependencies are required")
	}
	return &AdmissionService{repository: repository, waker: waker}, nil
}

func (service *AdmissionService) RequestManual(
	ctx context.Context,
	libraryID int64,
) (AdmissionResult, error) {
	if libraryID <= 0 {
		return AdmissionResult{}, ErrLibraryNotFound
	}
	result, err := service.repository.AdmitFullScan(ctx, libraryID, TriggerManual)
	if err != nil {
		return AdmissionResult{}, err
	}
	if !result.Coalesced {
		service.waker.Wake()
	}
	return result, nil
}

func (service *AdmissionService) RequestAutomaticFallback(
	ctx context.Context,
	libraryID int64,
) (AdmissionResult, error) {
	if libraryID <= 0 {
		return AdmissionResult{}, ErrLibraryNotFound
	}
	result, err := service.repository.AdmitFullScan(
		ctx,
		libraryID,
		TriggerScheduled,
	)
	if err != nil {
		return AdmissionResult{}, err
	}
	if !result.Coalesced {
		service.waker.Wake()
	}
	return result, nil
}

func (service *AdmissionService) RequestStartup(
	ctx context.Context,
) (StartupAdmissionSummary, error) {
	var summary StartupAdmissionSummary
	var afterID int64
	for {
		libraryIDs, err := service.repository.ListStartupLibraryIDs(
			ctx,
			afterID,
			startupLibraryPageSize,
		)
		if err != nil {
			if ctx.Err() != nil {
				return summary, nil
			}
			return summary, err
		}
		if len(libraryIDs) == 0 {
			return summary, nil
		}
		for _, libraryID := range libraryIDs {
			if libraryID <= afterID {
				return summary, errors.New("startup library page is not strictly ordered")
			}
			for {
				result, err := service.repository.AdmitFullScan(
					ctx,
					libraryID,
					TriggerStartup,
				)
				switch {
				case err == nil:
					if result.Coalesced {
						summary.Coalesced++
					} else {
						summary.Admitted++
						service.waker.Wake()
					}
				case errors.Is(err, ErrAdmissionCapacity):
					timer := time.NewTimer(startupCapacityRetry)
					select {
					case <-ctx.Done():
						timer.Stop()
						return summary, nil
					case <-timer.C:
						continue
					}
				case errors.Is(err, ErrAdmissionConflict),
					errors.Is(err, ErrLibraryNotFound):
					summary.Skipped++
				default:
					if ctx.Err() != nil {
						return summary, nil
					}
					return summary, err
				}
				break
			}
			afterID = libraryID
		}
		if len(libraryIDs) < startupLibraryPageSize {
			return summary, nil
		}
	}
}
