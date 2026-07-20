package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"storyboard/internal/openspec"
	projectregistry "storyboard/internal/registry"
)

// Server serves the read-only Storyboard API for one project. The registry
// phase will replace this single-project stepping stone with dynamic projects.
type Server struct {
	projectRoot string
	projectFS   fs.FS
	writeRoot   string
	logger      *slog.Logger
	events      *eventHub
	registry    *projectregistry.Store
	watchMu     sync.Mutex
	watchCtx    context.Context
	watchers    map[string]*projectWatcher
}

// New constructs a read-only API server rooted at projectRoot.
func New(projectRoot string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	store := projectregistry.NewMemory()
	if project, err := projectregistry.ProjectForPath(projectRoot); err == nil {
		store = projectregistry.NewMemory(project)
	}
	return &Server{
		projectRoot: projectRoot,
		projectFS:   os.DirFS(projectRoot),
		writeRoot:   projectRoot,
		logger:      logger,
		events:      newEventHub(),
		registry:    store,
		watchers:    make(map[string]*projectWatcher),
	}
}

// NewPersistent constructs the production server backed by a registry file.
// initialProject, when non-empty, is validated and registered before serving.
func NewPersistent(configPath, initialProject string, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}
	store, err := projectregistry.Open(configPath)
	if err != nil {
		return nil, err
	}
	if initialProject != "" {
		if _, err := store.Add(initialProject); err != nil {
			return nil, fmt.Errorf("registering initial project: %w", err)
		}
	}
	return &Server{logger: logger, events: newEventHub(), registry: store, watchers: make(map[string]*projectWatcher)}, nil
}

// NewWithFS is like New, but accepts an fs.FS for focused tests.
func NewWithFS(projectRoot string, projectFS fs.FS, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		projectRoot: projectRoot, projectFS: projectFS, logger: logger,
		events: newEventHub(), registry: projectregistry.NewMemory(), watchers: make(map[string]*projectWatcher),
	}
}

// Handler returns the complete HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/projects/current", s.handleCurrentProject)
	mux.HandleFunc("GET /api/projects", s.handleProjects)
	mux.HandleFunc("POST /api/projects", s.handleAddProject)
	mux.HandleFunc("DELETE /api/projects/{projectID}", s.handleRemoveProject)
	mux.HandleFunc("GET /api/projects/{projectID}/changes/{name}", s.handleRegisteredChangeDetail)
	mux.HandleFunc("POST /api/projects/{projectID}/changes/{name}/tasks/{id}/toggle", s.handleRegisteredTaskToggle)
	mux.HandleFunc("PUT /api/projects/{projectID}/changes/{name}/tasks/{id}/text", s.handleRegisteredTaskText)
	mux.HandleFunc("PUT /api/projects/{projectID}/changes/{name}/artifacts/proposal", s.handleRegisteredProposalText)
	mux.HandleFunc("GET /api/changes/{name}", s.handleChangeDetail)
	mux.HandleFunc("POST /api/changes/{name}/tasks/{id}/toggle", s.handleTaskToggle)
	mux.HandleFunc("PUT /api/changes/{name}/tasks/{id}/text", s.handleTaskText)
	mux.HandleFunc("PUT /api/changes/{name}/artifacts/proposal", s.handleProposalText)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	return s.logRequests(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCurrentProject(w http.ResponseWriter, _ *http.Request) {
	project, err := openspec.Discover(s.projectFS, s.projectRoot)
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) handleChangeDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := openspec.ReadChangeDetail(s.projectFS, r.PathValue("name"))
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

type toggleTaskRequest struct {
	Version openspec.FileVersion `json:"version"`
}

func (s *Server) handleTaskToggle(w http.ResponseWriter, r *http.Request) {
	if s.writeRoot == "" {
		writeAPIError(w, http.StatusInternalServerError, "writes_unavailable", "writes are unavailable for this project source")
		return
	}
	var request toggleTaskRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := openspec.ToggleTaskFile(s.writeRoot, r.PathValue("name"), r.PathValue("id"), request.Version)
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type taskTextRequest struct {
	Text    string               `json:"text"`
	Version openspec.FileVersion `json:"version"`
}

func (s *Server) handleTaskText(w http.ResponseWriter, r *http.Request) {
	if s.writeRoot == "" {
		writeAPIError(w, http.StatusInternalServerError, "writes_unavailable", "writes are unavailable for this project source")
		return
	}
	var request taskTextRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := openspec.UpdateTaskTextFile(
		s.writeRoot, r.PathValue("name"), r.PathValue("id"), request.Text, request.Version,
	)
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type proposalTextRequest struct {
	Content string               `json:"content"`
	Version openspec.FileVersion `json:"version"`
}

func (s *Server) handleProposalText(w http.ResponseWriter, r *http.Request) {
	if s.writeRoot == "" {
		writeAPIError(w, http.StatusInternalServerError, "writes_unavailable", "writes are unavailable for this project source")
		return
	}
	var request proposalTextRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := openspec.SaveProposalFile(s.writeRoot, r.PathValue("name"), request.Content, request.Version)
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) writeReadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, openspec.ErrChangeNotFound):
		writeAPIError(w, http.StatusNotFound, "change_not_found", err.Error())
	case errors.Is(err, openspec.ErrNotOpenSpecProject):
		writeAPIError(w, http.StatusUnprocessableEntity, "not_openspec_project", err.Error())
	case errors.Is(err, openspec.ErrTaskNotFound):
		writeAPIError(w, http.StatusNotFound, "task_not_found", err.Error())
	case errors.Is(err, openspec.ErrArtifactNotFound):
		writeAPIError(w, http.StatusNotFound, "artifact_not_found", err.Error())
	case errors.Is(err, openspec.ErrConflict):
		writeAPIError(w, http.StatusConflict, "file_conflict", err.Error())
	case errors.Is(err, openspec.ErrInvalidTaskLine):
		writeAPIError(w, http.StatusUnprocessableEntity, "invalid_task_line", err.Error())
	case errors.Is(err, openspec.ErrInvalidTaskText):
		writeAPIError(w, http.StatusUnprocessableEntity, "invalid_task_text", err.Error())
	default:
		s.logger.Error("reading OpenSpec project", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "read_failed", "could not read the OpenSpec project")
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		s.logger.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration", time.Since(started),
		)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("invalid JSON: request must contain one object")
	}
	return nil
}
