package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"storyboard/internal/directorybrowser"
	"storyboard/internal/openspec"
	projectregistry "storyboard/internal/registry"
)

// Server serves the registry-backed Storyboard API and its live project updates.
type Server struct {
	logger      *slog.Logger
	events      *eventHub
	registry    *projectregistry.Store
	watchMu     sync.Mutex
	watchCtx    context.Context
	watchers    map[string]*projectWatcher
	directories directoryLister
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
	return &Server{
		logger: logger, events: newEventHub(), registry: store,
		watchers: make(map[string]*projectWatcher), directories: directorybrowser.New(),
	}, nil
}

// Handler returns the complete HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/projects", s.handleProjects)
	mux.HandleFunc("POST /api/projects", s.handleAddProject)
	mux.HandleFunc("DELETE /api/projects/{projectID}", s.handleRemoveProject)
	mux.HandleFunc("GET /api/filesystem/directories", s.handleDirectories)
	mux.HandleFunc("GET /api/projects/{projectID}/changes/{name}", s.handleRegisteredChangeDetail)
	mux.HandleFunc("POST /api/projects/{projectID}/changes/{name}/tasks/{id}/toggle", s.handleRegisteredTaskToggle)
	mux.HandleFunc("PUT /api/projects/{projectID}/changes/{name}/tasks/{id}/text", s.handleRegisteredTaskText)
	mux.HandleFunc("PUT /api/projects/{projectID}/changes/{name}/artifacts/proposal", s.handleRegisteredProposalText)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.Handle("/", spaHandler())
	return restrictToLoopback(s.logRequests(mux))
}

func restrictToLoopback(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostname, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			hostname = r.Host
		}

		switch strings.ToLower(hostname) {
		case "127.0.0.1", "localhost", "::1":
			if r.Header.Get("Sec-Fetch-Site") != "cross-site" {
				next.ServeHTTP(w, r)
				return
			}
		}

		writeAPIError(w, http.StatusForbidden, "forbidden_host", "requests must come from a local browser context")
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
