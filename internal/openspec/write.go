package openspec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var checkboxPrefixPattern = regexp.MustCompile(`^[ \t]*-[ \t]+\[([ xX])\]`)
var taskTextPrefixPattern = regexp.MustCompile(`^[ \t]*-[ \t]+\[[ xX]\][ \t]+(?:\d+(?:\.\d+)*[ \t]+)?`)

// ToggleCheckboxLine flips the checkbox on a 1-based source line while
// preserving every other byte, including CRLF/LF endings and trailing data.
func ToggleCheckboxLine(content []byte, line int) ([]byte, error) {
	start, end, ok := lineRange(content, line)
	if !ok {
		return nil, ErrInvalidTaskLine
	}

	match := checkboxPrefixPattern.FindSubmatchIndex(content[start:end])
	if match == nil {
		return nil, ErrInvalidTaskLine
	}

	stateIndex := start + match[2]
	updated := bytes.Clone(content)
	if updated[stateIndex] == ' ' {
		updated[stateIndex] = 'x'
	} else {
		updated[stateIndex] = ' '
	}
	return updated, nil
}

// ReplaceTaskTextLine replaces only the description portion of one checkbox
// line. Indentation, checkbox state, id prefix, spacing, trailing whitespace,
// line endings, and every other source byte are preserved.
func ReplaceTaskTextLine(content []byte, line int, text string) ([]byte, error) {
	if bytes.ContainsAny([]byte(text), "\r\n") {
		return nil, ErrInvalidTaskText
	}
	start, end, ok := lineRange(content, line)
	if !ok {
		return nil, ErrInvalidTaskLine
	}
	lineBytes := content[start:end]
	prefix := taskTextPrefixPattern.FindIndex(lineBytes)
	if prefix == nil {
		return nil, ErrInvalidTaskLine
	}

	suffixStart := len(lineBytes)
	for suffixStart > prefix[1] && (lineBytes[suffixStart-1] == ' ' || lineBytes[suffixStart-1] == '\t') {
		suffixStart--
	}
	updated := make([]byte, 0, len(content)-suffixStart+prefix[1]+len(text))
	updated = append(updated, content[:start+prefix[1]]...)
	updated = append(updated, text...)
	updated = append(updated, content[start+suffixStart:]...)
	return updated, nil
}

// ToggleTaskFile verifies the caller's base version, flips one task checkbox,
// and atomically replaces tasks.md. Disk state always wins on conflict.
func ToggleTaskFile(projectRoot, changeName, taskID string, base FileVersion) (*ToggleResult, error) {
	changeDir, err := resolveChangeDir(projectRoot, changeName)
	if err != nil {
		return nil, err
	}
	tasksPath := filepath.Join(changeDir, "tasks.md")

	content, info, err := readVersionedFile(tasksPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("reading tasks.md: %w", err)
	}
	currentVersion := versionFor(content, info.ModTime())
	if !sameVersion(currentVersion, base) {
		return nil, ErrConflict
	}

	doc := ParseTasksDoc(content)
	task, found := findUniqueTask(doc, taskID)
	if !found {
		return nil, ErrTaskNotFound
	}
	updated, err := ToggleCheckboxLine(content, task.Line)
	if err != nil {
		return nil, err
	}
	if err := atomicReplace(tasksPath, updated, info.Mode(), base); err != nil {
		return nil, fmt.Errorf("writing tasks.md: %w", err)
	}

	newInfo, err := os.Stat(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("reading updated tasks.md version: %w", err)
	}
	task.Checked = !task.Checked
	return &ToggleResult{Task: task, Version: versionFor(updated, newInfo.ModTime())}, nil
}

// UpdateTaskTextFile verifies the caller's version and atomically replaces
// only one task's description. Disk state always wins on conflict.
func UpdateTaskTextFile(projectRoot, changeName, taskID, text string, base FileVersion) (*TaskTextResult, error) {
	changeDir, err := resolveChangeDir(projectRoot, changeName)
	if err != nil {
		return nil, err
	}
	tasksPath := filepath.Join(changeDir, "tasks.md")
	content, info, err := readVersionedFile(tasksPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("reading tasks.md: %w", err)
	}
	if !sameVersion(versionFor(content, info.ModTime()), base) {
		return nil, ErrConflict
	}

	task, found := findUniqueTask(ParseTasksDoc(content), taskID)
	if !found {
		return nil, ErrTaskNotFound
	}
	updated, err := ReplaceTaskTextLine(content, task.Line, text)
	if err != nil {
		return nil, err
	}
	if err := atomicReplace(tasksPath, updated, info.Mode(), base); err != nil {
		return nil, fmt.Errorf("writing tasks.md: %w", err)
	}
	newInfo, err := os.Stat(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("reading updated tasks.md version: %w", err)
	}
	task.Text = text
	return &TaskTextResult{Task: task, Version: versionFor(updated, newInfo.ModTime())}, nil
}

