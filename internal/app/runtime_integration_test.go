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
	"strings"
	"sync"
	"testing"
	"time"
)

const runtimeIntegrationTimeout = 10 * time.Second

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

func TestComposedAuthenticationHandlesConcurrentSetupAndSessionRequests(t *testing.T) {
	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     t.TempDir(),
			dataRoot:      t.TempDir(),
		},
	)
	if err != nil {
		t.Fatalf("composeConfiguration() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()
	defer func() {
		cancel()
		select {
		case runErr := <-result:
			if runErr != nil {
				t.Errorf("application.run() error = %v", runErr)
			}
		case <-time.After(runtimeIntegrationTimeout):
			t.Error("application did not stop after cancellation")
		}
	}()

	address := waitForListenAddress(t, application.http)
	client := &http.Client{Timeout: runtimeIntegrationTimeout}
	const concurrentRequests = 8
	type setupResult struct {
		status    int
		errorCode string
		body      string
		cookie    string
		err       error
	}
	results := make([]setupResult, concurrentRequests)
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range results {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			request, requestErr := http.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"http://"+address+"/api/v1/auth/setup",
				strings.NewReader(
					`{"username":"Administrator","displayName":"Administrator",`+
						`"password":"correct horse battery staple"}`,
				),
			)
			if requestErr != nil {
				results[index].err = requestErr
				return
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", "http://"+address)
			response, requestErr := client.Do(request)
			if requestErr != nil {
				results[index].err = requestErr
				return
			}
			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				results[index].err = errors.Join(readErr, closeErr)
				return
			}
			results[index].status = response.StatusCode
			results[index].body = string(body)
			var document struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(body, &document); err != nil {
				results[index].err = err
				return
			}
			results[index].errorCode = document.Error.Code
			for _, cookie := range response.Cookies() {
				if cookie.Name == "foliopath_session" {
					results[index].cookie = cookie.Value
				}
			}
		}()
	}
	close(start)
	group.Wait()

	successes := 0
	sessionCookie := ""
	for index, setup := range results {
		if setup.err != nil {
			t.Fatalf("concurrent setup %d error = %v", index, setup.err)
		}
		switch setup.status {
		case http.StatusCreated:
			successes++
			sessionCookie = setup.cookie
			if setup.cookie == "" {
				t.Fatalf("successful concurrent setup %d returned no cookie", index)
			}
		case http.StatusConflict:
			if setup.errorCode != "setup_in_progress" && setup.errorCode != "setup_closed" {
				t.Fatalf("concurrent setup %d error = %#v", index, setup)
			}
			for _, forbidden := range []string{
				"correct horse battery staple",
				"/app/data",
				"password_hash",
			} {
				if strings.Contains(setup.body, forbidden) {
					t.Fatalf("concurrent setup %d leaked %q: %s", index, forbidden, setup.body)
				}
			}
		default:
			t.Fatalf("concurrent setup %d = %#v", index, setup)
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent setups = %d, want 1", successes)
	}

	repeated := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodPost,
		"/api/v1/auth/setup",
		`{"username":"Other","displayName":"Other","password":"another valid password"}`,
		"",
		"",
	)
	if repeated.StatusCode != http.StatusConflict ||
		repeated.ErrorCode != "setup_closed" ||
		repeated.Cookie != "" {
		t.Fatalf("repeated setup response = %#v", repeated)
	}

	wrongPassword := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodPost,
		"/api/v1/auth/login",
		`{"username":"Administrator","password":"wrong password"}`,
		"",
		"",
	)
	unknownAdministrator := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodPost,
		"/api/v1/auth/login",
		`{"username":"Unknown","password":"wrong password"}`,
		"",
		"",
	)
	for name, response := range map[string]runtimeAuthenticationResponse{
		"wrong password":        wrongPassword,
		"unknown administrator": unknownAdministrator,
	} {
		if response.StatusCode != http.StatusUnauthorized ||
			response.ErrorCode != "invalid_credentials" ||
			response.Cookie != "" {
			t.Fatalf("%s response = %#v", name, response)
		}
		for _, forbidden := range []string{"wrong password", "Administrator", "Unknown"} {
			if strings.Contains(response.Body, forbidden) {
				t.Fatalf("%s response leaked %q: %s", name, forbidden, response.Body)
			}
		}
	}
	if wrongPassword.ErrorMessage != unknownAdministrator.ErrorMessage {
		t.Fatalf(
			"credential failure messages differ: %q != %q",
			wrongPassword.ErrorMessage,
			unknownAdministrator.ErrorMessage,
		)
	}

	type sessionResult struct {
		status int
		err    error
	}
	sessionResults := make([]sessionResult, 32)
	start = make(chan struct{})
	group = sync.WaitGroup{}
	for index := range sessionResults {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			request, requestErr := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"http://"+address+"/api/v1/auth/session",
				nil,
			)
			if requestErr != nil {
				sessionResults[index].err = requestErr
				return
			}
			request.AddCookie(&http.Cookie{
				Name:  "foliopath_session",
				Value: sessionCookie,
			})
			response, requestErr := client.Do(request)
			if requestErr != nil {
				sessionResults[index].err = requestErr
				return
			}
			_, readErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			sessionResults[index] = sessionResult{
				status: response.StatusCode,
				err:    errors.Join(readErr, closeErr),
			}
		}()
	}
	close(start)
	group.Wait()
	for index, session := range sessionResults {
		if session.err != nil || session.status != http.StatusOK {
			t.Fatalf("concurrent session %d = %#v", index, session)
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
	client := &http.Client{Timeout: runtimeIntegrationTimeout}
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
	assertAdministratorSessionHTTP(t, application, client, address, runNumber, sessionCookie)

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

func assertAdministratorSessionHTTP(
	t *testing.T,
	application *application,
	client *http.Client,
	address string,
	runNumber int,
	sessionCookie *string,
) {
	t.Helper()
	if application.authentication == nil {
		t.Fatal("composed application has no authentication service")
	}

	status := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodGet,
		"/api/v1/auth/status",
		"",
		"",
		"",
	)
	if runNumber == 1 {
		if status.StatusCode != http.StatusOK || !status.SetupRequired {
			t.Fatalf("initial authentication status = %#v", status)
		}
		established := runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodPost,
			"/api/v1/auth/setup",
			`{"username":"Administrator","displayName":"Administrator","password":"correct horse battery staple"}`,
			"",
			"",
		)
		if established.StatusCode != http.StatusCreated ||
			established.Cookie == "" ||
			established.CSRFToken == "" {
			t.Fatalf("setup response = %#v", established)
		}
		*sessionCookie = established.Cookie
		status = runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodGet,
			"/api/v1/auth/status",
			"",
			"",
			"",
		)
	}
	if status.StatusCode != http.StatusOK || status.SetupRequired {
		t.Fatalf("run %d authentication status = %#v", runNumber, status)
	}

	current := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodGet,
		"/api/v1/auth/session",
		"",
		*sessionCookie,
		"",
	)
	if current.StatusCode != http.StatusOK || current.CSRFToken == "" {
		t.Fatalf("run %d restore HTTP session = %#v", runNumber, current)
	}

	authorizedStatus := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodGet,
		"/api/v1/status",
		"",
		*sessionCookie,
		"",
	)
	if authorizedStatus.StatusCode != http.StatusOK {
		t.Fatalf("run %d authorized status = %#v", runNumber, authorizedStatus)
	}

	if runNumber == 2 {
		loggedIn := runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodPost,
			"/api/v1/auth/login",
			`{"username":"administrator","password":"correct horse battery staple"}`,
			"",
			"",
		)
		if loggedIn.StatusCode != http.StatusOK ||
			loggedIn.Cookie == "" ||
			loggedIn.CSRFToken == "" {
			t.Fatalf("login administrator after restart = %#v", loggedIn)
		}
		loggedOut := runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodPost,
			"/api/v1/auth/logout",
			"",
			loggedIn.Cookie,
			loggedIn.CSRFToken,
		)
		if loggedOut.StatusCode != http.StatusNoContent || !loggedOut.CookieExpired {
			t.Fatalf("logout response = %#v", loggedOut)
		}
		revoked := runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodGet,
			"/api/v1/auth/session",
			"",
			loggedIn.Cookie,
			"",
		)
		if revoked.StatusCode != http.StatusUnauthorized ||
			revoked.ErrorCode != "session_expired" {
			t.Fatalf("revoked session response = %#v", revoked)
		}
	}
}

