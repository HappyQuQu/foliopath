package scanner

import (
	"context"
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
