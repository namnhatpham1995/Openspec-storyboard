package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "--version prints version to stdout",
			args:       []string{"--version"},
			wantCode:   0,
			wantStdout: "storyboard dev",
			wantStderr: "",
		},
		{
			name:     "unknown flag fails and does not print version or notice",
			args:     []string{"--nope"},
			wantCode: 2,
		},
		{
			name:       "invalid port fails",
			args:       []string{"--port", "65536"},
			wantCode:   2,
			wantStderr: "--port must be between 0 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			got := run(tt.args, &stdout, &stderr)

			if got != tt.wantCode {
				t.Errorf("run() exit code = %d, want %d", got, tt.wantCode)
			}
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestRunContextGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	defer cancel()
	var stdout, stderr bytes.Buffer
	projectRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(projectRoot, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}

	code := runContext(ctx, []string{
		"--port", "0",
		"--no-open",
		"--project", projectRoot,
		"--config", filepath.Join(t.TempDir(), "config.json"),
	}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("runContext() exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "shutting down server") {
		t.Errorf("stderr = %q, want graceful shutdown log", stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "http://127.0.0.1:") {
		t.Errorf("stdout = %q, want selected loopback URL", stdout.String())
	}
}

func TestBrowserCommand(t *testing.T) {
	url := "http://127.0.0.1:12345"
	tests := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{goos: "windows", wantName: "rundll32", wantArgs: []string{"url.dll,FileProtocolHandler", url}},
		{goos: "darwin", wantName: "open", wantArgs: []string{url}},
		{goos: "linux", wantName: "xdg-open", wantArgs: []string{url}},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			name, args := browserCommand(tt.goos, url)
			if name != tt.wantName || strings.Join(args, "\x00") != strings.Join(tt.wantArgs, "\x00") {
				t.Errorf("browserCommand() = %q %q, want %q %q", name, args, tt.wantName, tt.wantArgs)
			}
		})
	}
}
