package openspec

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
)

// ReadChangeDetail loads one active or archived change and all of its raw
// markdown artifacts. name must be a single directory name, never a path.
func ReadChangeDetail(fsys fs.FS, name string) (*ChangeDetail, error) {
	if !fs.ValidPath(name) || path.Base(name) != name || name == "." {
		return nil, ErrChangeNotFound
	}

	changePath := path.Join("openspec/changes", name)
	archived := false
	if info, err := fs.Stat(fsys, changePath); err != nil || !info.IsDir() {
		changePath = path.Join("openspec/changes/archive", name)
		archived = true
		if info, err = fs.Stat(fsys, changePath); err != nil || !info.IsDir() {
			return nil, ErrChangeNotFound
		}
	}

	change, err := readChange(fsys, changePath, name, archived)
	if err != nil {
		return nil, err
	}

	artifactPaths, err := MarkdownArtifactPaths(fsys, changePath)
	if err != nil {
		return nil, fmt.Errorf("listing artifacts for %s: %w", name, err)
	}

	artifacts := make([]Artifact, 0, len(artifactPaths))
	for _, artifactPath := range artifactPaths {
		artifact, err := readArtifact(fsys, changePath, artifactPath)
		if err != nil {
			return nil, fmt.Errorf("reading artifact %s: %w", artifactPath, err)
		}
		artifacts = append(artifacts, artifact)
	}

	return &ChangeDetail{Change: change, ArtifactFiles: artifacts}, nil
}

// MarkdownArtifactPaths returns every markdown artifact discovered beneath a
// change path. The returned paths are relative to fsys, not to changePath.
func MarkdownArtifactPaths(fsys fs.FS, changePath string) ([]string, error) {
	var paths []string
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		candidate := path.Join(changePath, name)
		if fileExists(fsys, candidate) {
			paths = append(paths, candidate)
		}
	}

	specsPath := path.Join(changePath, "specs")
	err := fs.WalkDir(fsys, specsPath, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if !entry.IsDir() && path.Ext(entry.Name()) == ".md" {
			paths = append(paths, name)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}

func readArtifact(fsys fs.FS, changePath, artifactPath string) (Artifact, error) {
	content, err := fs.ReadFile(fsys, artifactPath)
	if err != nil {
		return Artifact{}, err
	}
	info, err := fs.Stat(fsys, artifactPath)
	if err != nil {
		return Artifact{}, err
	}

	sum := sha256.Sum256(content)
	relativePath := artifactPath[len(changePath)+1:]
	kind := artifactKind(relativePath)

	return Artifact{
		Kind:    kind,
		Path:    relativePath,
		Content: string(content),
		Version: FileVersion{ModTime: info.ModTime(), Hash: hex.EncodeToString(sum[:])},
	}, nil
}

func artifactKind(relativePath string) string {
	if path.Dir(relativePath) == "." {
		return relativePath[:len(relativePath)-len(path.Ext(relativePath))]
	}
	return "spec"
}
