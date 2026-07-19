package watch

import (
	"testing"
	"time"

	"storyboard/internal/openspec"
)

func TestDiffCheckboxAndArtifacts(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	before := &Snapshot{
		Project: projectWithTask(false),
		Files: map[string]string{
			"openspec/changes/demo/tasks.md":    "old-tasks",
			"openspec/changes/demo/proposal.md": "old-proposal",
		},
	}
	after := &Snapshot{
		Project: projectWithTask(true),
		Files: map[string]string{
			"openspec/changes/demo/tasks.md":    "new-tasks",
			"openspec/changes/demo/proposal.md": "new-proposal",
			"openspec/changes/demo/design.md":   "new-design",
		},
	}

	activities := Diff(before, after, now)
	want := map[string]bool{
		"tasks.md · 1.1 checked": true,
		"proposal.md edited":     true,
		"design.md added":        true,
	}
	if len(activities) != len(want) {
		t.Fatalf("activities = %+v", activities)
	}
	for _, activity := range activities {
		if !want[activity.Message] {
			t.Errorf("unexpected activity %q", activity.Message)
		}
		if !activity.Timestamp.Equal(now) {
			t.Errorf("timestamp = %v", activity.Timestamp)
		}
	}
}

func projectWithTask(checked bool) openspec.Project {
	return openspec.Project{Changes: []openspec.Change{{
		Name: "demo",
		Tasks: openspec.TasksDoc{Groups: []openspec.TaskGroup{{Tasks: []openspec.Task{{
			ID: "1.1", Checked: checked, Line: 2,
		}}}}},
	}}}
}
