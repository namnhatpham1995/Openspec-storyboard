// Command storyboard runs the Storyboard server: a local, portable board
// for viewing and managing OpenSpec projects.
package main

import (
	"flag"
	"fmt"
	"os"
)

// version is stamped at release build time via -ldflags (see design D9 / task 9.3).
// It stays "dev" for local `go run` / `go build` invocations.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("storyboard", version)
		return
	}

	fmt.Fprintln(os.Stderr, "storyboard", version, "- server not implemented yet")
}