type runtimeAuthenticationResponse struct {
	StatusCode    int
	SetupRequired bool
	CSRFToken     string
	Cookie        string
	CookieExpired bool
	ErrorCode     string
	ErrorMessage  string
	Body          string
}

func runtimeAuthenticationRequest(
	t *testing.T,
	client *http.Client,
	address string,
	method string,
	path string,
	body string,
	cookie string,
	csrfToken string,
) runtimeAuthenticationResponse {
	t.Helper()
	request, err := http.NewRequestWithContext(
		context.Background(),
		method,
		"http://"+address+path,
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, path, err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://"+address)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "foliopath_session", Value: cookie})
	}
	if csrfToken != "" {
		request.Header.Set("X-CSRF-Token", csrfToken)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, path, err)
	}
	document := struct {
		SetupRequired bool   `json:"setupRequired"`
		CSRFToken     string `json:"csrfToken"`
		Error         struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if response.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(bodyBytes, &document); err != nil {
			t.Fatalf("decode %s %s response: %v", method, path, err)
		}
	}
	result := runtimeAuthenticationResponse{
		StatusCode:    response.StatusCode,
		SetupRequired: document.SetupRequired,
		CSRFToken:     document.CSRFToken,
		ErrorCode:     document.Error.Code,
		ErrorMessage:  document.Error.Message,
		Body:          string(bodyBytes),
	}
	for _, responseCookie := range response.Cookies() {
		if responseCookie.Name != "foliopath_session" {
			continue
		}
		result.Cookie = responseCookie.Value
		result.CookieExpired = responseCookie.MaxAge < 0
	}
	return result
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
