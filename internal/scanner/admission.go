package scanner

import (
	"context"
	"errors"
)

var (
	ErrAdmissionConflict = errors.New("scan admission conflicts with library removal")
	ErrAdmissionCapacity = errors.New("scan admission capacity reached")
)

type AdmissionResult struct {
	Run       ScanRun
	Coalesced bool
}

type AdmissionRepository interface {
	AdmitFullScan(context.Context, int64, Trigger) (AdmissionResult, error)
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
