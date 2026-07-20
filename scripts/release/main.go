// Command release builds versioned Storyboard binaries for supported targets.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type target struct {
	goos   string
	goarch string
}

var supportedTargets = []target{
	{goos: "windows", goarch: "amd64"},
	{goos: "darwin", goarch: "amd64"},
	{goos: "darwin", goarch: "arm64"},
	{goos: "linux", goarch: "amd64"},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("release", flag.ContinueOnError)
	version := flags.String("version", "", "release version, for example v1.0.0")
	output := flags.String("output", "dist", "output directory")
	selected := flags.String("target", "", "build one target as os/arch (default: all)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*version) == "" {
		return errors.New("--version is required")
	}

	targets, err := selectTargets(*selected)
	if err != nil {
		return err
	}
	if err := runCommand("frontend", nil, "npm", "ci"); err != nil {
		return err
	}
	if err := runCommand("frontend", nil, "npm", "run", "build"); err != nil {
		return err
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	for _, target := range targets {
		artifact := filepath.Join(*output, artifactName(target))
		env := append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target.goos, "GOARCH="+target.goarch)
		ldflags := "-s -w -X main.version=" + *version
		if err := runCommand(".", env, "go", "build", "-trimpath", "-ldflags", ldflags, "-o", artifact, "./cmd/storyboard"); err != nil {
			return err
		}
		fmt.Println("built", artifact)
		if target.goos == runtime.GOOS && target.goarch == runtime.GOARCH {
			if err := verifyVersion(artifact, *version); err != nil {
				return err
			}
			fmt.Println("verified", artifact, "--version")
		}
	}
	return nil
}

func selectTargets(selected string) ([]target, error) {
	if selected == "" {
		return supportedTargets, nil
	}
	for _, candidate := range supportedTargets {
		if selected == candidate.goos+"/"+candidate.goarch {
			return []target{candidate}, nil
		}
	}
	return nil, fmt.Errorf("unsupported target %q", selected)
}

func artifactName(target target) string {
	name := "storyboard-" + target.goos + "-" + target.goarch
	if target.goos == "windows" {
		name += ".exe"
	}
	return name
}

func verifyVersion(artifact, version string) error {
	command := exec.Command(filepath.Clean(artifact), "--version")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("verify %s: %w", artifact, err)
	}
	want := "storyboard " + version
	if strings.TrimSpace(string(output)) != want {
		return fmt.Errorf("verify %s: got %q, want %q", artifact, bytes.TrimSpace(output), want)
	}
	return nil
}

func runCommand(directory string, env []string, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Dir = directory
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if env != nil {
		command.Env = env
	}
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
