package openspec

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArchiveChange(t *testing.T) {
	archiveTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	previousNow := archiveNow
	archiveNow = func() time.Time { return archiveTime }
	t.Cleanup(func() { archiveNow = previousNow })

	tests := []struct {
		name      string
		changeDir string
		prepare   func(t *testing.T, root string, base FileVersion)
		wantErr   error
	}{
		{name: "successful move"},
		{
			name: "stale version conflict",
			prepare: func(t *testing.T, root string, _ FileVersion) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, "openspec", "changes", "demo", "tasks.md"), []byte("- [x] 1.1 Updated\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: ErrConflict,
		},
		{
			name: "destination name collision",
			prepare: func(t *testing.T, root string, _ FileVersion) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(root, "openspec", "changes", "archive", "2026-07-24-demo"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: ErrArchiveNameConflict,
		},
		{name: "missing change", changeDir: "missing", wantErr: ErrChangeNotFound},
		{name: "already archived", changeDir: filepath.Join("archive", "demo"), wantErr: ErrChangeNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			changeDir := filepath.Join(root, "openspec", "changes", tt.changeDir)
			if tt.changeDir == "" {
				changeDir = filepath.Join(root, "openspec", "changes", "demo")
			}
			if tt.changeDir != "missing" {
				if err := os.MkdirAll(changeDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [x] 1.1 Done\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			tasksPath := filepath.Join(root, "openspec", "changes", "demo", "tasks.md")
			if tt.changeDir == filepath.Join("archive", "demo") {
				tasksPath = filepath.Join(root, "openspec", "changes", "archive", "demo", "tasks.md")
			}
			content, _ := os.ReadFile(tasksPath)
			info, _ := os.Stat(tasksPath)
			base := FileVersion{}
			if info != nil {
				base = versionFor(content, info.ModTime())
			}
			if tt.prepare != nil {
				tt.prepare(t, root, base)
			}

			result, err := ArchiveChange(root, "demo", base)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Name != "2026-07-24-demo" || result.Path != "openspec/changes/archive/2026-07-24-demo" {
				t.Errorf("result = %+v", result)
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(result.Path), "tasks.md")); err != nil {
				t.Errorf("archived tasks.md missing: %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, "openspec", "changes", "demo")); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("active change remains: %v", err)
			}
		})
	}
}
