package scanner

import (
	"context"
	"sync"
	"testing"
	"time"
)

type schedulerRepositoryStub struct {
	mutex    sync.Mutex
	interval *int64
	due      []int64
	admitted []int64
}

func (stub *schedulerRepositoryStub) GetScheduledScanIntervalHours(context.Context) (*int64, error) {
	stub.mutex.Lock()
	defer stub.mutex.Unlock()
	return stub.interval, nil
}

func (stub *schedulerRepositoryStub) ListDueLibraryIDs(
	_ context.Context, _ int64, afterID int64, limit int,
) ([]int64, error) {
	stub.mutex.Lock()
	defer stub.mutex.Unlock()
	var result []int64
	for _, id := range stub.due {
		if id > afterID && len(result) < limit {
			result = append(result, id)
		}
	}
	return result, nil
}

func (stub *schedulerRepositoryStub) AdmitFullScan(
	_ context.Context, libraryID int64, trigger Trigger,
) (AdmissionResult, error) {
	if trigger != TriggerScheduled {
		return AdmissionResult{}, ErrInvalidEntry
	}
	stub.mutex.Lock()
	defer stub.mutex.Unlock()
	stub.admitted = append(stub.admitted, libraryID)
	return AdmissionResult{Run: ScanRun{LibraryID: libraryID}}, nil
}

func TestSchedulerAdmitsDueLibrariesAndHonorsDisabledSetting(t *testing.T) {
	hours := int64(24)
	repository := &schedulerRepositoryStub{interval: &hours, due: []int64{1, 2}}
	work := &admissionWakeStub{}
	signal := &notificationStub{channel: make(chan struct{}, 1)}
	scheduler, err := NewScheduler(repository, work, signal, SchedulerOptions{
		PollInterval: time.Hour,
		Now:          func() time.Time { return time.UnixMilli(100_000_000) },
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	deadline := time.Now().Add(time.Second)
	for {
		repository.mutex.Lock()
		count := len(repository.admitted)
		repository.mutex.Unlock()
		if count == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scheduler did not admit due libraries")
		}
		time.Sleep(time.Millisecond)
	}
	repository.mutex.Lock()
	repository.interval = nil
	repository.due = append(repository.due, 3)
	repository.mutex.Unlock()
	signal.channel <- struct{}{}
	time.Sleep(10 * time.Millisecond)
	repository.mutex.Lock()
	count := len(repository.admitted)
	repository.mutex.Unlock()
	if count != 2 {
		t.Fatalf("disabled scheduler admitted %d libraries", count)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

type notificationStub struct{ channel chan struct{} }

func (stub *notificationStub) Notifications() <-chan struct{} { return stub.channel }
