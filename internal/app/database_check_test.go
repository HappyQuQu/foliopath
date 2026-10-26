package app

import (
	"context"
	"runtime"
	"testing"
)

func TestDatabaseCheckRejectsHostKernelAndDoesNotCompose(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("host-kernel boundary is verified on non-Linux; live Linux check runs in Docker")
	}
	if err := run(context.Background(), Input{Args: []string{"database-check"}, Environ: []string{"FOLIOPATH_UNKNOWN=must-not-compose"}}); err != errDatabaseCheckPlatform {
		t.Fatalf("got %v", err)
	}
}
