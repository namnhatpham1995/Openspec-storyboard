package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidProject = errors.New("project path is not a valid OpenSpec project")
	ErrNotFound       = errors.New("registered project not found")
)

// Project is one persisted project folder. ID is stable for its absolute path.
type Project struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type config struct {
	Projects []Project `json:"projects"`
	Recent   []string  `json:"recent"`
}

// Store serializes project registry mutations to one JSON config file.
type Store struct {
	mu     sync.RWMutex
	path   string
	config config
}

// DefaultPath returns the platform-native Storyboard registry path.
func DefaultPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "storyboard", "config.json"), nil
}

// Open loads a registry. Invalid JSON is moved aside and replaced with a
// valid empty registry so one damaged preference file cannot prevent startup.
func Open(name string) (*Store, error) {
	store := &Store{path: name}
	content, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading project registry: %w", err)
	}
	if err := json.Unmarshal(content, &store.config); err != nil {
		backup := name + ".corrupt-" + time.Now().UTC().Format("20060102T150405.000000000Z")
		if renameErr := os.Rename(name, backup); renameErr != nil {
			return nil, fmt.Errorf("backing up corrupt project registry: %w", renameErr)
		}
		if saveErr := store.saveLocked(); saveErr != nil {
			return nil, fmt.Errorf("resetting corrupt project registry: %w", saveErr)
		}
	}
	return store, nil
}

// NewMemory creates a non-persisted store for focused server tests.
func NewMemory(projects ...Project) *Store {
	return &Store{config: config{Projects: append([]Project(nil), projects...)}}
}

// List returns a snapshot of the registered entries, including disconnected
// paths so the UI can explain and remove them.
func (s *Store) List() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Project(nil), s.config.Projects...)
}

// Add validates and registers an OpenSpec project, deduplicated by path.
func (s *Store) Add(path string) (Project, error) {
	project, err := ProjectForPath(path)
	if err != nil {
		return Project{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.config.Projects {
		if samePath(existing.Path, project.Path) {
			s.markRecentLocked(existing.ID)
			if err := s.saveLocked(); err != nil {
				return Project{}, err
			}
			return existing, nil
		}
	}
	s.config.Projects = append(s.config.Projects, project)
	s.markRecentLocked(project.ID)
	if err := s.saveLocked(); err != nil {
		return Project{}, err
	}
	return project, nil
}

// Remove deletes one registry entry and its recent-project reference.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i, project := range s.config.Projects {
		if project.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return ErrNotFound
	}
	s.config.Projects = append(s.config.Projects[:index], s.config.Projects[index+1:]...)
	recent := s.config.Recent[:0]
	for _, recentID := range s.config.Recent {
		if recentID != id {
			recent = append(recent, recentID)
		}
	}
	s.config.Recent = recent
	return s.saveLocked()
}

// Get finds a project by stable id.
func (s *Store) Get(id string) (Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, project := range s.config.Projects {
		if project.ID == id {
			return project, true
		}
	}
	return Project{}, false
}

// ProjectForPath canonicalizes and validates one project path.
func ProjectForPath(path string) (Project, error) {
	absolute, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil || strings.TrimSpace(path) == "" {
		return Project{}, ErrInvalidProject
	}
	absolute = filepath.Clean(absolute)
	rootInfo, err := os.Stat(absolute)
	if err != nil || !rootInfo.IsDir() {
		return Project{}, ErrInvalidProject
	}
	openspecInfo, err := os.Stat(filepath.Join(absolute, "openspec"))
	if err != nil || !openspecInfo.IsDir() {
		return Project{}, ErrInvalidProject
	}
	sum := sha256.Sum256([]byte(normalizePath(absolute)))
	return Project{ID: hex.EncodeToString(sum[:8]), Path: absolute, Name: filepath.Base(absolute)}, nil
}

func (s *Store) markRecentLocked(id string) {
	recent := []string{id}
	for _, existing := range s.config.Recent {
		if existing != id {
			recent = append(recent, existing)
		}
	}
	if len(recent) > 10 {
		recent = recent[:10]
	}
	s.config.Recent = recent
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("creating project registry directory: %w", err)
	}
	content, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".storyboard-config-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, s.path); err != nil {
		return fmt.Errorf("replacing project registry: %w", err)
	}
	return nil
}

func samePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func normalizePath(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}
