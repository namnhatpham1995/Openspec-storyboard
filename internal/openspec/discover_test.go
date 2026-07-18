package openspec

import (
	"errors"
	"os"
	"testing"
)

func TestDiscover_RealProject(t *testing.T) {
	fsys := os.DirFS("testdata/real-project")

	project, err := Discover(fsys, "testdata/real-project")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if project.SchemaName != "spec-driven" {
		t.Errorf("SchemaName = %q, want %q", project.SchemaName, "spec-driven")
	}
	if len(project.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(project.Changes))
	}

	change := project.Changes[0]
	if change.Name != "build-storyboard-v1" {
		t.Errorf("Name = %q, want %q", change.Name, "build-storyboard-v1")
	}
	if change.Archived {
		t.Error("Archived = true, want false")
	}
	want := Artifacts{Proposal: true, Design: true, Specs: true, Tasks: true}
	if change.Artifacts != want {
		t.Errorf("Artifacts = %+v, want %+v", change.Artifacts, want)
	}
	// tasks.md has group 1 fully checked and later groups unchecked, so
	// the derived status must be in-progress, not complete or draft.
	if change.Status != StatusInProgress {
		t.Errorf("Status = %v, want %v", change.Status, StatusInProgress)
	}
	if !change.Tasks.Parseable {
		t.Error("Tasks.Parseable = false, want true")
	}
	if len(change.Tasks.Groups) == 0 {
		t.Error("Tasks.Groups is empty, want at least one group")
	}
}

func TestDiscover_EmptyChange(t *testing.T) {
	fsys := os.DirFS("testdata/empty-change")

	project, err := Discover(fsys, "testdata/empty-change")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(project.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(project.Changes))
	}
	change := project.Changes[0]

	want := Artifacts{Proposal: true}
	if change.Artifacts != want {
		t.Errorf("Artifacts = %+v, want %+v", change.Artifacts, want)
	}
	if change.Status != StatusDraft {
		t.Errorf("Status = %v, want %v", change.Status, StatusDraft)
	}
}

func TestDiscover_WithArchive(t *testing.T) {
	fsys := os.DirFS("testdata/with-archive")

	project, err := Discover(fsys, "testdata/with-archive")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(project.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2", len(project.Changes))
	}

	byName := make(map[string]Change, len(project.Changes))
	for _, c := range project.Changes {
		byName[c.Name] = c
	}

	active, ok := byName["active-one"]
	if !ok {
		t.Fatal("active-one not found in Changes")
	}
	if active.Archived {
		t.Error("active-one: Archived = true, want false")
	}
	if active.Status != StatusDraft {
		t.Errorf("active-one: Status = %v, want %v", active.Status, StatusDraft)
	}

	old, ok := byName["old-feature"]
	if !ok {
		t.Fatal("old-feature not found in Changes")
	}
	if !old.Archived {
		t.Error("old-feature: Archived = false, want true")
	}
	// Archived must win even though every task is checked.
	if old.Status != StatusArchived {
		t.Errorf("old-feature: Status = %v, want %v", old.Status, StatusArchived)
	}
}

func TestDiscover_MalformedTasks(t *testing.T) {
	fsys := os.DirFS("testdata/malformed-tasks")

	project, err := Discover(fsys, "testdata/malformed-tasks")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(project.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(project.Changes))
	}

	change := project.Changes[0]
	if !change.Artifacts.Tasks {
		t.Error("Artifacts.Tasks = false, want true (the file exists, it's just unparseable)")
	}
	if change.Tasks.Parseable {
		t.Error("Tasks.Parseable = true, want false")
	}
	total, _ := countTasks(change.Tasks)
	if total != 0 {
		t.Errorf("countTasks = %d, want 0", total)
	}
	// A present-but-empty tasks.md still counts as "beyond proposal".
	if change.Status != StatusInProgress {
		t.Errorf("Status = %v, want %v", change.Status, StatusInProgress)
	}
}

func TestDiscover_NoOpenSpecDir(t *testing.T) {
	fsys := os.DirFS("testdata/no-openspec")

	project, err := Discover(fsys, "testdata/no-openspec")
	if !errors.Is(err, ErrNotOpenSpecProject) {
		t.Fatalf("err = %v, want ErrNotOpenSpecProject", err)
	}
	if project != nil {
		t.Errorf("project = %+v, want nil", project)
	}
}

func TestDiscover_SchemaLineAfterComments(t *testing.T) {
	fsys := os.DirFS("testdata/schema-with-comments")

	project, err := Discover(fsys, "testdata/schema-with-comments")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if project.SchemaName != "spec-driven" {
		t.Errorf("SchemaName = %q, want %q", project.SchemaName, "spec-driven")
	}
	if len(project.Changes) != 0 {
		t.Errorf("len(Changes) = %d, want 0 (no changes/ dir at all)", len(project.Changes))
	}
}

func TestDiscover_NoSchemaLine(t *testing.T) {
	fsys := os.DirFS("testdata/no-schema-line")

	project, err := Discover(fsys, "testdata/no-schema-line")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if project.SchemaName != "" {
		t.Errorf("SchemaName = %q, want empty", project.SchemaName)
	}
}

func TestDiscover_OpenSpecIsAFile(t *testing.T) {
	fsys := os.DirFS("testdata/openspec-is-file")

	project, err := Discover(fsys, "testdata/openspec-is-file")
	if !errors.Is(err, ErrNotOpenSpecProject) {
		t.Fatalf("err = %v, want ErrNotOpenSpecProject", err)
	}
	if project != nil {
		t.Errorf("project = %+v, want nil", project)
	}
}

func TestDiscover_CRLFFile(t *testing.T) {
	fsys := os.DirFS("testdata/crlf")

	project, err := Discover(fsys, "testdata/crlf")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(project.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(project.Changes))
	}

	change := project.Changes[0]
	total, checked := countTasks(change.Tasks)
	if total != 2 || checked != 1 {
		t.Errorf("countTasks = (%d, %d), want (2, 1)", total, checked)
	}
	if change.Status != StatusInProgress {
		t.Errorf("Status = %v, want %v", change.Status, StatusInProgress)
	}
}
