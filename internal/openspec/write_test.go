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

func TestReplaceTaskTextLinePreservesSurroundingBytes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		line int
		text string
		want string
	}{
		{"unchecked LF", "intro\n- [ ] 1.1 Old text\nend\n", 2, "New text", "intro\n- [ ] 1.1 New text\nend\n"},
		{"checked CRLF", "\t-   [x]\t2.4\tOld text  \r\n", 1, "Rewritten", "\t-   [x]\t2.4\tRewritten  \r\n"},
		{"without id", "  - [ ] old text\n", 1, "new text", "  - [ ] new text\n"},
		{"empty replacement", "- [X] 1.2 old\t\n", 1, "", "- [X] 1.2 \t\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReplaceTaskTextLine([]byte(tt.in), tt.line, tt.text)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, []byte(tt.want)) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReplaceTaskTextLineRejectsStructureChanges(t *testing.T) {
	for _, text := range []string{"two\nlines", "two\rlines"} {
		if _, err := ReplaceTaskTextLine([]byte("- [ ] 1.1 old\n"), 1, text); !errors.Is(err, ErrInvalidTaskText) {
			t.Errorf("text %q: error = %v, want ErrInvalidTaskText", text, err)
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

func TestUpdateTaskTextFileAndConflict(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "demo")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(changeDir, "tasks.md")
	original := []byte("## Work\r\n\t- [x] 1.1 Keep this spacing  \r\n")
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	base := versionFor(original, info.ModTime())

	result, err := UpdateTaskTextFile(root, "demo", "1.1", "Edited description", base)
	if err != nil {
		t.Fatal(err)
	}
	if result.Task.Text != "Edited description" || !result.Task.Checked || result.Version.Hash == base.Hash {
		t.Errorf("result = %+v", result)
	}
	want := []byte("## Work\r\n\t- [x] 1.1 Edited description  \r\n")
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, want) {
		t.Errorf("file = %q, want %q", got, want)
	}
	if _, err := UpdateTaskTextFile(root, "demo", "1.1", "stale", base); !errors.Is(err, ErrConflict) {
		t.Errorf("stale edit error = %v, want ErrConflict", err)
	}
}

func TestSaveArtifactFileAndConflict(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "demo")
	if err := os.MkdirAll(filepath.Join(changeDir, "specs", "capability"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		relativePath string
		kind         string
		original     []byte
		updated      string
	}{
		{"proposal.md", "proposal", []byte("# Original\r\n\r\nKeep format.\r\n"), "# Edited proposal\r\n"},
		{"design.md", "design", []byte("# Design\n"), "# Edited design\n"},
		{"specs/capability/spec.md", "spec", []byte("# Spec\n"), "# Edited spec\n"},
	}
	var proposalBase FileVersion
	for _, tt := range tests {
		filePath := filepath.Join(changeDir, filepath.FromSlash(tt.relativePath))
		if err := os.WriteFile(filePath, tt.original, 0o640); err != nil {
			t.Fatal(err)
		}
		info, _ := os.Stat(filePath)
		base := versionFor(tt.original, info.ModTime())
		result, err := SaveArtifactFile(root, "demo", tt.relativePath, tt.updated, base)
		if err != nil {
			t.Fatal(err)
		}
		if result.Artifact.Kind != tt.kind || result.Artifact.Path != tt.relativePath || result.Artifact.Content != tt.updated || result.Artifact.Version.Hash == base.Hash {
			t.Errorf("%s result = %+v", tt.relativePath, result)
		}
		got, _ := os.ReadFile(filePath)
		if !bytes.Equal(got, []byte(tt.updated)) {
			t.Errorf("%s file = %q, want %q", tt.relativePath, got, tt.updated)
		}
		if tt.relativePath == "proposal.md" {
			proposalBase = base
		}
	}
	if _, err := SaveArtifactFile(root, "demo", "proposal.md", "stale", proposalBase); !errors.Is(err, ErrConflict) {
		t.Errorf("stale save error = %v, want ErrConflict", err)
	}
	for _, relativePath := range []string{"missing.md", "../outside.md"} {
		if _, err := SaveArtifactFile(root, "demo", relativePath, "blocked", FileVersion{}); !errors.Is(err, ErrArtifactNotFound) {
			t.Errorf("save %q error = %v, want ErrArtifactNotFound", relativePath, err)
		}
	}
}
