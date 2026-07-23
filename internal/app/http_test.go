package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHTTPComponentGracefullyDrainsInFlightRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	var startOnce sync.Once
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		startOnce.Do(func() { close(requestStarted) })
		<-releaseRequest
		writer.WriteHeader(http.StatusNoContent)
	})
	logger := newJSONLogger(io.Discard)

	component, service := newHTTPComponent("127.0.0.1:0", handler, logger)
	if err := component.start(context.Background()); err != nil {
		t.Fatalf("start HTTP component: %v", err)
	}
	client := &http.Client{Timeout: time.Second}

	requestResult := make(chan error, 1)
	go func() {
		response, err := client.Get("http://" + service.listenAddress())
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			err = response.Body.Close()
		}
		requestResult <- err
	}()
	<-requestStarted

	shutdownResult := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdownResult <- component.stop(ctx)
	}()

	select {
	case err := <-shutdownResult:
		t.Fatalf("shutdown returned before in-flight request completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseRequest)

	if err := <-requestResult; err != nil {
		t.Fatalf("in-flight request failed: %v", err)
	}
	if err := <-shutdownResult; err != nil {
		t.Fatalf("graceful shutdown: %v", err)
	}

	response, err := client.Get("http://" + service.listenAddress())
	if err == nil {
		_ = response.Body.Close()
		t.Fatal("HTTP component accepted a new request after shutdown")
	}
}

func TestHTTPComponentReportsListenFailure(t *testing.T) {
	logger := newJSONLogger(io.Discard)
	first, firstService := newHTTPComponent("127.0.0.1:0", http.NotFoundHandler(), logger)
	if err := first.start(context.Background()); err != nil {
		t.Fatalf("start first HTTP component: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := first.stop(ctx); err != nil {
			t.Errorf("stop first HTTP component: %v", err)
		}
	}()

	second, _ := newHTTPComponent(firstService.listenAddress(), http.NotFoundHandler(), logger)
	if err := second.start(context.Background()); err == nil {
		t.Fatal("second HTTP component unexpectedly bound an address already in use")
	}
}

func TestHTTPServerErrorWriterDropsRuntimeDetails(t *testing.T) {
	var output bytes.Buffer
	writer := safeHTTPErrorWriter{logger: newJSONLogger(&output)}
	message := []byte("panic serving: token=secret /app/data stack trace")

	written, err := writer.Write(message)
	if err != nil || written != len(message) {
		t.Fatalf("Write() = (%d, %v), want (%d, nil)", written, err, len(message))
	}
	logged := output.String()
	if !strings.Contains(logged, `"msg":"http.server_error"`) {
		t.Fatalf("safe HTTP log missing event: %s", logged)
	}
	for _, forbidden := range []string{"token=secret", "/app/data", "stack trace"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("safe HTTP log leaked %q: %s", forbidden, logged)
		}
	}
}
