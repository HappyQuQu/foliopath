package app

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	httpReadHeaderTimeout = 10 * time.Second
	httpIdleTimeout       = 60 * time.Second
	httpMaxHeaderBytes    = 1 << 20
)

type httpService struct {
	address string
	server  *http.Server
	done    chan error

	mutex    sync.RWMutex
	listener net.Listener
}

func newHTTPComponent(address string, handler http.Handler, logger *slog.Logger) (component, *httpService) {
	if logger == nil {
		logger = slog.Default()
	}
	service := &httpService{
		address: address,
		server: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: httpReadHeaderTimeout,
			IdleTimeout:       httpIdleTimeout,
			MaxHeaderBytes:    httpMaxHeaderBytes,
			ErrorLog:          log.New(safeHTTPErrorWriter{logger: logger}, "", 0),
		},
		done: make(chan error, 1),
	}

	return component{
		name:  "http",
		start: service.start,
		done:  service.done,
		stop:  service.stop,
	}, service
}

type safeHTTPErrorWriter struct {
	logger *slog.Logger
}

func (writer safeHTTPErrorWriter) Write(message []byte) (int, error) {
	writer.logger.Error("http.server_error")
	return len(message), nil
}

func (service *httpService) start(context.Context) error {
	listener, err := net.Listen("tcp", service.address)
	if err != nil {
		return err
	}

	service.mutex.Lock()
	service.listener = listener
	service.mutex.Unlock()

	go func() {
		err := service.server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		service.done <- err
		close(service.done)
	}()
	return nil
}

func (service *httpService) stop(ctx context.Context) error {
	return service.server.Shutdown(ctx)
}

func (service *httpService) listenAddress() string {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.listener == nil {
		return ""
	}
	return service.listener.Addr().String()
}
