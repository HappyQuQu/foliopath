package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/app"
)

const (
	exitOK       = 0
	exitFailure  = 1
	exitUsage    = 2
	usageMessage = `Usage:
  foliopath [serve] [--listen=IP:PORT]
  foliopath healthcheck
  foliopath version
  foliopath help

Environment:
  FOLIOPATH_LISTEN           Numeric IP and port (default 127.0.0.1:8080)
  FOLIOPATH_TRUSTED_PROXIES  Comma-separated trusted proxy IP CIDRs
`
)

var version = "dev"

type applicationRunner func(app.Input) error
type healthcheckRunner func() error

func main() {
	os.Exit(execute(
		os.Args[1:],
		os.Environ(),
		version,
		os.Stdout,
		os.Stderr,
		app.Run,
		checkReadiness,
	))
}

func execute(
	args []string,
	environ []string,
	buildVersion string,
	stdout io.Writer,
	stderr io.Writer,
	run applicationRunner,
	healthchecks ...healthcheckRunner,
) int {
	if len(args) == 1 {
		switch args[0] {
		case "healthcheck":
			if len(healthchecks) != 1 ||
				healthchecks[0] == nil ||
				healthchecks[0]() != nil {
				fmt.Fprintln(stderr, "foliopath: readiness check failed")
				return exitFailure
			}
			return exitOK
		case "version":
			fmt.Fprintf(stdout, "foliopath %s\n", normalizedVersion(buildVersion))
			return exitOK
		case "help", "-h", "--help":
			fmt.Fprint(stdout, usageMessage)
			return exitOK
		}
	}

	appArgs := args
	if len(args) > 0 {
		if args[0] == "serve" {
			appArgs = args[1:]
		} else if !strings.HasPrefix(args[0], "-") {
			fmt.Fprintf(stderr, "foliopath: unknown command %q\n", args[0])
			fmt.Fprint(stderr, usageMessage)
			return exitUsage
		}
	}

	if run == nil {
		fmt.Fprintln(stderr, "foliopath: startup failed")
		return exitFailure
	}
	if err := run(app.Input{
		Args:    slices.Clone(appArgs),
		Environ: slices.Clone(environ),
		Version: normalizedVersion(buildVersion),
	}); err != nil {
		// Detailed startup errors belong to the structured application logger.
		// The process boundary emits a stable message so paths, SQL, or secrets
		// cannot leak through an unclassified error.
		fmt.Fprintln(stderr, "foliopath: startup failed")
		return exitFailure
	}

	return exitOK
}

func normalizedVersion(buildVersion string) string {
	if buildVersion == "" {
		return "dev"
	}
	return buildVersion
}
