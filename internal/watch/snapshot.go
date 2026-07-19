package watch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"storyboard/internal/openspec"
)

// Snapshot is an immutable parsed project plus hashes for its source files.
type Snapshot struct {
	Project openspec.Project
	Files   map[string]string
}

// Activity is one human-readable transition observed between snapshots.
type Activity struct {
	Message   string    `json:"message"`
	File      string    `json:"file"`
	Change    string    `json:"change,omitempty"`
	TaskID    string    `json:"taskId,omitempty"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

// Capture parses one project and hashes its OpenSpec markdown files.
func Capture(projectRoot string) (*Snapshot, error) {
	project, err := openspec.Discover(os.DirFS(projectRoot), projectRoot)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string)
	openspecRoot := filepath.Join(projectRoot, "openspec")
	err = filepath.WalkDir(openspecRoot, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(name) != ".md" {
			return nil
		}
		content, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(projectRoot, name)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		files[filepath.ToSlash(relative)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hashing OpenSpec files: %w", err)
	}
	return &Snapshot{Project: *project, Files: files}, nil
}

// Diff derives activity entries from two consecutive project snapshots.
func Diff(before, after *Snapshot, now time.Time) []Activity {
	if before == nil || after == nil {
		return nil
	}
	var activities []Activity
	taskFiles := make(map[string]bool)
	beforeTasks := taskStates(before.Project)
	afterTasks := taskStates(after.Project)
	for key, next := range afterTasks {
		previous, ok := beforeTasks[key]
		if !ok || previous.Checked == next.Checked {
			continue
		}
		action := "unchecked"
		if next.Checked {
			action = "checked"
		}
		file := taskFile(next.Change, next.Archived)
		taskFiles[file] = true
		activities = append(activities, Activity{
			Message:   "tasks.md · " + next.ID + " " + action,
			File:      "tasks.md",
			Change:    next.Change,
			TaskID:    next.ID,
			Action:    action,
			Timestamp: now,
		})
	}

	allFiles := make(map[string]struct{}, len(before.Files)+len(after.Files))
	for name := range before.Files {
		allFiles[name] = struct{}{}
	}
	for name := range after.Files {
		allFiles[name] = struct{}{}
	}
	for name := range allFiles {
		if before.Files[name] == after.Files[name] || taskFiles[name] {
			continue
		}
		action := "edited"
		if _, existed := before.Files[name]; !existed {
			action = "added"
		} else if _, exists := after.Files[name]; !exists {
			action = "removed"
		}
		label := filepath.Base(filepath.FromSlash(name))
		activities = append(activities, Activity{
			Message:   label + " " + action,
			File:      name,
			Change:    changeFromPath(name),
			Action:    action,
			Timestamp: now,
		})
	}
	sort.Slice(activities, func(i, j int) bool { return activities[i].Message < activities[j].Message })
	return activities
}

type taskState struct {
	ID       string
	Change   string
	Archived bool
	Checked  bool
}

func taskStates(project openspec.Project) map[string]taskState {
	states := make(map[string]taskState)
	for _, change := range project.Changes {
		for _, group := range change.Tasks.Groups {
			for _, task := range group.Tasks {
				id := task.ID
				if id == "" {
					id = fmt.Sprintf("line-%d", task.Line)
				}
				key := fmt.Sprintf("%t/%s/%s", change.Archived, change.Name, id)
				states[key] = taskState{ID: id, Change: change.Name, Archived: change.Archived, Checked: task.Checked}
			}
		}
	}
	return states
}

func taskFile(change string, archived bool) string {
	if archived {
		return filepath.ToSlash(filepath.Join("openspec", "changes", "archive", change, "tasks.md"))
	}
	return filepath.ToSlash(filepath.Join("openspec", "changes", change, "tasks.md"))
}

func changeFromPath(name string) string {
	parts := strings.Split(filepath.ToSlash(name), "/")
	if len(parts) < 4 || parts[0] != "openspec" || parts[1] != "changes" {
		return ""
	}
	if parts[2] == "archive" && len(parts) >= 5 {
		return parts[3]
	}
	return parts[2]
}
