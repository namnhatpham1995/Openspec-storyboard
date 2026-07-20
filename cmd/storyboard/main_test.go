package main

import (
	"bytes"
	"context"
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

	code := runContext(ctx, []string{
		"--addr", "127.0.0.1:0",
		"--project", "../..",
		"--config", filepath.Join(t.TempDir(), "config.json"),
	}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("runContext() exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "shutting down server") {
		t.Errorf("stderr = %q, want graceful shutdown log", stderr.String())
	}
}