// SaveProposalFile replaces proposal.md verbatim after checking its base
// version and uses the same atomic, disk-wins write path as task mutations.
func SaveProposalFile(projectRoot, changeName, content string, base FileVersion) (*ArtifactWriteResult, error) {
	changeDir, err := resolveChangeDir(projectRoot, changeName)
	if err != nil {
		return nil, err
	}
	proposalPath := filepath.Join(changeDir, "proposal.md")
	current, info, err := readVersionedFile(proposalPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrArtifactNotFound
		}
		return nil, fmt.Errorf("reading proposal.md: %w", err)
	}
	if !sameVersion(versionFor(current, info.ModTime()), base) {
		return nil, ErrConflict
	}
	updated := []byte(content)
	if err := atomicReplace(proposalPath, updated, info.Mode(), base); err != nil {
		return nil, fmt.Errorf("writing proposal.md: %w", err)
	}
	newInfo, err := os.Stat(proposalPath)
	if err != nil {
		return nil, fmt.Errorf("reading updated proposal.md version: %w", err)
	}
	return &ArtifactWriteResult{Artifact: Artifact{
		Kind: "proposal", Path: "proposal.md", Content: content,
		Version: versionFor(updated, newInfo.ModTime()),
	}}, nil
}

func resolveChangeDir(projectRoot, changeName string) (string, error) {
	if changeName == "" || filepath.Base(changeName) != changeName || changeName == "." {
		return "", ErrChangeNotFound
	}
	for _, relative := range []string{
		filepath.Join("openspec", "changes", changeName),
		filepath.Join("openspec", "changes", "archive", changeName),
	} {
		candidate := filepath.Join(projectRoot, relative)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", ErrChangeNotFound
}

func findUniqueTask(doc TasksDoc, taskID string) (Task, bool) {
	var match Task
	found := false
	for _, group := range doc.Groups {
		for _, task := range group.Tasks {
			if task.ID != taskID {
				continue
			}
			if found {
				return Task{}, false
			}
			match = task
			found = true
		}
	}
	return match, found
}

func lineRange(content []byte, wanted int) (start, end int, ok bool) {
	if wanted < 1 {
		return 0, 0, false
	}
	start = 0
	line := 1
	for start <= len(content) {
		newline := bytes.IndexByte(content[start:], '\n')
		if newline < 0 {
			end = len(content)
		} else {
			end = start + newline
		}
		if end > start && content[end-1] == '\r' {
			end--
		}
		if line == wanted {
			return start, end, true
		}
		if newline < 0 {
			break
		}
		start += newline + 1
		line++
	}
	return 0, 0, false
}

func readVersionedFile(name string) ([]byte, os.FileInfo, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	content, err := io.ReadAll(file)
	return content, info, err
}

func versionFor(content []byte, modTime time.Time) FileVersion {
	sum := sha256.Sum256(content)
	return FileVersion{ModTime: modTime, Hash: hex.EncodeToString(sum[:])}
}

func sameVersion(current, base FileVersion) bool {
	return current.Hash == base.Hash && current.ModTime.Equal(base.ModTime)
}

func atomicReplace(name string, content []byte, mode os.FileMode, base FileVersion) (err error) {
	dir := filepath.Dir(name)
	temp, err := os.CreateTemp(dir, ".storyboard-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() {
		_ = temp.Close()
		if err != nil {
			_ = os.Remove(tempName)
		}
	}()

	if err = temp.Chmod(mode.Perm()); err != nil {
		return err
	}
	if _, err = temp.Write(content); err != nil {
		return err
	}
	if err = temp.Sync(); err != nil {
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}

	// Recheck after preparing the temp file so an editor save that landed
	// during this write does not get silently replaced.
	current, currentInfo, err := readVersionedFile(name)
	if err != nil {
		return err
	}
	if !sameVersion(versionFor(current, currentInfo.ModTime()), base) {
		return ErrConflict
	}
	return os.Rename(tempName, name)
}
