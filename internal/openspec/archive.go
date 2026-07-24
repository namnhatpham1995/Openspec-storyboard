package openspec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var archiveNow = time.Now

// ArchiveChange moves an active change directory under changes/archive after
// verifying that tasks.md still matches the caller's base version.
func ArchiveChange(projectRoot, changeName string, base FileVersion) (*ArchiveResult, error) {
	changeDir, err := resolveChangeDir(projectRoot, changeName)
	if err != nil {
		return nil, err
	}
	activeDir := filepath.Join(projectRoot, "openspec", "changes", changeName)
	if filepath.Clean(changeDir) != filepath.Clean(activeDir) {
		return nil, ErrChangeNotFound
	}

	tasksPath := filepath.Join(changeDir, "tasks.md")
	content, info, err := readVersionedFile(tasksPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrChangeNotFound
		}
		return nil, fmt.Errorf("reading tasks.md: %w", err)
	}
	if !sameVersion(versionFor(content, info.ModTime()), base) {
		return nil, ErrConflict
	}

	archivedName := archiveNow().Format("2006-01-02") + "-" + changeName
	archiveDir := filepath.Join(projectRoot, "openspec", "changes", "archive")
	destination := filepath.Join(archiveDir, archivedName)
	if _, err := os.Stat(destination); err == nil {
		return nil, fmt.Errorf("%w: a change named %q already exists in the archive; rename one and try again", ErrArchiveNameConflict, archivedName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("checking archive destination: %w", err)
	}
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating archive directory: %w", err)
	}
	if err := os.Rename(changeDir, destination); err != nil {
		return nil, fmt.Errorf("moving change to archive: %w", err)
	}
	return &ArchiveResult{
		Name: archivedName,
		Path: filepath.ToSlash(filepath.Join("openspec", "changes", "archive", archivedName)),
	}, nil
}
