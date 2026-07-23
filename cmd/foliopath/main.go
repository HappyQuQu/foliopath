package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/HappyQuQu/foliopath/internal/app"
)

const (
	exitOK       = 0
	exitFailure  = 1
	exitUsage    = 2
	usageMessage = `Usage:
  foliopath [serve] [options]
  foliopath version
  foliopath help
`
)

var version = "dev"

type applicationRunner func(app.Input) error

func main() {
	os.Exit(execute(os.Args[1:], os.Environ(), version, os.Stdout, os.Stderr, app.Run))
}

func execute(
	args []string,
	environ []string,
	buildVersion string,
	stdout io.Writer,
	stderr io.Writer,
	run applicationRunner,
) int {
	if len(args) == 1 {
		switch args[0] {
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
		if args[0] != "serve" {
			fmt.Fprintf(stderr, "foliopath: unknown command %q\n", args[0])
			fmt.Fprint(stderr, usageMessage)
			return exitUsage
		}
		appArgs = args[1:]
	}

	if run == nil {
		fmt.Fprintln(stderr, "foliopath: startup failed")
		return exitFailure
	}
	if err := run(app.Input{
		Args:    slices.Clone(appArgs),
		Environ: slices.Clone(environ),
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
