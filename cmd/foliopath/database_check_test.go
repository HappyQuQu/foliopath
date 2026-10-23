package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/app"
)

func TestDatabaseCheckOnlyReportsSuccessAfterCompletion(t *testing.T) {
	for _, fail := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		code := execute([]string{"database-check"}, nil, "test", &stdout, &stderr, func(input app.Input) error {
			if stdout.Len() != 0 {
				t.Fatal("reported success before check")
			}
			if len(input.Args) != 1 || input.Args[0] != "database-check" {
				t.Fatal("check not delegated")
			}
			if fail {
				return errors.New("private SQL /app/data secret")
			}
			return nil
		})
		if fail {
			if code != exitFailure || stdout.Len() != 0 || stderr.String() != "foliopath: database check failed\n" {
				t.Fatalf("failure: %d %q %q", code, stdout.String(), stderr.String())
			}
		} else {
			if code != exitOK || stdout.String() != "database check passed\n" || stderr.Len() != 0 {
				t.Fatalf("success: %d %q %q", code, stdout.String(), stderr.String())
			}
		}
	}
	called := false
	if code := execute([]string{"database-check", "/arbitrary/path"}, nil, "test", &bytes.Buffer{}, &bytes.Buffer{}, func(app.Input) error { called = true; return nil }); code != exitUsage || called {
		t.Fatal("path argument accepted")
	}
}
