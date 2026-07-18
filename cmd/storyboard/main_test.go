package main

import (
	"bytes"
	"strings"
	"testing"
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
			name:       "no args prints not-implemented notice to stderr",
			args:       nil,
			wantCode:   0,
			wantStdout: "",
			wantStderr: "storyboard dev - server not implemented yet",
		},
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
