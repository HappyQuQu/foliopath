// Command fs05-runtime is an isolated Stage 0 container and recovery probe.
// It is not the production FolioPath entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/HappyQuQu/foliopath/internal/library"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Printf("fs05 runtime spike failed: %v", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}
	switch command {
	case "serve":
		return serve()
	case "healthcheck":
		url := "http://127.0.0.1:8080/health/ready"
		if len(args) > 1 {
			url = args[1]
		}
		return healthcheck(url)
	case "seed":
		name := "Recovery Library"
		if len(args) > 1 {
			name = args[1]
		}
		return seed(name)
	case "verify":
		name := "Recovery Library"
		if len(args) > 1 {
			name = args[1]
		}
		return verify(name)
	case "version":
		fmt.Println(version)
		return nil
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func dataDirectory() string {
	if value := os.Getenv("FOLIOPATH_DATA_DIR"); value != "" {
		return value
	}
	return "/app/data"
}

func openStore(ctx context.Context) (*sqlitestore.Store, error) {
	dataDir := dataDirectory()
	if err := os.MkdirAll(filepath.Join(dataDir, "cache"), 0o750); err != nil {
		return nil, fmt.Errorf("prepare cache directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "tmp"), 0o750); err != nil {
		return nil, fmt.Errorf("prepare temporary directory: %w", err)
	}
	store, err := sqlitestore.Open(ctx, filepath.Join(dataDir, "foliopath.db"), sqlitestore.Options{
		BusyTimeout:        5 * time.Second,
		MaxOpenConnections: 4,
		MaxBatchSize:       500,
	})
	if err != nil {
		return nil, err
	}
	return store, nil
}

func serve() error {
	if info, err := os.Stat("/library"); err != nil || !info.IsDir() {
		return errors.New("/library is not an accessible directory")
	}
	store, err := openStore(context.Background())
	if err != nil {
		return fmt.Errorf("open data store: %w", err)
	}
	defer store.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = response.Write([]byte("live\n"))
	})
	mux.HandleFunc("GET /health/ready", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = response.Write([]byte("ready\n"))
	})
	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("fs05 runtime spike %s ready", version)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case received := <-signals:
		log.Printf("received %s; starting graceful shutdown", received)
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		log.Print("graceful shutdown complete")
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func healthcheck(url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %s", response.Status)
	}
	return nil
}

func seed(name string) error {
	store, err := openStore(context.Background())
	if err != nil {
		return err
	}
	defer store.Close()
	service, err := library.NewService(store)
	if err != nil {
		return err
	}
	_, err = service.Create(context.Background(), name, "recovery")
	return err
}

func verify(name string) error {
	store, err := openStore(context.Background())
	if err != nil {
		return err
	}
	defer store.Close()
	service, err := library.NewService(store)
	if err != nil {
		return err
	}
	libraries, err := service.List(context.Background())
	if err != nil {
		return err
	}
	for _, item := range libraries {
		if item.Name == name && item.RootRelativePath == "recovery" {
			fmt.Printf("verified library %q\n", name)
			return nil
		}
	}
	return fmt.Errorf("library %q was not restored", name)
}
