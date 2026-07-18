package openspec

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Discover parses an OpenSpec project. fsys is a filesystem rooted at the
// project folder (e.g. os.DirFS(path) in production, or an in-memory
// fstest.MapFS in tests); root is stored on the result only as a display
// label, since fs.FS values don't carry their own path.
//
// It returns ErrNotOpenSpecProject if fsys has no openspec/ directory, and
// never returns a partial *Project alongside an error.
func Discover(fsys fs.FS, root string) (*Project, error) {
	if info, err := fs.Stat(fsys, "openspec"); err != nil || !info.IsDir() {
		return nil, ErrNotOpenSpecProject
	}

	active, err := readChangeDirs(fsys, "openspec/changes", false)
	if err != nil {
		return nil, fmt.Errorf("reading active changes: %w", err)
	}

	archived, err := readChangeDirs(fsys, "openspec/changes/archive", true)
	if err != nil {
		return nil, fmt.Errorf("reading archived changes: %w", err)
	}

	changes := append(active, archived...)

	return &Project{
		Root:       root,
		SchemaName: readSchemaName(fsys, "openspec/config.yaml"),
		Changes:    changes,
	}, nil
}

// readChangeDirs lists the change directories directly under dir. When
// dir is the top-level changes directory, "archive" is skipped there and
// read separately by the caller with archived=true. A missing dir is not
// an error - it just means zero changes.
func readChangeDirs(fsys fs.FS, dir string, archived bool) ([]Change, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var changes []Change
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if dir == "openspec/changes" && entry.Name() == "archive" {
			continue
		}

		change, err := readChange(fsys, path.Join(dir, entry.Name()), entry.Name(), archived)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func readChange(fsys fs.FS, changePath, name string, archived bool) (Change, error) {
	artifacts := Artifacts{
		Proposal: fileExists(fsys, path.Join(changePath, "proposal.md")),
		Design:   fileExists(fsys, path.Join(changePath, "design.md")),
		Specs:    hasMarkdownFile(fsys, path.Join(changePath, "specs")),
		Tasks:    fileExists(fsys, path.Join(changePath, "tasks.md")),
	}

	var tasks TasksDoc
	if artifacts.Tasks {
		content, err := fs.ReadFile(fsys, path.Join(changePath, "tasks.md"))
		if err != nil {
			return Change{}, fmt.Errorf("reading %s/tasks.md: %w", changePath, err)
		}
		tasks = ParseTasksDoc(content)
	}

	return Change{
		Name:      name,
		Archived:  archived,
		Artifacts: artifacts,
		Tasks:     tasks,
		Status:    DeriveStatus(archived, artifacts, tasks),
	}, nil
}

func fileExists(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}

// hasMarkdownFile reports whether dir contains at least one .md file,
// searched recursively. A missing dir is treated as "no", not an error.
func hasMarkdownFile(fsys fs.FS, dir string) bool {
	found := false
	_ = fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // dir likely doesn't exist; nothing found
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

// readSchemaName leniently extracts the value of a top-level "schema:"
// line from config.yaml, ignoring comments. It returns "" if the file is
// missing or has no such line - this is not a hand-rolled YAML parser,
// just enough to read the one field Storyboard needs (design.md D3).
func readSchemaName(fsys fs.FS, configPath string) string {
	content, err := fs.ReadFile(fsys, configPath)
	if err != nil {
		return ""
	}

	for _, line := range splitLines(string(content)) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if after, ok := strings.CutPrefix(trimmed, "schema:"); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
