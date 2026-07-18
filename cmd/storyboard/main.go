// Command storyboard runs the Storyboard server: a local, portable board
// for viewing and managing OpenSpec projects.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// version is stamped at release build time via -ldflags (see design D9 / task 9.3).
// It stays "dev" for local `go run` / `go build` invocations.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run contains main's logic without calling os.Exit, so it can be unit
// tested directly. It returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("storyboard", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "storyboard", version)
		return 0
	}

	fmt.Fprintln(stderr, "storyboard", version, "- server not implemented yet")
	return 0
}
