package openspec

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestToggleCheckboxLinePreservesEveryOtherByte(t *testing.T) {
	tests := []struct {
		name string
		in   string
		line int
		want string
	}{
		{"unchecked LF", "intro\n- [ ] 1.1 Do it\nend\n", 2, "intro\n- [x] 1.1 Do it\nend\n"},
		{"checked LF", "- [x] 1.1 Done\n", 1, "- [ ] 1.1 Done\n"},
		{"uppercase checked", "- [X] 1.1 Done", 1, "- [ ] 1.1 Done"},
		{"CRLF", "intro\r\n- [ ] 1.1 Do it\r\nend\r\n", 2, "intro\r\n- [x] 1.1 Do it\r\nend\r\n"},
		{"extra spacing", "-   [ ] 1.1 Do it  \n", 1, "-   [x] 1.1 Do it  \n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToggleCheckboxLine([]byte(tt.in), tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, []byte(tt.want)) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToggleCheckboxLineRejectsInvalidTarget(t *testing.T) {
	for _, line := range []int{-1, 0, 1, 3} {
		_, err := ToggleCheckboxLine([]byte("prose\n- [ ] 1.1 task\n"), line)
		if !errors.Is(err, ErrInvalidTaskLine) {
			t.Errorf("line %d: error = %v", line, err)
		}
	}
}

func TestToggleTaskFileAndConflict(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "demo")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tasksPath := filepath.Join(changeDir, "tasks.md")
	original := []byte("## 1. Work\r\n\r\n- [ ] 1.1 Keep formatting  \r\n")
	if err := os.WriteFile(tasksPath, original, 0o640); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(tasksPath)
	base := versionFor(original, info.ModTime())

	result, err := ToggleTaskFile(root, "demo", "1.1", base)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Task.Checked || result.Version.Hash == base.Hash {
		t.Errorf("result = %+v", result)
	}
	got, _ := os.ReadFile(tasksPath)
	want := []byte("## 1. Work\r\n\r\n- [x] 1.1 Keep formatting  \r\n")
	if !bytes.Equal(got, want) {
		t.Errorf("file = %q, want %q", got, want)
	}
	if info, _ := os.Stat(tasksPath); runtime.GOOS != "windows" && info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640", info.Mode().Perm())
	}

	if _, err := ToggleTaskFile(root, "demo", "1.1", base); !errors.Is(err, ErrConflict) {
		t.Errorf("stale toggle error = %v, want ErrConflict", err)
	}
}

func TestToggleTaskFileRejectsMissingOrDuplicateTask(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "demo")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("- [ ] 1.1 First\n- [ ] 1.1 Duplicate\n")
	path := filepath.Join(changeDir, "tasks.md")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	base := versionFor(content, info.ModTime())

	for _, id := range []string{"1.1", "9.9"} {
		if _, err := ToggleTaskFile(root, "demo", id, base); !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("id %q: error = %v, want ErrTaskNotFound", id, err)
		}
	}
}
