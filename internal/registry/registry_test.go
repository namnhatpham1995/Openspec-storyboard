package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func makeProject(t *testing.T, parent, name string) string {
	t.Helper()
	root := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(root, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestAddPersistsDeduplicatesAndRemoves(t *testing.T) {
	root := t.TempDir()
	projectPath := makeProject(t, root, "alpha")
	configPath := filepath.Join(root, "config", "config.json")
	store, err := Open(configPath)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.Add(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := store.Add(filepath.Join(projectPath, "."))
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != project.ID || len(store.List()) != 1 {
		t.Fatalf("duplicate = %+v, projects = %+v", duplicate, store.List())
	}

	reloaded, err := Open(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.List(); len(got) != 1 || got[0].Path != project.Path {
		t.Fatalf("reloaded projects = %+v", got)
	}
	if err := reloaded.Remove(project.ID); err != nil {
		t.Fatal(err)
	}
	if len(reloaded.List()) != 0 {
		t.Fatal("project was not removed")
	}
	if err := reloaded.Remove(project.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second remove error = %v, want ErrNotFound", err)
	}
}

func TestAddRejectsInvalidProject(t *testing.T) {
	store := NewMemory()
	for _, path := range []string{"", t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		if _, err := store.Add(path); !errors.Is(err, ErrInvalidProject) {
			t.Errorf("Add(%q) error = %v, want ErrInvalidProject", path, err)
		}
	}
}

func TestOpenBacksUpCorruptConfigAndResets(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.json")
	if err := os.WriteFile(configPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := Open(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.List()) != 0 {
		t.Fatalf("reset projects = %+v", store.List())
	}
	matches, err := filepath.Glob(configPath + ".corrupt-*")
	if err != nil || len(matches) != 1 {
		t.Fatalf("backup matches = %v, error = %v", matches, err)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var reset config
	if err := json.Unmarshal(content, &reset); err != nil {
		t.Fatalf("reset config is invalid: %v", err)
	}
}
