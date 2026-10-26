package resourcecontrol

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBackgroundLimitSerializesWorkAndCanExpandLive(t *testing.T) {
	controller, err := NewController(Limits{Background: 1, Content: 4})
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
		t.Fatal("background limit allowed a second operation")
	case <-time.After(20 * time.Millisecond):
	}

	if err := controller.ApplyLimits(Limits{Background: 2, Content: 8}); err != nil {
		t.Fatal(err)
	}
	select {
	case release := <-acquired:
		release()
	case <-time.After(time.Second):
		t.Fatal("live limit expansion did not release a waiting operation")
	}
}

type testProcessor func(context.Context, string) error

func (p testProcessor) Process(ctx context.Context, item string) error { return p(ctx, item) }

func TestCancelledBackgroundDoesNotConsumeFreeSlotOrInvokeProcessor(t *testing.T) {
	controller, _ := NewController(Limits{Background: 1, Content: 1})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if release, err := controller.AcquireBackground(ctx); !errors.Is(err, context.Canceled) {
		if release != nil {
			release()
		}
		t.Fatalf("cancelled admission: %v", err)
	}
	processor, _ := LimitBackground(controller, testProcessor(func(context.Context, string) error {
		t.Fatal("cancelled task entered processor")
		return nil
	}))
	if err := processor.Process(ctx, "video"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	live, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	release, err := controller.AcquireBackground(live)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestMixedBackgroundProcessorsShareBudgetAndReleaseOnFailure(t *testing.T) {
	controller, _ := NewController(Limits{Background: 1, Content: 1})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	entered := make(chan string, 3)
	finish := make(chan struct{})
	failure := errors.New("synthetic inference failure")
	image, _ := LimitBackground(controller, testProcessor(func(ctx context.Context, item string) error {
		entered <- item
		select {
		case <-finish:
			return failure
		case <-ctx.Done():
			return ctx.Err()
		}
	}))
	video, _ := LimitBackground(controller, testProcessor(func(_ context.Context, item string) error { entered <- item; return nil }))
	imageDone := make(chan error, 1)
	go func() { imageDone <- image.Process(ctx, "image") }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	videoDone := make(chan error, 1)
	go func() { videoDone <- video.Process(ctx, "video") }()
	select {
	case item := <-entered:
		t.Fatalf("exceeded shared budget: %s", item)
	case <-time.After(20 * time.Millisecond):
	}
	// Foreground content is an independent budget even while background work waits.
	releaseContent, ok := controller.TryAcquireContent()
	if !ok {
		t.Fatal("background blocked independent content budget")
	}
	releaseContent()
	close(finish)
	if err := <-imageDone; !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if err := <-videoDone; err != nil {
		t.Fatal(err)
	}
	select {
	case item := <-entered:
		if item != "video" {
			t.Fatal(item)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestContentAdmissionIsBoundedAndLoweringDoesNotCancelHolders(t *testing.T) {
	controller, err := NewController(Limits{Background: 4, Content: 16})
	if err != nil {
		t.Fatal(err)
	}
	releases := make([]func(), 0, 5)
	for range 5 {
		release, ok := controller.TryAcquireContent()
		if !ok {
			t.Fatal("content admission rejected within its limit")
		}
		releases = append(releases, release)
	}
	if err := controller.ApplyLimits(Limits{Background: 1, Content: 4}); err != nil {
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
	controller, err := NewController(Limits{Background: 1, Content: 4})
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
