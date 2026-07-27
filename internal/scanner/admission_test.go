package scanner

import (
	"context"
	"errors"
	"testing"
)

type admissionRepositoryStub struct {
	result AdmissionResult
	err    error
}

func (stub admissionRepositoryStub) AdmitFullScan(
	context.Context,
	int64,
	Trigger,
) (AdmissionResult, error) {
	return stub.result, stub.err
}

func (stub admissionRepositoryStub) ListStartupLibraryIDs(
	context.Context,
	int64,
	int,
) ([]int64, error) {
	return nil, nil
}

type admissionWakeStub struct{ calls int }

func (stub *admissionWakeStub) Wake() { stub.calls++ }

func TestAdmissionServiceWakesOnlyForNewDurableWork(t *testing.T) {
	for _, test := range []struct {
		name      string
		coalesced bool
		wantWakes int
	}{
		{name: "new", wantWakes: 1},
		{name: "coalesced", coalesced: true, wantWakes: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			waker := &admissionWakeStub{}
			service, err := NewAdmissionService(admissionRepositoryStub{
				result: AdmissionResult{
					Run:       ScanRun{ID: 1, LibraryID: 2},
					Coalesced: test.coalesced,
				},
			}, waker)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.RequestManual(context.Background(), 2); err != nil {
				t.Fatal(err)
			}
			if waker.calls != test.wantWakes {
				t.Fatalf("wake calls = %d, want %d", waker.calls, test.wantWakes)
			}
		})
	}
}

type startupAdmissionRepositoryStub struct {
	ids         []int64
	admitCalls  map[int64]int
	seenTrigger []Trigger
}

func (stub *startupAdmissionRepositoryStub) ListStartupLibraryIDs(
	_ context.Context,
	afterID int64,
	limit int,
) ([]int64, error) {
	page := make([]int64, 0, limit)
	for _, id := range stub.ids {
		if id > afterID && len(page) < limit {
			page = append(page, id)
		}
	}
	return page, nil
}

func (stub *startupAdmissionRepositoryStub) AdmitFullScan(
	_ context.Context,
	libraryID int64,
	trigger Trigger,
) (AdmissionResult, error) {
	stub.admitCalls[libraryID]++
	stub.seenTrigger = append(stub.seenTrigger, trigger)
	switch libraryID {
	case 1:
		if stub.admitCalls[libraryID] == 1 {
			return AdmissionResult{}, ErrAdmissionCapacity
		}
		return AdmissionResult{Run: ScanRun{ID: 11, LibraryID: libraryID}}, nil
	case 2:
		return AdmissionResult{
			Run:       ScanRun{ID: 12, LibraryID: libraryID},
			Coalesced: true,
		}, nil
	case 3:
		return AdmissionResult{}, ErrAdmissionConflict
	default:
		return AdmissionResult{}, errors.New("unexpected library")
	}
}

func TestAdmissionServiceRequestsBoundedStartupScans(t *testing.T) {
	repository := &startupAdmissionRepositoryStub{
		ids:        []int64{1, 2, 3},
		admitCalls: make(map[int64]int),
	}
	waker := &admissionWakeStub{}
	service, err := NewAdmissionService(repository, waker)
	if err != nil {
		t.Fatal(err)
	}

	summary, err := service.RequestStartup(context.Background())
	if err != nil {
		t.Fatalf("RequestStartup() error = %v", err)
	}
	if summary != (StartupAdmissionSummary{Admitted: 1, Coalesced: 1, Skipped: 1}) {
		t.Fatalf("startup summary = %#v", summary)
	}
	if repository.admitCalls[1] != 2 {
		t.Fatalf("capacity retry calls = %d, want 2", repository.admitCalls[1])
	}
	if waker.calls != 1 {
		t.Fatalf("wake calls = %d, want 1", waker.calls)
	}
	for _, trigger := range repository.seenTrigger {
		if trigger != TriggerStartup {
			t.Fatalf("startup trigger = %q", trigger)
		}
	}
}
