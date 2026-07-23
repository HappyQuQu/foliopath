package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/app"
)

func TestExecuteDelegatesDefaultServeInput(t *testing.T) {
	var received app.Input
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := execute(
		nil,
		[]string{"FOLIOPATH_TEST=value"},
		"test",
		&stdout,
		&stderr,
		func(input app.Input) error {
			received = input
			return nil
		},
	)

	if code != exitOK {
		t.Fatalf("execute() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if len(received.Args) != 0 {
		t.Fatalf("delegated args = %q, want empty", received.Args)
	}
	if want := []string{"FOLIOPATH_TEST=value"}; !reflect.DeepEqual(received.Environ, want) {
		t.Fatalf("delegated environment = %q, want %q", received.Environ, want)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("unexpected output: stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestExecuteDelegatesServeArguments(t *testing.T) {
	var received app.Input

	code := execute(
		[]string{"serve", "--listen=127.0.0.1:8080"},
		nil,
		"test",
		&bytes.Buffer{},
		&bytes.Buffer{},
		func(input app.Input) error {
			received = input
			return nil
		},
	)

	if code != exitOK {
		t.Fatalf("execute() code = %d, want %d", code, exitOK)
	}
	if want := []string{"--listen=127.0.0.1:8080"}; !reflect.DeepEqual(received.Args, want) {
		t.Fatalf("delegated args = %q, want %q", received.Args, want)
	}
}

func TestExecuteDelegatesImplicitServeArguments(t *testing.T) {
	var received app.Input

	code := execute(
		[]string{"--listen=127.0.0.1:8080"},
		nil,
		"test",
		&bytes.Buffer{},
		&bytes.Buffer{},
		func(input app.Input) error {
			received = input
			return nil
		},
	)

	if code != exitOK {
		t.Fatalf("execute() code = %d, want %d", code, exitOK)
	}
	if want := []string{"--listen=127.0.0.1:8080"}; !reflect.DeepEqual(received.Args, want) {
		t.Fatalf("delegated args = %q, want %q", received.Args, want)
	}
}

func TestExecuteVersionDoesNotStartApplication(t *testing.T) {
	var stdout bytes.Buffer
	called := false

	code := execute(
		[]string{"version"},
		nil,
		"v1.2.3",
		&stdout,
		&bytes.Buffer{},
		func(app.Input) error {
			called = true
			return nil
		},
	)

	if code != exitOK {
		t.Fatalf("execute() code = %d, want %d", code, exitOK)
	}
	if got, want := stdout.String(), "foliopath v1.2.3\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
	if called {
		t.Fatal("version command started the application")
	}
}

func TestExecuteHelpDoesNotStartApplication(t *testing.T) {
	for _, command := range []string{"help", "-h", "--help"} {
		t.Run(command, func(t *testing.T) {
			var stdout bytes.Buffer
			called := false
			code := execute(
				[]string{command},
				nil,
				"test",
				&stdout,
				&bytes.Buffer{},
				func(app.Input) error {
					called = true
					return nil
				},
			)

			if code != exitOK {
				t.Fatalf("execute() code = %d, want %d", code, exitOK)
			}
			if !strings.Contains(stdout.String(), "foliopath [serve]") {
				t.Fatalf("help output missing usage: %q", stdout.String())
			}
			if !strings.Contains(stdout.String(), "FOLIOPATH_LISTEN") {
				t.Fatalf("help output missing listen environment: %q", stdout.String())
			}
			if called {
				t.Fatal("help command started the application")
			}
		})
	}
}

func TestExecuteRejectsUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer
	called := false

	code := execute(
		[]string{"scan"},
		nil,
		"test",
		&bytes.Buffer{},
		&stderr,
		func(app.Input) error {
			called = true
			return nil
		},
	)

	if code != exitUsage {
		t.Fatalf("execute() code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), `unknown command "scan"`) {
		t.Fatalf("stderr = %q, want unknown-command message", stderr.String())
	}
	if called {
		t.Fatal("unknown command started the application")
	}
}

func TestExecuteMapsStartupErrorWithoutLeakingDetails(t *testing.T) {
	var stderr bytes.Buffer

	code := execute(
		nil,
		nil,
		"test",
		&bytes.Buffer{},
		&stderr,
		func(app.Input) error {
			return errors.New("database /secret/path failed")
		},
	)

	if code != exitFailure {
		t.Fatalf("execute() code = %d, want %d", code, exitFailure)
	}
	if got, want := stderr.String(), "foliopath: startup failed\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if strings.Contains(stderr.String(), "/secret/path") {
		t.Fatalf("stderr leaked startup detail: %q", stderr.String())
	}
}

func TestNormalizedVersionDefaultsToDev(t *testing.T) {
	if got := normalizedVersion(""); got != "dev" {
		t.Fatalf("normalizedVersion(\"\") = %q, want dev", got)
	}
}
