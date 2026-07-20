// Package directorybrowser provides read-only navigation of local directories.
package directorybrowser

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	// ErrPathNotAbsolute is returned when a requested path is not absolute.
	ErrPathNotAbsolute = errors.New("directory path must be absolute")
	// ErrNotDirectory is returned when a requested path names a file.
	ErrNotDirectory = errors.New("path is not a directory")
)

// Directory is one navigable child directory.
type Directory struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Location is a platform-appropriate navigation shortcut.
type Location struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Listing describes the current directory and its navigation options.
type Listing struct {
	Path        string      `json:"path"`
	ParentPath  string      `json:"parentPath,omitempty"`
	Directories []Directory `json:"directories"`
	Locations   []Location  `json:"locations"`
}

// Browser lists local directories without modifying them.
type Browser struct {
	homeDir      func() (string, error)
	readDir      func(string) ([]os.DirEntry, error)
	stat         func(string) (fs.FileInfo, error)
	evalSymlinks func(string) (string, error)
	locations    func(string) ([]Location, error)
}

// New constructs a Browser backed by the operating system filesystem.
func New() *Browser {
	return &Browser{
		homeDir:      os.UserHomeDir,
		readDir:      os.ReadDir,
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		locations:    platformLocations,
	}
}

// List returns the requested directory's child directories. An empty path
// starts in the current user's home directory.
func (b *Browser) List(requestedPath string) (Listing, error) {
	path := requestedPath
	if path == "" {
		var err error
		path, err = b.homeDir()
		if err != nil {
			return Listing{}, err
		}
	}
	if !filepath.IsAbs(path) {
		return Listing{}, ErrPathNotAbsolute
	}

	canonicalPath, err := b.evalSymlinks(filepath.Clean(path))
	if err != nil {
		return Listing{}, err
	}
	canonicalPath, err = filepath.Abs(canonicalPath)
	if err != nil {
		return Listing{}, err
	}
	info, err := b.stat(canonicalPath)
	if err != nil {
		return Listing{}, err
	}
	if !info.IsDir() {
		return Listing{}, ErrNotDirectory
	}

	entries, err := b.readDir(canonicalPath)
	if err != nil {
		return Listing{}, err
	}
	directories := make([]Directory, 0, len(entries))
	for _, entry := range entries {
		isDirectory := entry.IsDir()
		if !isDirectory && entry.Type()&os.ModeSymlink != 0 {
			target, statErr := b.stat(filepath.Join(canonicalPath, entry.Name()))
			isDirectory = statErr == nil && target.IsDir()
		}
		if isDirectory {
			directories = append(directories, Directory{
				Name: entry.Name(),
				Path: filepath.Join(canonicalPath, entry.Name()),
			})
		}
	}
	sort.Slice(directories, func(i, j int) bool {
		left, right := strings.ToLower(directories[i].Name), strings.ToLower(directories[j].Name)
		if left == right {
			return directories[i].Name < directories[j].Name
		}
		return left < right
	})

	home, err := b.homeDir()
	if err != nil {
		return Listing{}, err
	}
	locations, err := b.locations(home)
	if err != nil {
		return Listing{}, err
	}
	locations = deduplicateLocations(locations)

	parentPath := filepath.Dir(canonicalPath)
	if parentPath == canonicalPath {
		parentPath = ""
	}
	return Listing{
		Path:        canonicalPath,
		ParentPath:  parentPath,
		Directories: directories,
		Locations:   locations,
	}, nil
}

func deduplicateLocations(locations []Location) []Location {
	result := make([]Location, 0, len(locations))
	seen := make(map[string]struct{}, len(locations))
	for _, location := range locations {
		if location.Path == "" {
			continue
		}
		path := filepath.Clean(location.Path)
		key := strings.ToLower(path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		location.Path = path
		result = append(result, location)
	}
	return result
}
