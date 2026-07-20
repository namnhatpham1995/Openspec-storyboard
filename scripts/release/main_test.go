package main

import "testing"

func TestArtifactName(t *testing.T) {
	tests := []struct {
		target target
		want   string
	}{
		{target{goos: "windows", goarch: "amd64"}, "storyboard-windows-amd64.exe"},
		{target{goos: "darwin", goarch: "arm64"}, "storyboard-darwin-arm64"},
		{target{goos: "linux", goarch: "amd64"}, "storyboard-linux-amd64"},
	}
	for _, tt := range tests {
		if got := artifactName(tt.target); got != tt.want {
			t.Errorf("artifactName(%+v) = %q, want %q", tt.target, got, tt.want)
		}
	}
}

func TestSelectTargets(t *testing.T) {
	selected, err := selectTargets("darwin/amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0].goos != "darwin" || selected[0].goarch != "amd64" {
		t.Fatalf("selectTargets() = %+v", selected)
	}
	if _, err := selectTargets("plan9/amd64"); err == nil {
		t.Fatal("selectTargets() accepted an unsupported target")
	}
}
