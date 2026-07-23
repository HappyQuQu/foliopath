package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

const runtimeIntegrationTimeout = 5 * time.Second

func TestComposedApplicationStartsServesStopsAndRestarts(t *testing.T) {
	dataRoot := t.TempDir()
	mediaRoot := t.TempDir()
	var sessionCookie string

	for runNumber := 1; runNumber <= 2; runNumber++ {
		runComposedApplication(t, dataRoot, mediaRoot, runNumber, &sessionCookie)
	}

	for _, relativePath := range []string{
		databaseFilename,
		"cache",
		"tmp",
	} {
		if _, err := os.Stat(filepath.Join(dataRoot, relativePath)); err != nil {
			t.Fatalf("application data %q: %v", relativePath, err)
		}
	}
}

func runComposedApplication(
	t *testing.T,
	dataRoot string,
	mediaRoot string,
	runNumber int,
	sessionCookie *string,
) {
	t.Helper()

	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     mediaRoot,
			dataRoot:      dataRoot,
		},
	)
	if err != nil {
		t.Fatalf("run %d composeConfiguration() error = %v", runNumber, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-result:
		default:
		}
	})

	address := waitForListenAddress(t, application.http)
	assertAdministratorSession(t, application.authentication, runNumber, sessionCookie)
	client := &http.Client{Timeout: time.Second}
	assertRuntimeResponse(t, client, address, "/health/ready", http.StatusOK, "ready")
	assertRuntimeResponse(t, client, address, "/health/live", http.StatusOK, "live")
	assertRuntimeResponse(
		t,
		client,
		address,
		"/api/v1/status",
		http.StatusUnauthorized,
		"authentication_required",
	)

	cancel()
	select {
	case runErr := <-result:
		if runErr != nil {
			t.Fatalf("run %d application.run() error = %v", runNumber, runErr)
		}
	case <-time.After(runtimeIntegrationTimeout):
		t.Fatalf("run %d application did not stop after cancellation", runNumber)
	}

	connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatalf("run %d HTTP listener %q remained open after shutdown", runNumber, address)
	}
}

func assertAdministratorSession(
	t *testing.T,
	authentication *auth.Service,
	runNumber int,
	sessionCookie *string,
) {
	t.Helper()
	if authentication == nil {
		t.Fatal("composed application has no authentication service")
	}

	state, err := authentication.SetupState(context.Background())
	if err != nil {
		t.Fatalf("run %d read setup state: %v", runNumber, err)
	}
	if runNumber == 1 {
		if state != auth.SetupRequired {
			t.Fatalf("initial setup state = %q, want %q", state, auth.SetupRequired)
		}
		established, err := authentication.Initialize(
			context.Background(),
			auth.InitializeParams{
				Username:    "Administrator",
				DisplayName: "Administrator",
				Password:    "correct horse battery staple",
			},
		)
		if err != nil {
			t.Fatalf("initialize administrator: %v", err)
		}
		*sessionCookie = established.CookieToken
		state, err = authentication.SetupState(context.Background())
		if err != nil {
			t.Fatalf("read initialized setup state: %v", err)
		}
	}
	if state != auth.SetupComplete {
		t.Fatalf("run %d setup state = %q, want %q", runNumber, state, auth.SetupComplete)
	}
	if _, err := authentication.Session(
		context.Background(),
		*sessionCookie,
	); err != nil {
		t.Fatalf("run %d restore initialized session: %v", runNumber, err)
	}
	if runNumber == 2 {
		loggedIn, err := authentication.Login(context.Background(), auth.LoginParams{
			Username: "administrator",
			Password: "correct horse battery staple",
		})
		if err != nil {
			t.Fatalf("login administrator after restart: %v", err)
		}
		if err := authentication.Logout(
			context.Background(),
			loggedIn.CookieToken,
		); err != nil {
			t.Fatalf("logout administrator: %v", err)
		}
		if _, err := authentication.Session(
			context.Background(),
			loggedIn.CookieToken,
		); !errors.Is(err, auth.ErrSessionExpired) {
			t.Fatalf("revoked session error = %v, want session expired", err)
		}
	}
}

func waitForListenAddress(t *testing.T, service *httpService) string {
	t.Helper()
	if service == nil {
		t.Fatal("composed application has no HTTP service")
	}

	deadline := time.Now().Add(runtimeIntegrationTimeout)
	for time.Now().Before(deadline) {
		if address := service.listenAddress(); address != "" {
			return address
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("HTTP service did not start")
	return ""
}

func assertRuntimeResponse(
	t *testing.T,
	client *http.Client,
	address string,
	path string,
	wantStatus int,
	wantValue string,
) {
	t.Helper()

	deadline := time.Now().Add(runtimeIntegrationTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"http://"+address+path,
			nil,
		)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		response, err := client.Do(request)
		if err != nil {
			lastErr = err
			time.Sleep(10 * time.Millisecond)
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			lastErr = errors.Join(readErr, closeErr)
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if response.StatusCode != wantStatus {
			t.Fatalf(
				"GET %s status = %d, want %d; body = %s",
				path,
				response.StatusCode,
				wantStatus,
				body,
			)
		}

		var document struct {
			Status string `json:"status"`
			Error  struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &document); err != nil {
			t.Fatalf("GET %s response is not JSON: %v", path, err)
		}
		if document.Status != wantValue && document.Error.Code != wantValue {
			t.Fatalf("GET %s response = %s, want value %q", path, body, wantValue)
		}
		if response.Header.Get("X-Request-ID") == "" {
			t.Fatalf("GET %s response has no X-Request-ID", path)
		}
		return
	}
	t.Fatalf("GET %s did not succeed: %v", path, lastErr)
}
