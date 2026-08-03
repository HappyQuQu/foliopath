package sqlite

import (
	"context"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/systemlog"
)

func TestSystemEventsAreBoundedAndFilterable(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	for _, event := range []systemlog.Event{
		{Level: systemlog.LevelInfo, Module: "application", EventCode: "application.started"},
		{Level: systemlog.LevelWarning, Module: "http", EventCode: "http.request_rejected", StatusCode: 409},
		{Level: systemlog.LevelError, Module: "http", EventCode: "http.request_failed", RequestID: "req_test", StatusCode: 500},
	} {
		if err := store.AppendSystemEvent(ctx, event, 2); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.ListSystemEvents(ctx, systemlog.Query{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Level != systemlog.LevelError ||
		events[1].Level != systemlog.LevelWarning {
		t.Fatalf("events = %#v", events)
	}
	errorsOnly, err := store.ListSystemEvents(ctx, systemlog.Query{
		Level: systemlog.LevelError, Limit: 10,
	})
	if err != nil || len(errorsOnly) != 1 || errorsOnly[0].RequestID != "req_test" {
		t.Fatalf("error events = %#v, %v", errorsOnly, err)
	}
}
