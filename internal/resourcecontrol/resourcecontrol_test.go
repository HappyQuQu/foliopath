package resourcecontrol

import (
	"context"
	"testing"
	"time"
)

func TestEcoProfileSerializesBackgroundWorkAndCanExpandLive(t *testing.T) {
	controller, err := NewController(ProfileEco)
	if err != nil {
		t.Fatal(err)
	}
	releaseFirst, err := controller.AcquireBackground(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer releaseFirst()

	acquired := make(chan func(), 1)
	go func() {
		release, acquireErr := controller.AcquireBackground(context.Background())
		if acquireErr == nil {
			acquired <- release
		}
	}()
	select {
	case release := <-acquired:
		release()
		t.Fatal("eco profile allowed a second background operation")
	case <-time.After(20 * time.Millisecond):
	}

	if err := controller.ApplyResourceProfile(ProfileBalanced); err != nil {
		t.Fatal(err)
	}
	select {
	case release := <-acquired:
		release()
	case <-time.After(time.Second):
		t.Fatal("live profile expansion did not release a waiting operation")
	}
}

func TestContentAdmissionIsBoundedAndLoweringDoesNotCancelHolders(t *testing.T) {
	controller, err := NewController(ProfilePerformance)
	if err != nil {
		t.Fatal(err)
	}
	releases := make([]func(), 0, 5)
	for range 5 {
		release, ok := controller.TryAcquireContent()
		if !ok {
			t.Fatal("performance content admission rejected within its limit")
		}
		releases = append(releases, release)
	}
	if err := controller.ApplyResourceProfile(ProfileEco); err != nil {
		t.Fatal(err)
	}
	releases[0]()
	if release, ok := controller.TryAcquireContent(); ok {
		release()
		t.Fatal("lowered limit admitted work while holders still exceed it")
	}
	for _, release := range releases[1:] {
		release()
	}
	if release, ok := controller.TryAcquireContent(); !ok {
		t.Fatal("content admission did not recover after releases")
	} else {
		release()
	}
}

func TestAcquireBackgroundHonorsCancellation(t *testing.T) {
	controller, err := NewController(ProfileEco)
	if err != nil {
		t.Fatal(err)
	}
	release, err := controller.AcquireBackground(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := controller.AcquireBackground(ctx); err != context.Canceled {
		t.Fatalf("acquire error = %v, want context.Canceled", err)
	}
}
